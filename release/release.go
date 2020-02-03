package release

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/terraform"
)

type readReleaseMetadataResult struct {
	metadata map[string]string
	err      error
}

// Run creates and runs the release container, returning a map of release metadata.
func Run(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, outputStream, errorStream io.Writer, env map[string]string) (map[string]string, error) {
	container, err := createReleaseContainer(dockerClient, image, codeDir, buildVolume, env)
	if err != nil {
		return nil, err
	}

	outputReadStream, outputWriteStream := io.Pipe()

	resultChannel := make(chan readReleaseMetadataResult)
	go handleReleaseOutput(outputReadStream, outputStream, resultChannel)

	if err := containers.Await(dockerClient, container, nil, outputWriteStream, errorStream, nil); err != nil {
		return nil, err
	}

	outputWriteStream.Close()

	props, err := dockerClient.InspectContainer(container.ID)
	if err != nil {
		return nil, err
	}

	if props.State.Running {
		panic("unexpected: release container still running")
	}
	if err := dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID}); err != nil {
		return nil, err
	}
	if props.State.ExitCode != 0 {
		return nil, errors.New("release container failed")
	}

	result := <-resultChannel
	return result.metadata, result.err
}

// handleReleaseOutput runs as a goroutine to buffer the container output, picking out the last line which contains the release metadata and sending it to the passed in result channel.
func handleReleaseOutput(readStream io.Reader, outputStream io.Writer, resultChannel chan readReleaseMetadataResult) {
	readScanner := bufio.NewScanner(readStream)
	var last []byte
	for readScanner.Scan() {
		last = readScanner.Bytes()
		n, err := outputStream.Write(last)
		if err != nil {
			resultChannel <- readReleaseMetadataResult{nil, err}
			return
		}
		if n != len(last) {
			resultChannel <- readReleaseMetadataResult{nil, errors.New("incomplete write")}
			return
		}
	}
	if err := readScanner.Err(); err != nil {
		resultChannel <- readReleaseMetadataResult{nil, err}
		return
	}
	var result map[string]string
	if err := json.Unmarshal(last, &result); err != nil {
		resultChannel <- readReleaseMetadataResult{nil, err}
	}
	resultChannel <- readReleaseMetadataResult{result, nil}
}

func createReleaseContainer(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, env map[string]string) (*docker.Container, error) {
	return dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: containers.RandomName("cdflow2-release"),
		Config: &docker.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/code",
			Env:          containers.MapToDockerEnv(env),
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{Type: "none"},
			Binds:     []string{codeDir + ":/code:ro", buildVolume.Name + ":/build"},
		},
	})
}

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
func RunCommand(env *command.GlobalEnvironment, version string) error {
	if !env.NoPullTerraform {
		if err := env.DockerClient.PullImage(docker.PullImageOptions{
			Repository:   containers.ImageWithTag(env.Manifest.TerraformImage),
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}
	savedTerraformImage, err := repoDigest(env.DockerClient, env.Manifest.TerraformImage)
	if err != nil {
		return err
	}

	if env.NoPullTerraform && savedTerraformImage == "" {
		savedTerraformImage = env.Manifest.TerraformImage
	} else if savedTerraformImage == "" {
		log.Panicln("no repo digest for ", env.Manifest.TerraformImage)
	}

	buildVolume, err := env.DockerClient.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		return err
	}
	defer env.DockerClient.RemoveVolume(buildVolume.Name)

	if err := terraform.InitInitial(
		env.DockerClient,
		savedTerraformImage,
		env.CodeDir,
		buildVolume,
		env.OutputStream,
		env.ErrorStream,
	); err != nil {
		return err
	}

	if !env.NoPullConfig {
		if err := env.DockerClient.PullImage(docker.PullImageOptions{
			Repository:   containers.ImageWithTag(env.Manifest.ConfigImage),
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}

	configContainer := config.NewContainer(env.DockerClient, env.Manifest.ConfigImage, buildVolume, env.ErrorStream)
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
	releaseEnv["TEAM"] = env.Manifest.Team
	releaseEnv["COMPONENT"] = env.Component
	releaseEnv["COMMIT"] = env.Commit

	releaseMetadata, err := Run(
		env.DockerClient,
		env.Manifest.ReleaseImage,
		env.CodeDir,
		buildVolume,
		env.OutputStream,
		env.ErrorStream,
		releaseEnv,
	)

	uploadReleaseResponse, err := configContainer.UploadRelease(
		savedTerraformImage,
		releaseMetadata,
	)

	if err := configContainer.RequestStop(); err != nil {
		log.Panicln("error stopping config container:", err)
	}

	fmt.Fprintln(env.ErrorStream, uploadReleaseResponse.Message)

	return nil
}
