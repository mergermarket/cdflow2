package setup_test

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/manifest"
	"github.com/mergermarket/cdflow2/monitoring"
	"github.com/mergermarket/cdflow2/setup"
	"github.com/mergermarket/cdflow2/test"
)

func TestRunCommand(t *testing.T) {
	// Given
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	dockerClient := test.GetDockerClient()

	// When
	if err := setup.RunCommand(
		&command.GlobalState{
			DockerClient: dockerClient,
			Component:    "test-component",
			Commit:       "test-commit",
			OutputStream: &outputBuffer,
			ErrorStream:  &errorBuffer,
			CodeDir:      test.GetConfig("TEST_ROOT") + "/test/release/sample-code",
			Manifest: &manifest.Manifest{
				Version: 2,
				Builds: map[string]manifest.ImageWithParamsAndEnvVars{
					"release": {
						Image: test.GetConfig("TEST_RELEASE_IMAGE"),
					},
				},
				Terraform: manifest.Terraform{
					Image: test.GetConfig("TEST_TERRAFORM_IMAGE"),
				},
				Config: manifest.ImageWithParams{
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
			MonitoringClient: monitoring.NewDatadogClient(),
		},
		map[string]string{},
	); err != nil {
		log.Fatalln("error running command:", err)
	}

	// Then
	if outputBuffer.String() != "output to stdout from setup, component: test-component, commit: test-commit\n" {
		log.Fatalln("unexpected output to stdout:", outputBuffer.String())
	}
	if !strings.Contains(errorBuffer.String(), "output to stderr from setup, needs: foo, bar\n") {
		log.Fatalln("unexpected output to stderr:", errorBuffer.String())
	}
}
