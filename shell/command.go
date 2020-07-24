package shell

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
	"golang.org/x/crypto/ssh/terminal"
)

// CommandArgs contains specific arguments to the deploy command.
type CommandArgs struct {
	EnvName   string
	Version   string
	ShellArgs []string
}

func handleArgs(arg string, commandArgs *CommandArgs, take func() (string, error)) (bool, error) {
	if arg == "--" {
		return true, nil
	} else if strings.HasPrefix(arg, "-") {
		return handleFlag(arg, commandArgs, take)
	}
	commandArgs.EnvName = arg
	return false, nil
}

func handleFlag(arg string, commandArgs *CommandArgs, take func() (string, error)) (bool, error) {
	if arg == "-v" || arg == "--version" {
		value, err := take()
		if err != nil {
			return false, err
		}
		commandArgs.Version = value
	} else if strings.HasPrefix(arg, "--version=") {
		commandArgs.Version = strings.TrimPrefix(arg, "--version=")
	}
	return false, nil
}

// ParseArgs parses command line arguments to the shell subcommand.
func ParseArgs(args []string) (*CommandArgs, error) {
	var result CommandArgs
	i := 0
	take := func() (string, error) {
		if i > len(args) {
			return "", errors.New("missing value")
		}
		i++
		return args[i], nil
	}
	for ; i < len(args); i++ {
		done, err := handleArgs(args[i], &result, take)
		if err != nil {
			return nil, err
		}
		if done {
			result.ShellArgs = args[i+1:]
			break
		}
	}
	if result.EnvName == "" {
		return nil, errors.New("Env missing value")
	}
	return &result, nil
}

// RunCommand runs the shell command.
func RunCommand(state *command.GlobalState, args *CommandArgs, env map[string]string) (returnedError error) {
	prepareTerraformResponse, buildVolume, err := config.SetupTerraform(state, args.EnvName, args.Version, env)
	if err != nil {
		return err
	}

	defer func() {
		if err := state.DockerClient.RemoveVolume(buildVolume); err != nil {
			if returnedError != nil {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
		}
	}()

	terraformContainer, err := terraform.NewContainer(
		state.DockerClient,
		state.Manifest.Terraform.Image,
		state.CodeDir,
		buildVolume,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := terraformContainer.Done(); err != nil {
			if returnedError != nil {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
		}
	}()

	if err := terraformContainer.ConfigureBackend(state.OutputStream, state.ErrorStream, prepareTerraformResponse, true); err != nil {
		return err
	}

	if err := terraformContainer.SwitchWorkspace(args.EnvName, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	oldState, e := terminal.MakeRaw(int(os.Stdin.Fd()))
	if e != nil {
		return e
	}
	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }()

	shellCommand := []string{"/bin/sh"}

	shellCommandandArgs := append(shellCommand, args.ShellArgs...)

	if err := terraformContainer.RunInteractiveCommand(shellCommandandArgs, prepareTerraformResponse.Env, os.Stdin, os.Stdout, os.Stderr); err != nil {
		return err
	}

	return nil
}
