package main

import (
	"bufio"
	"io"
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

	errReader, errWriter := io.Pipe()
	errScanner := bufio.NewScanner(errReader)
	if err := configContainer.start(errWriter); err != nil {
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

	if err := configContainer.uploadRelease("terraform:image"); err != nil {
		log.Fatalln("error in uploadRelease: ", err)
	}

	if !errScanner.Scan() {
		log.Fatalln("could not read from stderr: ", errScanner.Err())
	}
	if errScanner.Text() != "uploading release (terraform:image)" {
		log.Fatalf("unexpected output to stderr: '%s'", errScanner.Text())
	}

	if err := configContainer.stop(); err != nil {
		log.Fatalf("error stopping config container: %v", err)
	}

}
