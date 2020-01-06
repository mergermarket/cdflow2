package terraform_test

import (
	"bytes"
	"encoding/json"
	"log"
	"reflect"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/test"
)

// ReflectedInput is the message format returned from the fake terraform container that reflects its inputs.
type ReflectedInput struct {
	Args  []string
	Env   map[string]string
	Input string
	Cwd   string
	File  string
}

func TestTerraformInitInitial(t *testing.T) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	buildVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, buildVolume)

	if err := terraform.InitInitial(
		dockerClient,
		test.GetConfig("TEST_TERRAFORM_IMAGE"),
		test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
		buildVolume,
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

	buildOutput, err := test.ReadVolume(dockerClient, buildVolume)
	if err != nil {
		log.Panicln("could not read build volume:", err)
	}

	if !reflect.DeepEqual(buildOutput, map[string]string{"build-output-test": "build output"}) {
		log.Panicln("unexpected build output:", buildOutput)
	}
}

func TestTerraformDeployCommands(t *testing.T) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	//terraformContainer, err := NewTerraformContainer()

	if err := terraform.ConfigureBackend(
		dockerClient,
		test.GetConfig("TEST_TERRAFORM_IMAGE"),
		test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
		releaseVolume,
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

}
