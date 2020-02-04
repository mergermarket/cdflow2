package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

// Message is a generic request, in ordre to get the type
type Message struct {
	Action string
}

func main() {
	var version string
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	// for sending diagnostic info for the tests
	stderrEncoder := json.NewEncoder(os.Stderr)
	for scanner.Scan() {
		line := scanner.Bytes()
		var message Message
		if err := json.Unmarshal(line, &message); err != nil {
			log.Fatalln("error reading message:", err)
		}
		switch message.Action {
		case "configure_release":
			version = configureRelease(line, encoder, stderrEncoder)
		case "upload_release":
			uploadRelease(line, version, encoder, stderrEncoder)
		case "prepare_terraform":
			prepareTerraform(line, encoder, stderrEncoder)
		case "stop":
			os.Exit(0)
		default:
			log.Fatalln("unknown message type:", message.Action)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("error reading from stdin:", err)
	}
}

type configureReleaseRequest struct {
	Version string
	Config  map[string]interface{}
	Env     map[string]string
}

type configureReleaseResponse struct {
	Env map[string]string
}

func configureRelease(line []byte, encoder, stderrEncoder *json.Encoder) string {
	var request configureReleaseRequest
	if err := json.Unmarshal(line, &request); err != nil {
		log.Fatalln("error reading configure release request:", err)
	}
	stderrEncoder.Encode(map[string]interface{}{
		"Action":  "configure_release",
		"Request": &request,
	})
	if err := encoder.Encode(configureReleaseResponse{
		Env: map[string]string{
			"TEST_VERSION":                 request.Version,
			"TEST_RELEASE_VAR_FROM_ENV":    request.Env["TEST_ENV_VAR"],
			"TEST_RELEASE_VAR_FROM_CONFIG": fmt.Sprintf("%v", request.Config["TEST_CONFIG_VAR"]),
		},
	}); err != nil {
		log.Fatalln("error sending configure release response:", err)
	}
	return request.Version
}

type uploadReleaseRequest struct {
	TerraformImage  string
	ReleaseMetadata map[string]string
}

type uploadReleaseResponse struct {
	Message string
}

func uploadRelease(line []byte, version string, encoder, stderrEncoder *json.Encoder) {
	var request uploadReleaseRequest
	if err := json.Unmarshal(line, &request); err != nil {
		log.Fatalln("error reading upload release request:", err)
	}
	stderrEncoder.Encode(map[string]interface{}{
		"Action":  "upload_release",
		"Request": &request,
	})
	if err := encoder.Encode(uploadReleaseResponse{
		Message: "uploaded " + version,
	}); err != nil {
		log.Fatalln("error sending upload release response:", err)
	}
}

type prepareTerraformRequest struct {
	Version string
	EnvName string
	Config  map[string]interface{}
	Env     map[string]string
}

type prepareTerraformResponse struct {
	TerraformImage         string
	Env                    map[string]string
	TerraformBackendType   string
	TerraformBackendConfig map[string]string
}

func prepareTerraform(line []byte, encoder, stderrEncoder *json.Encoder) {
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
	var request prepareTerraformRequest
	if err := json.Unmarshal(line, &request); err != nil {
		log.Fatalln("error reading prepare terraform request:", err)
	}
	stderrEncoder.Encode(map[string]interface{}{
		"Action":  "prepare_terraform",
		"Request": &request,
		"PWD":     dir,
	})
	if err := encoder.Encode(prepareTerraformResponse{
		TerraformImage: fmt.Sprintf("%v", request.Config["terraform-digest"]),
		Env: map[string]string{
			"TEST_ENV_VAR":    request.Env["TEST_ENV_VAR"],
			"TEST_CONFIG_VAR": fmt.Sprintf("%v", request.Config["TEST_CONFIG_VAR"]),
		},
		TerraformBackendType: "a-terraform-backend-type",
		TerraformBackendConfig: map[string]string{
			"backend-config-key": "backend-config-value",
		},
	}); err != nil {
		log.Fatalln("error sending prepare terraform response:", err)
	}
}
