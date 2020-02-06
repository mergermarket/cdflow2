package config

import (
	"archive/tar"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/util"
)

// Container represents a config container.
type Container struct {
	dockerClient  *docker.Client
	container     *docker.Container
	completion    chan error
	image         string
	releaseVolume *docker.Volume
	reader        *bufio.Reader
	readStream    io.Reader
	writeStream   io.Writer
	errorStream   io.Writer
}

// NewContainer creates and returns a new config container.
func NewContainer(dockerClient *docker.Client, image string, releaseVolume *docker.Volume, errorStream io.Writer) *Container {
	return &Container{
		dockerClient:  dockerClient,
		completion:    make(chan error, 1),
		image:         image,
		releaseVolume: releaseVolume,
		errorStream:   errorStream,
	}
}

// Start starts the config container.
func (container *Container) Start() error {
	dockerContainer, err := container.createContainer()
	if err != nil {
		log.Fatalln("foo", err)
		return err
	}
	container.container = dockerContainer

	inputReadStream, inputWriteStream := io.Pipe()
	outputReadStream, outputWriteStream := io.Pipe()

	container.readStream = outputReadStream
	container.reader = bufio.NewReader(outputReadStream)
	container.writeStream = inputWriteStream

	started := make(chan error)
	go func() {
		container.completion <- containers.Await(container.dockerClient, dockerContainer, inputReadStream, outputWriteStream, container.errorStream, started)
		inputReadStream.Close()
		outputWriteStream.Close()
	}()
	return <-started
}

func (container *Container) createContainer() (*docker.Container, error) {
	name := util.RandomName("cdflow2-config")
	return container.dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Image:        container.image,
			OpenStdin:    true,
			StdinOnce:    true,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/release",
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{Type: "none"},
			Binds:     []string{container.releaseVolume.Name + ":/release"},
		},
	})
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
func (container *Container) Stop(n uint) error {
	return container.dockerClient.StopContainer(container.container.ID, n)
}

// Remove removes the config container.
func (container *Container) Remove() error {
	return container.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.container.ID,
	})
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
	Env map[string]string
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

	if err := container.dockerClient.UploadToContainer(container.container.ID, docker.UploadToContainerOptions{
		InputStream: buffer,
		Path:        "/",
	}); err != nil {
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
	return &response, nil
}
