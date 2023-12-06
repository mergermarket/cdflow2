package deploy

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// CommandArgs contains specific arguments to the deploy command.
type CommandArgs struct {
	EnvName                string
	Version                string
	PlanOnly               bool
	TerraformLogLevel      string
	StateShouldExist       *bool
	ErrorOnResourceDestroy bool
}

// ParseArgs parses command line arguments to the deploy subcommand.
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
		_, err := handleArgs(args[i], &result, take)
		if err != nil {
			return nil, err
		}
	}

	if result.EnvName == "" {
		return nil, errors.New("env argument is missing")
	}

	if result.Version == "" {
		return nil, errors.New("version argument is missing")
	}

	return &result, nil
}

func handleArgs(arg string, commandArgs *CommandArgs, take func() (string, error)) (bool, error) {
	if strings.HasPrefix(arg, "-") {
		return handleFlag(arg, commandArgs, take)
	} else if commandArgs.EnvName == "" {
		commandArgs.EnvName = arg
	} else if commandArgs.Version == "" {
		commandArgs.Version = arg
	} else {
		return false, errors.New("unknown deploy argument: " + arg)
	}
	return false, nil
}

func handleFlag(arg string, commandArgs *CommandArgs, take func() (string, error)) (bool, error) {
	var F = false

	if arg == "-p" || arg == "--plan-only" {
		commandArgs.PlanOnly = true
	} else if arg == "-n" || arg == "--new-state" {
		commandArgs.StateShouldExist = &F
	} else if arg == "-e" || arg == "--error-on-destroy" {
		commandArgs.ErrorOnResourceDestroy = true
	} else if arg == "-t" || arg == "--terraform-log-level" {
		value, err := take()
		if err != nil {
			return false, err
		}

		commandArgs.TerraformLogLevel = value
	} else {
		return false, errors.New("unknown deploy option: " + arg)
	}
	return false, nil
}

func hasResourceDelete(plan string) bool {
	return !strings.Contains(plan, "0 to destroy")
}

// RunCommand runs the release command.
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
		args.TerraformLogLevel,
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

	if err := terraformContainer.CopyTerraformLockIfExists(state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	if err := terraformContainer.ConfigureBackend(state.OutputStream, state.ErrorStream, prepareTerraformResponse, false); err != nil {
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

	commonConfigFile := "config/common.json"
	if _, err := os.Stat(commonConfigFile); !os.IsNotExist(err) {
		planCommand = append(planCommand, "-var-file=../"+commonConfigFile)
	}

	envConfigFilename := "config/" + args.EnvName + ".json"
	if _, err := os.Stat(envConfigFilename); !os.IsNotExist(err) {
		planCommand = append(planCommand, "-var-file=../"+envConfigFilename)
	}

	planCommand = append(
		planCommand,
		"-out="+planFilename,
	)

	fmt.Fprintf(
		state.ErrorStream,
		"\n%s\n%s\n\n",
		util.FormatInfo("creating plan"),
		util.FormatCommand(strings.Join(planCommand, " ")),
	)
	var planBuff strings.Builder
	multiWriter := io.MultiWriter(state.OutputStream, &planBuff)

	if err := terraformContainer.RunCommand(
		planCommand, prepareTerraformResponse.Env,
		multiWriter, state.ErrorStream,
	); err != nil {
		return err
	}

	if args.ErrorOnResourceDestroy {
		if hasResourceDelete(planBuff.String()) {
			return errors.New("the plan contains resources to be deleted")
		}
	}

	if args.PlanOnly {
		return nil
	}

	fmt.Fprintf(
		state.ErrorStream,
		"\n%s\n%s\n",
		util.FormatInfo("applying plan"),
		util.FormatCommand("terraform apply "+planFilename),
	)

	if err := terraformContainer.RunCommand(
		[]string{"terraform", "apply", planFilename}, prepareTerraformResponse.Env,
		state.OutputStream, state.ErrorStream,
	); err != nil {
		return err
	}

	return nil
}
