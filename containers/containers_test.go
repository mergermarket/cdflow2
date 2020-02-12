package containers_test

import (
	"bytes"
	"log"
	"reflect"
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
	if err := containers.EnsureImage(dockerClient, image); err != nil {
		log.Panicln("could not pull image:", err)
	}
	container, err := dockerClient.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			Cmd: []string{"/bin/sh", "-c", `
				echo foo bar baz
				echo one two three >&2
				echo baz bar foo
				echo three two one >&2
			`},
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

	if outputBuffer.String() != "foo bar baz\nbaz bar foo\n" {
		log.Panicf("unexpected output: %#v", outputBuffer.String())
	}
	if errorBuffer.String() != "one two three\nthree two one\n" {
		log.Panicf("unexpected output: %#v", outputBuffer.String())
	}
}

func TestMapToDockerEnv(test *testing.T) {
	result := containers.MapToDockerEnv(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	if !reflect.DeepEqual(result, []string{"key1=value1", "key2=value2"}) {
		log.Fatalln("unexpected docker env:", result)
	}
}

func TestImageWithTag(test *testing.T) {
	if containers.ImageWithTag("test") != "test:latest" {
		log.Fatalln("latest not added")
	}
	if containers.ImageWithTag("test:1") != "test:1" {
		log.Fatalln("tagged image should be no-op")
	}
}
