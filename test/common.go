package test

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mergermarket/cdflow2/containers"
)

func CreateDockerClient() *docker.Client {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		log.Panicln(err)
	}
	return client
}

func CreateVolume(dockerClient *docker.Client) *docker.Volume {
	volume, err := dockerClient.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		log.Panicln("could not create volume:", err)
	}
	return volume
}

func RemoveVolume(dockerClient *docker.Client, volume *docker.Volume) {
	if err := dockerClient.RemoveVolume(volume.Name); err != nil {
		log.Panicf("error removing volume %v: %v", volume.Name, err)
	}
}

func ReadVolume(dockerClient *docker.Client, volume *docker.Volume) (map[string]string, error) {
	image := "alpine:latest"
	if err := containers.EnsureImage(dockerClient, image); err != nil {
		log.Panicln("error pulling:", err)
	}
	container, err := dockerClient.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: image,
		},
		HostConfig: &docker.HostConfig{
			Binds: []string{volume.Name + ":/root:ro"},
		},
	})
	if err != nil {
		return nil, err
	}
	defer dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
	var buf bytes.Buffer
	if err := dockerClient.DownloadFromContainer(container.ID, docker.DownloadFromContainerOptions{
		OutputStream: &buf,
		Path:         "/root/",
	}); err != nil {
		return nil, err
	}
	tarReader := tar.NewReader(&buf)
	result := make(map[string]string)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if strings.HasSuffix(header.Name, "/") {
			continue
		}
		if err != nil {
			return nil, err
		}
		var contents bytes.Buffer
		io.Copy(&contents, tarReader)
		result[strings.TrimPrefix(header.Name, "root/")] = contents.String()
	}
	return result, nil
}

func GetConfig(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("environment variable %v not set - did you run ./test.sh?", name)
	}
	return value
}

// ReflectedInput is the message format returned from the fake terraform container that reflects its inputs.
type ReflectedInput struct {
	Args  []string
	Env   map[string]string
	Input string
	Cwd   string
	File  string
}

func CheckTerraformInitInitialReflectedInput(output []byte) {
	var input ReflectedInput
	if err := json.Unmarshal(output, &input); err != nil {
		log.Panicln("error parsing json:", err)
	}

	// interface is that the code is mapped to /code and the terraform is in the infra subfolder
	if !reflect.DeepEqual(input.Args, []string{"init", "/code/infra"}) {
		log.Fatalf("unexpected args: %v", input.Args)
	}

	// interface is that the mapped in cwd is /build
	if input.Cwd != "/build" {
		log.Fatalf("unexpected cwd: %v", input.Cwd)
	}

	if input.File != "sample content" {
		log.Fatalf("code not mapped as /code - file contents: %v", input.File)
	}
}
