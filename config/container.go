package config

import (
	"archive/tar"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/util"
)

// Container represents a config container.
type Container struct {
	state         *command.GlobalState
	ID            string
	completion    chan error
	image         string
	releaseVolume string
	reader        *bufio.Reader
	readStream    io.Reader
	writeStream   io.Writer
	errorStream   io.Writer
}

// NewContainer creates and returns a new config container.
func NewContainer(state *command.GlobalState, image, releaseVolume string, errorStream io.Writer) *Container {
	return &Container{
		state:         state,
		completion:    make(chan error, 1),
		image:         image,
		releaseVolume: releaseVolume,
		errorStream:   errorStream,
	}
}

// Start starts the config container.
func (configContainer *Container) Start() error {
	id, err := configContainer.createContainer()
	if err != nil {
		return err
	}
	configContainer.ID = id

	inputReadStream, inputWriteStream := io.Pipe()
	outputReadStream, outputWriteStream := io.Pipe()

	configContainer.readStream = outputReadStream
	configContainer.reader = bufio.NewReader(outputReadStream)
	configContainer.writeStream = inputWriteStream

	started := make(chan error, 1)
	go func() {
		configContainer.completion <- containers.Await(configContainer.state, configContainer.ID, inputReadStream, outputWriteStream, configContainer.errorStream, started)
		inputReadStream.Close()
		outputWriteStream.Close()
	}()
	return <-started
}

func (configContainer *Container) createContainer() (string, error) {
	id := util.RandomName("cdflow2-config")
	if _, err := configContainer.state.DockerClient.ContainerCreate(
		configContainer.state.DockerContext,
		&containertypes.Config{
			Image:        configContainer.image,
			OpenStdin:    true,
			StdinOnce:    true,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/release",
		},
		&container.HostConfig{
			LogConfig: container.LogConfig{Type: "none"},
			Binds:     []string{configContainer.releaseVolume + ":/release"},
		},
		nil,
		id,
	); err != nil {
		return "", fmt.Errorf("error creating config container: %w", err)
	}
	return id, nil
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

// Stop stops the container.
func (configContainer *Container) Stop(timeout *time.Duration) error {
	return configContainer.state.DockerClient.ContainerStop(configContainer.state.DockerContext, configContainer.ID, timeout)
}

// Remove removes the config container.
func (configContainer *Container) Remove() error {
	return configContainer.state.DockerClient.ContainerRemove(
		configContainer.state.DockerContext,
		configContainer.ID,
		types.ContainerRemoveOptions{},
	)
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
	return <-configContainer.completion
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

	if err := configContainer.state.DockerClient.CopyToContainer(
		configContainer.state.DockerContext,
		configContainer.ID,
		"/",
		buffer,
		types.CopyToContainerOptions{},
	); err != nil {
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
