package terraform

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mergermarket/cdflow2/docker"
)

// InitInitial runs terraform init as part of the release in order to download providers and modules.
func InitInitial(dockerClient docker.Iface, image, codeDir string, buildVolume string, outputStream, errorStream io.Writer) error {

	fmt.Fprintf(
		errorStream,
		"\nInitialising terraform...\n\n$ %v\n\n",
		"terraform init -backend=false infra/",
	)

	return dockerClient.Run(&docker.RunOptions{
		Image:        image,
		WorkingDir:   "/code",
		Cmd:          []string{"init", "-backend=false", "infra/"},
		Env:          []string{"TF_IN_AUTOMATION=true", "TF_INPUT=0", "TF_DATA_DIR=/build/.terraform"},
		Binds:        []string{codeDir + ":/code:ro", buildVolume + ":/build"},
		NamePrefix:   "cdflow2-terraform-init",
		OutputStream: outputStream,
		ErrorStream:  errorStream,
	})
}

// Container stores information about a running terraform container for running terraform commands in.
type Container struct {
	dockerClient docker.Iface
	id           string
	done         chan error
}

// NewContainer creates and returns a terraformContainer for running terraform commands in.
func NewContainer(dockerClient docker.Iface, image, codeDir string, releaseVolume string) (*Container, error) {

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
			WorkingDir:   "/code",
			Entrypoint:   []string{"/bin/sleep"},
			Cmd:          []string{strconv.Itoa(365 * 24 * 60 * 60)}, // a long time!
			Env:          []string{"TF_IN_AUTOMATION=true", "TF_INPUT=0", "TF_DATA_DIR=/build/.terraform"},
			Started:      started,
			Init:         true,
			NamePrefix:   "cdflow2-terraform",
			Binds: []string{
				codeDir + ":/code:ro",
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
		}, nil
	case err := <-done:
		return nil, fmt.Errorf("could not start terraform container: %w\nOutput: %v", err, outputBuffer.String())
	}
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

// ConfigureBackend runs terraform init as part of the release in order to download providers and modules.
func (terraformContainer *Container) ConfigureBackend(outputStream, errorStream io.Writer, backendConfig map[string]string) error {
	command := make([]string, 0)
	command = append(command, "terraform")
	command = append(command, "init")
	command = append(command, "-get=false")
	command = append(command, "-get-plugins=false")

	for _, pair := range DictToSortedPairs(backendConfig) {
		command = append(command, "-backend-config="+pair.Key+"="+pair.Value)
	}

	command = append(command, "infra/")

	fmt.Fprintf(
		errorStream,
		"\nConfiguring terraform backend...\n\n$ %v\n",
		//strings.Join(command, " "),
		"terraform init -get=false -get-plugins=false -backend-config=... -backend-config=... infra/",
	)

	if err := terraformContainer.RunCommand(command, outputStream, errorStream); err != nil {
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
		"\nSwitching workspace...\n\n$ terraform workspace %s %s\n",
		command, name,
	)

	if err := terraformContainer.RunCommand([]string{"terraform", "workspace", command, name, "infra/"}, outputStream, errorStream); err != nil {
		return err
	}

	return nil
}

// listWorkspaces lists the terraform workspaces and returns them as a set
func (terraformContainer *Container) listWorkspaces(errorStream io.Writer) (map[string]bool, error) {
	var outputBuffer bytes.Buffer

	fmt.Fprintf(
		errorStream,
		"\nListing workspaces...\n\n$ terraform workspace list infra/\n",
	)

	if err := terraformContainer.RunCommand([]string{"terraform", "workspace", "list", "infra/"}, &outputBuffer, errorStream); err != nil {
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

// RunCommand execs a command inside the terraform container.
func (terraformContainer *Container) RunCommand(cmd []string, outputStream, errorStream io.Writer) error {
	return terraformContainer.dockerClient.Exec(&docker.ExecOptions{
		ID:           terraformContainer.id,
		Cmd:          cmd,
		OutputStream: outputStream,
		ErrorStream:  errorStream,
	})
}

// Done stops and removes the terraform container.
func (terraformContainer *Container) Done() error {
	if err := terraformContainer.dockerClient.Stop(terraformContainer.id, 10*time.Second); err != nil {
		return err
	}
	return <-terraformContainer.done
}
