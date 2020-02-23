package command_test

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/manifest"
	release "github.com/mergermarket/cdflow2/release/command"
	"github.com/mergermarket/cdflow2/test"
	"github.com/mergermarket/cdflow2/util"
)

func TestRunCommand(t *testing.T) {

	// Given
	var outputBuffer util.Buffer
	var errorBuffer util.Buffer

	// When
	if err := release.RunCommand(
		&command.GlobalState{
			DockerClient: test.GetDockerClient(),
			Component:    "test-component",
			Commit:       "test-commit",
			OutputStream: &outputBuffer,
			ErrorStream:  &errorBuffer,
			CodeDir:      test.GetConfig("TEST_ROOT") + "/test/release/sample-code",
			Manifest: &manifest.Manifest{
				Version: 2,
				Team:    "test-team",
				Builds: map[string]manifest.Build{
					"release": manifest.Build{
						Image: test.GetConfig("TEST_RELEASE_IMAGE"),
					},
				},
				Terraform: manifest.Terraform{
					Image: test.GetConfig("TEST_TERRAFORM_IMAGE"),
				},
				Config: manifest.Config{
					Image: test.GetConfig("TEST_CONFIG_IMAGE"),
					Params: map[string]interface{}{
						"test-manifest-config-key": "test-manifest-config-value",
					},
				},
			},
			GlobalArgs: &command.GlobalArgs{
				NoPullConfig:    true,
				NoPullRelease:   true,
				NoPullTerraform: true,
			},
		},
		"test-version",
		map[string]string{},
	); err != nil {
		log.Fatalln("error running command:", err)
	}

	// Then
	lines := strings.Split(errorBuffer.String(), "\n")
	if len(lines) != 7 || lines[6] != "" {
		log.Panicln("expected six lines with a trailing newline (empty string), got lines:", lines)
	}

	test.CheckTerraformInitInitialReflectedInput([]byte(lines[0]))

	checkConfigureReleaseOutput(lines[1])

	if lines[2] != "message to stderr from release" {
		log.Panicln("unexpected output of release:", lines[2])
	}

	if lines[3] != "docker status: OK" {
		log.Panicln("unexpected output of release:", lines[2])
	}

	fmt.Println(lines[5])
	checkUploadReleaseOutput(lines[4])

	if lines[5] != "uploaded test-version" {
		log.Panic("expected 'uploaded test-version' message from config container, got:", lines[5])
	}
}

func checkConfigureReleaseOutput(debugOutput string) {
	var decoded struct {
		Action  string
		Request struct {
			Version string
			Config  map[string]interface{}
		}
	}

	if err := json.Unmarshal([]byte(debugOutput), &decoded); err != nil {
		log.Panicln("error decoding configure release debug output:", err)
	}

	if decoded.Action != "configure_release" {
		log.Panicln("unexpected action for configure releaes:", decoded.Action)
	}

	if decoded.Request.Version != "test-version" {
		log.Panicln("unexpected version passed to configure release:", decoded.Request.Version)
	}

	if decoded.Request.Config["test-manifest-config-key"] != "test-manifest-config-value" {
		log.Panicln("unexpected config value:", decoded.Request.Config["test-manifest-config-key"])
	}
}

func checkUploadReleaseOutput(debugOutput string) {
	var decoded struct {
		Action  string
		Request struct {
			TerraformImage string
		}
		ReleaseMetadata map[string]map[string]string
	}
	if err := json.Unmarshal([]byte(debugOutput), &decoded); err != nil {
		log.Panicln("error decoding upload release debug output:", err, "'"+debugOutput+"'")
	}

	if decoded.Action != "upload_release" {
		log.Panicln("unexpected action for upload releaes:", decoded.Action)
	}

	expectedTerraformImage := test.GetConfig("TEST_TERRAFORM_REPO_DIGEST")
	if decoded.Request.TerraformImage != expectedTerraformImage {
		log.Panicln("expected terraform repo digest: ", expectedTerraformImage, ", got:", decoded.Request.TerraformImage)
	}

	if decoded.ReleaseMetadata["release"]["component_from_defaults"] != "test-component" {
		log.Panicln("expected component test-component, got:", decoded.ReleaseMetadata["component_from_defaults"])
	}

	if decoded.ReleaseMetadata["release"]["commit_from_defaults"] != "test-commit" {
		log.Panicln("expected commit test-commit, got:", decoded.ReleaseMetadata["commit_from_defaults"])
	}

	if decoded.ReleaseMetadata["release"]["version"] != "test-version" {
		log.Panicln("unexpected version from release metadata:", decoded.ReleaseMetadata["release"]["version"])
	}
	if decoded.ReleaseMetadata["release"]["commit"] != "test-commit" {
		log.Panicln("unexpected commit from release metadata:", decoded.ReleaseMetadata["release"]["commit"])
	}
	if decoded.ReleaseMetadata["release"]["component"] != "test-component" {
		log.Panicln("unexpected component from release metadata:", decoded.ReleaseMetadata["release"]["component"])
	}
	if decoded.ReleaseMetadata["release"]["team"] != "test-team" {
		log.Panicln("unexpected team from release metadata:", decoded.ReleaseMetadata["release"]["team"])
	}
}
