package official

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/registry"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/mergermarket/cdflow2/docker"
	"github.com/mergermarket/cdflow2/util"
)

const (
	cdflowDockerAuthPrefix = "CDFLOW2_DOCKER_AUTH_"
)

// Client is a concrete implementation of our docker interface that uses the official client library.
type Client struct {
	client      *client.Client
	debugVolume string
}

// NewClient creates and returns a new client.
func NewClient() (*Client, error) {
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{
		client: client,
	}, nil
}

// SetDebugVolume sets a volume that will be mapped to /debug in each container, for an out of band way to get data out for testing.
func (dockerClient *Client) SetDebugVolume(volume string) {
	dockerClient.debugVolume = volume
}

// Run runs a container (much like `docker run` in the cli).
func (dockerClient *Client) Run(options *docker.RunOptions) error {
	stdin := false
	if options.InputStream != nil {
		stdin = true
	}
	binds := options.Binds
	if dockerClient.debugVolume != "" {
		binds = append(binds, dockerClient.debugVolume+":/debug")
	}
	response, err := dockerClient.client.ContainerCreate(
		context.Background(),
		&container.Config{
			Image:        options.Image,
			OpenStdin:    stdin,
			AttachStdin:  stdin,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   options.WorkingDir,
			Entrypoint:   options.Entrypoint,
			Cmd:          options.Cmd,
			Env:          options.Env,
		},
		&container.HostConfig{
			LogConfig: container.LogConfig{Type: "none"},
			Binds:     binds,
			Init:      &options.Init,
		},
		nil,
		nil,
		util.RandomName(options.NamePrefix),
	)
	if err != nil {
		return err
	}

	statusChannel := dockerClient.waitForContainerExit(response.ID)

	if err := dockerClient.runContainer(response.ID, options.InputStream, options.OutputStream, options.ErrorStream, options.Started); err != nil {
		return err
	}

	status := <-statusChannel
	if status.err != nil {
		return err
	}

	if status.exitCode != options.SuccessStatus {
		extra := ""
		if err := dockerClient.RemoveContainer(response.ID); err != nil {
			extra = "\nerror removing container: " + err.Error()
		}
		return fmt.Errorf("container exited with unsuccessful exit code %d%s", status.exitCode, extra)
	}

	if options.BeforeRemove != nil {
		if err := options.BeforeRemove(response.ID); err != nil {
			return fmt.Errorf("error in BeforeRemove function for container: %w", err)
		}
	}
	return dockerClient.RemoveContainer(response.ID)
}

type status struct {
	exitCode int
	err      error
}

func (dockerClient *Client) waitForContainerExit(id string) chan status {
	resultChannel, errChannel := dockerClient.client.ContainerWait(context.Background(), id, container.WaitConditionNextExit)

	statusChannel := make(chan status)
	go func() {
		var status status
		select {
		case result := <-resultChannel:
			if result.Error != nil {
				status.err = fmt.Errorf("error waiting for container: %s", result.Error.Message)
			} else {
				status.exitCode = int(result.StatusCode)
			}
		case err := <-errChannel:
			status.err = err
		}
		statusChannel <- status
	}()

	return statusChannel
}

func (dockerClient *Client) runContainer(id string, inputStream io.Reader, outputStream, errorStream io.Writer, started chan string) error {
	stdin := false
	if inputStream != nil {
		stdin = true
	}
	hijackedResponse, err := dockerClient.client.ContainerAttach(context.Background(), id, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Stdin:  stdin,
	})
	if err != nil {
		return err
	}

	return dockerClient.streamHijackedResponse(hijackedResponse, inputStream, outputStream, errorStream, func() error {
		if err := dockerClient.client.ContainerStart(
			context.Background(),
			id,
			types.ContainerStartOptions{},
		); err != nil {
			return err
		}
		if started != nil {
			started <- id
		}
		return nil
	})
}

// EnsureImage pulls an image if it does not exist locally.
func (dockerClient *Client) EnsureImage(image string, outputStream io.Writer) error {
	// TODO bit lax, this should check the error type
	if _, _, err := dockerClient.client.ImageInspectWithRaw(
		context.Background(),
		image,
	); err == nil {
		return nil
	}
	return dockerClient.PullImage(image, outputStream)
}

