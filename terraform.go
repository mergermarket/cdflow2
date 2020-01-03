package main

import (
	"errors"
	"io"

	docker "github.com/fsouza/go-dockerclient"
)

// createTerraformContainer creates and returns a container for running terraform.
func createTerraformContainer(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume) (*docker.Container, error) {
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

// terraformInitInitial runs terraform init as part of the release in order to download providers and modules.
func terraformInitInitial(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, outputStream, errorStream io.Writer) error {
	container, err := createTerraformContainer(dockerClient, image, codeDir, buildVolume)
	if err != nil {
		return err
	}

	if err := awaitContainer(dockerClient, container, nil, outputStream, errorStream, nil); err != nil {
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

// terraformConfigureBackend runs terraform init as part of the release in order to download providers and modules.
func terraformConfigureBackend(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, outputStream, errorStream io.Writer) error {
	container, err := createTerraformContainer(dockerClient, image, codeDir, buildVolume)
	if err != nil {
		return err
	}

	if err := awaitContainer(dockerClient, container, nil, outputStream, errorStream, nil); err != nil {
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
