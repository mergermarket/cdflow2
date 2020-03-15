package container_test

import (
	"bytes"
	"log"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/release/container"
	"github.com/mergermarket/cdflow2/test"
)

func TestRelese(t *testing.T) {
	// Given
	dockerClient := test.GetDockerClient()

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	buildVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, buildVolume)

	codeDir := test.GetConfig("TEST_ROOT")+"/test/release/sample-code"

	// When
	releaseMetadata, err := container.Run(
		dockerClient,
		test.GetConfig("TEST_RELEASE_IMAGE"),
		codeDir,
		buildVolume,
		&outputBuffer,
		&errorBuffer,
		map[string]string{
			"VERSION":         "test-version",
			"TEAM":            "test-team",
			"COMPONENT":       "test_component",
			"COMMIT":          "test-commit",
			"BUILD_ID":        "test-build-id",
			"TEST_VERSION":    "test-version",
			"MANIFEST_PARAMS": "{}",
		},
	)
	if err != nil {
		log.Panicln("unexpected error: ", err)
	}

	// Then
	if errorBuffer.String() != "message to stderr from release\ndocker status: OK\n" {
		log.Panicf("unexpected stderr output: '%v'", errorBuffer.String())
	}

	if !reflect.DeepEqual(releaseMetadata, map[string]string{
		"release_var_from_env":    "release value from env",
		"version_from_defaults":   "test-version",
		"team_from_defaults":      "test-team",
		"component_from_defaults": "test_component",
		"commit_from_defaults":    "test-commit",
		"build_id_from_defaults":  "test-build-id",
		"test_from_config":        "test-version",
		"manifest_params":         "{}",
		"code_dir":                codeDir,
	}) {
		log.Panicf("unexpected release metadata: %v\n", releaseMetadata)
	}
}
