package deploy_test

import (
	"bytes"
	"log"
	"testing"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/deploy"
	"github.com/mergermarket/cdflow2/manifest"
	"github.com/mergermarket/cdflow2/test"
)

func TestRunCommand(t *testing.T) {

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	if err := deploy.RunCommand(
		&command.GlobalState{
			DockerClient: test.CreateDockerClient(),
			OutputStream: &outputBuffer,
			ErrorStream:  &errorBuffer,
			CodeDir:      test.GetConfig("TEST_ROOT") + "/test/release/sample-code",
			Component:    "test-component",
			Commit:       "test-commit",
			Manifest: &manifest.Manifest{
				Version:        2,
				ConfigImage:    test.GetConfig("TEST_CONFIG_IMAGE"),
				TerraformImage: test.GetConfig("TEST_TERRAFORM_IMAGE"),
			},
			NoPullConfig:    true,
			NoPullTerraform: true,
		},
		"test-env",
		"test-version",
	); err != nil {
		log.Fatalln("error running command:", err, errorBuffer.String())
	}
}
