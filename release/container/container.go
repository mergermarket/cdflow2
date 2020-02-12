package container

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/util"
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

	resultChannel := make(chan readReleaseMetadataResult, 1)
	go func() {
		result, err := handleReleaseOutput(outputReadStream, outputStream)
		resultChannel <- readReleaseMetadataResult{result, err}
	}()

	if err := containers.Await(dockerClient, container, nil, outputWriteStream, errorStream, nil); err != nil {
		return nil, err
	}

	if err := outputWriteStream.Close(); err != nil {
		return nil, fmt.Errorf("error closing pipe for container output: %v", err)
	}

	//if err := dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID}); err != nil {
	//	return nil, err
	//}
	result := <-resultChannel

	if result.err != nil {
		output, err := exec.Command("docker", "logs", container.ID).CombinedOutput()
		fmt.Printf("container output: '%v', err: %v", output, err)
	}

	return result.metadata, result.err
}

type tailBuffer struct {
	data []byte
}

func (buffer *tailBuffer) Write(data []byte) (int, error) {
	bufferSize := 10 * 1024
	buffer.data = append(buffer.data, data...)
	if len(buffer.data) > bufferSize {
		buffer.data = buffer.data[len(buffer.data)-bufferSize:]
	}
	return len(data), nil
}

func getLastLine(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("could not get last line of release data - there was no output")
	}
	if data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	newlinePosition := bytes.LastIndex(data, []byte{'\n'})
	if newlinePosition == -1 {
		return nil, fmt.Errorf("could not get last line of release data - no newline found")
	}
	return data[newlinePosition+1:], nil
}

// handleReleaseOutput reads from the read stream and writes to the write stream, picking out and returning the release metadata.
func handleReleaseOutput(readStream io.Reader, outputStream io.Writer) (map[string]string, error) {
	tee := io.TeeReader(readStream, outputStream)
	var buffer tailBuffer
	io.Copy(&buffer, tee)
	lastLine, err := getLastLine(buffer.data)
	if err != nil {
		return nil, err
	}
	var result map[string]string
	if err := json.Unmarshal(lastLine, &result); err != nil {
		return nil, err
	}
	return result, nil
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
			//LogConfig: docker.LogConfig{Type: "none"},
			Binds: []string{
				codeDir + ":/code:ro",
				buildVolume.Name + ":/build",
				"/var/run/docker.sock:/var/run/docker.sock",
			},
		},
	})
}
