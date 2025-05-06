package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/release/container"
	"github.com/mergermarket/cdflow2/terraform"
)

type terraformResult struct {
	savedTerraformImage string
	err                 error
}

type output struct {
	stdout bool
	output []byte
	err    error
}

// CommandArgs contains specific arguments to the deploy command.
type CommandArgs struct {
	ReleaseData       map[string]string
	Version           string
	TerraformLogLevel string
}

func parseReleaseData(value string) (map[string]string, error) {
	dataStrings := strings.SplitN(value, "=", 2)
	if len(dataStrings) == 2 {
		return map[string]string{dataStrings[0]: dataStrings[1]}, nil
	} else {
		return nil, errors.New("release data not in the correct format")
	}
}

func handleArgs(arg string, commandArgs *CommandArgs, take func() (string, error)) (bool, error) {
	if strings.HasPrefix(arg, "-") {
		return handleFlag(arg, commandArgs, take)
	} else if commandArgs.Version == "" {
		commandArgs.Version = arg
	} else {
		return false, errors.New("unknown release argument: " + arg)
	}
	return false, nil
}

func handleFlag(arg string, commandArgs *CommandArgs, take func() (string, error)) (bool, error) {
	if arg == "-r" || arg == "--release-data" {
		value, err := take()
		if err != nil {
			return false, err
		}
		releaseData, err := parseReleaseData(value)
		if err != nil {
			return false, err
		}
		for k, v := range releaseData {
			commandArgs.ReleaseData[k] = v
		}
	} else if arg == "-t" || arg == "--terraform-log-level" {
		value, err := take()
		if err != nil {
			return false, err
		}

		commandArgs.TerraformLogLevel = value
	} else {
		return false, errors.New("unknown release option: " + arg)
	}
	return false, nil
}

// ParseArgs parses command line arguments to the shell subcommand.
func ParseArgs(args []string) (*CommandArgs, error) {
	var result CommandArgs
	result.ReleaseData = make(map[string]string)

	i := 0
	take := func() (string, error) {
		i++
		if i >= len(args) {
			return "", errors.New("missing value")
		}

		return args[i], nil
	}
	for ; i < len(args); i++ {
		_, err := handleArgs(args[i], &result, take)
		if err != nil {
			return nil, err
		}
	}

	if result.Version == "" {
		return nil, errors.New("version argument is missing")
	}

	return &result, nil
}

func pipeToOutput(stdout bool, reader io.Reader, outputChan chan *output) {
	for {
		buffer := make([]byte, 10*1024)
		n, err := reader.Read(buffer)
		outputChan <- &output{stdout, buffer[:n], err}
		if err != nil {
			break
		}
	}
}

func getOutputCapture() (chan *output, io.WriteCloser, io.WriteCloser) {
	outputReader, outputWriter := io.Pipe()
	errorReader, errorWriter := io.Pipe()
	outputChan := make(chan *output, 10*1024)
	go pipeToOutput(true, outputReader, outputChan)
	go pipeToOutput(false, errorReader, outputChan)
	return outputChan, outputWriter, errorWriter
}

func streamOutput(terraformOutputChan chan *output, outputStream, errorStream io.Writer) error {
	eofs := 0
	for {
		terraformOutput := <-terraformOutputChan
		if len(terraformOutput.output) > 0 {
			if terraformOutput.stdout {
				outputStream.Write(terraformOutput.output)
			} else {
				errorStream.Write(terraformOutput.output)
			}
		}
		if terraformOutput.err == io.EOF {
			eofs++
			if eofs == 2 {
				return nil
			}
		} else if terraformOutput.err != nil {
			return terraformOutput.err
		}
	}
}

