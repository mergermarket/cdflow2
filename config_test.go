package main

import (
	"log"
	"reflect"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
)

func createDockerClient() *docker.Client {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		log.Panicln(err)
	}
	return client
}

func createVolume(dockerClient *docker.Client) *docker.Volume {
	volume, err := dockerClient.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		log.Panicln("could not create volume:", err)
	}
	return volume
}

func removeVolume(dockerClient *docker.Client, volume *docker.Volume) {
	if err := dockerClient.RemoveVolume(volume.Name); err != nil {
		log.Panicf("error removing volume %v: %v", volume.Name, err)
	}
}

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

	env, err := configContainer.configureRelease(
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

	if !reflect.DeepEqual(env, map[string]string{
		"TEST_VERSION":                 "test-version",
		"TEST_RELEASE_VAR_FROM_CONFIG": "config value",
		"TEST_RELEASE_VAR_FROM_ENV":    "env value",
	}) {
		log.Panicln("unexpected env in response:", env)
	}

	if err := configContainer.uploadRelease("terraform:image"); err != nil {
		log.Panicln("error in uploadRelease:", err)
	}

	if err := configContainer.stop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}
}

func TestConfigDeploy(t *testing.T) {

	dockerClient, configContainer, buildVolume := setupConfigContainer()
	defer removeVolume(dockerClient, buildVolume)
	defer removeConfigContainer(configContainer)

	terraformImage, env, err := configContainer.prepareTerraform("test-version")
	if err != nil {
		log.Panicln(err)
	}

	if !reflect.DeepEqual(env, map[string]string{"EnvKey": "EnvValue"}) {
		log.Panicln("unexpected env:", env)
	}

	if terraformImage != "terraform:image-for-test-version" {
		log.Panicln("unexpected terraform image:", terraformImage)
	}

	if err := configContainer.stop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}
}
