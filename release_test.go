package main

import (
	"bytes"
	"log"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/release"
)

func TestRelese(t *testing.T) {
	dockerClient := createDockerClient()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	buildVolume := createVolume(dockerClient)
	defer removeVolume(dockerClient, buildVolume)

	releaseMetadata, err := release.Run(
		dockerClient,
		getConfig("TEST_RELEASE_IMAGE"),
		getConfig("TEST_ROOT")+"/test/release/sample-code",
		buildVolume,
		&outputBuffer,
		&errorBuffer,
	)
	if err != nil {
		log.Panicln("unexpected error: ", err)
	}

	if errorBuffer.String() != "message to stderr\n" {
		log.Panicf("unexpected stderr output: '%v'", errorBuffer.String())
	}
	if errorBuffer.String() != "message to stderr\n" {
		log.Panicf("unexpected stderr output: '%v'", errorBuffer.String())
	}

	if !reflect.DeepEqual(releaseMetadata, map[string]string{
		"release_var_from_env": "release value from env",
	}) {
		log.Panicf("unexpected release metadata: %v\n", releaseMetadata)
	}

}
