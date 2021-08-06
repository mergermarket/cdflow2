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

	"github.com/mergermarket/cdflow2/docker"
	"github.com/mergermarket/cdflow2/docker/official"
)

// GetDockerClient returns a docker client for testing (panics).
func GetDockerClient() docker.Iface {
	dockerClient, err := official.NewClient()
	if err != nil {
		log.Panicln("error creating docker client:", err)
	}
	return dockerClient
}

// GetDockerClientWithDebugVolume returns a docker client and debug volume for testing.
func GetDockerClientWithDebugVolume() (docker.Iface, string) {
	dockerClient := GetDockerClient()
	debugVolume := CreateVolume(dockerClient)
	dockerClient.SetDebugVolume(debugVolume)
	return dockerClient, debugVolume
}

// CreateVolume creates a volume (panics).
func CreateVolume(dockerCient docker.Iface) string {
	volume, err := dockerCient.CreateVolume("")
	if err != nil {
		log.Panicln("could not create volume:", err)
	}
	return volume
}

// RemoveVolume removes a docker volume - outputs a warning if it fails to avoid masking another error.
func RemoveVolume(dockerClient docker.Iface, volume string) {
	if err := dockerClient.RemoveVolume(volume); err != nil {
		log.Printf("error removing volume %v: %v\n", volume, err)
	}
}

// ReadVolume reads all the files in a volume as a map of path strings to byte slices of the file contents (panics).
func ReadVolume(dockerClient docker.Iface, volume string) (map[string][]byte, error) {
	image := "alpine:latest"
	if err := dockerClient.EnsureImage(image, os.Stderr); err != nil {
		log.Panicln("error pulling:", err)
	}

	container, err := dockerClient.CreateContainer(&docker.CreateContainerOptions{
		Image: image,
		Binds: []string{volume + ":/root:ro"},
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := dockerClient.RemoveContainer(container); err != nil {
			log.Fatalln("could not remove container for reading volume:", err)
		}
	}()
	reader, err := dockerClient.CopyFromContainer(container, "/root/")
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)
	result := make(map[string][]byte)
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
		result[strings.TrimPrefix(header.Name, "root/")] = contents.Bytes()
	}
	return result, nil
}

// GetConfig gets a config value from the environment (panics).
func GetConfig(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Panicf("environment variable %v not set - did you run ./test.sh?", name)
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

// CheckTerraformInitInitialReflectedInput checks the debug output for the terraform init during release (panics).
func CheckTerraformInitInitialReflectedInput(output []byte) {
	var input ReflectedInput
	if err := json.Unmarshal(output, &input); err != nil {
		log.Panicln("error parsing json:", err)
	}

	// interface is that the code is mapped to /code and the terraform is in the infra subfolder
	if !reflect.DeepEqual(input.Args, []string{"init", "-backend=false"}) {
		log.Fatalf("unexpected args: %v", input.Args)
	}

	if input.Cwd != "/code/infra" {
		log.Fatalf("unexpected cwd: %v", input.Cwd)
	}

	if input.File != "sample content" {
		log.Fatalf("code not mapped as /code - file contents: %v", input.File)
	}
}

// CheckTerraformWorkspaceList checks the debug output for the terraform list workspace command during workspace selection in deployment (panics).
func CheckTerraformWorkspaceList(line []byte) {
	var input ReflectedInput
	if err := json.Unmarshal(line, &input); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(input.Args, []string{"workspace", "list"}) {
		log.Panicln("unexpected args for workspace list:", input.Args)
	}
}

// CheckTerraformWorkspaceNew checks the debug output for the terraform workspace new command during workspace selections in deployment.
func CheckTerraformWorkspaceNew(line []byte, workspaceName string) {
	var input ReflectedInput
	if err := json.Unmarshal(line, &input); err != nil {
		log.Panicln("error parsing json:", err)
	}

	if !reflect.DeepEqual(input.Args, []string{"workspace", "new", workspaceName}) {
		log.Panicln("unexpected args for workspace new:", input.Args)
	}
}

// DumpLines outputs a set of lines with indentation.
func DumpLines(lines [][]byte) string {
	result := ""
	for _, line := range lines {
		result += "  " + string(line) + "\n"
	}
	return result
}
