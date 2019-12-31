package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"

	docker "github.com/fsouza/go-dockerclient"
)

type configContainer struct {
	dockerClient *docker.Client
	container    *docker.Container
	completion   chan error
	image        string
	buildVolume  *docker.Volume
	reader       *bufio.Reader
	readStream   io.Reader
	writeStream  io.Writer
}

func NewConfigContainer(dockerClient *docker.Client, image string, buildVolume *docker.Volume) *configContainer {
	return &configContainer{
		dockerClient: dockerClient,
		completion:   make(chan error, 1),
		image:        image,
		buildVolume:  buildVolume,
	}
}

func (self *configContainer) start() error {
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
		self.completion <- awaitContainer(self.dockerClient, container, inputReadStream, outputWriteStream, os.Stderr, started)
		inputReadStream.Close()
		outputWriteStream.Close()
	}()
	return <-started
}

func (self *configContainer) createContainer() (*docker.Container, error) {
	return self.dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: "config",
		Config: &docker.Config{
			Image:        self.image,
			OpenStdin:    true,
			StdinOnce:    true,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/build",
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{Type: "none"},
			Binds:     []string{self.buildVolume.Name + ":/build"},
		},
	})
}

func (self *configContainer) readline() ([]byte, error) {
	line, err := self.reader.ReadBytes('\n')
	if err == io.EOF {
		return line, errors.New("config container disconnected")
	}
	return line, err
}

func (self *configContainer) write(message []byte) error {
	n, err := self.writeStream.Write(message)
	if err != nil {
		return err
	}
	if n != len(message) {
		return errors.New("incomplete write to container")
	}
	return nil
}

func (self *configContainer) stopContainer(n uint) error {
	return self.dockerClient.StopContainer(self.container.ID, n)
}

func (self *configContainer) removeContainer() error {
	return self.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID: self.container.ID,
	})
}

type stopRequest struct {
	Action string
}

func (self *configContainer) stop() error {
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

func (self *configContainer) configureRelease(
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

func (self *configContainer) uploadRelease(
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

func (self *configContainer) prepareTerraform(
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
