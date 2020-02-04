package terraform

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/containers"
)

// createTerraformInitContainer creates and returns a container for running terraform init to download providers and modules.
func createTerraformInitContainer(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume) (*docker.Container, error) {
	return dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: containers.RandomName("cdflow2-terraform"),
		Config: &docker.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/build",
			Cmd:          []string{"init", "/code/infra"},
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{Type: "none"},
			Binds:     []string{codeDir + ":/code:ro", buildVolume.Name + ":/build"},
		},
	})
}

// InitInitial runs terraform init as part of the release in order to download providers and modules.
func InitInitial(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, outputStream, errorStream io.Writer) error {
	container, err := createTerraformInitContainer(dockerClient, image, codeDir, buildVolume)
	if err != nil {
		return err
	}

	if err := containers.Await(dockerClient, container, nil, outputStream, errorStream, nil); err != nil {
		return err
	}

	props, err := dockerClient.InspectContainer(container.ID)
	if err != nil {
		return err
	}

	if props.State.Running {
		panic("unexpected: terraform container still running")
	}
	if err := dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID}); err != nil {
		return err
	}
	if props.State.ExitCode != 0 {
		return errors.New("terraform container failed")
	}
	return nil
}

// Container stores information about a running terraform container for running terraform commands in.
type Container struct {
	dockerClient *docker.Client
	container    *docker.Container
}

// NewContainer creates and returns a terraformContainer for running terraform commands in.
func NewContainer(dockerClient *docker.Client, image, codeDir string, releaseVolume *docker.Volume) (*Container, error) {

	dockerContainer, err := dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: containers.RandomName("cdflow2-terraform"),
		Config: &docker.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/code",
			Entrypoint:   []string{"/bin/sleep"},
			Cmd:          []string{strconv.Itoa(365 * 24 * 60 * 60)}, // a long time!
		},
		HostConfig: &docker.HostConfig{
			Init:      true,
			LogConfig: docker.LogConfig{Type: "none"},
			Binds:     []string{codeDir + ":/code:ro", releaseVolume.Name + ":/release"},
		},
	})
	if err != nil {
		return nil, err
	}

	if err := dockerClient.StartContainer(dockerContainer.ID, nil); err != nil {
		return nil, err
	}

	return &Container{
		dockerClient: dockerClient,
		container:    dockerContainer,
	}, nil
}

// BackendConfigParameter is like a tuple of key and value for use in a slice (rather than a map as that wouldn't preserve order).
type BackendConfigParameter struct {
	Key   string
	Value string
}

// ConfigureBackend runs terraform init as part of the release in order to download providers and modules.
func (container *Container) ConfigureBackend(outputStream, errorStream io.Writer, backendConfig []BackendConfigParameter) error {

	command := make([]string, 0)
	command = append(command, "terraform")
	command = append(command, "init")
	command = append(command, "-get=false")
	command = append(command, "-get-plugins=false")

	for _, param := range backendConfig {
		command = append(command, "-backend-config="+param.Key+"="+param.Value)
	}

	if err := container.runCommand(command, outputStream, errorStream); err != nil {
		return err
	}

	return nil
}

// SwitchWorkspace switched to a named workspace, creating it if necessary.
func (container *Container) SwitchWorkspace(name string, outputStream, errorStream io.Writer) error {
	workspaces, err := container.listWorkspaces(errorStream)
	if err != nil {
		return err
	}

	command := "new"
	if workspaces[name] {
		command = "select"
	}

	if err := container.runCommand([]string{"terraform", "workspace", command, name}, outputStream, errorStream); err != nil {
		return err
	}

	return nil
}

// listWorkspaces lists the terraform workspaces and returns them as a set
func (container *Container) listWorkspaces(errorStream io.Writer) (map[string]bool, error) {
	var outputBuffer bytes.Buffer

	if err := container.runCommand([]string{"terraform", "workspace", "list"}, &outputBuffer, errorStream); err != nil {
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

// runCommand execs a command inside the terraform container.
func (container *Container) runCommand(command []string, outputStream, errorStream io.Writer) error {
	exec, err := container.dockerClient.CreateExec(docker.CreateExecOptions{
		Container:    container.container.ID,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          command,
	})
	if err != nil {
		return err
	}

	if err := container.dockerClient.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: outputStream,
		ErrorStream:  errorStream,
	}); err != nil {
		return err
	}
	return nil
}

// Done stops and removes the terraform container.
func (container *Container) Done() error {
	if err := container.dockerClient.StopContainer(container.container.ID, 10); err != nil {
		return err
	}
	return container.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.container.ID,
	})
}