func terraformRelease(state *command.GlobalState, buildVolume string, outputStream, errorStream io.Writer, logLevel string) (image string, returnedError error) {
	dockerClient := state.DockerClient

	if !state.GlobalArgs.NoPullTerraform {
		fmt.Fprintf(errorStream, "\nPulling terraform image %v...\n\n", state.Manifest.Terraform.Image)
		if err := dockerClient.PullImage(state.Manifest.Terraform.Image, errorStream); err != nil {
			return "", fmt.Errorf("error pulling terraform image: %w", err)
		}
	}

	repoDigests, err := dockerClient.GetImageRepoDigests(state.Manifest.Terraform.Image)
	if err != nil {
		return "", err
	}
	if len(repoDigests) == 0 {
		return "", fmt.Errorf("no docker repo digest(s) available for image %v", state.Manifest.Terraform.Image)
	}
	savedTerraformImage := repoDigests[0]

	if state.GlobalArgs.NoPullTerraform && savedTerraformImage == "" {
		savedTerraformImage = state.Manifest.Terraform.Image
	} else if savedTerraformImage == "" {
		log.Panicln("no repo digest for ", state.Manifest.Terraform.Image)
	}

	terraformContainer, err := terraform.NewContainer(
		state.DockerClient,
		savedTerraformImage,
		state.CodeDir,
		buildVolume,
		logLevel,
	)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := terraformContainer.Done(); err != nil {
			if returnedError != nil {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
		}
	}()

	return savedTerraformImage, terraformContainer.InitInitial(outputStream, errorStream)
}

