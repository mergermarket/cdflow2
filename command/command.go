package command

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/manifest"
)

// GlobalState contains common to all commands.
type GlobalState struct {
	Component       string
	Commit          string
	NoPullConfig    bool
	NoPullRelease   bool
	NoPullTerraform bool
	CodeDir         string
	Manifest        *manifest.Manifest
	DockerClient    *docker.Client
	OutputStream    io.Writer
	ErrorStream     io.Writer
}

// GetGlobalState collects info common to every command.
func GetGlobalState() (string, []string, *GlobalState, error) {
	var state GlobalState

	var err error

	state.CodeDir, err = os.Getwd()
	if err != nil {
		return "", []string{}, nil, err
	}

	state.Manifest, err = manifest.Load(state.CodeDir)
	if err != nil {
		return "", []string{}, nil, err
	}
	if state.Manifest.Version != 2 {
		return "", []string{}, nil, errors.New("cdflow.yaml version must be 2 for cdflow2")
	}
	state.DockerClient, err = docker.NewClientFromEnv()
	if err != nil {
		return "", []string{}, nil, err
	}

	command, remainingArgs := ParseArgs(os.Args[1:], &state)

	if state.Component == "" {
		state.Component, err = GetComponentFromGit()
		if err != nil {
			return "", []string{}, nil, err
		}
	}

	state.Commit, err = GetCommitFromGit()
	if err != nil {
		return "", []string{}, nil, err
	}

	state.OutputStream = os.Stdout
	state.ErrorStream = os.Stderr

	return command, remainingArgs, &state, nil
}

// ParseArgs takes arguments and splits them into global and remaining args.
func ParseArgs(args []string, state *GlobalState) (string, []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" || args[i] == "--component" {
			if i+1 == len(args) {
				break
			}
			state.Component = args[i+1]
			i++
		} else if args[i] == "--no-pull-config" {
			state.NoPullConfig = true
		} else if args[i] == "--no-pull-release" {
			state.NoPullRelease = true
		} else if args[i] == "--no-pull-terraform" {
			state.NoPullTerraform = true
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
		return "", errors.New("could not get component name from git (git config remote.origin.url): " + err.Error())
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
		return "", errors.New("could not get commit from git (git rev-parse HEAD): " + err.Error())
	}
	return strings.TrimSpace(string(output)), nil
}
