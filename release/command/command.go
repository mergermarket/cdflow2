package command

import (
	"fmt"
	"log"
	"os"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/release/container"
	"github.com/mergermarket/cdflow2/terraform"
	"github.com/mergermarket/cdflow2/util"
)

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, version string) error {
	// TODO too long, split this function
	if !state.GlobalArgs.NoPullTerraform {
		if err := state.DockerClient.PullImage(docker.PullImageOptions{
			Repository:   containers.ImageWithTag(state.Manifest.TerraformImage),
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}
	savedTerraformImage, err := containers.RepoDigest(state.DockerClient, state.Manifest.TerraformImage)
	if err != nil {
		return err
	}

	if state.GlobalArgs.NoPullTerraform && savedTerraformImage == "" {
		savedTerraformImage = state.Manifest.TerraformImage
	} else if savedTerraformImage == "" {
		log.Panicln("no repo digest for ", state.Manifest.TerraformImage)
	}

	buildVolume, err := state.DockerClient.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		return err
	}
	defer state.DockerClient.RemoveVolume(buildVolume.Name)

	if err := terraform.InitInitial(
		state.DockerClient,
		savedTerraformImage,
		state.CodeDir,
		buildVolume,
		state.OutputStream,
		state.ErrorStream,
	); err != nil {
		return err
	}

	if !state.GlobalArgs.NoPullConfig {
		if err := state.DockerClient.PullImage(docker.PullImageOptions{
			Repository:   containers.ImageWithTag(state.Manifest.ConfigImage),
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}

	configContainer := config.NewContainer(state.DockerClient, state.Manifest.ConfigImage, buildVolume, state.ErrorStream)
	if err := configContainer.Start(); err != nil {
		return err
	}
	defer func() {
		if err := configContainer.Remove(); err != nil {
			if err := configContainer.Stop(5); err != nil {
				log.Panicln("failed to remove and then to stop the container:", err)
			}
			if err := configContainer.Remove(); err != nil {
				log.Panicln("error removing config container:", err)
			}
		}
	}()

	configureReleaseResponse, err := configContainer.ConfigureRelease(version, state.Manifest.Config, util.GetEnv(os.Environ()))
	if err != nil {
		return err
	}

	releaseEnv := configureReleaseResponse.Env
	// these are built in and cannot be overridden by the config container (since choosing the clashing name would likely be an accident)
	releaseEnv["VERSION"] = version
	releaseEnv["TEAM"] = state.Manifest.Team
	releaseEnv["COMPONENT"] = state.Component
	releaseEnv["COMMIT"] = state.Commit

	releaseMetadata := make(map[string]map[string]string)
	for buildID, buildImage := range state.Manifest.Builds {
		metadata, err := container.Run(
			state.DockerClient,
			buildImage,
			state.CodeDir,
			buildVolume,
			state.OutputStream,
			state.ErrorStream,
			releaseEnv,
		)
		if err != nil {
			return err
		}
		metadata["version"] = version
		metadata["commit"] = state.Commit
		metadata["component"] = state.Component
		metadata["team"] = state.Manifest.Team
		releaseMetadata[buildID] = metadata
	}
	if _, ok := releaseMetadata["release"]; !ok {
		releaseMetadata["release"] = map[string]string{
			"version":   version,
			"commit":    state.Commit,
			"component": state.Component,
			"team":      state.Manifest.Team,
		}
	}

	if err := configContainer.WriteReleaseMetadata(releaseMetadata); err != nil {
		return err
	}

	uploadReleaseResponse, err := configContainer.UploadRelease(
		savedTerraformImage,
	)
	if err := configContainer.RequestStop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}

	fmt.Fprintln(state.ErrorStream, uploadReleaseResponse.Message)

	return nil
}
