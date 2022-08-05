package command_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/manifest"
	release "github.com/mergermarket/cdflow2/release/command"
	"github.com/mergermarket/cdflow2/test"
)

func Equals(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestParseArgsWhenEnv(t *testing.T) {

	assertMatchArgs := func(t *testing.T, gotArgs, wantArgs *release.CommandArgs) {
		t.Helper()
		if gotArgs.Version != wantArgs.Version {
			t.Errorf("Version: got %s want %s", gotArgs.Version, wantArgs.Version)
		}
		if !Equals(gotArgs.ReleaseData, wantArgs.ReleaseData) {
			t.Errorf("ReleaseData: got %s want %s", gotArgs.ReleaseData, wantArgs.ReleaseData)
		}
	}

	assertError := func(t *testing.T, got, want error) {
		t.Helper()
		if got != nil && got.Error() != want.Error() {
			t.Errorf("Error: got %s want %s", got, want)
		}
	}

	t.Run("version", func(t *testing.T) {
		args := []string{"beta"}

		gotArgs, gotError := release.ParseArgs(args)

		var wantArgs release.CommandArgs
		wantArgs.Version = "beta"

		var wantError error = nil

		assertMatchArgs(t, gotArgs, &wantArgs)
		assertError(t, gotError, wantError)

	})

	t.Run("--release-data and version", func(t *testing.T) {
		args := []string{"--release-data", "foo=bar", "version1"}

		gotArgs, gotError := release.ParseArgs(args)

		var wantArgs release.CommandArgs
		wantArgs.ReleaseData = map[string]string{"foo": "bar"}
		wantArgs.Version = "version1"

		var wantError error = nil

		assertMatchArgs(t, gotArgs, &wantArgs)
		assertError(t, gotError, wantError)

	})

	t.Run("multiple --release-data flags", func(t *testing.T) {
		args := []string{"--release-data", "foo=bar", "--release-data", "more=data", "version1"}

		gotArgs, gotError := release.ParseArgs(args)

		var wantArgs release.CommandArgs
		wantArgs.ReleaseData = map[string]string{"foo": "bar", "more": "data"}
		wantArgs.Version = "version1"

		var wantError error = nil

		assertMatchArgs(t, gotArgs, &wantArgs)
		assertError(t, gotError, wantError)

	})

	t.Run("--release-data in wrong format", func(t *testing.T) {
		args := []string{"--release-data", "foo:bar", "version1"}

		_, gotError := release.ParseArgs(args)

		var wantError error = errors.New("Release data not in the correct format")

		assertError(t, gotError, wantError)

	})

	t.Run("empty args", func(t *testing.T) {
		args := []string{}

		_, gotError := release.ParseArgs(args)

		var wantError error = errors.New("Env missing value")

		assertError(t, gotError, wantError)

	})

	t.Run("missing version", func(t *testing.T) {
		args := []string{"--release-data", "foo=bar"}

		_, gotError := release.ParseArgs(args)

		var wantError error = errors.New("missing value")

		assertError(t, gotError, wantError)

	})

}

func TestRunCommand(t *testing.T) {

	// Given
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	// When
	if err := release.RunCommand(
		&command.GlobalState{
			DockerClient: dockerClient,
			Component:    "test-component",
			Commit:       "test-commit",
			OutputStream: &outputBuffer,
			ErrorStream:  &errorBuffer,
			CodeDir:      test.GetConfig("TEST_ROOT") + "/test/release/sample-code",
			Manifest: &manifest.Manifest{
				Version: 2,
				Builds: map[string]manifest.ImageWithParams{
					"buildid": {
						Image:  test.GetConfig("TEST_RELEASE_IMAGE"),
						Params: map[string]interface{}{"a": "b"},
					},
				},
				Terraform: manifest.Terraform{
					Image: test.GetConfig("TEST_TERRAFORM_IMAGE"),
				},
				Config: manifest.ImageWithParams{
					Image: test.GetConfig("TEST_CONFIG_IMAGE"),
					Params: map[string]interface{}{
						"test-manifest-config-key": "test-manifest-config-value",
					},
				},
			},
			GlobalArgs: &command.GlobalArgs{
				NoPullConfig:    true,
				NoPullRelease:   true,
				NoPullTerraform: true,
			},
		},
		release.CommandArgs{
			Version: "test-version",
		},
		map[string]string{},
	); err != nil {
		log.Fatalln("error running command:", err, errorBuffer.String())
	}

	// Then
	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		t.Fatal("error getting debug info:", err)
	}

	test.CheckTerraformInitInitialReflectedInput(debugInfo["terraform"])

	checkConfigureReleaseOutput(t, debugInfo["configure-release.json"])

	checkUploadReleaseOutput(t, debugInfo["upload-release.json"])

	if !strings.Contains(errorBuffer.String(), "message to stderr from release\ndocker status: OK\n") {
		t.Fatal("unexpected output of release:", errorBuffer.String())
	}

	if !strings.Contains(errorBuffer.String(), "uploaded test-version\n") {
		t.Fatalf("expected %q to contain %q", errorBuffer.String(), "uploaded test-version\n")
	}
}

