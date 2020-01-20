package terraform

import (
	"errors"
	"io"
	"strconv"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/containers"
)

// createTerraformInitContainer creates and returns a container for running terraform init to download providers and modules.
func createTerraformInitContainer(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume) (*docker.Container, error) {
	return dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: "terraform",
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

type terraformContainer struct {
	dockerClient *docker.Client
	container    *docker.Container
}

func NewTerraformContainer(dockerClient *docker.Client, image, codeDir string, releaseVolume *docker.Volume) (*terraformContainer, error) {

	container, err := dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: "terraform",
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

	if err := dockerClient.StartContainer(container.ID, nil); err != nil {
		return nil, err
	}

	self := terraformContainer{
		dockerClient: dockerClient,
		container:    container,
	}

	self.container = container
	return &self, nil
}

type BackendConfigParameter struct {
	Key   string
	Value string
}

// ConfigureBackend runs terraform init as part of the release in order to download providers and modules.
func (self *terraformContainer) ConfigureBackend(outputStream, errorStream io.Writer, backendConfig []BackendConfigParameter) error {

	command := make([]string, 0)
	command = append(command, "terraform")
	command = append(command, "init")
	command = append(command, "-get=false")
	command = append(command, "-get-plugins=false")

	for _, param := range backendConfig {
		command = append(command, "-backend-config="+param.Key+"="+param.Value)
	}

	exec, err := self.dockerClient.CreateExec(docker.CreateExecOptions{
		Container:    self.container.ID,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          command,
	})
	if err != nil {
		return err
	}

	if err := self.dockerClient.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: outputStream,
		ErrorStream:  errorStream,
	}); err != nil {
		return err
	}

	return nil
}

// Done stops and removes the terraform container.
func (self *terraformContainer) Done() error {
	if err := self.dockerClient.StopContainer(self.container.ID, 10); err != nil {
		return err
	}
	return self.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID: self.container.ID,
	})
}
