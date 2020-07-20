package shell

import (
	"fmt"
	"os"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
	"golang.org/x/crypto/ssh/terminal"
)

// CommandArgs contains specific arguments to the deploy command.
type CommandArgs struct {
	EnvName string
	Version string
}

// ParseArgs parses command line arguments to the shell subcommand.
func ParseArgs(args []string) (*CommandArgs, bool) {
	var result CommandArgs
	for i, arg := range args {
		if arg == "-v" || arg == "--version" {
			result.Version = args[i+1]
		}
		if result.EnvName == "" {
			result.EnvName = arg
		} else {
			return nil, false
		}
	}
	if result.EnvName == "" {
		return nil, false
	}
	return &result, true
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

	shellCommand := []string{
		"/bin/sh",
	}

	if err := terraformContainer.RunInteractiveCommand(shellCommand, prepareTerraformResponse.Env, os.Stdin, os.Stdout, os.Stderr); err != nil {
		return err
	}

	return nil
}
