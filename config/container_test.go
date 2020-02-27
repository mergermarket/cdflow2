package config_test

import (
	"bytes"
	"encoding/json"
	"log"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/test"
)

func TestConfigRelease(t *testing.T) {
	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var errorBuffer bytes.Buffer

	var configureReleaseResponse *config.ConfigureReleaseConfigResponse
	var uploadReleaseResponse *config.UploadReleaseResponse

	// When
	func() {
		configContainer, err := config.NewContainer(dockerClient, test.GetConfig("TEST_CONFIG_IMAGE"), releaseVolume, &errorBuffer)
		if err != nil {
			log.Panicln("error creating config container:", err)
		}
		defer func() {
			if err := configContainer.Done(); err != nil {
				log.Panicln("error stopping config container:", err)
			}
		}()

		configureReleaseResponse, err = configContainer.ConfigureRelease(
			"test-version",
			map[string]interface{}{
				"TEST_CONFIG_VAR": "config value",
			},
			map[string]string{
				"TEST_ENV_VAR": "env value",
			},
		)
		if err != nil {
			log.Panicln("error in configureRelease:", err, errorBuffer.String())
		}

		configContainer.WriteReleaseMetadata(map[string]map[string]string{
			"release": {
				"metadata-key": "metadata-value",
			},
		})

		uploadReleaseResponse, err = configContainer.UploadRelease("terraform:image")
		if err != nil {
			log.Panicln("error in uploadRelease:", err)
		}

		if err := configContainer.RequestStop(); err != nil {
			log.Panicln("error stopping config container:", err)
		}
	}()

	// Then

	if !reflect.DeepEqual(configureReleaseResponse.Env, map[string]string{
		"TEST_VERSION":                 "test-version",
		"TEST_RELEASE_VAR_FROM_CONFIG": "config value",
		"TEST_RELEASE_VAR_FROM_ENV":    "env value",
	}) {
		log.Panicln("unexpected env in response:", configureReleaseResponse.Env)
	}

	if uploadReleaseResponse.Message != "uploaded test-version" {
		log.Panicln("unexpected message:", uploadReleaseResponse.Message)
	}

	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		log.Panicln("error getting debug info:", err)
	}

	var configureReleaseDebugOutput map[string]interface{}
	if err := json.Unmarshal(debugInfo["configure-release.json"], &configureReleaseDebugOutput); err != nil {
		log.Panicln("error decoding configure release debug output:", err)
	}

	if configureReleaseDebugOutput["Action"] != "configure_release" {
		log.Panicln("expected configure_release, got ", configureReleaseDebugOutput["Action"])
	}

	var uploadReleaseDebugOutput map[string]interface{}
	if err := json.Unmarshal(debugInfo["upload-release.json"], &uploadReleaseDebugOutput); err != nil {
		log.Panicln("error decoding upload release debug output:", err)
	}

	if uploadReleaseDebugOutput["Action"] != "upload_release" {
		log.Panicln("expected upload_release, got ", uploadReleaseDebugOutput["Action"])
	}
}

func TestConfigDeploy(t *testing.T) {
	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	releaseVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, releaseVolume)

	var errorBuffer bytes.Buffer
	var prepareTerraformResponse *config.PrepareTerraformResponse

	// When
	func() {
		configContainer, err := config.NewContainer(dockerClient, test.GetConfig("TEST_CONFIG_IMAGE"), releaseVolume, &errorBuffer)
		if err != nil {
			log.Panicln("error creating config container:", err)
		}
		defer func() {
			if err := configContainer.Done(); err != nil {
				log.Panicln("error stopping config container:", err)
			}
		}()

		prepareTerraformResponse, err = configContainer.PrepareTerraform(
			"test-version",
			"test-env",
			map[string]interface{}{
				"TEST_CONFIG_VAR": "config value",
			},
			map[string]string{
				"TEST_ENV_VAR":     "env value",
				"TERRAFORM_DIGEST": "test terraform image digest",
			},
		)
		if err != nil {
			log.Panicln(err)
		}

		if err := configContainer.RequestStop(); err != nil {
			log.Panicln("error stopping config container:", err)
		}
	}()

	// Then
	if !reflect.DeepEqual(prepareTerraformResponse.Env, map[string]string{
		"TEST_ENV_VAR":    "env value",
		"TEST_CONFIG_VAR": "config value",
	}) {
		log.Panicln("unexpected env:", prepareTerraformResponse.Env)
	}

	if prepareTerraformResponse.TerraformImage != "test terraform image digest" {
		log.Panicln("unexpected terraform image:", prepareTerraformResponse.TerraformImage)
	}

	if prepareTerraformResponse.TerraformBackendType != "a-terraform-backend-type" {
		log.Panicln("unexpected terraform backend type:", prepareTerraformResponse.TerraformBackendType)
	}

	if !reflect.DeepEqual(prepareTerraformResponse.TerraformBackendConfig, map[string]string{"backend-config-key": "backend-config-value"}) {
		log.Panicln("unexpected terraform backend config:", prepareTerraformResponse.TerraformBackendConfig)
	}

	releaseData, err := test.ReadVolume(dockerClient, releaseVolume)
	if err != nil {
		log.Panicln("could not read release volume:", err)
	}

	if !reflect.DeepEqual(releaseData, map[string][]byte{"test": []byte("unpacked")}) {
		log.Panicln("unexpected release data:", releaseData)
	}

	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		log.Panicln("error getting debug info:", err)
	}

	var prepareTerraformDebugOutput struct {
		Action  string
		Request struct {
			EnvName string
		}
	}

	if err := json.Unmarshal(debugInfo["prepare-terraform.json"], &prepareTerraformDebugOutput); err != nil {
		log.Panicln("error decoding prepare terraform debug output:", err)
	}

	if prepareTerraformDebugOutput.Action != "prepare_terraform" {
		log.Panicln("expected prepare_terraform, got ", prepareTerraformDebugOutput.Action)
	}

	if prepareTerraformDebugOutput.Request.EnvName != "test-env" {
		log.Panicln("expected env name test-env passed to prepare terraform, got:", prepareTerraformDebugOutput.Request.EnvName)
	}
}
