package containers

import (
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strings"
	"time"

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
	return fmt.Sprintf("%s-%x-%x", prefix, time.Now().UnixNano(), rand.Int())
}

// MapToDockerEnv converts from a map[string]string to the []string that docker expects (with key and value separated by an equals sign).
func MapToDockerEnv(input map[string]string) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	// sort to make it stable for testing
	sort.Strings(keys)
	var result []string
	for _, key := range keys {
		result = append(result, fmt.Sprintf("%s=%s", key, input[key]))
	}
	return result
}

// ImageWithTag takes a docker image and adds the :latest tag if there is no tag present.
func ImageWithTag(image string) string {
	if strings.Contains(image, ":") {
		return image
	}
	return image + ":latest"
}
