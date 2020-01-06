package containers_test

import (
	"bytes"
	"log"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/test"
)

func TestAwait(t *testing.T) {
	dockerClient := test.CreateDockerClient()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	image := "alpine:latest"
	if err := dockerClient.PullImage(docker.PullImageOptions{
		Repository: image,
	}, docker.AuthConfiguration{}); err != nil {
		log.Panicln("error pulling:", err)
	}
	container, err := dockerClient.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			Cmd:          []string{"echo", "hello"},
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{Type: "none"},
		},
	})
	if err != nil {
		log.Panicln("error creating container:", err)
	}

	if err := containers.Await(dockerClient, container, nil, &outputBuffer, &errorBuffer, nil); err != nil {
		log.Panicln("error running container:", err)
	}

	if outputBuffer.String() != "hello\n" {
		log.Panicln("unexpected output:", outputBuffer.String())
	}
}
