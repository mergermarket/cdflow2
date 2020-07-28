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
	if result.EnvName == "" || result.Version == "" {
		return nil, false
	}
	return &result, true
}

// RunCommand runs the release command.
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

	if err := terraformContainer.ConfigureBackend(state.OutputStream, state.ErrorStream, prepareTerraformResponse, false); err != nil {
		return err
	}

	if err := terraformContainer.SwitchWorkspace(args.EnvName, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	// terraform container has been initialized
	// command specific stuff below

	planCommand := []string{
		"terraform",
		"plan",
		"-destroy",
		"-var-file=/build/release-metadata.json",
		"infra/",
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
		util.FormatCommand("terraform destroy -auto-approve"),
	)

	if err := terraformContainer.RunCommand(
		[]string{"terraform", "destroy", "-auto-approve", "-var-file=/build/release-metadata.json", "infra/"}, prepareTerraformResponse.Env,
		state.OutputStream, state.ErrorStream,
	); err != nil {
		return err
	}

	return nil
}