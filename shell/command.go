package shell

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
)

// CommandArgs contains specific arguments to the deploy command.
type CommandArgs struct {
	EnvName          string
	Version          string
	ShellArgs        []string
	StateShouldExist *bool
}

func isTty(stream os.File) bool {
	stat, _ := stream.Stat()
	if stat.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return true
}

func handleArgs(arg string, commandArgs *CommandArgs, take func() (string, error)) (bool, error) {
	if arg == "--" {
		return true, nil
	} else if strings.HasPrefix(arg, "-") {
		return handleFlag(arg, commandArgs, take)
	} else if commandArgs.EnvName == "" {
		commandArgs.EnvName = arg
		return false, nil
	} else {
		commandArgs.ShellArgs = append(commandArgs.ShellArgs, arg)
	}
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
	} else {
		return false, errors.New("Unknown global option: " + arg)
	}
	return false, nil
}

// ParseArgs parses command line arguments to the shell subcommand.
func ParseArgs(args []string) (*CommandArgs, error) {
	var result CommandArgs
	var T = true
	result.StateShouldExist = &T // set default to true
	i := 0
	take := func() (string, error) {
		i++
		if i >= len(args) {
			return "", errors.New("missing value")
		}

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
	prepareTerraformResponse, buildVolume, terraformImage, err := config.SetupTerraform(state, args.StateShouldExist, args.EnvName, args.Version, env)
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
		terraformImage,
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

	shellCommand := []string{"/bin/sh"}
	shellCommandWithArgs := append(shellCommand, args.ShellArgs...)

	interactive := false
	if len(args.ShellArgs) < 1 {
		interactive = true
	}

	tty := isTty(*os.Stdin)

	if err := terraformContainer.RunInteractiveCommand(
		shellCommandWithArgs,
		prepareTerraformResponse.Env,
		state.InputStream,
		state.OutputStream,
		state.ErrorStream,
		tty,
		interactive); err != nil {
		return err
	}

	return nil
}
