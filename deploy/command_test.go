package deploy_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/deploy"
	"github.com/mergermarket/cdflow2/manifest"
	"github.com/mergermarket/cdflow2/test"
)

func TestRunCommand(t *testing.T) {

	// Given
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	state := &command.GlobalState{
		DockerClient: dockerClient,
		OutputStream: &outputBuffer,
		ErrorStream:  &errorBuffer,
		CodeDir:      test.GetConfig("TEST_ROOT") + "/test/release/sample-code",
		Component:    "test-component",
		Commit:       "test-commit",
		Manifest: &manifest.Manifest{
			Version: 2,
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
			NoPullTerraform: true,
		},
	}

	repoDigests, err := state.DockerClient.GetImageRepoDigests(test.GetConfig("TEST_TERRAFORM_IMAGE"))
	if err != nil {
		t.Fatal("could not get repo digests for terraform container:", err)
	}
	if len(repoDigests) == 0 {
		t.Fatal("no repo digests for terraform container", test.GetConfig("TEST_TERRAFORM_IMAGE"))
	}
	terraformDigest := repoDigests[0]

	// When
	if err := deploy.RunCommand(state, "test-env", "test-version", map[string]string{
		"TERRAFORM_DIGEST": terraformDigest,
	}); err != nil {
		t.Fatal("error running deploy command:", err, errorBuffer.String())
	}

	// Then
	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		t.Fatal("error getting debug info:", err)
	}

	checkPrepareTerraformOutput(t, debugInfo["prepare-terraform.json"])

	lines := bytes.Split(debugInfo["terraform"], []byte{'\n'})
	if len(lines) != 6 || len(lines[5]) != 0 {
		t.Fatalf("expected five lines with a trailing newline (empty string), got %v lines:\n%v", len(lines), test.DumpLines(lines))
	}

	// TODO check terraform init

	test.CheckTerraformWorkspaceList(lines[1])
	test.CheckTerraformWorkspaceNew(lines[2], "test-env")

	planFilename := checkTerraformPlanOutput(t, lines[3])
	checkTerraformApplyOutput(t, lines[4], planFilename)
}

func checkPrepareTerraformOutput(t *testing.T, debugOutput []byte) {
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

	if err := json.Unmarshal(debugOutput, &decoded); err != nil {
		t.Fatal("error decoding prepare terraform debug output:", err)
	}

	if decoded.Action != "prepare_terraform" {
		t.Fatal("expected action prepare_terraform got:", decoded.Action)
	}
	if decoded.Request.Version != "test-version" {
		t.Fatal("expected version test-version got:", decoded.Request.Version)
	}
	if decoded.Request.EnvName != "test-env" {
		t.Fatal("expected env test-env got:", decoded.Request.EnvName)
	}
	if decoded.Request.Config["test-manifest-config-key"] != "test-manifest-config-value" {
		t.Fatal("expected config from manifest got:", decoded.Request.Config)
	}
	if decoded.PWD != "/release" {
		t.Fatal("expected prepare_terraform to run in /release got:", decoded.PWD)
	}

}

func checkTerraformPlanOutput(t *testing.T, output []byte) string {
	var input test.ReflectedInput
	if err := json.Unmarshal(output, &input); err != nil {
		t.Fatal("error parsing json:", err)
	}

	planFilename := strings.TrimPrefix(input.Args[2], "-out=")

	if !reflect.DeepEqual(input.Args, []string{
		"plan",
		"-var-file=/build/release-metadata.json",
		"-out=" + planFilename,
		"infra/",
	}) {
		t.Fatal("unexpected terraform plan args:", input.Args)
	}
	return planFilename
}

func checkTerraformApplyOutput(t *testing.T, output []byte, planFilename string) {
	var input test.ReflectedInput
	if err := json.Unmarshal(output, &input); err != nil {
		t.Fatal("error parsing json:", err)
	}

	if !reflect.DeepEqual(input.Args, []string{
		"apply",
		planFilename,
	}) {
		t.Fatal("unexpected terraform apply args:", input.Args)
	}
}
