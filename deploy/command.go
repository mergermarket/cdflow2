package deploy

import (
	"log"

	"github.com/docker/docker/api/types/volume"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, envName, version string, env map[string]string) error {
	// TODO too long, consider factoring some parts out

	if err := containers.MaybePullImage(!state.GlobalArgs.NoPullConfig, state, state.Manifest.Config.Image, "config"); err != nil {
		return err
	}

	buildVolume, err := state.DockerClient.VolumeCreate(state.DockerContext, volume.VolumeCreateBody{})
	if err != nil {
		return err
	}
	defer state.DockerClient.VolumeRemove(state.DockerContext, buildVolume.Name, false)

	configContainer := config.NewContainer(state, state.Manifest.Config.Image, buildVolume.Name, state.ErrorStream)
	if err := configContainer.Start(); err != nil {
		return err
	}
	defer func() {
		if err := configContainer.RequestStop(); err != nil {
			log.Panicln("error stopping config container:", err)
		}
		if err := configContainer.Remove(); err != nil {
			log.Panicln("error removing config container:", err)
		}
	}()

	prepareTerraformResponse, err := configContainer.PrepareTerraform(version, envName, state.Manifest.Config.Params, env)
	if err != nil {
		return err
	}

	if err := containers.MaybePullImage(!state.GlobalArgs.NoPullTerraform, state, state.Manifest.Terraform.Image, "terraform"); err != nil {
		return err
	}

	terraformContainer, err := terraform.NewContainer(
		state,
		prepareTerraformResponse.TerraformImage,
		state.CodeDir,
		buildVolume.Name,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := terraformContainer.Done(); err != nil {
			log.Fatalln("error stopping terraform container:", err)
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
