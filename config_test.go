package main

import (
	"log"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	/*
		dockerClient, err := docker.NewClientFromEnv()
		if err != nil {
			log.Fatal(err)
		}
	*/
	buildDir, err := tempdir()
	if err != nil {
		log.Fatalf("could not make tempdir: %v", err)
	}
	defer os.RemoveAll(buildDir)

	/*
		configIn := make(chan string)
		configOut := make(chan string)

		eg := errgroup.Group{}

			//terraformInit(dockerClient, manifest.TerraformImage, dir)
			configContainer := createConfigContainer(dockerClient, getConfig("TEST_CONFIG_IMAGE"), buildDir)
			eg.Go(func() error {
				return awaitConfigContainer(dockerClient, configContainer, configIn, configOut)
			})
	*/
}
