package container

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/docker"
)

// GetReleaseRequirements runs the container in order to get requirements.
func GetReleaseRequirements(state *command.GlobalState, buildID, image string, errorStream io.Writer) (map[string]interface{}, error) {
	fmt.Fprintf(state.ErrorStream, "\nGetting requirements for build: '%v' (%v)...\n", buildID, image)
	var outputBuffer bytes.Buffer
	if err := state.DockerClient.Run(&docker.RunOptions{
		Image:        image,
		OutputStream: &outputBuffer,
		ErrorStream:  errorStream,
		NamePrefix:   "cdflow2-release-reqirements",
		Cmd:          []string{"requirements"},
	}); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.NewDecoder(&outputBuffer).Decode(&result); err != nil {
		return nil, err
	}
	fmt.Fprintf(state.ErrorStream, "Got: %v\n", result)
	return result, nil
}

// Run creates and runs the release container, returning a map of release metadata.
func Run(dockerClient docker.Iface, image, codeDir, buildVolume string, outputStream, errorStream io.Writer, env map[string]string) (map[string]string, error) {

	var releaseMetadata map[string]string

	return releaseMetadata, dockerClient.Run(&docker.RunOptions{
		Image:        image,
		OutputStream: outputStream,
		ErrorStream:  errorStream,
		WorkingDir:   "/code",
		Env:          mapToDockerEnv(env),
		Binds: []string{
			codeDir + ":/code:ro",
			buildVolume + ":/build",
			"/var/run/docker.sock:/var/run/docker.sock",
		},
		NamePrefix: "cdflow2-release",
		BeforeRemove: func(id string) error {
			result, err := getReleaseMetadataFromContainer(dockerClient, id)
			if err != nil {
				return fmt.Errorf("could not get release metadata from container: %w", err)
			}
			releaseMetadata = result
			return nil
		},
	})
}

func getReleaseMetadataFromContainer(dockerClient docker.Iface, id string) (returnedMetadata map[string]string, returnedError error) {
	reader, err := dockerClient.CopyFromContainer(id, "/release-metadata.json")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			if returnedError != nil {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
			return
		}
	}()

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

func mapToDockerEnv(input map[string]string) []string {
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
