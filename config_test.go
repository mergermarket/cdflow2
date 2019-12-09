package main

import (
	"log"
	"os"
	"reflect"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
)

func TestConfig(t *testing.T) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	buildDir, err := tempdir()
	if err != nil {
		log.Fatalf("could not make tempdir: %v", err)
	}
	defer os.RemoveAll(buildDir)

	configContainer := NewConfigContainer(dockerClient, getConfig("TEST_CONFIG_IMAGE"), buildDir)

	if err := configContainer.start(); err != nil {
		log.Fatalf("error running config container: %v", err)
	}

	env, err := configContainer.configureRelease(
		map[string]interface{}{
			"TEST_CONFIG_VAR": "config value",
		},
		map[string]string{
			"TEST_ENV_VAR": "env value",
		},
	)
	if err != nil {
		log.Fatalf("error in configureRelease: %v", err)
	}

	if !reflect.DeepEqual(env, map[string]string{"TEST_RELEASE_VAR_FROM_CONFIG": "config value", "TEST_RELEASE_VAR_FROM_ENV": "env value"}) {
		log.Fatalf("unexpected env in response: %v", env)
	}

	if err := configContainer.stop(); err != nil {
		log.Fatalf("error stopping config container: %v", err)
	}
}
