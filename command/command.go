package command

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
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

  cdflow2 [ -c COMPONENT_NAME ] COMMAND [ ARGS ]

Commands:

  release VERSION       - build and publish a new software artefact
  deploy ENV VERSION    - create & update infrastructure using software artefact
  help [ COMMAND ]      - displayed detailed help and usage information for a command
`

// Run runs the cdflow2 command.
func Run() {
	rand.Seed(time.Now().UnixNano())

	codeDir, err := os.Getwd()
	if err != nil {
		log.Fatalln("could not get working directory:", err)
	}
	manifest, err := config.LoadManifest(codeDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading cdflow.yaml:", err)
	}
	if manifest.Version != 2 {
		fmt.Fprintf(os.Stderr, "cdflow.yaml version must be 2 for cdflow2")
		os.Exit(1)
	}
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatalln("could not initialise docker client:", err)
	}

	globalArgs, remainingArgs := ParseArgs(os.Args[1:])

	component := globalArgs.Component
	if component == "" {
		component, err = GetComponentFromGit()
		if err != nil {
			log.Fatalln("could not get component from git:", err)
		}
	}

	commit, err := GetCommitFromGit()
	if err != nil {
		log.Fatalln("could not get commit from git:", err)
	}

	if globalArgs.Command == "release" {
		if err := release.RunCommand(dockerClient, os.Stdout, os.Stderr, codeDir, component, commit, remainingArgs, manifest); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		fmt.Println(help)
		os.Exit(1)
	}
}

// GlobalArgs contains global arguments.
type GlobalArgs struct {
	Command   string
	Component string
}

// ParseArgs takes arguments and splits them into global and remaining args.
func ParseArgs(args []string) (*GlobalArgs, []string) {
	var globalArgs GlobalArgs
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" || args[i] == "--component" {
			if i+1 == len(args) {
				return &globalArgs, []string{}
			}
			globalArgs.Component = args[i+1]
			i++
		} else {
			globalArgs.Command = args[i]
			return &globalArgs, args[i+1:]
		}
	}
	return &globalArgs, []string{}
}

// GetComponentFromGit gets the last part of the git repo name to use as a default component name.
func GetComponentFromGit() (string, error) {
	output, err := exec.Command("git", "config", "remote.origin.url").Output()
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.TrimSpace(string(output)), "/")
	name := parts[len(parts)-1]
	if strings.HasSuffix(name, ".git") {
		name = name[:len(name)-4]
	}
	return name, nil
}

// GetCommitFromGit runs git in order to get the current commit.
func GetCommitFromGit() (string, error) {
	output, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
