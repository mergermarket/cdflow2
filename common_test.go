package main

import (
	"archive/tar"
	"bytes"
	"io"
	"log"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

func createDockerClient() *docker.Client {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		log.Panicln(err)
	}
	return client
}

func createVolume(dockerClient *docker.Client) *docker.Volume {
	volume, err := dockerClient.CreateVolume(docker.CreateVolumeOptions{})
	if err != nil {
		log.Panicln("could not create volume:", err)
	}
	return volume
}

func removeVolume(dockerClient *docker.Client, volume *docker.Volume) {
	if err := dockerClient.RemoveVolume(volume.Name); err != nil {
		log.Panicf("error removing volume %v: %v", volume.Name, err)
	}
}

func readVolume(dockerClient *docker.Client, volume *docker.Volume) (map[string]string, error) {
	container, err := dockerClient.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: "hello-world",
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
