package release_test

import (
	"bytes"
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

func TestRunCommand(t *testing.T) {
	dockerClient := test.CreateDockerClient()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	if err := release.RunCommand(
		dockerClient,
		&outputBuffer,
		&errorBuffer,
		test.GetConfig("TEST_ROOT")+"/test/release/sample-code",
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
	if len(lines) != 3 || lines[2] != "" {
		log.Panicln("expected two lines with a trailing newline (empty string), got lines:", lines)
	}

	test.CheckTerraformInitInitialReflectedInput([]byte(lines[0]))

	if lines[1] != "configure_release" {
		log.Panicln("expected configure_release in config container, found:", lines[1])
	}
}
