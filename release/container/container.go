package container

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/util"
)

func getReleaseMetadataFromContainer(dockerClient *docker.Client, id string) (map[string]string, error) {
	var buffer bytes.Buffer
	if err := dockerClient.DownloadFromContainer(id, docker.DownloadFromContainerOptions{
		OutputStream: &buffer,
		Path:         "/release-metadata.json",
	}); err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(&buffer)

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
func Run(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, outputStream, errorStream io.Writer, env map[string]string) (map[string]string, error) {
	container, err := createReleaseContainer(dockerClient, image, codeDir, buildVolume, env)
	if err != nil {
		return nil, err
	}

	if err := containers.Await(dockerClient, container, nil, outputStream, errorStream, nil); err != nil {
		return nil, err
	}

	/*if err := outputStream.Close(); err != nil {
		return nil, fmt.Errorf("error closing pipe for container output: %v", err)
	}*/

	releaseMetadata, err := getReleaseMetadataFromContainer(dockerClient, container.ID)
	if err != nil {
		return nil, fmt.Errorf("could not get release metadata from container: %w", err)
	}

	if err := dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID}); err != nil {
		return nil, err
	}

	return releaseMetadata, nil
}

func createReleaseContainer(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, env map[string]string) (*docker.Container, error) {
	return dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: util.RandomName("cdflow2-release"),
		Config: &docker.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/code",
			Env:          containers.MapToDockerEnv(env),
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{Type: "none"},
			Binds: []string{
				codeDir + ":/code:ro",
				buildVolume.Name + ":/build",
				"/var/run/docker.sock:/var/run/docker.sock",
			},
		},
	})
}
