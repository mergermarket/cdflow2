package terraform

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mergermarket/cdflow2/docker"
)

// InitInitial runs terraform init as part of the release in order to download providers and modules.
func InitInitial(dockerClient docker.Iface, image, codeDir string, buildVolume string, outputStream, errorStream io.Writer) error {
	return dockerClient.Run(&docker.RunOptions{
		Image:        image,
		WorkingDir:   "/build",
		Cmd:          []string{"init", "/code/infra"},
		Env:          []string{"TF_IN_AUTOMATION=true"},
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
			OutputStream:  &outputBuffer,
			ErrorStream:   &outputBuffer,
			WorkingDir:    "/code",
			Entrypoint:    []string{"/bin/sleep"},
			Cmd:           []string{strconv.Itoa(365 * 24 * 60 * 60)}, // a long time!
			Env:           []string{"TF_IN_AUTOMATION=true"},
			Started:       started,
			Init:          true,
			NamePrefix:    "cdflow2-terraform",
			Binds:         []string{codeDir + ":/code:ro", releaseVolume + ":/release"},
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

// BackendConfigParameter is like a tuple of key and value for use in a slice (rather than a map as that wouldn't preserve order).
type BackendConfigParameter struct {
	Key   string
	Value string
}

// ConfigureBackend runs terraform init as part of the release in order to download providers and modules.
func (terraformContainer *Container) ConfigureBackend(outputStream, errorStream io.Writer, backendConfig []BackendConfigParameter) error {
	command := make([]string, 0)
	command = append(command, "terraform")
	command = append(command, "init")
	command = append(command, "-get=false")
	command = append(command, "-get-plugins=false")

	for _, param := range backendConfig {
		command = append(command, "-backend-config="+param.Key+"="+param.Value)
	}

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

	if err := terraformContainer.RunCommand([]string{"terraform", "workspace", command, name}, outputStream, errorStream); err != nil {
		return err
	}

	return nil
}

// listWorkspaces lists the terraform workspaces and returns them as a set
func (terraformContainer *Container) listWorkspaces(errorStream io.Writer) (map[string]bool, error) {
	var outputBuffer bytes.Buffer

	if err := terraformContainer.RunCommand([]string{"terraform", "workspace", "list"}, &outputBuffer, errorStream); err != nil {
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