// PullProgressDetail is the progress returned from docker for an image pull.
type PullProgressDetail struct {
	Current  int64
	Total    int64
	Progress string
}

// PullMessage is the line format returned from dockder during an image pull.
type PullMessage struct {
	Status         string
	ProgressDetail PullProgressDetail
	ID             string
}

func writePullProgress(reader io.ReadCloser, outputStream io.Writer) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var message PullMessage
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			return err
		}
		if message.Status != "Downloading" && message.Status != "Extracting" {
			if message.ID != "" {
				fmt.Fprintf(outputStream, "%s: %s\n", message.ID, message.Status)
			} else {
				fmt.Fprintln(outputStream, message.Status)
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}
	return reader.Close()
}

func getRegistryAuth(image string, outputStream io.Writer) (string, error) {
	username, password, err := getRegistryCredentials(image)
	if err != nil {
		fmt.Fprintf(outputStream, "Unable to get registry credentials, fallback to legacy method: %v\n\n", err)
	}

	if username == "" || password == "" {
		username, password = getRegistryCredentialsLegacy(image)
	}

	authBytes, err := json.Marshal(
		types.AuthConfig{
			Username: username,
			Password: password,
		},
	)

	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(authBytes), nil
}

func getRegistryCredentials(image string) (username, password string, err error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", "", err
	}

	imageRegistry := strings.ToUpper(
		strings.NewReplacer(
			".", "_",
			":", "_",
			"-", "_",
		).Replace(reference.Domain(named)),
	)

	return os.Getenv(cdflowDockerAuthPrefix + imageRegistry + "_USERNAME"), os.Getenv(cdflowDockerAuthPrefix + imageRegistry + "_PASSWORD"), nil
}

func getRegistryCredentialsLegacy(image string) (username, password string) {
	imageRegistry := registry.IndexHostname
	if strings.Count(image, "/") > 1 {
		imageRegistry = strings.Split(image, "/")[0]
	}

	imageRegistry = strings.ToUpper(
		strings.NewReplacer(
			".", "_",
			":", "_",
			"-", "_",
		).Replace(imageRegistry),
	)

	return os.Getenv(cdflowDockerAuthPrefix + imageRegistry + "_USERNAME"), os.Getenv(cdflowDockerAuthPrefix + imageRegistry + "_PASSWORD")
}

// PullImage pulls and image.
func (dockerClient *Client) PullImage(image string, outputStream io.Writer) error {
	RegistryAuth, err := getRegistryAuth(image, outputStream)
	imagePullOptions := types.ImagePullOptions{}

	if err == nil {
		imagePullOptions.RegistryAuth = RegistryAuth
	}

	reader, err := dockerClient.client.ImagePull(
		context.Background(),
		image,
		imagePullOptions,
	)
	if err != nil {
		return err
	}

	return writePullProgress(reader, outputStream)
}

// GetImageRepoDigests inspects an image and pulls out the repo digests.
func (dockerClient *Client) GetImageRepoDigests(image string) ([]string, error) {
	details, _, err := dockerClient.client.ImageInspectWithRaw(context.Background(), image)
	if err != nil {
		return nil, err
	}
	return details.RepoDigests, nil
}

