package destroy

import (
	"fmt"
	"strings"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// CommandArgs contains specific arguments to the deploy command.
type CommandArgs struct {
	EnvName  string
	Version  string
	PlanOnly bool
}

// ParseArgs parses command line arguments to the deploy subcommand.
func ParseArgs(args []string) (*CommandArgs, bool) {
	var result CommandArgs
	for _, arg := range args {
		if arg == "-p" || arg == "--plan-only" {
			result.PlanOnly = true
		} else if result.EnvName == "" {
			result.EnvName = arg
		} else if result.Version == "" {
			result.Version = arg
		} else {
			return nil, false
		}
	}
	if result.EnvName == "" {
		return nil, false
	}
	return &result, true
}

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, args *CommandArgs, env map[string]string) (returnedError error) {
	prepareTerraformResponse, buildVolume, terraformImage, err := config.SetupTerraform(state, args.EnvName, args.Version, env)
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

	planCommand := []string{
		"terraform",
		"plan",
		"-destroy",
		"infra/",
	}

	destroyCommand := []string{
		"terraform",
		"destroy",
		"-auto-approve",
		"infra/",
	}

	if args.Version != "" {
		planCommand = append(
			planCommand[:len(planCommand)-1],
			"-var-file=/build/release-metadata.json",
			"infra/",
		)
		destroyCommand = append(
			destroyCommand[:len(destroyCommand)-1],
			"-var-file=/build/release-metadata.json",
			"infra/",
		)
	}

	fmt.Fprintf(
		state.ErrorStream,
		"\n%s\n%s\n\n",
		util.FormatInfo("generating plan"),
		util.FormatCommand(strings.Join(planCommand, " ")),
	)

	if err := terraformContainer.RunCommand(
		planCommand, prepareTerraformResponse.Env,
		state.OutputStream, state.ErrorStream,
	); err != nil {
		return err
	}

	if args.PlanOnly {
		return nil
	}

	fmt.Fprintf(
		state.ErrorStream,
		"\n%s\n%s\n",
		util.FormatInfo("applying plan"),
		util.FormatCommand(strings.Join(destroyCommand, " ")),
	)

	if err := terraformContainer.RunCommand(
		destroyCommand, prepareTerraformResponse.Env,
		state.OutputStream, state.ErrorStream,
	); err != nil {
		return err
	}

	return nil
}
