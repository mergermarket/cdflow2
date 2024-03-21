package config

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/docker"
	"github.com/mergermarket/cdflow2/util"
)

// Container represents a config container.
type Container struct {
	dockerClient docker.Iface
	id           string
	done         chan error
	finished     bool
	errorStream  io.Writer
}

// NewContainer creates and returns a new config container.
func NewContainer(state *command.GlobalState, image, releaseVolume string) (*Container, error) {
	dockerClient := state.DockerClient

	cacheVolume, err := util.GetCacheVolume(dockerClient)
	if err != nil {
		return nil, err
	}

	started := make(chan string, 1)
	defer close(started) // does not error so no named returns

	done := make(chan error, 1)

	container := &Container{
		dockerClient: dockerClient,
		done:         done,
		errorStream:  state.ErrorStream,
	}

	go func() {
		options := docker.RunOptions{
			NamePrefix:   "cdflow2-config",
			Image:        image,
			OutputStream: state.OutputStream,
			ErrorStream:  state.ErrorStream,
			Started:      started,
		}
		if releaseVolume == "" { // setup doesn't need a volume
			options.WorkingDir = "/"
		} else {
			options.WorkingDir = "/release"
			options.Binds = []string{
				releaseVolume + ":/release",
				cacheVolume + ":/cache",
			}
		}
		err := dockerClient.Run(&options)
		container.finished = true
		done <- err
	}()

	select {
	case id := <-started:
		container.id = id
		return container, nil
	case err := <-done:
		return nil, err
	}
}

func (configContainer *Container) request(request interface{}, response interface{}) error {
	var rawRequest bytes.Buffer
	if err := json.NewEncoder(&rawRequest).Encode(request); err != nil {
		return err
	}
	var errors bytes.Buffer
	var rawResponse bytes.Buffer
	if err := configContainer.dockerClient.Exec(&docker.ExecOptions{
		ID:           configContainer.id,
		Cmd:          []string{"/app", "forward"},
		InputStream:  &rawRequest,
		OutputStream: &rawResponse,
		ErrorStream:  &errors,
	}); err != nil {
		return err
	}
	if len(rawResponse.Bytes()) == 0 {
		return fmt.Errorf("no response returned")
	}
	if err := json.NewDecoder(&rawResponse).Decode(response); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}
	return nil
}

type Monitoring struct {
	APIKey string
	Data   map[string]string
}

// ReleaseRequirements contains a list of needs.
type ReleaseRequirements struct {
	Needs []string
}

type setupConfigRequest struct {
	Action              string
	Config              map[string]interface{}
	Env                 map[string]string
	Component           string
	Commit              string
	ReleaseRequirements map[string]*ReleaseRequirements
}

type SetupConfigResponse struct {
	Monitoring Monitoring
	Success    bool
}

// Setup requests the container does setup.
func (configContainer *Container) Setup(
	config map[string]interface{},
	env map[string]string,
	component, commit string,
	releaseRequirements map[string]*ReleaseRequirements,
) (*SetupConfigResponse, error) {
	response := &SetupConfigResponse{}
	if err := configContainer.request(&setupConfigRequest{
		Action:              "setup",
		Config:              config,
		Env:                 env,
		Component:           component,
		Commit:              commit,
		ReleaseRequirements: releaseRequirements,
	}, response); err != nil {
		return nil, err
	}
	if !response.Success {
		return response, command.Failure(1)
	}
	return response, nil
}

type configureReleaseConfigRequest struct {
	Action              string
	Version             string
	Component           string
	Commit              string
	Config              map[string]interface{}
	Env                 map[string]string
	ReleaseRequirements map[string]*ReleaseRequirements
}

// ConfigureReleaseConfigResponse contains the response to the configure release request.
type ConfigureReleaseConfigResponse struct {
	Env                map[string]map[string]string
	AdditionalMetadata map[string]string
	Monitoring         Monitoring
	Success            bool
}

// ConfigureRelease requests the container configures the release and returns the response.
func (configContainer *Container) ConfigureRelease(
	version, component, commit string,
	config map[string]interface{},
	env map[string]string,
	releaseRequirements map[string]*ReleaseRequirements,
) (*ConfigureReleaseConfigResponse, error) {
	var response ConfigureReleaseConfigResponse
	if err := configContainer.request(&configureReleaseConfigRequest{
		Action:              "configure_release",
		Version:             version,
		Component:           component,
		Commit:              commit,
		Config:              config,
		Env:                 env,
		ReleaseRequirements: releaseRequirements,
	}, &response); err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, command.Failure(1)
	}
	return &response, nil
}

// WriteReleaseMetadata copies the release metadata file into the release volume via the config container.
func (configContainer *Container) WriteReleaseMetadata(releaseMetadata map[string]map[string]string) error {
	encoded, err := json.Marshal(releaseMetadata)
	if err != nil {
		return err
	}

	if err := configContainer.CopyFileToRelease("release-metadata.json", encoded); err != nil {
		return err
	}

	return nil
}

