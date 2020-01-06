package config_test

import (
	"log"
	"reflect"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/test"
)

func removeConfigContainer(configContainer *config.ConfigContainer) {
	configContainer.StopContainer(5)
	if err := configContainer.Remove(); err != nil {
		log.Panicln("could not remove config container:", err)
	}
}

func setupConfigContainer() (*docker.Client, *config.ConfigContainer, *docker.Volume) {
	dockerClient := test.CreateDockerClient()

	releaseVolume := test.CreateVolume(dockerClient)

	configContainer := config.NewConfigContainer(dockerClient, test.GetConfig("TEST_CONFIG_IMAGE"), releaseVolume)

	if err := configContainer.Start(); err != nil {
		log.Panicln("error running config container:", err)
	}
	return dockerClient, configContainer, releaseVolume
}

func TestConfigRelease(t *testing.T) {
	dockerClient, configContainer, releaseVolume := setupConfigContainer()
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

	if err := configContainer.Stop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}
}

func TestConfigDeploy(t *testing.T) {

	dockerClient, configContainer, releaseVolume := setupConfigContainer()
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

	if err := configContainer.Stop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}
}
