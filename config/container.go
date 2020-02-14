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
func (container *Container) Start() error {
	id, err := container.createContainer()
	if err != nil {
		return err
	}
	container.ID = id

	inputReadStream, inputWriteStream := io.Pipe()
	outputReadStream, outputWriteStream := io.Pipe()

	container.readStream = outputReadStream
	container.reader = bufio.NewReader(outputReadStream)
	container.writeStream = inputWriteStream

	started := make(chan error, 1)
	go func() {
		container.completion <- containers.Await(container.state, container.ID, inputReadStream, outputWriteStream, container.errorStream, started)
		inputReadStream.Close()
		outputWriteStream.Close()
	}()
	return <-started
}

func (container *Container) createContainer() (string, error) {
	id := util.RandomName("cdflow2-config")
	if _, err := container.state.DockerClient.ContainerCreate(
		container.state.DockerContext,
		&containertypes.Config{
			Image:        container.image,
			OpenStdin:    true,
			StdinOnce:    true,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/release",
		},
		nil,
		nil,
		id,
	); err != nil {
		return "", fmt.Errorf("error creating config container: %w", err)
	}
	return id, nil
}

func (container *Container) readline() ([]byte, error) {
	line, err := container.reader.ReadBytes('\n')
	if err == io.EOF {
		return line, errors.New("config container disconnected")
	}
	return line, err
}

func (container *Container) write(message []byte) error {
	n, err := container.writeStream.Write(message)
	if err != nil {
		return err
	}
	if n != len(message) {
		return errors.New("incomplete write to container")
	}
	return nil
}

// Stop stops the container.
func (container *Container) Stop(timeout *time.Duration) error {
	return container.state.DockerClient.ContainerStop(container.state.DockerContext, container.ID, timeout)
}

// Remove removes the config container.
func (container *Container) Remove() error {
	return container.state.DockerClient.ContainerRemove(
		container.state.DockerContext,
		container.ID,
		types.ContainerRemoveOptions{},
	)
}

type stopRequest struct {
	Action string
}

// RequestStop sends a message to the config container asking it to stop gracefully.
func (container *Container) RequestStop() error {
	request, err := json.Marshal(&stopRequest{Action: "stop"})
	if err != nil {
		return err
	}
	if err := container.write(append(request, '\n')); err != nil {
		return err
	}
	return <-container.completion
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
func (container *Container) ConfigureRelease(
	version string,
	config map[string]interface{},
	env map[string]string,
) (*ConfigureReleaseConfigResponse, error) {
	request, err := json.Marshal(&configureReleaseConfigRequest{Action: "configure_release", Version: version, Config: config, Env: env})
	if err != nil {
		return nil, err
	}
	if err := container.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := container.readline()
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
func (container *Container) WriteReleaseMetadata(releaseMetadata map[string]map[string]string) error {
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

	if err := container.state.DockerClient.CopyToContainer(
		container.state.DockerContext,
		container.ID,
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
func (container *Container) UploadRelease(terraformImage string) (*UploadReleaseResponse, error) {
	request, err := json.Marshal(&uploadReleaseRequest{Action: "upload_release", TerraformImage: terraformImage})
	if err != nil {
		return nil, err
	}
	if err := container.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := container.readline()
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
func (container *Container) PrepareTerraform(
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
	if err := container.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := container.readline()
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