func (configContainer *Container) CopyFileToRelease(filename string, content []byte) error {

	buffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(buffer)

	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "/release/" + filename,
		Mode: 0644,
		Size: int64(len(content)),
	}); err != nil {
		return err
	}

	if _, err := tarWriter.Write(content); err != nil {
		return err
	}

	if err := tarWriter.Close(); err != nil {
		return err
	}

	if err := configContainer.dockerClient.CopyToContainer(configContainer.id, "/", buffer); err != nil {
		return err
	}
	return nil
}

type uploadReleaseRequest struct {
	Action         string
	TerraformImage string
}

// UploadReleaseResponse contains the response to the upload release request.
type UploadReleaseResponse struct {
	Message string
	Success bool
}

// UploadRelease requests that the config container uploads the release and returns the response.
func (configContainer *Container) UploadRelease(terraformImage string) (*UploadReleaseResponse, error) {
	var response UploadReleaseResponse
	if err := configContainer.request(&uploadReleaseRequest{
		Action:         "upload_release",
		TerraformImage: terraformImage,
	}, &response); err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, errors.New("config container failed to upload release")
	}
	return &response, nil
}

type prepareTerraformRequest struct {
	Action           string
	Version          string
	Component        string
	Commit           string
	Config           map[string]interface{}
	Env              map[string]string
	EnvName          string
	StateShouldExist *bool
}

// TerraformBackendConfigParameter
type TerrafromBackendConfigParameter struct {
	Value        string
	DisplayValue string
}

// PrepareTerraformResponse contains the response to the prepare terraform request.
type PrepareTerraformResponse struct {
	TerraformImage                   string
	Env                              map[string]string
	TerraformBackendType             string
	TerraformBackendConfig           map[string]string
	TerraformBackendConfigParameters map[string]*TerrafromBackendConfigParameter
	Monitoring                       Monitoring
	Success                          bool
}

// PrepareTerraform requests that the config container prepares for running terraform and returns the response.
func (configContainer *Container) PrepareTerraform(
	version, component, commit, envName string,
	stateShouldExist *bool,
	config map[string]interface{},
	env map[string]string,
) (*PrepareTerraformResponse, error) {

	var response PrepareTerraformResponse
	if err := configContainer.request(&prepareTerraformRequest{
		Action:           "prepare_terraform",
		Config:           config,
		Env:              env,
		EnvName:          envName,
		Version:          version,
		Component:        component,
		Commit:           commit,
		StateShouldExist: stateShouldExist,
	}, &response); err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, errors.New("config container failed to prepare for running terraform")
	}
	return &response, nil
}

// SetupTerraform creates the config container and prepares terraform in one.
func SetupTerraform(state *command.GlobalState, stateShouldExist *bool, envName, version string, env map[string]string) (_ *PrepareTerraformResponse, returnedBuildVolume string, terraformImage string, returnedError error) {
	dockerClient := state.DockerClient

	if err := Pull(state); err != nil {
		return nil, "", "", err
	}

	buildVolume, err := dockerClient.CreateVolume("")
	if err != nil {
		return nil, "", "", err
	}

	configContainer, err := NewContainer(state, state.Manifest.Config.Image, buildVolume)
	if err != nil {
		return nil, "", "", err
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

	prepareTerraformResponse, err := configContainer.PrepareTerraform(version, state.Component, state.Commit, envName, stateShouldExist, state.Manifest.Config.Params, env)
	if err != nil {
		return nil, "", "", err
	}

	state.MonitoringClient.APIKey = prepareTerraformResponse.Monitoring.APIKey
	state.MonitoringClient.ConfigData = prepareTerraformResponse.Monitoring.Data

	imageName := prepareTerraformResponse.TerraformImage
	if version == "" {
		imageName = state.Manifest.Terraform.Image
	}
	if !state.GlobalArgs.NoPullTerraform {
		if err := dockerClient.EnsureImage(imageName, state.ErrorStream); err != nil {
			return nil, "", "", fmt.Errorf("error pulling terraform image %v: %w", imageName, err)
		}
	}
	return prepareTerraformResponse, buildVolume, imageName, nil
}

// Done stops and removes the config container.
func (configContainer *Container) Done() error {
	if !configContainer.finished {
		if err := configContainer.dockerClient.Stop(configContainer.id, 2); err != nil {
			return err
		}
	}
	return <-configContainer.done
}

// Pull pulls the config image.
func Pull(state *command.GlobalState) error {
	if state.GlobalArgs.NoPullConfig {
		return nil
	}
	fmt.Fprintf(state.ErrorStream, "\nPulling config image %v...\n\n", state.Manifest.Config.Image)
	if err := state.DockerClient.PullImage(state.Manifest.Config.Image, state.ErrorStream); err != nil {
		return fmt.Errorf("error pulling config image: %w", err)
	}
	return nil
}
