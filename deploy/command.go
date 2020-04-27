package deploy

import (
	"fmt"
	"os"
	"strings"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, envName, version string, env map[string]string) (returnedError error) {
	prepareTerraformResponse, buildVolume, err := config.SetupTerraform(state, envName, version, env)
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

	terraformContainer.ConfigureBackend(state.OutputStream, state.ErrorStream, prepareTerraformResponse.TerraformBackendConfig)

	if err := terraformContainer.SwitchWorkspace(envName, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	planFilename := "/build/" + util.RandomName("plan")

	planCommand := []string{
		"terraform",
		"plan",
		"-var-file=/build/release-metadata.json",
	}

	envConfigFilename := "config/" + envName + ".json"
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
		"\nCreating plan...\n\n$ %s\n",
		strings.Join(planCommand, " "),
	)

	if err := terraformContainer.RunCommand(planCommand, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	fmt.Fprintf(
		state.ErrorStream,
		"\nApplying plan...\n\n$ terraform apply %s\n",
		planFilename,
	)

	if err := terraformContainer.RunCommand(
		[]string{"terraform", "apply", planFilename},
		state.OutputStream, state.ErrorStream,
	); err != nil {
		return err
	}

	return nil
}
