package config

import (
	"archive/tar"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/docker"
)

// Container represents a config container.
type Container struct {
	dockerClient docker.Iface
	id           string
	done         chan error
	reader       *bufio.Reader
	writeStream  io.WriteCloser
	finished     bool
}

// NewContainer creates and returns a new config container.
func NewContainer(dockerClient docker.Iface, image, releaseVolume string, errorStream io.Writer) (*Container, error) {
	started := make(chan string, 1)
	defer close(started) // does not error so no named returns

	done := make(chan error, 1)

	inputReadStream, inputWriteStream := io.Pipe()
	outputReadStream, outputWriteStream := io.Pipe()
	container := &Container{
		dockerClient: dockerClient,
		done:         done,
		reader:       bufio.NewReader(outputReadStream),
		writeStream:  inputWriteStream,
	}

	go func() {
		err := dockerClient.Run(&docker.RunOptions{
			NamePrefix:   "cdflow2-config",
			Image:        image,
			InputStream:  inputReadStream,
			OutputStream: outputWriteStream,
			ErrorStream:  errorStream,
			WorkingDir:   "/release",
			Binds:        []string{releaseVolume + ":/release"},
			Started:      started,
		})
		container.finished = true
		if err := inputReadStream.Close(); err != nil {
			log.Panicln("error closing read stream:", err)
		}
		if err := outputWriteStream.Close(); err != nil {
			log.Panicln("error closing write stream:", err)
		}
		done <- err
	}()

	select {
	case id := <-started:
		container.id = id
		return container, nil
	case err := <-done:
		return nil, err
	}
}

func (configContainer *Container) readline() ([]byte, error) {
	line, err := configContainer.reader.ReadBytes('\n')
	if err == io.EOF {
		return line, errors.New("config container disconnected")
	}
	return line, err
}

func (configContainer *Container) write(message []byte) error {
	n, err := configContainer.writeStream.Write(message)
	if err != nil {
		return err
	}
	if n != len(message) {
		return errors.New("incomplete write to container")
	}
	return nil
}

type stopRequest struct {
	Action string
}

// RequestStop sends a message to the config container asking it to stop gracefully.
func (configContainer *Container) RequestStop() error {
	request, err := json.Marshal(&stopRequest{Action: "stop"})
	if err != nil {
		return err
	}
	if err := configContainer.write(append(request, '\n')); err != nil {
		return err
	}
	return nil
}

type configureReleaseConfigRequest struct {
	Action  string
	Version string
	Config  map[string]interface{}
	Env     map[string]string
}

// ConfigureReleaseConfigResponse contains the response to the configure release request.
type ConfigureReleaseConfigResponse struct {
	Env     map[string]string
	Success bool
}

// ConfigureRelease requests the container configures the release and returns the response.
func (configContainer *Container) ConfigureRelease(
	version string,
	config map[string]interface{},
	env map[string]string,
) (*ConfigureReleaseConfigResponse, error) {
	request, err := json.Marshal(&configureReleaseConfigRequest{Action: "configure_release", Version: version, Config: config, Env: env})
	if err != nil {
		return nil, err
	}
	if err := configContainer.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := configContainer.readline()
	if err != nil {
		return nil, err
	}
	var response ConfigureReleaseConfigResponse
	if err := json.Unmarshal(received, &response); err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, errors.New("config container failed to prepare configuration for release")
	}
	return &response, nil
}

// WriteReleaseMetadata copies the release metadata file into the release volume via the config container.
func (configContainer *Container) WriteReleaseMetadata(releaseMetadata map[string]map[string]string) error {
	encoded, err := json.Marshal(releaseMetadata)
	if err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(buffer)

	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "release/release-metadata.json",
		Mode: 0644,
		Size: int64(len(encoded)),
	}); err != nil {
		return err
	}

	if _, err := tarWriter.Write(encoded); err != nil {
		return err
	}

	if err := tarWriter.Close(); err != nil {
		return err
	}

	if err := configContainer.dockerClient.CopyToContainer(configContainer.id, "/", buffer); err != nil {
		return err
	}
	return nil
}

