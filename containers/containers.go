package containers

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mergermarket/cdflow2/command"
)

// EnsureImage pulls an image if it does not exist locally.
func EnsureImage(state *command.GlobalState, image string, outputStream io.Writer) error {
	// TODO bit lax, this should check the error type
	if _, _, err := state.DockerClient.ImageInspectWithRaw(
		state.DockerContext,
		image,
	); err == nil {
		return nil
	}
	reader, err := state.DockerClient.ImagePull(
		state.DockerContext,
		image,
		types.ImagePullOptions{},
	)
	if err != nil {
		return err
	}
	_, err = io.Copy(outputStream, reader)
	return err

}

// Await waits for a container runs a container and waits for it to finish.
func Await(state *command.GlobalState, container string, inputStream io.ReadCloser, outputStream, errorStream io.Writer, started chan error) error {
	stdin := false
	if inputStream != nil {
		stdin = true
	}
	hijackedResponse, err := state.DockerClient.ContainerAttach(
		state.DockerContext,
		container,
		types.ContainerAttachOptions{
			Stream: true,
			Stdout: true,
			Stderr: true,
			Stdin:  stdin,
		},
	)
	if err != nil {
		return err
	}

	return StreamHijackedResponse(state, hijackedResponse, inputStream, outputStream, errorStream, func() error {
		err := state.DockerClient.ContainerStart(
			state.DockerContext,
			container,
			types.ContainerStartOptions{},
		)
		if started != nil {
			started <- err
		}
		return err
	})
}

// StreamHijackedResponse copies input and output to and from the attached container connection.
func StreamHijackedResponse(state *command.GlobalState, hijackedResponse types.HijackedResponse, inputStream io.ReadCloser, outputStream, errorStream io.Writer, start func() error) error {
	if inputStream != nil {
		go func() {
			defer inputStream.Close()
			defer hijackedResponse.CloseWrite()
			io.Copy(hijackedResponse.Conn, inputStream)
		}()
	}

	outputDone := make(chan error, 1)
	defer close(outputDone)
	go func() {
		defer hijackedResponse.Close()
		_, err := stdcopy.StdCopy(outputStream, errorStream, hijackedResponse.Reader)
		outputDone <- err
	}()

	if err := start(); err != nil {
		return err
	}

	for {
		select {
		case err := <-outputDone:
			return err
		case <-state.DockerContext.Done():
			return state.DockerContext.Err()
		}
	}
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
func RepoDigest(state *command.GlobalState, image string) (string, error) {
	details, _, err := state.DockerClient.ImageInspectWithRaw(state.DockerContext, image)
	if err != nil {
		return "", err
	}
	if len(details.RepoDigests) == 0 {
		return "", nil
	}
	return details.RepoDigests[0], nil
}

// MaybePullImage conditionally pulls a docker image based on a boolean flag.
func MaybePullImage(doIt bool, state *command.GlobalState, image, imageDescription string) error {
	if !doIt {
		return nil
	}
	reader, err := state.DockerClient.ImagePull(state.DockerContext, ImageWithTag(image), types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("error pulling %s image: %w", imageDescription, err)
	}
	if _, err := io.Copy(state.OutputStream, reader); err != nil {
		return fmt.Errorf("error while pulling %s image: %w", imageDescription, err)
	}
	return nil
}
