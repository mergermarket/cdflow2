package command

import (
	"fmt"
	"log"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/release/container"
	"github.com/mergermarket/cdflow2/terraform"
)

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, version string, env map[string]string) (returnedError error) {

	dockerClient := state.DockerClient

	if !state.GlobalArgs.NoPullTerraform {
		fmt.Fprintf(state.ErrorStream, "\nPulling terraform image %v...\n\n", state.Manifest.Terraform.Image)
		if err := dockerClient.PullImage(state.Manifest.Terraform.Image, state.ErrorStream); err != nil {
			return fmt.Errorf("error pulling terraform image: %w", err)
		}
	}

	repoDigests, err := dockerClient.GetImageRepoDigests(state.Manifest.Terraform.Image)
	if err != nil {
		return err
	}
	if len(repoDigests) == 0 {
		return fmt.Errorf("no docker repo digest(s) available for image %v", state.Manifest.Terraform.Image)
	}
	savedTerraformImage := repoDigests[0]

	if state.GlobalArgs.NoPullTerraform && savedTerraformImage == "" {
		savedTerraformImage = state.Manifest.Terraform.Image
	} else if savedTerraformImage == "" {
		log.Panicln("no repo digest for ", state.Manifest.Terraform.Image)
	}

	buildVolume, err := dockerClient.CreateVolume()
	if err != nil {
		return err
	}
	defer func() {
		if err := dockerClient.RemoveVolume(buildVolume); err != nil {
			if returnedError != nil {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
			return
		}
	}()

	if err := terraform.InitInitial(
		dockerClient,
		savedTerraformImage,
		state.CodeDir,
		buildVolume,
		state.OutputStream,
		state.ErrorStream,
	); err != nil {
		return err
	}

	if err := config.Pull(state); err != nil {
		return err
	}

	message, err := buildAndUploadRelease(state, buildVolume, version, savedTerraformImage, env)
	if err != nil {
		return err
	}

	// not in the above function to ensure docker output flushed before that finishes
	fmt.Fprintln(state.ErrorStream, message)

	return nil
}

func buildAndUploadRelease(state *command.GlobalState, buildVolume, version, savedTerraformImage string, env map[string]string) (returnedMessage string, returnedError error) {
	dockerClient := state.DockerClient
	configContainer, err := config.NewContainer(state, state.Manifest.Config.Image, buildVolume)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := configContainer.Done(); err != nil {
			if returnedError != nil {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
			return
		}
	}()

	configureReleaseResponse, err := configContainer.ConfigureRelease(version, state.Manifest.Config.Params, env)
	if err != nil {
		return "", err
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
			dockerClient,
			build.Image,
			state.CodeDir,
			buildVolume,
			state.OutputStream,
			state.ErrorStream,
			releaseEnv,
		)
		if err != nil {
			return "", fmt.Errorf("error running release '%v': %w", buildID, err)
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
		return "", err
	}

	uploadReleaseResponse, err := configContainer.UploadRelease(
		savedTerraformImage,
	)
	if err != nil {
		return "", fmt.Errorf("error uploading release: %w", err)
	}
	return uploadReleaseResponse.Message, nil
}
