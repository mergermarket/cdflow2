package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	docker "github.com/fsouza/go-dockerclient"
)

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

/*
	cdflow release:
		terraform container: terraform init (no config container dependency yet)
		config container: get environment for artefact build & publish (e.g. aws creds for docker push)
		release container: do build and publish
		config container: upload release archive

*/
func main() {
	fmt.Println("hello world")
}

func terraformInit(dockerClient *docker.Client, image, codeDir, buildDir string, outputStream, errorStream io.Writer) error {
	container, err := createTerraformContainer(dockerClient, image, codeDir, buildDir)
	if err != nil {
		return err
	}

	if err := awaitContainer(dockerClient, container, outputStream, errorStream); err != nil {
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
		return errors.New("container failed")
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

func awaitContainer(dockerClient *docker.Client, container *docker.Container, outputStream io.Writer, errorStream io.Writer) error {
	attached := make(chan error)
	finished := make(chan error)
	go func() {
		waiter, err := dockerClient.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
			Container:    container.ID,
			OutputStream: outputStream,
			ErrorStream:  errorStream,
			Stream:       true,
			Stdout:       true,
			Stderr:       true,
			Stdin:        false,
		})
		attached <- err
		if err != nil {
			return
		}
		finished <- waiter.Wait()
	}()

	if err := <-attached; err != nil {
		return err
	}

	if err := dockerClient.StartContainer(container.ID, nil); err != nil {
		return err
	}

	if err := <-finished; err != nil {
		return err
	}
	return nil
}
