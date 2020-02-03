package deploy

import (
	"log"
	"os"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/containers"
)

// RunCommand runs the release command.
func RunCommand(env *command.GlobalEnvironment, envName string, version string) error {

	if !env.NoPullConfig {
		if err := env.DockerClient.PullImage(docker.PullImageOptions{
			Repository:   containers.ImageWithTag(env.Manifest.ConfigImage),
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}

	buildVolume, err := env.DockerClient.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		return err
	}
	defer env.DockerClient.RemoveVolume(buildVolume.Name)

	configContainer := config.NewContainer(env.DockerClient, env.Manifest.ConfigImage, buildVolume, env.ErrorStream)
	if err := configContainer.Start(); err != nil {
		return err
	}
	defer func() {
		if err := configContainer.Remove(); err != nil {
			log.Panicln("error removing config container:", err)
		}
	}()

	if err := configContainer.RequestStop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}

	return nil
}
