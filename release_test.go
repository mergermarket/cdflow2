package main

import (
	"bytes"
	"log"
	"os"
	"reflect"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
)

func TestRelese(t *testing.T) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	buildDir, err := tempdir()
	if err != nil {
		log.Fatalf("could not make tempdir: %v", err)
	}
	defer os.RemoveAll(buildDir)

	releaseMetadata, err := runRelease(
		dockerClient,
		getConfig("TEST_RELEASE_IMAGE"),
		getConfig("TEST_ROOT")+"/test/release/sample-code",
		buildDir,
		&outputBuffer,
		&errorBuffer,
	)
	if err != nil {
		log.Fatalln("unexpected error: ", err)
	}

	if errorBuffer.String() != "message to stderr\n" {
		log.Fatalf("unexpected stderr output: '%v'", errorBuffer.String())
	}
	if errorBuffer.String() != "message to stderr\n" {
		log.Fatalf("unexpected stderr output: '%v'", errorBuffer.String())
	}

	if !reflect.DeepEqual(releaseMetadata, map[string]string{
		"release_var_from_env": "release value from env",
	}) {
		log.Fatalf("unexpected release metadata: %v\n", releaseMetadata)
	}

}