func checkConfigureReleaseOutput(t *testing.T, debugOutput []byte) {
	var decoded struct {
		Action  string
		Request struct {
			Version string
			Config  map[string]interface{}
		}
	}

	if err := json.Unmarshal(debugOutput, &decoded); err != nil {
		t.Fatal("error decoding configure release debug output:", err)
	}

	if decoded.Action != "configure_release" {
		t.Fatal("unexpected action for configure releaes:", decoded.Action)
	}

	if decoded.Request.Version != "test-version" {
		t.Fatal("unexpected version passed to configure release:", decoded.Request.Version)
	}

	if decoded.Request.Config["test-manifest-config-key"] != "test-manifest-config-value" {
		t.Fatal("unexpected config value:", decoded.Request.Config["test-manifest-config-key"])
	}
}

func checkUploadReleaseOutput(t *testing.T, debugOutput []byte) {
	var decoded struct {
		Action  string
		Request struct {
			TerraformImage string
		}
		ReleaseMetadata map[string]map[string]string
	}
	if err := json.Unmarshal(debugOutput, &decoded); err != nil {
		t.Fatalf("error decoding upload release debug output: %v, '%v'", err, string(debugOutput))
	}

	if decoded.Action != "upload_release" {
		t.Fatal("unexpected action for upload releaes:", decoded.Action)
	}

	expectedTerraformImage := test.GetConfig("TEST_TERRAFORM_REPO_DIGEST")
	if decoded.Request.TerraformImage != expectedTerraformImage {
		t.Fatal("expected terraform repo digest: ", expectedTerraformImage, ", got:", decoded.Request.TerraformImage)
	}

	if decoded.ReleaseMetadata["buildid"]["component_from_defaults"] != "test-component" {
		t.Fatal("expected component test-component, got:", decoded.ReleaseMetadata["buildid"]["component_from_defaults"])
	}

	if decoded.ReleaseMetadata["buildid"]["commit_from_defaults"] != "test-commit" {
		t.Fatal("expected commit test-commit, got:", decoded.ReleaseMetadata["buildid"]["commit_from_defaults"])
	}

	if decoded.ReleaseMetadata["release"]["version"] != "test-version" {
		t.Fatal("unexpected version from release metadata:", decoded.ReleaseMetadata["release"]["version"])
	}
	if decoded.ReleaseMetadata["release"]["commit"] != "test-commit" {
		t.Fatal("unexpected commit from release metadata:", decoded.ReleaseMetadata["release"]["commit"])
	}
	if decoded.ReleaseMetadata["release"]["component"] != "test-component" {
		t.Fatal("unexpected component from release metadata:", decoded.ReleaseMetadata["release"]["component"])
	}
	if decoded.ReleaseMetadata["release"]["foo"] != "bar" {
		t.Fatal("unexpected config provided release metadata:", decoded.ReleaseMetadata["release"]["foo"])
	}

	if decoded.ReleaseMetadata["buildid"]["manifest_params"] != "{\"a\":\"b\"}" {
		t.Fatal("unexpected manifest_params:", decoded.ReleaseMetadata["buildid"]["manifest_params"])
	}
}
