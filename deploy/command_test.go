package deploy_test

import (
	"bytes"
	"encoding/json"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/deploy"
	"github.com/mergermarket/cdflow2/manifest"
	"github.com/mergermarket/cdflow2/test"
)

func TestRunCommand(t *testing.T) {

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	state := &command.GlobalState{
		DockerClient: test.GetDockerClient(),
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
			Config: manifest.Config{
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
		log.Panicln("could not get repo digests for terraform container:", err)
	}
	if len(repoDigests) == 0 {
		log.Panicln("no repo digests for terraform container", test.GetConfig("TEST_TERRAFORM_IMAGE"))
	}
	terraformDigest := repoDigests[0]
	if err := deploy.RunCommand(state, "test-env", "test-version", map[string]string{
		"TERRAFORM_DIGEST": terraformDigest,
	}); err != nil {
		log.Fatalln("error running deploy command:", err, errorBuffer.String())
	}

	lines := strings.Split(errorBuffer.String(), "\n")
	if len(lines) != 6 || lines[5] != "" {
		log.Panicln("expected five lines with a trailing newline (empty string), got lines:", len(lines))
	}

	checkPrepareTerraformOutput(lines[0])

	test.CheckTerraformWorkspaceList(lines[1])
	test.CheckTerraformWorkspaceNew(lines[2], "test-env")

	planFilename := checkTerraformPlanOutput(lines[3])
	checkTerraformApplyOutput(lines[4], planFilename)
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

func checkTerraformPlanOutput(output string) string {
	var input test.ReflectedInput
	if err := json.Unmarshal([]byte(output), &input); err != nil {
		log.Panicln("error parsing json:", err)
	}

	planFilename := strings.TrimPrefix(input.Args[4], "-out=")

	if !reflect.DeepEqual(input.Args, []string{
		"plan",
		"-input=false",
		"-var-file=/release/release-metadata.json",
		"-var-file=config/test-env.json",
		"-out=" + planFilename,
		"infra/",
	}) {
		log.Panicln("unexpected terraform plan args:", input.Args)
	}
	return planFilename
}

func checkTerraformApplyOutput(output, planFilename string) {
	var input test.ReflectedInput
	if err := json.Unmarshal([]byte(output), &input); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(input.Args, []string{
		"apply",
		"-input=false",
		planFilename,
	}) {
		log.Panicln("unexpected terraform apply args:", input.Args)
	}
}