// Exec execs a process in a docker container (like `docker exec` in the cli).
func (dockerClient *Client) Exec(options *docker.ExecOptions) error {
	stdin := false
	if options.InputStream != nil {
		stdin = true
	}
	var env []string
	for key, value := range options.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	exec, err := dockerClient.client.ContainerExecCreate(
		context.Background(),
		options.ID,
		types.ExecConfig{
			AttachStdin:  stdin,
			AttachStdout: true,
			AttachStderr: true,
			Cmd:          options.Cmd,
			Env:          env,
			Tty:          options.Tty,
			WorkingDir:   options.WorkingDir,
		},
	)
	if err != nil {
		return fmt.Errorf("error creating docker exec: %w", err)
	}

	attachResponse, err := dockerClient.client.ContainerExecAttach(
		context.Background(),
		exec.ID,
		types.ExecStartCheck{},
	)
	if err != nil {
		return fmt.Errorf("error attaching to docker exec: %w", err)
	}
	defer attachResponse.Close() // does not return error

	if options.Tty && options.Interactive {
		width, height, err := terminal.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			width = 150
			height = 25

			_, _ = fmt.Fprintf(options.ErrorStream, "\n%s\n", util.FormatInfo(fmt.Sprintf("unable to get terminal size, using default width: %d and height: %d, err: %v", width, height, err)))
		}

		err = dockerClient.client.ContainerExecResize(context.Background(), exec.ID, types.ResizeOptions{
			Width:  uint(width),
			Height: uint(height),
		})
		if err != nil {
			_, _ = fmt.Fprintf(options.ErrorStream, "\n%s\n", util.FormatInfo(fmt.Sprintf("unable to set tty size: %v", err)))
		}

		// to print newline 'correctly' after setting raw mode (e.g. with fmt.Fprintf...), use '\r\n' instead of just '\n'
		// https://github.com/golang/go/issues/50761#issuecomment-1019372593
		oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }()
	}

	if err := dockerClient.streamHijackedResponse(
		attachResponse,
		options.InputStream,
		options.OutputStream,
		options.ErrorStream,
		func() error {
			return nil
		},
	); err != nil {
		return fmt.Errorf("error streaming data from exec: %w", err)
	}

	details, err := dockerClient.client.ContainerExecInspect(
		context.Background(),
		exec.ID,
	)
	if err != nil {
		return fmt.Errorf("error inspecting exec: %w", err)
	}

	if details.ExitCode != 0 {
		return fmt.Errorf("exec process exited with error status code %d", details.ExitCode)
	}

	return nil
}

// Stop stops a container.
func (dockerClient *Client) Stop(id string, timeout time.Duration) error {
	return dockerClient.client.ContainerStop(context.Background(), id, &timeout)
}

// CreateVolume creates a docker volume and returns its ID.
func (dockerClient *Client) CreateVolume(name string) (string, error) {
	volume, err := dockerClient.client.VolumeCreate(context.Background(), volume.VolumeCreateBody{
		Name: name,
	})
	if err != nil {
		return "", err
	}
	return volume.Name, nil
}

// RemoveVolume removes a docker volume given its ID.
func (dockerClient *Client) RemoveVolume(id string) error {
	return dockerClient.client.VolumeRemove(context.Background(), id, false)
}

// VolumeExists checks if a named volume exists.
func (dockerClient *Client) VolumeExists(name string) (bool, error) {
	_, err := dockerClient.client.VolumeInspect(context.Background(), name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateContainer creates a docker container.
func (dockerClient *Client) CreateContainer(options *docker.CreateContainerOptions) (string, error) {
	container, err := dockerClient.client.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: options.Image,
		},
		&container.HostConfig{
			Binds: options.Binds,
		},
		nil,
		nil,
		"",
	)
	if err != nil {
		return "", err
	}
	return container.ID, nil
}

// RemoveContainer removes a docker container.
func (dockerClient *Client) RemoveContainer(id string) error {
	return dockerClient.client.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{RemoveVolumes: true})
}

// CopyFromContainer returns a tar stream for a path within a container (like `docker cp CONTAINER -`).
func (dockerClient *Client) CopyFromContainer(id string, path string) (io.ReadCloser, error) {
	reader, _, err := dockerClient.client.CopyFromContainer(context.Background(), id, path)
	return reader, err
}

// CopyToContainer takes a tar stream and copies it into the container.
func (dockerClient *Client) CopyToContainer(id string, path string, reader io.Reader) error {
	return dockerClient.client.CopyToContainer(context.Background(), id, path, reader, types.CopyToContainerOptions{})
}

func (dockerClient *Client) streamHijackedResponse(hijackedResponse types.HijackedResponse, inputStream io.Reader, outputStream, errorStream io.Writer, start func() error) error {
	if inputStream != nil {
		go func() {
			defer hijackedResponse.CloseWrite()         // add to error below
			io.Copy(hijackedResponse.Conn, inputStream) // expected error here - catch and check
		}()
	}
	defer hijackedResponse.Close() // no error return value

	if err := start(); err != nil {
		return err
	}

	_, err := stdcopy.StdCopy(outputStream, errorStream, hijackedResponse.Reader)
	return err
}
