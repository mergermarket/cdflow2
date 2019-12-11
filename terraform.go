package main

import (
	"errors"
	"io"

	docker "github.com/fsouza/go-dockerclient"
)

func terraformInit(dockerClient *docker.Client, image, codeDir, buildDir string, outputStream, errorStream io.Writer) error {
	container, err := createTerraformContainer(dockerClient, image, codeDir, buildDir)
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

func createTerraformContainer(dockerClient *docker.Client, image, codeDir, buildDir string) (*docker.Container, error) {
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
			Binds:     []string{codeDir + ":/code:ro", buildDir + ":/build"},
		},
	})
}
