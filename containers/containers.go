package containers

import (
	"fmt"
	"io"
	"sort"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

// EnsureImage pulls an image if it does not exist locally.
func EnsureImage(dockerClient *docker.Client, image string) error {
	if _, err := dockerClient.InspectImage(image); err == nil {
		return nil
	}
	return dockerClient.PullImage(docker.PullImageOptions{
		Repository: image,
	}, docker.AuthConfiguration{})
}

// Await waits for a container runs a container and waits for it to finish.
func Await(dockerClient *docker.Client, container *docker.Container, inputStream io.Reader, outputStream, errorStream io.Writer, started chan error) error {
	// TODO argument list too complex, refactor to struct?
	// TODO too complex, consider factoring some functionality out
	attached := make(chan error)
	detached := make(chan error)
	stdin := false
	if inputStream != nil {
		stdin = true
	}
	go func() {
		waiter, err := dockerClient.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
			Container:    container.ID,
			InputStream:  inputStream,
			OutputStream: outputStream,
			ErrorStream:  errorStream,
			RawTerminal:  true,
			Stream:       true,
			Stdout:       true,
			Stderr:       true,
			Stdin:        stdin,
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

	if err := <-detached; err != nil {
		return err
	}
	props, err := dockerClient.InspectContainer(container.ID)
	if err != nil {
		return err
	}

	if props.State.Running {
		return fmt.Errorf("unexpected: release container still running")
	}

	if props.State.ExitCode != 0 {
		return fmt.Errorf("container failed with status %v", props.State.ExitCode)
	}

	return nil
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

// RepoDigest returns the first image digest for a docker image.
func RepoDigest(dockerClient *docker.Client, image string) (string, error) {
	details, err := dockerClient.InspectImage(image)
	if err != nil {
		return "", err
	}
	if len(details.RepoDigests) == 0 {
		return "", nil
	}
	return details.RepoDigests[0], nil
}
