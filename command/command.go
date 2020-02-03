package command

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/config"
)

// GlobalEnvironment contains common to all commands.
type GlobalEnvironment struct {
	Component       string
	Commit          string
	NoPullConfig    bool
	NoPullRelease   bool
	NoPullTerraform bool
	CodeDir         string
	Manifest        *config.Manifest
	DockerClient    *docker.Client
	OutputStream    io.Writer
	ErrorStream     io.Writer
}

// GetGlobalEnv collects info common to every command.
func GetGlobalEnv() (string, []string, *GlobalEnvironment) {
	var env GlobalEnvironment

	var err error

	env.CodeDir, err = os.Getwd()
	if err != nil {
		log.Fatalln("could not get working directory:", err)
	}

	env.Manifest, err = config.LoadManifest(env.CodeDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading cdflow.yaml:", err)
	}
	if env.Manifest.Version != 2 {
		fmt.Fprintf(os.Stderr, "cdflow.yaml version must be 2 for cdflow2")
		os.Exit(1)
	}
	env.DockerClient, err = docker.NewClientFromEnv()
	if err != nil {
		log.Fatalln("could not initialise docker client:", err)
	}

	command, remainingArgs := ParseArgs(os.Args[1:], &env)

	if env.Component == "" {
		env.Component, err = GetComponentFromGit()
		if err != nil {
			log.Fatalln("could not get component from git:", err)
		}
	}

	env.Commit, err = GetCommitFromGit()
	if err != nil {
		log.Fatalln("could not get commit from git:", err)
	}

	env.OutputStream = os.Stdout
	env.ErrorStream = os.Stderr

	return command, remainingArgs, &env
}

// ParseArgs takes arguments and splits them into global and remaining args.
func ParseArgs(args []string, env *GlobalEnvironment) (string, []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" || args[i] == "--component" {
			if i+1 == len(args) {
				break
			}
			env.Component = args[i+1]
			i++
		} else if args[i] == "--no-pull-config" {
			env.NoPullConfig = true
		} else if args[i] == "--no-pull-release" {
			env.NoPullRelease = true
		} else if args[i] == "--no-pull-terraform" {
			env.NoPullTerraform = true
		} else {
			return args[i], args[i+1:]
		}
	}
	return "", []string{}
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
