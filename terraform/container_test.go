package terraform_test

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/test"
)

func TestTerraformInitInitial(t *testing.T) {
	// Given
	dockerClient := test.GetDockerClient()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	buildVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, buildVolume)

	// When
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

	// Then
	if outputBuffer.String() != "message to stdout\n" {
		log.Fatalf("unexpected stdout output: '%v'", outputBuffer.String())
	}

	test.CheckTerraformInitInitialReflectedInput(errorBuffer.Bytes())

	buildOutput, err := test.ReadVolume(dockerClient, buildVolume)
	if err != nil {
		log.Panicln("could not read build volume:", err)
	}

	if !reflect.DeepEqual(buildOutput, map[string]string{"build-output-test": "build output"}) {
		log.Panicln("unexpected build output:", buildOutput)
	}
}

func TestTerraformConfigureBackend(t *testing.T) {
	// Given
	dockerClient := test.GetDockerClient()

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	// When
	func {
		terraformContainer, err := terraform.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TERRAFORM_IMAGE"),
			test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
			releaseVolume,
			os.Stdout,
			os.Stderr,
		)
		if err != nil {
			log.Fatalln("error creating terraform container:", err)
		}
		defer func() {
			if err := terraformContainer.Done(); err != nil {
				log.Panicln("error cleaning up terraform container:", err)
			}
		}()

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
	}()

	// Then
	if outputBuffer.String() != "message to stdout\n" {
		log.Panicf("unexpected stdout output: '%v'", outputBuffer.String())
	}

	var input test.ReflectedInput
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
	// Given
	dockerClient := test.GetDockerClient()

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	workspaceName := "existing-workspace"

	// When
	func () {
		terraformContainer, err := terraform.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TERRAFORM_IMAGE"),
			test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
			releaseVolume,
			os.Stdout,
			os.Stderr,
		)
		if err != nil {
			log.Fatalln("error creating terraform container:", err)
		}

		defer func() {
			if err := terraformContainer.Done(); err != nil {
				log.Panicln("error cleaning up terraform container:", err)
			}
		}()

		if err := terraformContainer.SwitchWorkspace(
			workspaceName,
			&outputBuffer,
			&errorBuffer,
		); err != nil {
			log.Panicln("error switching workspace:", err)
		}
	}()

	// Then
	lines := strings.Split(errorBuffer.String(), "\n")
	if len(lines) != 3 || lines[2] != "" {
		log.Panicln("expected two lines with a trailing newline (empty string), got lines:", lines)
	}

	var listInput test.ReflectedInput
	if err := json.Unmarshal([]byte(lines[0]), &listInput); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(listInput.Args, []string{"workspace", "list"}) {
		log.Panicln("unexpected args for workspace list:", listInput.Args)
	}

	var selectInput test.ReflectedInput
	if err := json.Unmarshal([]byte(lines[1]), &selectInput); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(selectInput.Args, []string{"workspace", "select", workspaceName}) {
		log.Panicln("unexpected args for workspace select:", selectInput.Args)
	}
}

func TestSwitchWorkspaceNew(t *testing.T) {
	// When
	dockerClient := test.GetDockerClient()

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	workspaceName := "new-workspace"

	// When
	func () {
		terraformContainer, err := terraform.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TERRAFORM_IMAGE"),
			test.GetConfig("TEST_ROOT")+"/test/terraform/sample-code",
			releaseVolume,
			os.Stdout,
			os.Stderr,
		)
		if err != nil {
			log.Fatalln("error creating terraform container:", err)
		}
		defer func() {
			if err := terraformContainer.Done(); err != nil {
				log.Panicln("error cleaning up terraform container:", err)
			}
		}()

		if err := terraformContainer.SwitchWorkspace(
			workspaceName,
			&outputBuffer,
			&errorBuffer,
		); err != nil {
			log.Panicln("error switching workspace:", err)
		}
	}()

	// Then
	lines := strings.Split(errorBuffer.String(), "\n")
	if len(lines) != 3 || lines[2] != "" {
		log.Panicln("expected two lines with a trailing newline (empty string), got lines:", lines)
	}

	test.CheckTerraformWorkspaceList(lines[0])

	test.CheckTerraformWorkspaceNew(lines[1], workspaceName)
}
