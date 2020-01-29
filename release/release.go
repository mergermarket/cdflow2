package release

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/terraform"
)

type readReleaseMetadataResult struct {
	metadata map[string]string
	err      error
}

// Run creates and runs the release container, returning a map of release metadata.
func Run(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume, outputStream, errorStream io.Writer) (map[string]string, error) {
	container, err := createReleaseContainer(dockerClient, image, codeDir, buildVolume)
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

func createReleaseContainer(dockerClient *docker.Client, image, codeDir string, buildVolume *docker.Volume) (*docker.Container, error) {
	return dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: containers.RandomName("cdflow2-release"),
		Config: &docker.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/code",
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{Type: "none"},
			Binds:     []string{codeDir + ":/code:ro", buildVolume.Name + ":/build"},
		},
	})
}

// Args contains parsed command line options.
type Args struct {
	NoPullConfig    *bool
	NoPullTerraform *bool
	NoPullRelease   *bool
}

// ParseArgs takes the command line arguments to the release command, and returns then parsed into an Args struct.
func ParseArgs(args []string) (*Args, error) {
	flagset := flag.NewFlagSet("cdflow2 release", flag.ContinueOnError)

	var result Args
	result.NoPullConfig = flagset.Bool("no-pull-config", false, "don't pull the config image (image must exist)")
	result.NoPullTerraform = flagset.Bool("no-pull-terraform", false, "don't pull the terraform image (image must exist)")
	result.NoPullRelease = flagset.Bool("no-pull-release", false, "don't pull the release image (image must exist)")

	if err := flagset.Parse(args); err != nil {
		return nil, err
	}

	return &result, nil
}

// RunCommand runs the release command.
func RunCommand(dockerClient *docker.Client, outputStream, errorStream io.Writer, codeDir string, inputArgs []string, manifest *config.Manifest) error {
	args, err := ParseArgs(inputArgs)
	if err != nil {
		return err
	}

	if !*args.NoPullTerraform {
		if err := dockerClient.PullImage(docker.PullImageOptions{
			Repository:   manifest.TerraformImage,
			OutputStream: os.Stderr,
		}, docker.AuthConfiguration{}); err != nil {
			return err
		}
	}

	buildVolume, err := dockerClient.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		return err
	}
	defer dockerClient.RemoveVolume(buildVolume.Name)

	if err := terraform.InitInitial(
		dockerClient,
		manifest.TerraformImage,
		codeDir,
		buildVolume,
		outputStream,
		errorStream,
	); err != nil {
		return err

	}

	return nil
}
