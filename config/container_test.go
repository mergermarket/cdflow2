package config_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/test"
)

func TestConfigRelease(t *testing.T) {
	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	state := &command.GlobalState{
		DockerClient: dockerClient,
		OutputStream: &outputBuffer,
		ErrorStream:  &errorBuffer,
	}

	var configureReleaseResponse *config.ConfigureReleaseConfigResponse
	var uploadReleaseResponse *config.UploadReleaseResponse

	// When
	func() {
		configContainer, err := config.NewContainer(state, test.GetConfig("TEST_CONFIG_IMAGE"), releaseVolume)
		if err != nil {
			t.Fatal("error creating config container:", err)
		}
		defer func() {
			if err := configContainer.Done(); err != nil {
				t.Fatal("error stopping config container:", err)
			}
		}()

		configureReleaseResponse, err = configContainer.ConfigureRelease(
			"test-version",
			"test-component",
			"test-commit",
			map[string]interface{}{
				"TEST_CONFIG_VAR": "config value",
			},
			map[string]string{
				"TEST_ENV_VAR": "env value",
			},
			map[string]*config.ReleaseRequirements{
				"release": {
					Needs: []string{"need1"},
				},
			},
		)
		if err != nil {
			t.Fatal("error in configureRelease:", err, errorBuffer.String())
		}

		configContainer.WriteReleaseMetadata(map[string]map[string]string{
			"release": {
				"metadata-key": "metadata-value",
			},
		})

		uploadReleaseResponse, err = configContainer.UploadRelease("terraform:image")
		if err != nil {
			t.Fatal("error in uploadRelease:", err)
		}
	}()

	// Then

	if !reflect.DeepEqual(configureReleaseResponse.Env, map[string]map[string]string{
		"release": {
			"TEST_VERSION":                 "test-version",
			"TEST_COMPONENT":               "test-component",
			"TEST_COMMIT":                  "test-commit",
			"TEST_RELEASE_VAR_FROM_CONFIG": "config value",
			"TEST_RELEASE_VAR_FROM_ENV":    "env value",
		},
	}) {
		t.Fatal("unexpected env in response:", configureReleaseResponse.Env)
	}

	if uploadReleaseResponse.Message != "uploaded test-version" {
		t.Fatal("unexpected message:", uploadReleaseResponse.Message)
	}

	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		t.Fatal("error getting debug info:", err)
	}

	var configureReleaseDebugOutput struct {
		Action              string
		ReleaseRequirements map[string]struct {
			Needs []string
		}
	}
	if err := json.Unmarshal(debugInfo["configure-release.json"], &configureReleaseDebugOutput); err != nil {
		t.Fatal("error decoding configure release debug output:", err)
	}

	if configureReleaseDebugOutput.Action != "configure_release" {
		t.Fatal("expected configure_release, got ", configureReleaseDebugOutput.Action)
	}

	needs := configureReleaseDebugOutput.ReleaseRequirements["release"].Needs
	if !reflect.DeepEqual(needs, []string{"need1"}) {
		t.Fatal("unexpected release needs:", configureReleaseDebugOutput.ReleaseRequirements["release"])
	}

	var uploadReleaseDebugOutput map[string]interface{}
	if err := json.Unmarshal(debugInfo["upload-release.json"], &uploadReleaseDebugOutput); err != nil {
		t.Fatal("error decoding upload release debug output:", err)
	}

	if uploadReleaseDebugOutput["Action"] != "upload_release" {
		t.Fatal("expected upload_release, got ", uploadReleaseDebugOutput["Action"])
	}
}

func TestConfigDeploy(t *testing.T) {
	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer
	var stateShouldExist = true

	state := &command.GlobalState{
		DockerClient: dockerClient,
		OutputStream: &outputBuffer,
		ErrorStream:  &errorBuffer,
	}

	var prepareTerraformResponse *config.PrepareTerraformResponse

	// When
	func() {
		configContainer, err := config.NewContainer(state, test.GetConfig("TEST_CONFIG_IMAGE"), releaseVolume)
		if err != nil {
			t.Fatal("error creating config container:", err)
		}
		defer func() {
			if err := configContainer.Done(); err != nil {
				t.Fatal("error stopping config container:", err)
			}
		}()

		prepareTerraformResponse, err = configContainer.PrepareTerraform(
			"test-version",
			"test-component",
			"test-commit",
			"test-env",
			&stateShouldExist,
			map[string]interface{}{
				"TEST_CONFIG_VAR": "config value",
			},
			map[string]string{
				"TEST_ENV_VAR":     "env value",
				"TERRAFORM_DIGEST": "test terraform image digest",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Then
	if !reflect.DeepEqual(prepareTerraformResponse.Env, map[string]string{
		"TEST_ENV_VAR":    "env value",
		"TEST_CONFIG_VAR": "config value",
	}) {
		t.Fatal("unexpected env:", prepareTerraformResponse.Env)
	}

	if prepareTerraformResponse.TerraformImage != "test terraform image digest" {
		t.Fatal("unexpected terraform image:", prepareTerraformResponse.TerraformImage)
	}

	if prepareTerraformResponse.TerraformBackendType != "a-terraform-backend-type" {
		t.Fatal("unexpected terraform backend type:", prepareTerraformResponse.TerraformBackendType)
	}

	if !reflect.DeepEqual(prepareTerraformResponse.TerraformBackendConfig, map[string]string{"backend-config-key": "backend-config-value"}) {
		t.Fatal("unexpected terraform backend config:", prepareTerraformResponse.TerraformBackendConfig)
	}

	releaseData, err := test.ReadVolume(dockerClient, releaseVolume)
	if err != nil {
		t.Fatal("could not read release volume:", err)
	}

	if !reflect.DeepEqual(releaseData, map[string][]byte{"test": []byte("unpacked")}) {
		t.Fatal("unexpected release data:", releaseData)
	}

	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		t.Fatal("error getting debug info:", err)
	}

	var prepareTerraformDebugOutput struct {
		Action  string
		Request struct {
			EnvName string
		}
	}

	if err := json.Unmarshal(debugInfo["prepare-terraform.json"], &prepareTerraformDebugOutput); err != nil {
		t.Fatal("error decoding prepare terraform debug output:", err)
	}

	if prepareTerraformDebugOutput.Action != "prepare_terraform" {
		t.Fatal("expected prepare_terraform, got ", prepareTerraformDebugOutput.Action)
	}

	if prepareTerraformDebugOutput.Request.EnvName != "test-env" {
		t.Fatal("expected env name test-env passed to prepare terraform, got:", prepareTerraformDebugOutput.Request.EnvName)
	}
}
