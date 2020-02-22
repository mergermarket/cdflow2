package official_test

import (
	"bytes"
	"log"
	"testing"

	"github.com/mergermarket/cdflow2/docker"
	"github.com/mergermarket/cdflow2/docker/official"
)

func TestRun(t *testing.T) {
	// Given
	dockerClient, err := official.NewClient()
	if err != nil {
		log.Fatalln("error creating doker client:", err)
	}

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	image := "alpine:latest"
	if err := dockerClient.EnsureImage(image, nil); err != nil {
		log.Panicln("could not pull image:", err)
	}

	// When
	if err := dockerClient.Run(&docker.RunOptions{
		Image:        image,
		OutputStream: &outputBuffer,
		ErrorStream:  &errorBuffer,
		Cmd: []string{"/bin/sh", "-c", `
			echo foo bar baz
			echo one two three >&2
			echo baz bar foo
			echo three two one >&2
		`},
		NamePrefix: "cdflow2-test-official",
	}); err != nil {
		log.Panicln("error running container:", err)
	}

	// Then
	if outputBuffer.String() != "foo bar baz\nbaz bar foo\n" {
		log.Panicf("unexpected output: %#v", outputBuffer.String())
	}
	if errorBuffer.String() != "one two three\nthree two one\n" {
		log.Panicf("unexpected output: %#v", outputBuffer.String())
	}
}
