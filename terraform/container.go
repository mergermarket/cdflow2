package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/containers"
	"github.com/mergermarket/cdflow2/util"
)

// createTerraformInitContainer creates and returns a container for running terraform init to download providers and modules.
func createTerraformInitContainer(state *command.GlobalState, image, codeDir string, buildVolume string) (string, error) {
	response, err := state.DockerClient.ContainerCreate(
		state.DockerContext,
		&container.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/build",
			Cmd:          []string{"init", "/code/infra"},
			Env:          []string{"TF_IN_AUTOMATION=true"},
		},
		&container.HostConfig{
			LogConfig: container.LogConfig{Type: "none"},
			Binds:     []string{codeDir + ":/code:ro", buildVolume + ":/build"},
		},
		nil,
		util.RandomName("cdflow2-terraform-init"),
	)
	if err != nil {
		return "", err
	}
	return response.ID, nil
}

// InitInitial runs terraform init as part of the release in order to download providers and modules.
func InitInitial(state *command.GlobalState, image, codeDir string, buildVolume string, outputStream, errorStream io.Writer) error {
	container, err := createTerraformInitContainer(state, image, codeDir, buildVolume)
	if err != nil {
		return err
	}

	if err := containers.Await(state, container, nil, outputStream, errorStream, nil); err != nil {
		return err
	}

	if err := state.DockerClient.ContainerRemove(
		state.DockerContext,
		container,
		types.ContainerRemoveOptions{},
	); err != nil {
		return err
	}

	return nil
}

// Container stores information about a running terraform container for running terraform commands in.
type Container struct {
	state *command.GlobalState
	ID    string
}

// NewContainer creates and returns a terraformContainer for running terraform commands in.
func NewContainer(state *command.GlobalState, image, codeDir string, releaseVolume string) (*Container, error) {
	init := true
	dockerContainer, err := state.DockerClient.ContainerCreate(
		state.DockerContext,
		&container.Config{
			Image:        image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/code",
			Entrypoint:   []string{"/bin/sleep"},
			Cmd:          []string{strconv.Itoa(365 * 24 * 60 * 60)}, // a long time!
			Env:          []string{"TF_IN_AUTOMATION=true"},
		},
		&container.HostConfig{
			Init:      &init,
			LogConfig: container.LogConfig{Type: "none"},
			Binds:     []string{codeDir + ":/code:ro", releaseVolume + ":/release"},
		},
		nil,
		util.RandomName("cdflow2-terraform"),
	)
	if err != nil {
		return nil, err
	}

	if err := state.DockerClient.ContainerStart(state.DockerContext, dockerContainer.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	return &Container{
		state: state,
		ID:    dockerContainer.ID,
	}, nil
}

// BackendConfigParameter is like a tuple of key and value for use in a slice (rather than a map as that wouldn't preserve order).
type BackendConfigParameter struct {
	Key   string
	Value string
}

// ConfigureBackend runs terraform init as part of the release in order to download providers and modules.
func (configContainer *Container) ConfigureBackend(outputStream, errorStream io.Writer, backendConfig []BackendConfigParameter) error {

	command := make([]string, 0)
	command = append(command, "terraform")
	command = append(command, "init")
	command = append(command, "-get=false")
	command = append(command, "-get-plugins=false")

	for _, param := range backendConfig {
		command = append(command, "-backend-config="+param.Key+"="+param.Value)
	}

	if err := configContainer.RunCommand(command, outputStream, errorStream); err != nil {
		return err
	}

	return nil
}

// SwitchWorkspace switched to a named workspace, creating it if necessary.
func (configContainer *Container) SwitchWorkspace(name string, outputStream, errorStream io.Writer) error {
	workspaces, err := configContainer.listWorkspaces(errorStream)
	if err != nil {
		return err
	}

	command := "new"
	if workspaces[name] {
		command = "select"
	}

	if err := configContainer.RunCommand([]string{"terraform", "workspace", command, name}, outputStream, errorStream); err != nil {
		return err
	}

	return nil
}

// listWorkspaces lists the terraform workspaces and returns them as a set
func (configContainer *Container) listWorkspaces(errorStream io.Writer) (map[string]bool, error) {
	var outputBuffer bytes.Buffer

	if err := configContainer.RunCommand([]string{"terraform", "workspace", "list"}, &outputBuffer, errorStream); err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, line := range strings.Split(outputBuffer.String(), "\n") {
		for _, word := range strings.Fields(line) {
			if word != "*" {
				result[word] = true
			}
		}
	}

	return result, nil
}

// RunCommand execs a command inside the terraform container.
func (configContainer *Container) RunCommand(command []string, outputStream, errorStream io.Writer) error {
	exec, err := configContainer.state.DockerClient.ContainerExecCreate(
		configContainer.state.DockerContext,
		configContainer.ID,
		types.ExecConfig{
			AttachStdout: true,
			AttachStderr: true,
			Cmd:          command,
		},
	)
	if err != nil {
		return err
	}

	attachResponse, err := configContainer.state.DockerClient.ContainerExecAttach(
		configContainer.state.DockerContext,
		exec.ID,
		types.ExecStartCheck{},
	)
	if err != nil {
		return err
	}
	defer attachResponse.Close()

	if err := containers.StreamHijackedResponse(configContainer.state, attachResponse, nil, outputStream, errorStream, func() error {
		return configContainer.state.DockerClient.ContainerExecStart(
			configContainer.state.DockerContext,
			exec.ID,
			types.ExecStartCheck{},
		)
	}); err != nil {
		return fmt.Errorf("error streaming data from terraform exec: %w", err)
	}

	details, err := configContainer.state.DockerClient.ContainerExecInspect(
		configContainer.state.DockerContext,
		exec.ID,
	)
	if err != nil {
		return err
	}

	if details.ExitCode != 0 {
		return errors.New("processed exited with error status code " + string(details.ExitCode))
	}

	return nil
}

// Done stops and removes the terraform container.
func (configContainer *Container) Done() error {
	timeout := 10 * time.Second
	if err := configContainer.state.DockerClient.ContainerStop(
		configContainer.state.DockerContext,
		configContainer.ID,
		&timeout,
	); err != nil {
		return err
	}
	return configContainer.state.DockerClient.ContainerRemove(
		configContainer.state.DockerContext,
		configContainer.ID,
		types.ContainerRemoveOptions{},
	)
}
