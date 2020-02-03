package release_test

import (
	"bytes"
	"encoding/json"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/release"
	"github.com/mergermarket/cdflow2/test"
)

func TestRelese(t *testing.T) {
	dockerClient := test.CreateDockerClient()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	buildVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, buildVolume)

	releaseMetadata, err := release.Run(
		dockerClient,
		test.GetConfig("TEST_RELEASE_IMAGE"),
		test.GetConfig("TEST_ROOT")+"/test/release/sample-code",
		buildVolume,
		&outputBuffer,
		&errorBuffer,
		map[string]string{
			"VERSION":      "test-version",
			"TEAM":         "test-team",
			"COMPONENT":    "test_component",
			"COMMIT":       "test-commit",
			"TEST_VERSION": "test-version",
		},
	)
	if err != nil {
		log.Panicln("unexpected error: ", err)
	}

	if errorBuffer.String() != "message to stderr from release\n" {
		log.Panicf("unexpected stderr output: '%v'", errorBuffer.String())
	}
	if errorBuffer.String() != "message to stderr from release\n" {
		log.Panicf("unexpected stderr output: '%v'", errorBuffer.String())
	}

	if !reflect.DeepEqual(releaseMetadata, map[string]string{
		"release_var_from_env":    "release value from env",
		"version_from_defaults":   "test-version",
		"team_from_defaults":      "test-team",
		"component_from_defaults": "test_component",
		"commit_from_defaults":    "test-commit",
		"test_from_config":        "test-version",
	}) {
		log.Panicf("unexpected release metadata: %v\n", releaseMetadata)
	}
}

func TestParseArgsDefaults(t *testing.T) {
	args, err := release.ParseArgs([]string{"test-version"})
	if err != nil {
		log.Fatalln("error parsing empty args:", err)
	}
	if args.NoPullConfig {
		log.Fatalln("default for --no-pull-config true when it should be false")
	}
	if args.NoPullRelease {
		log.Fatalln("default for --no-pull-release true when it should be false")
	}
	if args.NoPullTerraform {
		log.Fatalln("default for --no-pull-terraform true when it should be false")
	}
}

func TestParseArgsNoPullConfig(t *testing.T) {
	args, err := release.ParseArgs([]string{"test-version", "--no-pull-config"})
	if err != nil {
		log.Fatalln("error parsing --no-pull-config args:", err)
	}
	if !args.NoPullConfig {
		log.Fatalln("--no-pull-config should be true")
	}
}

func TestParseArgsNoPullRelease(t *testing.T) {
	args, err := release.ParseArgs([]string{"--no-pull-release", "test-version"})
	if err != nil {
		log.Fatalln("error parsing --no-pull-release args:", err)
	}
	if !args.NoPullRelease {
		log.Fatalln("--no-pull-release should be true")
	}
}

func TestParseArgsNoPullTerraform(t *testing.T) {
	args, err := release.ParseArgs([]string{"test-version", "--no-pull-terraform"})
	if err != nil {
		log.Fatalln("error parsing --no-pull-terraform args:", err)
	}
	if !args.NoPullTerraform {
		log.Fatalln("--no-pull-terraform should be true")
	}
}

func checkConfigureReleaseOutput(debugOutput string) {
	var decoded struct {
		Action  string
		Request map[string]interface{}
	}

	if err := json.Unmarshal([]byte(debugOutput), &decoded); err != nil {
		log.Panicln("error decoding configure release debug output:", err)
	}

	if decoded.Action != "configure_release" {
		log.Panicln("unexpected action for configure releaes:", decoded.Action)
	}

	if decoded.Request["Version"] != "test-version" {
		log.Panicln("unexpected version passed to configure release:", decoded.Request["Version"])
	}
}

func checkUploadReleaseOutput(debugOutput string) {
	var decoded struct {
		Action  string
		Request struct {
			ReleaseMetadata map[string]string
			TerraformImage  string
		}
	}
	if err := json.Unmarshal([]byte(debugOutput), &decoded); err != nil {
		log.Panicln("error decoding upload release debug output:", err)
	}

	if decoded.Action != "upload_release" {
		log.Panicln("unexpected action for upload releaes:", decoded.Action)
	}

	expectedTerraformImage := test.GetConfig("TEST_TERRAFORM_REPO_DIGEST")
	if decoded.Request.TerraformImage != expectedTerraformImage {
		log.Panicln("expected terraform repo digest: ", expectedTerraformImage, ", got:", decoded.Request.TerraformImage)
	}

	if decoded.Request.ReleaseMetadata["component_from_defaults"] != "test-component" {
		log.Panicln("expected component test-component, got:", decoded.Request.ReleaseMetadata["component_from_defaults"])
	}

	if decoded.Request.ReleaseMetadata["commit_from_defaults"] != "test-commit" {
		log.Panicln("expected commit test-commit, got:", decoded.Request.ReleaseMetadata["commit_from_defaults"])
	}
}

func TestRunCommand(t *testing.T) {
	dockerClient := test.CreateDockerClient()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	if err := release.RunCommand(
		dockerClient,
		&outputBuffer,
		&errorBuffer,
		test.GetConfig("TEST_ROOT")+"/test/release/sample-code",
		"test-component",
		"test-commit",
		[]string{"test-version", "--no-pull-release", "--no-pull-config", "--no-pull-terraform"},
		&config.Manifest{
			Version:        2,
			ReleaseImage:   test.GetConfig("TEST_RELEASE_IMAGE"),
			ConfigImage:    test.GetConfig("TEST_CONFIG_IMAGE"),
			TerraformImage: test.GetConfig("TEST_TERRAFORM_IMAGE"),
		},
	); err != nil {
		log.Fatalln("error running command:", err, errorBuffer.String())
	}

	lines := strings.Split(errorBuffer.String(), "\n")
	if len(lines) != 6 || lines[5] != "" {
		log.Panicln("expected six lines with a trailing newline (empty string), got lines:", len(lines))
	}

	test.CheckTerraformInitInitialReflectedInput([]byte(lines[0]))

	checkConfigureReleaseOutput(lines[1])

	if lines[2] != "message to stderr from release" {
		log.Panicln("unexpected output of release:", lines[2])
	}

	checkUploadReleaseOutput(lines[3])

	if lines[4] != "uploaded test-version" {
		log.Panic("expected 'uploaded test-version' message from config container, got:", lines[4])
	}
}
