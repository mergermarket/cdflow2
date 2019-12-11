package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
)

type ReflectedInput struct {
	Args  []string
	Env   map[string]string
	Input string
	Cwd   string
	File  string
}

func TestTerraform(t *testing.T) {
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
	if err := terraformInit(
		dockerClient,
		getConfig("TEST_TERRAFORM_IMAGE"),
		getConfig("TEST_ROOT")+"/test/terraform/sample-code",
		buildDir,
		&outputBuffer,
		&errorBuffer,
	); err != nil {
		log.Fatalln("unexpected error: ", err)
	}

	if errorBuffer.String() != "message to stderr\n" {
		log.Fatalf("unexpected stderr output: '%v'", errorBuffer.String())
	}

	var output ReflectedInput
	json.Unmarshal(outputBuffer.Bytes(), &output)

	// interface is that the code is mapped to /code and the terraform is in the infra subfolder
	if !reflect.DeepEqual(output.Args, []string{"init", "/code/infra"}) {
		log.Fatalf("unexpected args: %v", output.Args)
	}

	// interface is that the mapped in cwd is /build
	if output.Cwd != "/build" {
		log.Fatalf("unexpected cwd: %v", output.Cwd)
	}

	if output.File != "sample content" {
		log.Fatalf("code not mapped as /code - file contents: %v", output.File)
	}

	buildOutput, err := ioutil.ReadFile(buildDir + "/build-output-test")
	if err != nil {
		log.Fatalf("could not read build output: %v", err)
	}
	if string(buildOutput) != "build output" {
		log.Fatalf("unexpected contents of test build output file: %v", string(buildOutput))
	}
}
