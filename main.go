package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/release"
)

/*
	cdflow release:
		terraform container: terraform init (no config container dependency yet)
		config container: get environment for artefact build & publish (e.g. aws creds for docker push)
		release container: do build and publish
		config container: upload release archive

*/

const help string = `
Usage:

  cdflow2 COMMAND [ ARGS ]

Commands:

  release VERSION       - build and publish a new software artefact
  deploy ENV VERSION    - create & update infrastructure using software artefact
  help [ COMMAND ]      - displayed detailed help and usage information for a command
`

func main() {
	rand.Seed(time.Now().UnixNano())
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	codeDir, err := os.Getwd()
	if err != nil {
		log.Fatalln("could not get working directory:", err)
	}
	manifest, err := config.LoadManifest(codeDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading cdflow.yaml:", err)
	}
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatalln("could not initialise docker client:", err)
	}
	if command == "release" {
		if err := release.RunCommand(dockerClient, os.Stdout, os.Stderr, codeDir, os.Args[2:], manifest); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		fmt.Println(help)
		os.Exit(1)
	}
}
