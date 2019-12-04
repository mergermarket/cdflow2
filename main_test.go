package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

func getConfig(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("environment variable %v not set - did you run ./test.sh?", name)
	}
	return value
}

type ReflectedInput struct {
	Args  []string
	Env   map[string]string
	Input string
	Cwd   string
	File  string
}

func tempdir() (string, error) {
	// not using the native TempDir since the directory is not configured to be sharable with docker
	// containers on OSX by default :-(
	rand.Seed(time.Now().UnixNano())
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	dir := "/tmp/cdflow2-test-" + string(b)
	if err := os.Mkdir(dir, 0777); err != nil {
		return "", err
	}
	return dir, nil
}

func TestRelease(t *testing.T) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	buildDir, err := tempdir()
	if err != nil {
		log.Fatalf("could not make tempdir: %v", err)
	}
	defer os.RemoveAll(buildDir)
	terraformInit(dockerClient, getConfig("TEST_TERRAFORM_IMAGE"), getConfig("TEST_ROOT")+"/test/terraform/sample-code", buildDir, &outputBuffer, &errorBuffer)

	if errorBuffer.String() != "message to stderr\n" {
		log.Fatalf("unexpected stderr output: '%v'", errorBuffer.String())
	}

	var output ReflectedInput
	json.Unmarshal(outputBuffer.Bytes(), &output)

	// interface is that the code is mapped to /code and the terraform is in the infra subfolder
	if !reflect.DeepEqual(output.Args, []string{"init", "/code/infra"}) {
		log.Fatalf("unexpected args: %v", output.Args)
	}

	// interface is that the mapped in cwd is /build
	if output.Cwd != "/build" {
		log.Fatalf("unexpected cwd: %v", output.Cwd)
	}

	if output.File != "sample content" {
		log.Fatalf("code not mapped as /code - file contents: %v", output.File)
	}
}
