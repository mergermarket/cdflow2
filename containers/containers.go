package containers

import (
	"fmt"
	"io"
	"math/rand"

	docker "github.com/fsouza/go-dockerclient"
)

func EnsureImage(dockerClient *docker.Client, image string) error {
	if _, err := dockerClient.InspectImage(image); err == nil {
		return nil
	}
	return dockerClient.PullImage(docker.PullImageOptions{
		Repository: image,
	}, docker.AuthConfiguration{})
}

func Await(dockerClient *docker.Client, container *docker.Container, inputStream io.Reader, outputStream, errorStream io.Writer, started chan error) error {
	attached := make(chan error)
	detached := make(chan error)
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
		detached <- waiter.Wait()
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

	return <-detached
}

// RandomName creates a random name with a prefix so container names don't clash.
func RandomName(prefix string) string {
	return fmt.Sprintf("%s-%x", prefix, rand.Int())
}
