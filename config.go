package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"

	docker "github.com/fsouza/go-dockerclient"
)

type configContainer struct {
	dockerClient *docker.Client
	container    *docker.Container
	completion   chan error
	image        string
	dir          string
	readScanner  *bufio.Scanner
	readStream   *io.PipeReader
	writeStream  *io.PipeWriter
}

func NewConfigContainer(dockerClient *docker.Client, image, dir string) *configContainer {
	return &configContainer{
		dockerClient: dockerClient,
		completion:   make(chan error),
		image:        image,
		dir:          dir,
	}
}

func (self *configContainer) start(errorStream io.Writer) error {
	container, err := self.createContainer()
	if err != nil {
		return err
	}
	self.container = container

	inputReadStream, inputWriteStream := io.Pipe()
	outputReadStream, outputWriteStream := io.Pipe()

	self.readStream = outputReadStream
	self.readScanner = bufio.NewScanner(outputReadStream)
	self.writeStream = inputWriteStream

	started := make(chan error)
	go func() {
		self.completion <- awaitContainer(self.dockerClient, container, inputReadStream, outputWriteStream, errorStream, started)
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
			Binds:     []string{self.dir + ":/build"},
		},
	})
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

func (self *configContainer) read() ([]byte, error) {
	if !self.readScanner.Scan() {
		return nil, self.readScanner.Err()
	}
	return self.readScanner.Bytes(), nil
}

func (self *configContainer) wait() error {
	if err := <-self.completion; err != nil {
		return err
	}
	if err := self.dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: self.container.ID}); err != nil {
		return err
	}
	return nil
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
	// no response for a stop
	self.readStream.Close()
	self.writeStream.Close()
	return self.wait()
}

type configureReleaseConfigRequest struct {
	Action string
	Config map[string]interface{}
	Env    map[string]string
}

type configureReleaseConfigResponse struct {
	Env map[string]string
}

func (self *configContainer) configureRelease(config map[string]interface{}, env map[string]string) (map[string]string, error) {
	request, err := json.Marshal(&configureReleaseConfigRequest{Action: "configure_release", Config: config, Env: env})
	if err != nil {
		return nil, err
	}
	if err := self.write(append(request, '\n')); err != nil {
		return nil, err
	}
	received, err := self.read()
	if err != nil {
		return nil, err
	}
	var response configureReleaseConfigResponse
	if err := json.Unmarshal(received, &response); err != nil {
		return nil, err
	}
	return response.Env, nil
}

type uploadReleaseRequest struct {
	Action         string
	TerraformImage string
}

type uploadReleaseResponse struct {
}

func (self *configContainer) uploadRelease(terraformImage string) error {
	request, err := json.Marshal(&uploadReleaseRequest{Action: "upload_release", TerraformImage: terraformImage})
	if err != nil {
		return err
	}
	if err := self.write(append(request, '\n')); err != nil {
		return err
	}
	received, err := self.read()
	if err != nil {
		return err
	}
	var response uploadReleaseResponse
	if err := json.Unmarshal(received, &response); err != nil {
		return err
	}
	return nil
}
