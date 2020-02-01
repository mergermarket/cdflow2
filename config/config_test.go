package config_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"reflect"
	"strings"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/test"
)

func removeConfigContainer(configContainer *config.Container) {
	configContainer.Stop(5)
	if err := configContainer.Remove(); err != nil {
		log.Panicln("could not remove config container:", err)
	}
}

func setupConfigContainer(errorStream io.Writer) (*docker.Client, *config.Container, *docker.Volume) {
	dockerClient := test.CreateDockerClient()

	releaseVolume := test.CreateVolume(dockerClient)

	configContainer := config.NewContainer(dockerClient, test.GetConfig("TEST_CONFIG_IMAGE"), releaseVolume, errorStream)

	if err := configContainer.Start(); err != nil {
		log.Panicln("error running config container:", err)
	}
	return dockerClient, configContainer, releaseVolume
}

func TestConfigRelease(t *testing.T) {
	var errorBuffer bytes.Buffer
	dockerClient, configContainer, releaseVolume := setupConfigContainer(&errorBuffer)
	defer test.RemoveVolume(dockerClient, releaseVolume)
	defer removeConfigContainer(configContainer)

	response, err := configContainer.ConfigureRelease(
		"test-version",
		map[string]interface{}{
			"TEST_CONFIG_VAR": "config value",
		},
		map[string]string{
			"TEST_ENV_VAR": "env value",
		},
	)
	if err != nil {
		log.Panicln("error in configureRelease:", err)
	}

	if !reflect.DeepEqual(response.Env, map[string]string{
		"TEST_VERSION":                 "test-version",
		"TEST_RELEASE_VAR_FROM_CONFIG": "config value",
		"TEST_RELEASE_VAR_FROM_ENV":    "env value",
	}) {
		log.Panicln("unexpected env in response:", response.Env)
	}

	uploadReleaseResponse, err := configContainer.UploadRelease(
		"terraform:image",
		map[string]string{
			"metadata-key": "metadata-value",
		},
	)
	if err != nil {
		log.Panicln("error in uploadRelease:", err)
	}

	if uploadReleaseResponse.Message != "uploaded test-version" {
		log.Panicln("unexpected message:", uploadReleaseResponse.Message)
	}

	if err := configContainer.RequestStop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}

	lines := strings.Split(errorBuffer.String(), "\n")
	if len(lines) != 3 || lines[2] != "" {
		log.Panicln("expected two lines with a trailing newline (empty string), got lines:", lines)
	}

	var configureReleaseDebugOutput map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &configureReleaseDebugOutput); err != nil {
		log.Panicln("error decoding configure release debug output:", err)
	}

	if configureReleaseDebugOutput["Action"] != "configure_release" {
		log.Panicln("expected configure_release, got ", configureReleaseDebugOutput["Action"])
	}

	var uploadReleaseDebugOutput map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &uploadReleaseDebugOutput); err != nil {
		log.Panicln("error decoding upload release debug output:", err)
	}

	if uploadReleaseDebugOutput["Action"] != "upload_release" {
		log.Panicln("expected upload_release, got ", uploadReleaseDebugOutput["Action"])
	}
}

func TestConfigDeploy(t *testing.T) {
	var errorBuffer bytes.Buffer
	dockerClient, configContainer, releaseVolume := setupConfigContainer(&errorBuffer)
	defer test.RemoveVolume(dockerClient, releaseVolume)
	defer removeConfigContainer(configContainer)

	response, err := configContainer.PrepareTerraform(
		"test-version",
		map[string]interface{}{
			"TEST_CONFIG_VAR": "config value",
		},
		map[string]string{
			"TEST_ENV_VAR": "env value",
		},
	)
	if err != nil {
		log.Panicln(err)
	}

	if !reflect.DeepEqual(response.Env, map[string]string{
		"TEST_ENV_VAR":    "env value",
		"TEST_CONFIG_VAR": "config value",
	}) {
		log.Panicln("unexpected env:", response.Env)
	}

	if response.TerraformImage != "terraform:image-for-test-version" {
		log.Panicln("unexpected terraform image:", response.TerraformImage)
	}

	if response.TerraformBackendType != "a-terraform-backend-type" {
		log.Panicln("unexpected terraform backend type:", response.TerraformBackendType)
	}

	if !reflect.DeepEqual(response.TerraformBackendConfig, map[string]string{"backend-config-key": "backend-config-value"}) {
		log.Panicln("unexpected terraform backend config:", response.TerraformBackendConfig)
	}

	releaseData, err := test.ReadVolume(dockerClient, releaseVolume)
	if err != nil {
		log.Panicln("could not read release volume:", err)
	}

	if !reflect.DeepEqual(releaseData, map[string]string{"test": "unpacked"}) {
		log.Panicln("unexpected release data:", releaseData)
	}

	if err := configContainer.RequestStop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}

	var prepareTerraformDebugOutput map[string]interface{}
	if err := json.Unmarshal(errorBuffer.Bytes(), &prepareTerraformDebugOutput); err != nil {
		log.Panicln("error decoding prepare terraform debug output:", err)
	}

	if prepareTerraformDebugOutput["Action"] != "prepare_terraform" {
		log.Panicln("expected prepare_terraform, got ", prepareTerraformDebugOutput["Action"])
	}
}
