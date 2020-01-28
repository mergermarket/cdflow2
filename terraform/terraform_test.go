package terraform_test

import (
	"bytes"
	"encoding/json"
	"log"
	"reflect"
	"strings"
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

	if outputBuffer.String() != "message to stdout\n" {
		log.Fatalf("unexpected stdout output: '%v'", outputBuffer.String())
	}

	var input ReflectedInput
	if err := json.Unmarshal(errorBuffer.Bytes(), &input); err != nil {
		log.Panicln("error parsing json:", err)
	}

	// interface is that the code is mapped to /code and the terraform is in the infra subfolder
	if !reflect.DeepEqual(input.Args, []string{"init", "/code/infra"}) {
		log.Fatalf("unexpected args: %v", input.Args)
	}

	// interface is that the mapped in cwd is /build
	if input.Cwd != "/build" {
		log.Fatalf("unexpected cwd: %v", input.Cwd)
	}

	if input.File != "sample content" {
		log.Fatalf("code not mapped as /code - file contents: %v", input.File)
	}

	buildOutput, err := test.ReadVolume(dockerClient, buildVolume)
	if err != nil {
		log.Panicln("could not read build volume:", err)
	}

	if !reflect.DeepEqual(buildOutput, map[string]string{"build-output-test": "build output"}) {
		log.Panicln("unexpected build output:", buildOutput)
	}
}

func TestTerraformConfigureBackend(t *testing.T) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	terraformContainer, err := terraform.NewTerraformContainer(
		dockerClient,
		test.GetConfig("TEST_TERRAFORM_IMAGE"),
		test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
		releaseVolume,
	)
	defer terraformContainer.Done()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	if err := terraformContainer.ConfigureBackend(
		&outputBuffer,
		&errorBuffer,
		[]terraform.BackendConfigParameter{
			terraform.BackendConfigParameter{"key1", "value1"},
			terraform.BackendConfigParameter{"key2", "value2"},
		},
	); err != nil {
		log.Panicln("unexpected error: ", err)
	}

	if outputBuffer.String() != "message to stdout\n" {
		log.Panicf("unexpected stdout output: '%v'", outputBuffer.String())
	}

	var input ReflectedInput
	if err := json.Unmarshal(errorBuffer.Bytes(), &input); err != nil {
		log.Panicln("error parsing json:", err)
	}
	if !reflect.DeepEqual(input.Args, []string{
		"init",
		"-get=false",
		"-get-plugins=false",
		"-backend-config=key1=value1",
		"-backend-config=key2=value2",
	}) {
		log.Panicln("unexpected args:", input.Args)
	}
}

func TestSwitchWorkspaceExisting(t *testing.T) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	terraformContainer, err := terraform.NewTerraformContainer(
		dockerClient,
		test.GetConfig("TEST_TERRAFORM_IMAGE"),
		test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
		releaseVolume,
	)
	defer terraformContainer.Done()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	workspaceName := "existing-workspace"

	if err := terraformContainer.SwitchWorkspace(
		workspaceName,
		&outputBuffer,
		&errorBuffer,
	); err != nil {
		log.Panicln("error switching workspace:", err)
	}

	lines := strings.Split(errorBuffer.String(), "\n")
	if len(lines) != 3 || lines[2] != "" {
		log.Panicln("expected two lines with a trailing newline (empty string), got lines:", lines)
	}

	var listInput ReflectedInput
	if err := json.Unmarshal([]byte(lines[0]), &listInput); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(listInput.Args, []string{"workspace", "list"}) {
		log.Panicln("unexpected args for workspace list:", listInput.Args)
	}

	var selectInput ReflectedInput
	if err := json.Unmarshal([]byte(lines[1]), &selectInput); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(selectInput.Args, []string{"workspace", "select", workspaceName}) {
		log.Panicln("unexpected args for workspace select:", selectInput.Args)
	}
}

func TestSwitchWorkspaceNew(t *testing.T) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	terraformContainer, err := terraform.NewTerraformContainer(
		dockerClient,
		test.GetConfig("TEST_TERRAFORM_IMAGE"),
		test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
		releaseVolume,
	)
	defer terraformContainer.Done()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	workspaceName := "new-workspace"

	if err := terraformContainer.SwitchWorkspace(
		workspaceName,
		&outputBuffer,
		&errorBuffer,
	); err != nil {
		log.Panicln("error switching workspace:", err)
	}

	lines := strings.Split(errorBuffer.String(), "\n")
	if len(lines) != 3 || lines[2] != "" {
		log.Panicln("expected two lines with a trailing newline (empty string), got lines:", lines)
	}

	var listInput ReflectedInput
	if err := json.Unmarshal([]byte(lines[0]), &listInput); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(listInput.Args, []string{"workspace", "list"}) {
		log.Panicln("unexpected args for workspace list:", listInput.Args)
	}

	var newInput ReflectedInput
	if err := json.Unmarshal([]byte(lines[1]), &newInput); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(newInput.Args, []string{"workspace", "new", workspaceName}) {
		log.Panicln("unexpected args for workspace new:", newInput.Args)
	}
}
