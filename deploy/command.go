package deploy

import (
	"log"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, envName, version string, env map[string]string) error {
	dockerClient := state.DockerClient

	terraformImage, buildVolume, err := config.SetupTerraform(state, envName, version, env)
	if err != nil {
		return err
	}

	defer func() {
		if err := dockerClient.RemoveVolume(buildVolume); err != nil {
			log.Panicln("error removing build release volume:", err)
		}
	}()

	terraformContainer, err := terraform.NewContainer(
		dockerClient,
		terraformImage,
		state.CodeDir,
		buildVolume,
		state.OutputStream,
		state.ErrorStream,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := terraformContainer.Done(); err != nil {
			log.Fatalln("error cleaning up terraform container:", err)
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
		"-var-file=config/test-env.json",
		"-out=" + planFilename,
		"infra/",
	}, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	if err := terraformContainer.RunCommand([]string{
		"terraform",
		"apply",
		"-input=false",
		planFilename,
	}, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	return nil
}
