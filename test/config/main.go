package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	common "github.com/mergermarket/cdflow2-config-common"
)

// Message is a generic request, in order to get the type
type Message struct {
	Action string
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "forward" {
		common.Forward(os.Stdin, os.Stdout, "")
	} else {
		common.Listen(NewHandler(), "", nil)
	}
}

type handler struct{}

// NewHandler returns a new handler.
func NewHandler() common.Handler {
	return &handler{}
}

func writeDebug(data interface{}, path string) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Panicf("error opening %v for write: %v\n", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Panicf("error closing %v: %v\n", path, err)
		}
	}()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		log.Panicf("error serialising %v as json to %v: %v", data, path, err)
	}
}

// Setup handles a setup request in order to pipeline setup.
func (*handler) Setup(request *common.SetupRequest, response *common.SetupResponse) error {
	fmt.Println("output to stdout from setup, component: " + request.Component + ", commit: " + request.Commit + ", team: " + request.Team)
	fmt.Fprintln(os.Stderr, "output to stderr from setup")
	return nil
}

// ConfigureRelease handles a configure release request in order to prepare for the release container to be ran.
func (*handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse) error {
	writeDebug(map[string]interface{}{
		"Action":  "configure_release",
		"Request": &request,
	}, "/debug/configure-release.json")
	response.Env = map[string]string{
		"TEST_VERSION":                 request.Version,
		"TEST_COMPONENT":               request.Component,
		"TEST_COMMIT":                  request.Commit,
		"TEST_TEAM":                    request.Team,
		"TEST_RELEASE_VAR_FROM_ENV":    request.Env["TEST_ENV_VAR"],
		"TEST_RELEASE_VAR_FROM_CONFIG": fmt.Sprintf("%v", request.Config["TEST_CONFIG_VAR"]),
	}
	return nil
}

// UploadRelease handles an upload release request in order to upload the release after the release container is run.
func (*handler) UploadRelease(
	request *common.UploadReleaseRequest,
	response *common.UploadReleaseResponse,
	version string,
	config map[string]interface{},
) error {
	var releaseMetadata map[string]map[string]string
	data, err := ioutil.ReadFile("/release/release-metadata.json")
	if err != nil {
		log.Panicln("could not read /release/release-metadata.json:", err)
	}
	if err := json.Unmarshal(data, &releaseMetadata); err != nil {
		log.Panicln("could not decode /release/release-metadata.json:", err)
	}
	writeDebug(map[string]interface{}{
		"Action":          "upload_release",
		"Request":         &request,
		"ReleaseMetadata": releaseMetadata,
	}, "/debug/upload-release.json")
	response.Message = "uploaded " + version
	return nil
}

// PrepareTerraform handles a prepare terraform request in order to provide configuration for terraform during a deploy, destroy, etc.
func (*handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse) error {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalln("could not get working directory:", err)
	}
	if dir != "/release" {
		log.Fatalln("expected PWD /release, got:", dir)
	}
	if err := ioutil.WriteFile("/release/test", []byte("unpacked"), 0644); err != nil {
		log.Fatalln("could not write to /release/test:", err)
	}
	writeDebug(map[string]interface{}{
		"Action":  "prepare_terraform",
		"Request": &request,
		"PWD":     dir,
	}, "/debug/prepare-terraform.json")
	response.TerraformImage = request.Env["TERRAFORM_DIGEST"]
	response.Env = map[string]string{
		"TEST_ENV_VAR":    request.Env["TEST_ENV_VAR"],
		"TEST_CONFIG_VAR": fmt.Sprintf("%v", request.Config["TEST_CONFIG_VAR"]),
	}
	response.TerraformBackendType = "a-terraform-backend-type"
	response.TerraformBackendConfig = map[string]string{
		"backend-config-key": "backend-config-value",
	}
	return nil
}
