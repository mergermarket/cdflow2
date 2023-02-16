package terraform

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/docker"
	"github.com/mergermarket/cdflow2/util"
)

const (
	terraformDataDir = "/build/.terraform"
)

// Container stores information about a running terraform container for running terraform commands in.
type Container struct {
	dockerClient docker.Iface
	id           string
	done         chan error
	codeDir      string
}

// NewContainer creates and returns a terraformContainer for running terraform commands in.
func NewContainer(dockerClient docker.Iface, image, codeDir string, releaseVolume string) (*Container, error) {
	infraDir := filepath.Join(codeDir, "infra")
	if _, err := os.Stat(infraDir); err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(infraDir, 0755)
			if err != nil {
				return nil, fmt.Errorf("unable to create infra directory: %v", err)
			}
		}

		return nil, fmt.Errorf("unable to check infra directory existence: %v", err)
	}

	started := make(chan string, 1)
	defer close(started)

	done := make(chan error, 1)

	var outputBuffer bytes.Buffer

	go func() {
		done <- dockerClient.Run(&docker.RunOptions{
			Image: image,
			// output to user in case there's an error (e.g. terraform container doesn't have /bin/sleep)
			OutputStream: &outputBuffer,
			ErrorStream:  &outputBuffer,
			WorkingDir:   "/code/infra",
			Entrypoint:   []string{"/bin/sleep"},
			Cmd:          []string{strconv.Itoa(365 * 24 * 60 * 60)}, // a long time!
			Env:          []string{"TF_IN_AUTOMATION=true", "TF_INPUT=0", "TF_DATA_DIR=" + terraformDataDir},
			Started:      started,
			Init:         true,
			NamePrefix:   "cdflow2-terraform",
			Binds: []string{
				codeDir + ":/code",
				releaseVolume + ":/build",
			},
			SuccessStatus: 128 + 15, // sleep will be killed with SIGTERM
		})
	}()

	select {
	case id := <-started:
		return &Container{
			dockerClient: dockerClient,
			id:           id,
			done:         done,
			codeDir:      codeDir,
		}, nil
	case err := <-done:
		return nil, fmt.Errorf("could not start terraform container: %w\nOutput: %v", err, outputBuffer.String())
	}
}

// NamedTerrafromBackendConfigParameter is a terraform backend config parameter with a name.
type NamedTerrafromBackendConfigParameter struct {
	Name      string
	Parameter *config.TerrafromBackendConfigParameter
}

// SortTerraformBackendConfigParameters sorts a map of terraform backend config parameters.
func SortTerraformBackendConfigParameters(input map[string]*config.TerrafromBackendConfigParameter) []NamedTerrafromBackendConfigParameter {
	var names []string
	for name := range input {
		names = append(names, name)
	}
	result := make([]NamedTerrafromBackendConfigParameter, len(input))
	sort.Strings(names)
	for i, name := range names {
		result[i].Name = name
		result[i].Parameter = input[name]
	}
	return result
}

// Pair is item in a map.
type Pair struct {
	Key   string
	Value string
}

// DictToSortedPairs takes a map of strings and returns a list of Pairs sorted by key (meh go).
func DictToSortedPairs(input map[string]string) []Pair {
	var keys []string
	for k := range input {
		keys = append(keys, k)
	}
	result := make([]Pair, len(input))
	sort.Strings(keys)
	for i, k := range keys {
		result[i].Key = k
		result[i].Value = input[k]
	}
	return result
}

const backendTemplate = `
/*
This is a partial backend configuration - see:

  https://www.terraform.io/docs/backends/config.html#partial-configuration

There's no need to add any additional configuration as this is provided by the
config container you are using. This file can safely be ignored or committed - run
the following from the project root to ignore it:

  echo backend.tf >> infra/.gitignore
  git commit -m 'ignore generated backend.tf file' infra/.gitignore

*/
terraform {
	backend "%s" {}
}
`

