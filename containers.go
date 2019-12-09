package main

import (
	"io"

	docker "github.com/fsouza/go-dockerclient"
)

func awaitContainer(dockerClient *docker.Client, container *docker.Container, inputStream io.Reader, outputStream, errorStream io.Writer, started chan error) error {
	attached := make(chan error)
	finished := make(chan error)
	go func() {
		waiter, err := dockerClient.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
			Container:    container.ID,
			InputStream:  inputStream,
			OutputStream: outputStream,
			ErrorStream:  errorStream,
			Stream:       true,
			Stdout:       true,
			Stderr:       true,
			Stdin:        true,
		})
		attached <- err
		if err != nil {
			return
		}
		finished <- waiter.Wait()
	}()

	if err := <-attached; err != nil {
		if started != nil {
			started <- err
		}
		return err
	}

	if err := dockerClient.StartContainer(container.ID, nil); err != nil {
		if started != nil {
			started <- err
		}
		return err
	}
	if started != nil {
		started <- nil
	}

	if err := <-finished; err != nil {
		return err
	}
	return nil
}
