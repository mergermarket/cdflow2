package deploy

import (
	"log"
	"os"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, envName, version string) error {
	// TODO too long, consider factoring some parts out
	if !state.NoPullConfig {
		if err := state.DockerClient.PullImage(docker.PullImageOptions{
			Repository:   containers.ImageWithTag(state.Manifest.ConfigImage),
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}

	buildVolume, err := state.DockerClient.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		return err
	}
	defer state.DockerClient.RemoveVolume(buildVolume.Name)

	configContainer := config.NewContainer(state.DockerClient, state.Manifest.ConfigImage, buildVolume, state.ErrorStream)
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

	prepareTerraformResponse, err := configContainer.PrepareTerraform(version, envName, state.Manifest.Config, util.GetEnv(os.Environ()))
	if err != nil {
		return err
	}

	if !state.NoPullTerraform {
		if err := state.DockerClient.PullImage(docker.PullImageOptions{
			Repository:   prepareTerraformResponse.TerraformImage,
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}

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
			log.Fatalln("error stopping terraform container:", err)
		}
	}()

	if err := terraformContainer.SwitchWorkspace(envName, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	if err := terraformContainer.RunCommand([]string{
		"terraform",
		"plan",
		"-input=false",
		"-var-file=release-metadata-VERSION.json",
		"-var-file=config/test-env.json",
		"-out=plan-TIMESTAMP",
		"infra/",
	}, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	if err := terraformContainer.RunCommand([]string{
		"terraform",
		"apply",
		"-input=false",
		"plan-TIMESTAMP",
	}, state.OutputStream, state.ErrorStream); err != nil {
		return err
	}

	return nil
}