type uploadReleaseRequest struct {
	Action         string
	TerraformImage string
}

// UploadReleaseResponse contains the response to the upload release request.
type UploadReleaseResponse struct {
	Message string
	Success bool
}

// UploadRelease requests that the config container uploads the release and returns the response.
func (configContainer *Container) UploadRelease(terraformImage string) (*UploadReleaseResponse, error) {
	request, err := json.Marshal(&uploadReleaseRequest{Action: "upload_release", TerraformImage: terraformImage})
	if err != nil {
		return nil, err
	}
	if err := configContainer.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := configContainer.readline()
	if err != nil {
		return nil, err
	}
	var response UploadReleaseResponse
	if err := json.Unmarshal(received, &response); err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, errors.New("config container failed to upload release")
	}
	return &response, nil
}

type prepareTerraformRequest struct {
	Action  string
	Version string
	Config  map[string]interface{}
	Env     map[string]string
	EnvName string
}

// PrepareTerraformResponse contains the response to the prepare terraform request.
type PrepareTerraformResponse struct {
	TerraformImage         string
	Env                    map[string]string
	TerraformBackendType   string
	TerraformBackendConfig map[string]string
	Success                bool
}

// PrepareTerraform requests that the config container prepares for running terraform and returns the response.
func (configContainer *Container) PrepareTerraform(
	version, envName string,
	config map[string]interface{},
	env map[string]string,
) (*PrepareTerraformResponse, error) {
	request, err := json.Marshal(&prepareTerraformRequest{
		Action:  "prepare_terraform",
		Config:  config,
		Env:     env,
		EnvName: envName,
		Version: version,
	})
	if err != nil {
		return nil, err
	}
	if err := configContainer.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := configContainer.readline()
	if err != nil {
		return nil, err
	}
	var response PrepareTerraformResponse
	if err := json.Unmarshal(received, &response); err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, errors.New("config container failed to prepare for running terraform")
	}
	return &response, nil
}

// SetupTerraform creates the config container and prepares terraform in one.
func SetupTerraform(state *command.GlobalState, envName, version string, env map[string]string) (returnedTerraformImage, returnedBuildVolume string, returnedError error) {
	dockerClient := state.DockerClient

	if !state.GlobalArgs.NoPullConfig {
		if err := dockerClient.PullImage(state.Manifest.Config.Image, state.ErrorStream); err != nil {
			return "", "", fmt.Errorf("error pulling config image: %w", err)
		}
	}

	buildVolume, err := dockerClient.CreateVolume()
	if err != nil {
		return "", "", err
	}

	configContainer, err := NewContainer(dockerClient, state.Manifest.Config.Image, buildVolume, state.ErrorStream)
	if err != nil {
		return "", "", err
	}
	defer func() { // meh defer
		if err := configContainer.RequestStop(); err != nil {
			if returnedError != nil {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
			return
		}
		if err := configContainer.Done(); err != nil {
			if returnedError != nil {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
			return
		}
	}()

	prepareTerraformResponse, err := configContainer.PrepareTerraform(version, envName, state.Manifest.Config.Params, env)
	if err != nil {
		return "", "", err
	}

	if !state.GlobalArgs.NoPullTerraform {
		if err := dockerClient.EnsureImage(prepareTerraformResponse.TerraformImage, state.ErrorStream); err != nil {
			return "", "", fmt.Errorf("error pulling terraform image %v: %w", prepareTerraformResponse.TerraformImage, err)
		}
	}
	return prepareTerraformResponse.TerraformImage, buildVolume, nil
}

// Done stops and removes the config container.
func (configContainer *Container) Done() error {
	if !configContainer.finished {
		if err := configContainer.dockerClient.Stop(configContainer.id, 10*time.Second); err != nil {
			return err
		}
	}
	if err := configContainer.writeStream.Close(); err != nil {
		return fmt.Errorf("error closing pipe to config container: %w", err)
	}
	return <-configContainer.done
}
