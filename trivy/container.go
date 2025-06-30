package trivy

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/mergermarket/cdflow2/docker"
)

type Config struct {
	errorOnFindings bool
}

type Container struct {
	dockerClient docker.Iface
	id           string
	done         chan error
	codeDir      string
	config       Config
}

const CODE_DIR = "/code"
const CONFIG_ERROR_ON_FINDINGS = "errorOnFindings"

func NewContainer(dockerClient docker.Iface,
	image,
	codeDir string,
	params map[string]interface{}) (*Container, error) {

	config, err := GetConfig(params)
	if err != nil {
		return nil, fmt.Errorf("error getting trivy config: %w", err)
	}
	started := make(chan string, 1)
	defer close(started)

	done := make(chan error, 1)

	var outputBuffer bytes.Buffer

	go func() {
		done <- dockerClient.Run(
			&docker.RunOptions{
				Image:        image,
				OutputStream: &outputBuffer,
				ErrorStream:  &outputBuffer,
				WorkingDir:   CODE_DIR,
				Entrypoint:   []string{"/bin/sleep"},
				Cmd:          []string{strconv.Itoa(365 * 24 * 60 * 60)},
				Started:      started,
				Init:         true, // Use init to ensure the container is killed properly
				NamePrefix:   "cdflow2-trivy-",
				Binds: []string{
					codeDir + ":" + CODE_DIR,
					"/var/run/docker.sock:/var/run/docker.sock",
				},
				SuccessStatus: 128 + 15, // 128 + SIGTERM
			})
	}()

	select {
	case id := <-started:
		return &Container{
			dockerClient: dockerClient,
			id:           id,
			done:         done,
			codeDir:      codeDir,
			config:       config,
		}, nil
	case err := <-done:
		return nil, fmt.Errorf("could not start trivy container: %w\nOutput: %v", err, outputBuffer.String())

	}
}

func (trivyContainer *Container) ScanRepository(outputStream, errorStream io.Writer) error {
	cmd := []string{
		"trivy",
		"fs",
		"--severity", "CRITICAL",
		"--ignore-unfixed",
		"--exit-code", trivyContainer.setExitCode(),
		CODE_DIR,
	}
	return trivyContainer.dockerClient.Exec(
		&docker.ExecOptions{
			ID:           trivyContainer.id,
			Cmd:          cmd,
			OutputStream: outputStream,
			ErrorStream:  errorStream,
			Tty:          false,
		})
}

func (trivyContainer *Container) ScanImage(image string, outputStream, errorStream io.Writer) error {
	cmd := []string{
		"trivy",
		"image",
		"--severity", "CRITICAL",
		"--ignore-unfixed",
		"--exit-code", trivyContainer.setExitCode(),
		image,
	}
	return trivyContainer.dockerClient.Exec(
		&docker.ExecOptions{
			ID:           trivyContainer.id,
			Cmd:          cmd,
			OutputStream: outputStream,
			ErrorStream:  errorStream,
			Tty:          false,
		})
}

func (trivyContainer *Container) Done() error {
	if err := trivyContainer.dockerClient.Stop(trivyContainer.id, 10); err != nil {
		return err
	}
	return <-trivyContainer.done
}

func GetConfig(params map[string]interface{}) (Config, error) {
	config := Config{
		errorOnFindings: true,
	}
	if val, ok := params[CONFIG_ERROR_ON_FINDINGS]; ok {
		if errorOnFindings, ok := val.(bool); ok {
			config.errorOnFindings = errorOnFindings
		}
	}
	return config, nil
}

func (trivyContainer *Container) setExitCode() string {
	if trivyContainer.config.errorOnFindings {
		return "1"
	}
	return "0"
}
