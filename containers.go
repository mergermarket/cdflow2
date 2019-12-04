package main

import (
	"io"

	docker "github.com/fsouza/go-dockerclient"
)

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
