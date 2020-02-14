package container

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/util"
)

func getReleaseMetadataFromContainer(state *command.GlobalState, id string) (map[string]string, error) {
	reader, _, err := state.DockerClient.CopyFromContainer(
		state.DockerContext,
		id,
		"/release-metadata.json",
	)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)

	if _, err := tarReader.Next(); err != nil {
		return nil, err
	}

	var untarred bytes.Buffer
	io.Copy(&untarred, tarReader)

	var result map[string]string
	if err := json.Unmarshal(untarred.Bytes(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Run creates and runs the release container, returning a map of release metadata.
func Run(state *command.GlobalState, image, codeDir, buildVolume string, outputStream, errorStream io.Writer, env map[string]string) (map[string]string, error) {
	container, err := createReleaseContainer(state, image, codeDir, buildVolume, env)
	if err != nil {
		return nil, err
	}

	if err := containers.Await(state, container, nil, outputStream, errorStream, nil); err != nil {
		return nil, err
	}

	releaseMetadata, err := getReleaseMetadataFromContainer(state, container)
	if err != nil {
		return nil, fmt.Errorf("could not get release metadata from container: %w", err)
	}

	if err := state.DockerClient.ContainerRemove(
		state.DockerContext,
		container,
		types.ContainerRemoveOptions{},
	); err != nil {
		return nil, err
	}

	return releaseMetadata, nil
}

func createReleaseContainer(state *command.GlobalState, image, codeDir, buildVolume string, env map[string]string) (string, error) {
	response, err := state.DockerClient.ContainerCreate(
		state.DockerContext,
		&container.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/code",
			Env:          containers.MapToDockerEnv(env),
		},
		&container.HostConfig{
			LogConfig: container.LogConfig{Type: "none"},
			Binds: []string{
				codeDir + ":/code:ro",
				buildVolume + ":/build",
				"/var/run/docker.sock:/var/run/docker.sock",
			},
		},
		nil,
		util.RandomName("cdflow2-release"),
	)
	if err != nil {
		return "", nil
	}
	return response.ID, nil
}
