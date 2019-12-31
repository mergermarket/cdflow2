package main

import (
	"log"
	"reflect"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
)

func removeConfigContainer(configContainer *configContainer) {
	configContainer.stopContainer(5)
	if err := configContainer.removeContainer(); err != nil {
		log.Panicln("could not remove config container:", err)
	}
}

func setupConfigContainer() (*docker.Client, *configContainer, *docker.Volume) {
	dockerClient := createDockerClient()

	buildVolume := createVolume(dockerClient)

	configContainer := NewConfigContainer(dockerClient, getConfig("TEST_CONFIG_IMAGE"), buildVolume)

	if err := configContainer.start(); err != nil {
		log.Panicln("error running config container:", err)
	}
	return dockerClient, configContainer, buildVolume
}

func TestConfigRelease(t *testing.T) {
	dockerClient, configContainer, buildVolume := setupConfigContainer()
	defer removeVolume(dockerClient, buildVolume)
	defer removeConfigContainer(configContainer)

	response, err := configContainer.configureRelease(
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

	uploadReleaseResponse, err := configContainer.uploadRelease(
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

	if err := configContainer.stop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}
}

func TestConfigDeploy(t *testing.T) {

	dockerClient, configContainer, buildVolume := setupConfigContainer()
	defer removeVolume(dockerClient, buildVolume)
	defer removeConfigContainer(configContainer)

	response, err := configContainer.prepareTerraform(
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

	if err := configContainer.stop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}
}
