package deploy

import (
	"fmt"
	"log"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, envName, version string, env map[string]string) error {
	// TODO too long, consider factoring some parts out

	dockerClient := state.DockerClient

	if !state.GlobalArgs.NoPullConfig {
		if err := dockerClient.PullImage(state.Manifest.Config.Image, state.ErrorStream); err != nil {
			return fmt.Errorf("error pulling config image: %w", err)
		}
	}

	buildVolume, err := dockerClient.CreateVolume()
	if err != nil {
		return err
	}
	defer func() {
		if err := dockerClient.RemoveVolume(buildVolume); err != nil {
			log.Panicln("error removing build release volume:", err)
		}
	}()

	terraformImage, err := prepareTerraform(state, envName, version, buildVolume, env)
	if err != nil {
		return err
	}

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

func prepareTerraform(state *command.GlobalState, envName, version, buildVolume string, env map[string]string) (string, error) {
	dockerClient := state.DockerClient

	configContainer, err := config.NewContainer(dockerClient, state.Manifest.Config.Image, buildVolume, state.ErrorStream)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := configContainer.RequestStop(); err != nil {
			log.Panicln("error stopping config container:", err)
		}
		if err := configContainer.Done(); err != nil {
			log.Panicln("error cleaning up config container:", err)
		}
	}()

	prepareTerraformResponse, err := configContainer.PrepareTerraform(version, envName, state.Manifest.Config.Params, env)
	if err != nil {
		return "", err
	}

	if !state.GlobalArgs.NoPullTerraform {
		if err := dockerClient.EnsureImage(prepareTerraformResponse.TerraformImage, state.ErrorStream); err != nil {
			return "", fmt.Errorf("error pulling terraform image %v: %w", prepareTerraformResponse.TerraformImage, err)
		}
	}
	return prepareTerraformResponse.TerraformImage, nil
}