// RunCommand runs the release command.
func RunCommand(state *command.GlobalState, releaseArgs CommandArgs, env map[string]string) (returnedError error) {

	dockerClient := state.DockerClient
	terraformOutputChan, terraformOutputStream, terraformErrorStream := getOutputCapture()
	terraformResultChan := make(chan *terraformResult, 1)

	buildVolume, err := dockerClient.CreateVolume("")
	if err != nil {
		return err
	}
	defer func() {
		<-terraformResultChan // wait for terraformRelease to finish, otherwise buildVolume will be in use and cannot be deleted
		if err := dockerClient.RemoveVolume(buildVolume); err != nil {
			if returnedError != nil && returnedError.Error() != "" {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
			return
		}
	}()

	go func() {
		savedTerraformImage, err := terraformRelease(state, buildVolume, terraformOutputStream, terraformErrorStream, releaseArgs.TerraformLogLevel)
		terraformOutputStream.Close()
		terraformErrorStream.Close()
		terraformResultChan <- &terraformResult{savedTerraformImage, err}
		close(terraformResultChan)
	}()

	if err := config.Pull(state); err != nil {
		return err
	}

	message, err := buildAndUploadRelease(state, buildVolume, releaseArgs.Version, releaseArgs.ReleaseData, terraformResultChan, terraformOutputChan, env)
	if err != nil {
		return err
	}

	// not in the above function to ensure docker output flushed before that finishes
	fmt.Fprintln(state.ErrorStream, message)

	return nil
}

func buildAndUploadRelease(state *command.GlobalState, buildVolume, version string, releaseData map[string]string, terraformResultChan chan *terraformResult, terraformOutputChan chan *output, env map[string]string) (returnedMessage string, returnedError error) {

	releaseRequirements, err := GetReleaseRequirements(state)
	if err != nil {
		return "", err
	}

	dockerClient := state.DockerClient
	configContainer, err := config.NewContainer(state, state.Manifest.Config.Image, buildVolume)
	if err != nil {
		return "", err
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

	fmt.Print("\ncdflow2: getting release configuration...\n\n")

	configureReleaseResponse, err := configContainer.ConfigureRelease(
		version,
		state.Component,
		state.Commit,
		state.Manifest.Config.Params,
		env,
		releaseRequirements,
	)
	if err != nil {
		return "", err
	}

	releaseEnv := configureReleaseResponse.Env
	state.MonitoringClient.APIKey = configureReleaseResponse.Monitoring.APIKey
	state.MonitoringClient.ConfigData = configureReleaseResponse.Monitoring.Data

	releaseMetadata := make(map[string]map[string]string)
	for buildID, build := range state.Manifest.Builds {
		env := releaseEnv[buildID]

		if env == nil {
			env = make(map[string]string)
		}
		
		// Iterate over each environment variable name in build.EnvVars,
		// retrieve its value from the environment, and add it to the env map
		// without printing the value to avoid leaking secrets.
		for _, envvar := range build.EnvVars {
			v := os.Getenv(envvar)
			if v == "" {
				fmt.Printf("\nWarning: Environment variable %s is not set in host environment.\n\n", envvar)
			} else {
				fmt.Printf("\n\nAdding Environment variable %s into cdflow release container.\n\n", envvar)
			}
			env[envvar] = v
		}

		// these are built in and cannot be overridden by the config container (since choosing the clashing name would likely be an accident)
		env["VERSION"] = version
		env["COMPONENT"] = state.Component
		env["COMMIT"] = state.Commit
		env["BUILD_ID"] = buildID
		manifestParams, err := json.Marshal(build.Params)
		if err != nil {
			return "", err
		}
		env["MANIFEST_PARAMS"] = string(manifestParams)
		metadata, err := container.Run(
			dockerClient,
			build.Image,
			state.CodeDir,
			buildVolume,
			state.OutputStream,
			state.ErrorStream,
			env,
		)
		if err != nil {
			return "", fmt.Errorf("cdflow2: error running build '%v' - %w", buildID, err)
		}
		releaseMetadata[buildID] = metadata
	}
	if releaseMetadata["release"] == nil {
		releaseMetadata["release"] = make(map[string]string)
	}
	releaseMetadata["release"]["version"] = version
	releaseMetadata["release"]["commit"] = state.Commit
	releaseMetadata["release"]["component"] = state.Component
	for k, v := range releaseData {
		releaseMetadata["release"][k] = v
	}
	for k, v := range configureReleaseResponse.AdditionalMetadata {
		releaseMetadata["release"][k] = v
	}

	if err := configContainer.WriteReleaseMetadata(releaseMetadata); err != nil {
		return "", err
	}

	terraformResult := <-terraformResultChan
	if err := streamOutput(terraformOutputChan, state.OutputStream, state.ErrorStream); err != nil {
		return "", err
	}

	if terraformResult.err != nil {
		return "", terraformResult.err
	}
	fmt.Fprintf(state.OutputStream, "Checking for .terraform.lock.hcl \n")
	if _, err := os.Stat("./infra/.terraform.lock.hcl"); err == nil {
		fmt.Fprintf(state.OutputStream, "	Adding .terraform.lock.hcl to release \n")
		b, err := ioutil.ReadFile("./infra/.terraform.lock.hcl")
		if err != nil {
			return "", fmt.Errorf("error on reading .terraform.lock.hcl %w", err)
		}
		if err := configContainer.CopyFileToRelease(".terraform.lock.hcl", b); err != nil {
			return "", err
		}

	}

	fmt.Print("\ncdflow2: uploading release...\n\n")

	uploadReleaseResponse, err := configContainer.UploadRelease(
		terraformResult.savedTerraformImage,
	)
	if err != nil {
		return "", fmt.Errorf("error uploading release: %w", err)
	}

	return uploadReleaseResponse.Message, nil
}

// GetReleaseRequirements runs the release containers in order to get their requirements.
func GetReleaseRequirements(state *command.GlobalState) (map[string]*config.ReleaseRequirements, error) {
	result := make(map[string]*config.ReleaseRequirements)
	for buildID, build := range state.Manifest.Builds {
		if !state.GlobalArgs.NoPullRelease {
			fmt.Fprintf(state.ErrorStream, "\nPulling build image (%v): %v...\n\n", buildID, build.Image)
			if err := state.DockerClient.PullImage(build.Image, state.ErrorStream); err != nil {
				return nil, fmt.Errorf("error pulling build image (%v): %w", buildID, err)
			}
		}
		requirements, err := container.GetReleaseRequirements(state, buildID, build.Image, state.ErrorStream)
		if err != nil {
			return nil, err
		}
		result[buildID] = requirements
	}
	return result, nil
}
