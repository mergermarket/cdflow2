package deploy

import (
	"fmt"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, envName, version string, env map[string]string) (returnedError error) {
	terraformImage, buildVolume, err := config.SetupTerraform(state, envName, version, env)
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

	if err := terraformContainer.SwitchWorkspace(envName, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	planFilename := util.RandomName("plan")

	if err := terraformContainer.RunCommand([]string{
		"terraform",
		"plan",
		"-input=false",
		"-var-file=/release/release-metadata.json",
		"-var-file=config/" + envName + ".json",
		"-out=" + planFilename,
		"infra/",
	}, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	if err := terraformContainer.RunCommand(
		[]string{"terraform", "apply", "-input=false", planFilename},
		state.OutputStream, state.ErrorStream,
	); err != nil {
		return err
	}

	return nil
}
