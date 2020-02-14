package test

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/containers"
)

func GetDockerClient() *client.Client {
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalln("error creating docker client:", err)
	}
	return client
}

// CreateState creates and returns a global state for testing.
func CreateState() *command.GlobalState {
	state := command.GlobalState{
		DockerClient:  GetDockerClient(),
		DockerContext: context.Background(),
	}
	return &state
}

func CreateVolume(state *command.GlobalState) string {
	volume, err := state.DockerClient.VolumeCreate(
		state.DockerContext,
		volume.VolumeCreateBody{},
	)
	if err != nil {
		log.Panicln("could not create volume:", err)
	}
	return volume.Name
}

func RemoveVolume(state *command.GlobalState, volume string) {
	if err := state.DockerClient.VolumeRemove(
		state.DockerContext,
		volume,
		false,
	); err != nil {
		log.Panicf("error removing volume %v: %v", volume, err)
	}
}

func ReadVolume(state *command.GlobalState, volume string) (map[string]string, error) {
	image := "alpine:latest"
	if err := containers.EnsureImage(state, image, nil); err != nil {
		log.Panicln("error pulling:", err)
	}
	container, err := state.DockerClient.ContainerCreate(
		state.DockerContext,
		&container.Config{
			Image: image,
		},
		&container.HostConfig{
			Binds: []string{volume + ":/root:ro"},
		},
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := state.DockerClient.ContainerRemove(
			state.DockerContext,
			container.ID,
			types.ContainerRemoveOptions{},
		); err != nil {
			log.Fatalln("could not remove container for reading volume:", err)
		}
	}()
	reader, _, err := state.DockerClient.CopyFromContainer(
		state.DockerContext,
		container.ID,
		"/root/",
	)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)
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

// CheckTerraformInitInitialReflectedInput checks the debug output for the terraform init during release.
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

// CheckTerraformWorkspaceList checks the debug output for the terraform list workspace command during workspace selection in deployment.
func CheckTerraformWorkspaceList(line string) {
	var input ReflectedInput
	if err := json.Unmarshal([]byte(line), &input); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(input.Args, []string{"workspace", "list"}) {
		log.Panicln("unexpected args for workspace list:", input.Args)
	}
}

// CheckTerraformWorkspaceNew checks the debug output for the terraform workspace new command during workspace selections in deployment.
func CheckTerraformWorkspaceNew(line, workspaceName string) {
	var input ReflectedInput
	if err := json.Unmarshal([]byte(line), &input); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(input.Args, []string{"workspace", "new", workspaceName}) {
		log.Panicln("unexpected args for workspace new:", input.Args)
	}
}
