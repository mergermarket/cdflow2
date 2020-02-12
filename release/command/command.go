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
			Repository:   containers.ImageWithTag(state.Manifest.Terraform.Image),
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}
	savedTerraformImage, err := containers.RepoDigest(state.DockerClient, state.Manifest.Terraform.Image)
	if err != nil {
		return err
	}

	if state.GlobalArgs.NoPullTerraform && savedTerraformImage == "" {
		savedTerraformImage = state.Manifest.Terraform.Image
	} else if savedTerraformImage == "" {
		log.Panicln("no repo digest for ", state.Manifest.Terraform.Image)
	}

	buildVolume, err := state.DockerClient.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		return err
	}
	//defer state.DockerClient.RemoveVolume(buildVolume.Name)

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
			Repository:   containers.ImageWithTag(state.Manifest.Config.Image),
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}

	configContainer := config.NewContainer(state.DockerClient, state.Manifest.Config.Image, buildVolume, state.ErrorStream)
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

	configureReleaseResponse, err := configContainer.ConfigureRelease(version, state.Manifest.Config.Params, util.GetEnv(os.Environ()))
	if err != nil {
		return fmt.Errorf("error configuring release: %w", err)
	}

	releaseEnv := configureReleaseResponse.Env
	// these are built in and cannot be overridden by the config container (since choosing the clashing name would likely be an accident)
	releaseEnv["VERSION"] = version
	releaseEnv["TEAM"] = state.Manifest.Team
	releaseEnv["COMPONENT"] = state.Component
	releaseEnv["COMMIT"] = state.Commit

	releaseMetadata := make(map[string]map[string]string)
	for buildID, build := range state.Manifest.Builds {
		metadata, err := container.Run(
			state.DockerClient,
			build.Image,
			state.CodeDir,
			buildVolume,
			state.OutputStream,
			state.ErrorStream,
			releaseEnv,
		)
		if err != nil {
			return fmt.Errorf("error running release '%v': %w", buildID, err)
		}
		releaseMetadata[buildID] = metadata
	}
	if releaseMetadata["release"] == nil {
		releaseMetadata["release"] = make(map[string]string)
	}
	releaseMetadata["release"]["version"] = version
	releaseMetadata["release"]["commit"] = state.Commit
	releaseMetadata["release"]["component"] = state.Component
	releaseMetadata["release"]["team"] = state.Manifest.Team

	if err := configContainer.WriteReleaseMetadata(releaseMetadata); err != nil {
		return err
	}

	uploadReleaseResponse, err := configContainer.UploadRelease(
		savedTerraformImage,
	)
	if err != nil {
		return fmt.Errorf("error uploading release: %w", err)
	}
	if err := configContainer.RequestStop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}

	fmt.Fprintln(state.ErrorStream, uploadReleaseResponse.Message)

	return nil
}
