package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	common.Run(NewHandler(), os.Stdin, os.Stdout, os.Stderr)
}

type handler struct{}

// NewHandler returns a new handler.
func NewHandler() common.Handler {
	return &handler{}
}

// ConfigureRelease handles a configure release request in order to prepare for the release container to be ran.
func (*handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse, errorStream io.Writer) error {
	if err := json.NewEncoder(errorStream).Encode(map[string]interface{}{
		"Action":  "configure_release",
		"Request": &request,
	}); err != nil {
		return err
	}
	response.Env = map[string]string{
		"TEST_VERSION":                 request.Version,
		"TEST_RELEASE_VAR_FROM_ENV":    request.Env["TEST_ENV_VAR"],
		"TEST_RELEASE_VAR_FROM_CONFIG": fmt.Sprintf("%v", request.Config["TEST_CONFIG_VAR"]),
	}
	return nil
}

// UploadRelease handles an upload release request in order to upload the release after the release container is run.
func (*handler) UploadRelease(request *common.UploadReleaseRequest, response *common.UploadReleaseResponse, errorStream io.Writer, version string) error {
	var releaseMetadata map[string]map[string]string
	data, err := ioutil.ReadFile("/release/release-metadata.json")
	if err != nil {
		log.Panicln("could not read /release/release-metadata.json:", err)
	}
	if err := json.Unmarshal(data, &releaseMetadata); err != nil {
		log.Panicln("could not decode /release/release-metadata.json:", err)
	}
	if err := json.NewEncoder(errorStream).Encode(map[string]interface{}{
		"Action":          "upload_release",
		"Request":         &request,
		"ReleaseMetadata": releaseMetadata,
	}); err != nil {
		return err
	}
	response.Message = "uploaded " + version
	return nil
}

// PrepareTerraform handles a prepare terraform request in order to provide configuration for terraform during a deploy, destroy, etc.
func (*handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse, errorStream io.Writer) error {
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
	if err := json.NewEncoder(errorStream).Encode(map[string]interface{}{
		"Action":  "prepare_terraform",
		"Request": &request,
		"PWD":     dir,
	}); err != nil {
		return err
	}
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
