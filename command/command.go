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

// GlobalArgs represents the global (non command specific) arguments.
type GlobalArgs struct {
	Command         string
	Component       string
	Commit          string
	NoPullConfig    bool
	NoPullRelease   bool
	NoPullTerraform bool
}

// GlobalState contains common to all commands.
type GlobalState struct {
	GlobalArgs   *GlobalArgs
	Component    string
	Commit       string
	CodeDir      string
	Manifest     *manifest.Canonical
	DockerClient *docker.Client
	OutputStream io.Writer
	ErrorStream  io.Writer
}

// GetGlobalState collects info common to every command.
func GetGlobalState(globalArgs *GlobalArgs) (*GlobalState, error) {
	var state GlobalState

	state.GlobalArgs = globalArgs

	var err error

	state.CodeDir, err = os.Getwd()
	if err != nil {
		return nil, err
	}

	state.Manifest, err = manifest.Load(state.CodeDir)
	if err != nil {
		return nil, err
	}
	if state.Manifest.Version != 2 {
		return nil, errors.New("cdflow.yaml version must be 2 for cdflow2")
	}
	state.DockerClient, err = docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	if globalArgs.Component == "" {
		state.Component, err = GetComponentFromGit()
		if err != nil {
			return nil, err
		}
	} else {
		state.Component = globalArgs.Component
	}

	if globalArgs.Commit == "" {
		state.Commit, err = GetCommitFromGit()
		if err != nil {
			return nil, err
		}
	} else {
		state.Commit = globalArgs.Commit
	}

	state.OutputStream = os.Stdout
	state.ErrorStream = os.Stderr

	return &state, nil
}

// ParseArgs takes arguments and splits them into global and remaining args.
func ParseArgs(args []string) (*GlobalArgs, []string, error) {
	var globalArgs GlobalArgs
	remainingArgs := []string{}
	i := 0
	for ; i < len(args); i++ {
		if args[i] == "-c" || args[i] == "--component" {
			if i+1 == len(args) {
				return nil, remainingArgs, errors.New("no value for component parameter")
			}
			globalArgs.Component = args[i+1]
			i++
		} else if args[i] == "--no-pull-config" {
			globalArgs.NoPullConfig = true
		} else if args[i] == "--no-pull-release" {
			globalArgs.NoPullRelease = true
		} else if args[i] == "--no-pull-terraform" {
			globalArgs.NoPullTerraform = true
		} else if args[i] == "help" || args[i] == "--help" || args[i] == "-h" {
			globalArgs.Command = "help"
			remainingArgs = args[i+1:]
			break
		} else if args[i] == "version" || args[i] == "--version" || args[i] == "-v" {
			globalArgs.Command = "version"
			remainingArgs = args[i+1:]
			break
		} else if strings.HasPrefix(args[i], "-") {
			return nil, remainingArgs, errors.New("Unknown global option: " + args[i])
		} else {
			globalArgs.Command = args[i]
			remainingArgs = args[i+1:]
			break
		}
	}
	return &globalArgs, remainingArgs, nil
}

// GetComponentFromGit gets the last part of the git repo name to use as a default component name.
func GetComponentFromGit() (string, error) {
	output, err := exec.Command("git", "config", "remote.origin.url").Output()
	if err != nil {
		return "", errors.New(
			"could not get component name from git (git config remote.origin.url): " + err.Error() + "\n" +
				"If git is not available you can pass the component name with the --component global option.\n",
		)
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
		return "", errors.New(
			"could not get commit from git (git rev-parse HEAD): " + err.Error() + "\n" +
				"If git is not available you can pass the commit with the --commit global option\n",
		)
	}
	return strings.TrimSpace(string(output)), nil
}
