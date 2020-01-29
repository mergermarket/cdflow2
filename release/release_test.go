package release_test

import (
	"bytes"
	"log"
	"reflect"
	"testing"

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
	args, err := release.ParseArgs([]string{})
	if err != nil {
		log.Fatalln("error parsing empty args:", err)
	}
	if *args.NoPullConfig {
		log.Fatalln("default for --no-pull-config true when it should be false")
	}
	if *args.NoPullRelease {
		log.Fatalln("default for --no-pull-release true when it should be false")
	}
	if *args.NoPullTerraform {
		log.Fatalln("default for --no-pull-terraform true when it should be false")
	}
}

func TestParseArgsNoPullConfig(t *testing.T) {
	args, err := release.ParseArgs([]string{"--no-pull-config"})
	if err != nil {
		log.Fatalln("error parsing --no-pull-config args:", err)
	}
	if !*args.NoPullConfig {
		log.Fatalln("--no-pull-config should be true")
	}
}

func TestParseArgsNoPullRelease(t *testing.T) {
	args, err := release.ParseArgs([]string{"--no-pull-release"})
	if err != nil {
		log.Fatalln("error parsing --no-pull-release args:", err)
	}
	if !*args.NoPullRelease {
		log.Fatalln("--no-pull-release should be true")
	}
}

func TestParseArgsNoPullTerraform(t *testing.T) {
	args, err := release.ParseArgs([]string{"--no-pull-terraform"})
	if err != nil {
		log.Fatalln("error parsing --no-pull-terraform args:", err)
	}
	if !*args.NoPullTerraform {
		log.Fatalln("--no-pull-terraform should be true")
	}
}
