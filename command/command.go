package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/mergermarket/cdflow2/docker"
	"github.com/mergermarket/cdflow2/docker/official"
	"github.com/mergermarket/cdflow2/manifest"
)

// Failure represents a non-zero exit status without the need for further output.
type Failure int

// Error outputs and empty string - the reason for failure will have already been output to the user.
func (Failure) Error() string {
	return ""
}

// GlobalArgs represents the global (non command specific) arguments.
type GlobalArgs struct {
	Command         string
	Component       string
	Commit          string
	NoPullConfig    bool
	NoPullRelease   bool
	NoPullTerraform bool
	Quiet           bool
}

// GlobalState contains common to all commands.
type GlobalState struct {
	GlobalArgs   *GlobalArgs
	Component    string
	Commit       string
	CodeDir      string
	Manifest     *manifest.Manifest
	InputStream  io.Reader
	OutputStream io.Writer
	ErrorStream  io.Writer
	DockerClient docker.Iface
}

// GetGlobalState collects info common to every command.
func GetGlobalState(globalArgs *GlobalArgs, repoShouldExist bool) (*GlobalState, error) {
	var state GlobalState

	state.GlobalArgs = globalArgs

	var err error

	state.CodeDir, err = os.Getwd()
	if err != nil {
		return nil, err
	}

	state.InputStream = os.Stdin
	state.OutputStream = os.Stdout
	state.ErrorStream = os.Stderr

	dockerClient, err := official.NewClient()
	if err != nil {
		return nil, fmt.Errorf("error creating docker client: %w", err)
	}
	state.DockerClient = dockerClient

	if repoShouldExist {
		state.Manifest, err = manifest.Load(state.CodeDir)
		if err != nil {
			return nil, err
		}
		if state.Manifest.Version != 2 {
			return nil, errors.New("cdflow.yaml version must be 2 for cdflow2")
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
	}

	return &state, nil
}

func handleArg(arg string, globalArgs *GlobalArgs, take func() (string, error)) (bool, error) {
	if strings.HasPrefix(arg, "-") {
		if handleSimpleFlag(arg, globalArgs) {
			return false, nil
		}
		return handleFlag(arg, globalArgs, take)
	}
	globalArgs.Command = arg
	return true, nil
}

func handleSimpleFlag(arg string, globalArgs *GlobalArgs) bool {
	if arg == "--no-pull-config" {
		globalArgs.NoPullConfig = true
		return true
	} else if arg == "--no-pull-release" {
		globalArgs.NoPullRelease = true
		return true
	} else if arg == "--no-pull-terraform" {
		globalArgs.NoPullTerraform = true
		return true
	} else if arg == "--quiet" || arg == "-q" {
		globalArgs.Quiet = true
		return true
	}
	return false
}

func handleFlag(arg string, globalArgs *GlobalArgs, take func() (string, error)) (bool, error) {
	if arg == "-c" || arg == "--component" {
		value, err := take()
		if err != nil {
			return false, err
		}
		globalArgs.Component = value
	} else if strings.HasPrefix(arg, "--component=") {
		globalArgs.Component = strings.TrimPrefix(arg, "--component=")
	} else if arg == "--commit" {
		value, err := take()
		if err != nil {
			return false, err
		}
		globalArgs.Commit = value
	} else if strings.HasPrefix(arg, "--commit=") {
		globalArgs.Commit = strings.TrimPrefix(arg, "--commit=")
	} else if arg == "--help" || arg == "-h" {
		globalArgs.Command = "help"
		return true, nil
	} else if arg == "--version" || arg == "-v" {
		globalArgs.Command = "version"
		return true, nil
	} else {
		return false, errors.New("Unknown global option: " + arg)
	}
	return false, nil
}

// ParseArgs takes arguments and splits them into global and remaining args.
func ParseArgs(args []string) (*GlobalArgs, []string, error) {
	var globalArgs GlobalArgs
	remainingArgs := []string{}
	i := 0
	take := func() (string, error) {
		i++
		if i >= len(args) {
			return "", errors.New("missing value")
		}

		return args[i], nil
	}
	for ; i < len(args); i++ {
		done, err := handleArg(args[i], &globalArgs, take)
		if err != nil {
			return nil, remainingArgs, err
		}
		if done {
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
