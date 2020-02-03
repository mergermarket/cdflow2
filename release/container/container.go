package container

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/containers"
)

type readReleaseMetadataResult struct {
	metadata map[string]string
	err      error
}

// Run creates and runs the release container, returning a map of release metadata.
func Run(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, outputStream, errorStream io.Writer, env map[string]string) (map[string]string, error) {
	container, err := createReleaseContainer(dockerClient, image, codeDir, buildVolume, env)
	if err != nil {
		return nil, err
	}

	outputReadStream, outputWriteStream := io.Pipe()

	resultChannel := make(chan readReleaseMetadataResult)
	go handleReleaseOutput(outputReadStream, outputStream, resultChannel)

	if err := containers.Await(dockerClient, container, nil, outputWriteStream, errorStream, nil); err != nil {
		return nil, err
	}

	outputWriteStream.Close()

	props, err := dockerClient.InspectContainer(container.ID)
	if err != nil {
		return nil, err
	}

	if props.State.Running {
		panic("unexpected: release container still running")
	}
	if err := dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID}); err != nil {
		return nil, err
	}
	if props.State.ExitCode != 0 {
		return nil, errors.New("release container failed")
	}

	result := <-resultChannel
	return result.metadata, result.err
}

// handleReleaseOutput runs as a goroutine to buffer the container output, picking out the last line which contains the release metadata and sending it to the passed in result channel.
func handleReleaseOutput(readStream io.Reader, outputStream io.Writer, resultChannel chan readReleaseMetadataResult) {
	readScanner := bufio.NewScanner(readStream)
	var last []byte
	for readScanner.Scan() {
		last = readScanner.Bytes()
		n, err := outputStream.Write(last)
		if err != nil {
			resultChannel <- readReleaseMetadataResult{nil, err}
			return
		}
		if n != len(last) {
			resultChannel <- readReleaseMetadataResult{nil, errors.New("incomplete write")}
			return
		}
	}
	if err := readScanner.Err(); err != nil {
		resultChannel <- readReleaseMetadataResult{nil, err}
		return
	}
	var result map[string]string
	if err := json.Unmarshal(last, &result); err != nil {
		resultChannel <- readReleaseMetadataResult{nil, err}
	}
	resultChannel <- readReleaseMetadataResult{result, nil}
}

func createReleaseContainer(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, env map[string]string) (*docker.Container, error) {
	return dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: containers.RandomName("cdflow2-release"),
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
			Binds:     []string{codeDir + ":/code:ro", buildVolume.Name + ":/build"},
		},
	})
}
