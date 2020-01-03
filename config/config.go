package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"

	docker "github.com/fsouza/go-dockerclient"
	containers "github.com/mergermarket/cdflow2/containers"
)

type ConfigContainer struct {
	dockerClient  *docker.Client
	container     *docker.Container
	completion    chan error
	image         string
	releaseVolume *docker.Volume
	reader        *bufio.Reader
	readStream    io.Reader
	writeStream   io.Writer
}

func NewConfigContainer(dockerClient *docker.Client, image string, releaseVolume *docker.Volume) *ConfigContainer {
	return &ConfigContainer{
		dockerClient:  dockerClient,
		completion:    make(chan error, 1),
		image:         image,
		releaseVolume: releaseVolume,
	}
}

func (self *ConfigContainer) Start() error {
	container, err := self.createContainer()
	if err != nil {
		return err
	}
	self.container = container

	inputReadStream, inputWriteStream := io.Pipe()
	outputReadStream, outputWriteStream := io.Pipe()

	self.readStream = outputReadStream
	self.reader = bufio.NewReader(outputReadStream)
	self.writeStream = inputWriteStream

	started := make(chan error)
	go func() {
		self.completion <- containers.Await(self.dockerClient, container, inputReadStream, outputWriteStream, os.Stderr, started)
		inputReadStream.Close()
		outputWriteStream.Close()
	}()
	return <-started
}

func (self *ConfigContainer) createContainer() (*docker.Container, error) {
	return self.dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: "config",
		Config: &docker.Config{
			Image:        self.image,
			OpenStdin:    true,
			StdinOnce:    true,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/release",
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{Type: "none"},
			Binds:     []string{self.releaseVolume.Name + ":/release"},
		},
	})
}

func (self *ConfigContainer) readline() ([]byte, error) {
	line, err := self.reader.ReadBytes('\n')
	if err == io.EOF {
		return line, errors.New("config container disconnected")
	}
	return line, err
}

func (self *ConfigContainer) write(message []byte) error {
	n, err := self.writeStream.Write(message)
	if err != nil {
		return err
	}
	if n != len(message) {
		return errors.New("incomplete write to container")
	}
	return nil
}

func (self *ConfigContainer) StopContainer(n uint) error {
	return self.dockerClient.StopContainer(self.container.ID, n)
}

func (self *ConfigContainer) Remove() error {
	return self.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID: self.container.ID,
	})
}

type stopRequest struct {
	Action string
}

func (self *ConfigContainer) Stop() error {
	request, err := json.Marshal(&stopRequest{Action: "stop"})
	if err != nil {
		return err
	}
	if err := self.write(append(request, '\n')); err != nil {
		return err
	}
	return <-self.completion
}

type configureReleaseConfigRequest struct {
	Action  string
	Version string
	Config  map[string]interface{}
	Env     map[string]string
}

type configureReleaseConfigResponse struct {
	Env map[string]string
}

func (self *ConfigContainer) ConfigureRelease(
	version string,
	config map[string]interface{},
	env map[string]string,
) (*configureReleaseConfigResponse, error) {
	request, err := json.Marshal(&configureReleaseConfigRequest{Action: "configure_release", Version: version, Config: config, Env: env})
	if err != nil {
		return nil, err
	}
	if err := self.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := self.readline()
	if err != nil {
		return nil, err
	}
	var response configureReleaseConfigResponse
	if err := json.Unmarshal(received, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

type uploadReleaseRequest struct {
	Action          string
	TerraformImage  string
	ReleaseMetadata map[string]string
}

type uploadReleaseResponse struct {
	Message string
}

func (self *ConfigContainer) UploadRelease(
	terraformImage string,
	releaseMetadata map[string]string,
) (*uploadReleaseResponse, error) {
	request, err := json.Marshal(&uploadReleaseRequest{Action: "upload_release", TerraformImage: terraformImage, ReleaseMetadata: releaseMetadata})
	if err != nil {
		return nil, err
	}
	if err := self.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := self.readline()
	if err != nil {
		return nil, err
	}
	var response uploadReleaseResponse
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
}

type prepareTerraformResponse struct {
	TerraformImage         string
	Env                    map[string]string
	TerraformBackendType   string
	TerraformBackendConfig map[string]string
}

func (self *ConfigContainer) PrepareTerraform(
	version string,
	config map[string]interface{},
	env map[string]string,
) (*prepareTerraformResponse, error) {
	request, err := json.Marshal(&prepareTerraformRequest{
		Action:  "prepare_terraform",
		Config:  config,
		Env:     env,
		Version: version,
	})
	if err != nil {
		return nil, err
	}
	if err := self.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := self.readline()
	if err != nil {
		return nil, err
	}
	var response prepareTerraformResponse
	if err := json.Unmarshal(received, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
