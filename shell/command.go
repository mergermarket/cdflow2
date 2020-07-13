package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// CommandArgs contains specific arguments to the deploy command.
type CommandArgs struct {
	EnvName string
	Version string
}

// ParseArgs parses command line arguments to the shell subcommand.
func ParseArgs(args []string) (*CommandArgs, bool) {
	var result CommandArgs
	for _, arg := range args {
		if result.EnvName == "" {
			result.EnvName = arg
		} else if result.Version == "" {
			result.Version = arg
		} else {
			return nil, false
		}
	}
	if result.EnvName == "" || result.Version == "" {
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
		prepareTerraformResponse.TerraformImage,
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

	if err := terraformContainer.ConfigureBackend(state.OutputStream, state.ErrorStream, prepareTerraformResponse); err != nil {
		return err
	}

	if err := terraformContainer.SwitchWorkspace(args.EnvName, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	planFilename := "/build/" + util.RandomName("plan")

	planCommand := []string{
		"terraform",
		"plan",
		"-var-file=/build/release-metadata.json",
	}

	envConfigFilename := "config/" + args.EnvName + ".json"
	if _, err := os.Stat(envConfigFilename); !os.IsNotExist(err) {
		planCommand = append(planCommand, "-var-file="+envConfigFilename)
	}

	planCommand = append(
		planCommand,
		"-out="+planFilename,
		"infra/",
	)

	fmt.Fprintf(
		state.ErrorStream,
		"\n%s\n%s\n\n",
		util.FormatInfo("creating plan"),
		util.FormatCommand(strings.Join(planCommand, " ")),
	)

	if err := terraformContainer.RunCommand(
		planCommand, prepareTerraformResponse.Env,
		state.OutputStream, state.ErrorStream,
	); err != nil {
		return err
	}

	shellCommand := []string{
		"/bin/sh",
	}

	if err := terraformContainer.RunInteractiveCommand(shellCommand, prepareTerraformResponse.Env, os.Stdin, os.Stdout, os.Stderr); err != nil {
		return err
	}

	return nil
}