func (terraformContainer *Container) createPartialBackendConfig(codeDir, backendType string) error {
	infraDir := path.Join(codeDir, "infra")
	backendConfigFilepath := path.Join(infraDir, "backend.tf")
	_, err := os.Stat(backendConfigFilepath)
	if err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(infraDir, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(backendConfigFilepath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, backendTemplate, backendType); err != nil {
		return err
	}
	return nil
}

// InitInitial runs terraform init as part of the release in order to download providers and modules.
func (terraformContainer *Container) InitInitial(outputStream, errorStream io.Writer) error {
	fmt.Fprintf(
		errorStream,
		"\n%s\n%s\n\n",
		util.FormatInfo("initialising terraform"),
		util.FormatCommand("terraform init -backend=false"),
	)

	err := terraformContainer.RunCommand([]string{"terraform", "init", "-backend=false"}, nil, outputStream, errorStream)
	if err != nil {
		return err
	}

	return terraformContainer.removeProviders(errorStream)
}

func (terraformContainer *Container) removeProviders(errorStream io.Writer) error {
	var output bytes.Buffer

	err := terraformContainer.RunCommand([]string{"sh", "-c", "terraform version | head -n 1"}, nil, &output, errorStream)
	if err != nil {
		fmt.Fprintf(errorStream, "\n%s\n\n", util.FormatInfo(fmt.Sprintf("unable to run terraform version command: %v", err)))
		// keep providers just to be sure when can't get version
		return nil
	}

	version := strings.TrimPrefix(output.String(), "Terraform")
	version = strings.TrimSpace(version)

	semver, ok := util.ParseSemver(version)
	if !ok {
		fmt.Fprintf(errorStream, "\n%s\n", util.FormatInfo(fmt.Sprintf("unable to parse Terraform version: %s", version)))
		// keep providers just to be sure when can't parse version
		return nil
	}

	if semver.Major == 0 && semver.Minor < 14 {
		if semver.Minor == 13 {
			fmt.Fprintf(errorStream, "\n\n%s\n\n",
				util.FormatWarning("WARNING! You're using terraform 0.13.x version, which is not recommended anymore for cdflow2.\n"+
					"Please upgrade to 0.14 or newer otherwise you will see increased build times when running 'release' command.\n"+
					"See this PR for details: https://github.com/mergermarket/cdflow2/pull/33."))
		}

		return nil
	}

	ok, err = terraformContainer.CheckFileExists("/code/infra/.terraform.lock.hcl", errorStream)
	if !ok {
		if err != nil {
			return err
		}

		return fmt.Errorf("terraform lock file not exists, terraform version: %s", version)
	}

	// 'plugins' the legacy cache path
	for _, p := range []string{"plugins", "providers"} {
		var outputBuffer bytes.Buffer

		fullPath := fmt.Sprintf("%s/%s", terraformDataDir, p)
		command := fmt.Sprintf("test -d %s && rm -rf %s || echo -n 'not found'", fullPath, fullPath)

		err := terraformContainer.RunCommand([]string{"sh", "-c", command}, nil, &outputBuffer, errorStream)
		if err != nil {
			return err
		}

		output := outputBuffer.String()
		if output != "" && output != "not found" {
			return fmt.Errorf("unable to remove '%s' directory, error: '%s'", fullPath, output)
		}
	}

	return nil
}

// ConfigureBackend runs terraform init as part of the release in order to download providers and modules.
func (terraformContainer *Container) ConfigureBackend(outputStream, errorStream io.Writer, terraformResponse *config.PrepareTerraformResponse, download bool) error {
	if err := terraformContainer.createPartialBackendConfig(terraformContainer.codeDir, terraformResponse.TerraformBackendType); err != nil {
		return err
	}

	command := make([]string, 0)
	command = append(command, "terraform")
	command = append(command, "init")
	if !download {
		command = append(command, "-get=false")
	}

	displayCommand := make([]string, len(command))
	copy(displayCommand, command)

	// the old style, to be removed
	for _, pair := range DictToSortedPairs(terraformResponse.TerraformBackendConfig) {
		command = append(command, "-backend-config="+pair.Key+"="+pair.Value)
		displayCommand = append(displayCommand, "-backend-config="+pair.Key+"=...")
	}

	for _, namedParameter := range SortTerraformBackendConfigParameters(terraformResponse.TerraformBackendConfigParameters) {
		format := "-backend-config=" + namedParameter.Name + "=%s"
		command = append(command, fmt.Sprintf(format, namedParameter.Parameter.Value))
		displayValue := namedParameter.Parameter.Value
		if namedParameter.Parameter.DisplayValue != "" {
			displayValue = "[" + namedParameter.Parameter.DisplayValue + "]"
		}
		displayCommand = append(displayCommand, fmt.Sprintf(format, displayValue))
	}

	fmt.Fprintf(
		errorStream,
		"\n%s\n%s\n",
		util.FormatInfo("configuring terraform backend"),
		strings.Join(displayCommand, " "),
	)

	if err := terraformContainer.RunCommand(command, map[string]string{}, outputStream, errorStream); err != nil {
		return err
	}

	return nil
}

// SwitchWorkspace switched to a named workspace, creating it if necessary.
func (terraformContainer *Container) SwitchWorkspace(name string, outputStream, errorStream io.Writer) error {
	workspaces, err := terraformContainer.listWorkspaces(errorStream)
	if err != nil {
		return err
	}

	command := "new"
	if workspaces[name] {
		command = "select"
	}

	fmt.Fprintf(
		errorStream,
		"\n%s\n%s\n\n",
		util.FormatInfo("switching workspace"),
		util.FormatCommand("terraform workspace "+command+" "+name),
	)

	if err := terraformContainer.RunCommand([]string{"terraform", "workspace", command, name}, map[string]string{}, outputStream, errorStream); err != nil {
		return err
	}

	return nil
}

// listWorkspaces lists the terraform workspaces and returns them as a set
func (terraformContainer *Container) listWorkspaces(errorStream io.Writer) (map[string]bool, error) {
	var outputBuffer bytes.Buffer

	fmt.Fprintf(
		errorStream,
		"\n%s\n%s\n",
		util.FormatInfo("listing workspaces"),
		util.FormatCommand("terraform workspace list"),
	)

	if err := terraformContainer.RunCommand([]string{"terraform", "workspace", "list"}, map[string]string{}, &outputBuffer, errorStream); err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, line := range strings.Split(outputBuffer.String(), "\n") {
		for _, word := range strings.Fields(line) {
			if word != "*" {
				result[word] = true
			}
		}
	}

	return result, nil
}

func (terraformContainer *Container) CopyTerraformLockIfExists(outputStream, errorStream io.Writer) error {
	lockExists, err := terraformContainer.CheckFileExists("/build/.terraform.lock.hcl", errorStream)
	if err != nil {
		return err
	}

	if !lockExists {
		fmt.Fprintf(
			errorStream,
			"\n%s\n%s\n",
			util.FormatInfo(".terraform.lock.hcl not found"),
			util.FormatCommand(""),
		)
		return nil
	}

	fmt.Fprintf(
		errorStream,
		"\n%s\n%s\n",
		util.FormatInfo("copying .terraform.lock.hcl from release"),
		util.FormatCommand("cp /build/.terraform.lock.hcl /code/infra/"),
	)

	if err := terraformContainer.RunCommand([]string{"cp", "/build/.terraform.lock.hcl", "/code/infra/"}, map[string]string{}, outputStream, errorStream); err != nil {
		return err
	}

	return nil
}

func (terraformContainer *Container) CheckFileExists(path string, errorStream io.Writer) (bool, error) {
	var outputBuffer bytes.Buffer
	command := fmt.Sprintf("test -f %s && echo exists || echo none", path)

	if err := terraformContainer.RunCommand([]string{"sh", "-c", command}, map[string]string{}, &outputBuffer, errorStream); err != nil {
		return false, err
	}

	if outputBuffer.String() == "exists\n" {
		return true, nil
	}

	return false, nil
}

// RunCommand execs a command inside the terraform container.
func (terraformContainer *Container) RunCommand(cmd []string, env map[string]string, outputStream, errorStream io.Writer) error {
	return terraformContainer.dockerClient.Exec(&docker.ExecOptions{
		ID:           terraformContainer.id,
		Cmd:          cmd,
		Env:          env,
		OutputStream: outputStream,
		ErrorStream:  errorStream,
		Tty:          false,
	})
}

// RunInteractiveCommand execs a command inside the terraform container.
func (terraformContainer *Container) RunInteractiveCommand(
	cmd []string,
	env map[string]string,
	inputStream io.Reader,
	outputStream,
	errorStream io.Writer,
	tty bool,
	interactive bool) error {
	return terraformContainer.dockerClient.Exec(&docker.ExecOptions{
		ID:           terraformContainer.id,
		Cmd:          cmd,
		Env:          env,
		InputStream:  inputStream,
		OutputStream: outputStream,
		ErrorStream:  errorStream,
		Tty:          tty,
		Interactive:  interactive,
	})
}

// Done stops and removes the terraform container.
func (terraformContainer *Container) Done() error {
	if err := terraformContainer.dockerClient.Stop(terraformContainer.id, 10*time.Second); err != nil {
		return err
	}
	return <-terraformContainer.done
}
