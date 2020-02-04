package deploy_test

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/deploy"
	"github.com/mergermarket/cdflow2/manifest"
	"github.com/mergermarket/cdflow2/test"
)

func TestRunCommand(t *testing.T) {

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	dockerClient := test.CreateDockerClient()

	terraformDigest, err := containers.RepoDigest(dockerClient, test.GetConfig("TEST_TERRAFORM_IMAGE"))
	if err != nil {
		log.Panicln("could not get repo digest for terraform container:", err)
	}

	if err := deploy.RunCommand(
		&command.GlobalState{
			DockerClient: dockerClient,
			OutputStream: &outputBuffer,
			ErrorStream:  &errorBuffer,
			CodeDir:      test.GetConfig("TEST_ROOT") + "/test/release/sample-code",
			Component:    "test-component",
			Commit:       "test-commit",
			Manifest: &manifest.Manifest{
				Version:        2,
				ConfigImage:    test.GetConfig("TEST_CONFIG_IMAGE"),
				TerraformImage: test.GetConfig("TEST_TERRAFORM_IMAGE"),
				Config: map[string]interface{}{
					"test-manifest-config-key": "test-manifest-config-value",
					"terraform-digest":         terraformDigest,
				},
			},
			NoPullConfig:    true,
			NoPullTerraform: true,
		},
		"test-env",
		"test-version",
	); err != nil {
		log.Fatalln("error running deploy command:", err, errorBuffer.String())
	}

	lines := strings.Split(errorBuffer.String(), "\n")
	if len(lines) != 4 || lines[3] != "" {
		log.Panicln("expected three lines with a trailing newline (empty string), got lines:", len(lines))
	}

	checkPrepareTerraformOutput(lines[0])

	test.CheckTerraformWorkspaceList(lines[1])
	test.CheckTerraformWorkspaceNew(lines[2], "test-env")
}

func checkPrepareTerraformOutput(debugOutput string) {
	var decoded struct {
		Action  string
		Request struct {
			Version string
			EnvName string
			Config  map[string]interface{}
			Env     map[string]string
		}
		PWD string
	}

	if err := json.Unmarshal([]byte(debugOutput), &decoded); err != nil {
		log.Panicln("error decoding prepare terraform debug output:", err)
	}

	if decoded.Action != "prepare_terraform" {
		log.Panicln("expected action prepare_terraform got:", decoded.Action)
	}
	if decoded.Request.Version != "test-version" {
		log.Panicln("expected version test-version got:", decoded.Request.Version)
	}
	if decoded.Request.EnvName != "test-env" {
		log.Panicln("expected env test-env got:", decoded.Request.EnvName)
	}
	if decoded.Request.Config["test-manifest-config-key"] != "test-manifest-config-value" {
		log.Panicln("expected config from manifest got:", decoded.Request.Config)
	}
	if decoded.PWD != "/release" {
		log.Panicln("expected prepare_terraform to run in /release got:", decoded.PWD)
	}

}
