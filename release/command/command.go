package command

import (
	"fmt"
	"log"
	"os"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/release/container"
	"github.com/mergermarket/cdflow2/terraform"
)

func getEnv() map[string]string {
	result := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		result[pair[0]] = pair[1]
	}
	return result
}

func repoDigest(dockerClient *docker.Client, image string) (string, error) {
	details, err := dockerClient.InspectImage(image)
	if err != nil {
		return "", err
	}
	if len(details.RepoDigests) == 0 {
		return "", nil
	}
	return details.RepoDigests[0], nil
}

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, version string) error {
	if !state.NoPullTerraform {
		if err := state.DockerClient.PullImage(docker.PullImageOptions{
			Repository:   containers.ImageWithTag(state.Manifest.TerraformImage),
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}
	savedTerraformImage, err := repoDigest(state.DockerClient, state.Manifest.TerraformImage)
	if err != nil {
		return err
	}

	if state.NoPullTerraform && savedTerraformImage == "" {
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

	if !state.NoPullConfig {
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
			log.Panicln("error removing config container:", err)
		}
	}()

	configureReleaseResponse, err := configContainer.ConfigureRelease(version, map[string]interface{}{}, getEnv())
	if err != nil {
		return err
	}

	releaseEnv := configureReleaseResponse.Env
	// these are built in and cannot be overridden by the config container (since choosing the clashing name would likely be an accident)
	releaseEnv["VERSION"] = version
	releaseEnv["TEAM"] = state.Manifest.Team
	releaseEnv["COMPONENT"] = state.Component
	releaseEnv["COMMIT"] = state.Commit

	releaseMetadata, err := container.Run(
		state.DockerClient,
		state.Manifest.ReleaseImage,
		state.CodeDir,
		buildVolume,
		state.OutputStream,
		state.ErrorStream,
		releaseEnv,
	)

	uploadReleaseResponse, err := configContainer.UploadRelease(
		savedTerraformImage,
		releaseMetadata,
	)

	if err := configContainer.RequestStop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}

	fmt.Fprintln(state.ErrorStream, uploadReleaseResponse.Message)

	return nil
}
