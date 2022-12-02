package shell_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/manifest"
	"github.com/mergermarket/cdflow2/monitoring"
	"github.com/mergermarket/cdflow2/shell"
	"github.com/mergermarket/cdflow2/test"
)

func TestRunCommand(t *testing.T) {

	// Given
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	state := &command.GlobalState{
		DockerClient: dockerClient,
		InputStream:  nil,
		OutputStream: &outputBuffer,
		ErrorStream:  &errorBuffer,
		CodeDir:      test.GetConfig("TEST_ROOT") + "/test/release/sample-code",
		Component:    "test-component",
		Commit:       "test-commit",
		Manifest: &manifest.Manifest{
			Version: 2,
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
			NoPullTerraform: true,
		},
		MonitoringClient: monitoring.NewDatadogClient(),
	}

	repoDigests, err := state.DockerClient.GetImageRepoDigests(test.GetConfig("TEST_TERRAFORM_IMAGE"))
	if err != nil {
		t.Fatal("could not get repo digests for terraform container:", err)
	}
	if len(repoDigests) == 0 {
		t.Fatal("no repo digests for terraform container", test.GetConfig("TEST_TERRAFORM_IMAGE"))
	}
	terraformDigest := repoDigests[0]
	args, _ := shell.ParseArgs([]string{"test-env", "-v", "test-version", "--", "-c", "terraform -v"})

	// When
	if err := shell.RunCommand(state, args, map[string]string{
		"TERRAFORM_DIGEST": terraformDigest,
	}); err != nil {
		t.Fatal("error running shell command:", err, errorBuffer.String())
	}

	// Then
	debugInfo, err := test.ReadVolume(dockerClient, debugVolume)
	if err != nil {
		t.Fatal("error getting debug info:", err)
	}

	checkPrepareTerraformOutput(t, debugInfo["prepare-terraform.json"])

	lines := bytes.Split(debugInfo["terraform"], []byte{'\n'})
	if len(lines) != 5 {
		t.Fatalf("expected five lines, got %v lines:\n%v", len(lines), debugInfo)
	}

	// TODO check terraform init

	test.CheckTerraformWorkspaceList(lines[1])
	test.CheckTerraformWorkspaceNew(lines[2], "test-env")

	checkCdflowShellOutput(t, lines[3])

}

func checkCdflowShellOutput(t *testing.T, output []byte) {
	var input test.ReflectedInput
	if err := json.Unmarshal(output, &input); err != nil {
		t.Fatal("error parsing json:", err)
	}

	if !reflect.DeepEqual(input.Args, []string{"-v"}) {

		t.Fatal("unexpected terraform args:", input.Args)
	}
}
func checkPrepareTerraformOutput(t *testing.T, debugOutput []byte) {
	var decoded struct {
		Action  string
		Request struct {
			Version string
			EnvName string
			Config  map[string]interface{}
			Env     map[string]string
		}
		PWD string
	}

	if err := json.Unmarshal(debugOutput, &decoded); err != nil {
		t.Fatal("error decoding prepare terraform debug output:", err)
	}

	if decoded.Action != "prepare_terraform" {
		t.Fatal("expected action prepare_terraform got:", decoded.Action)
	}
	if decoded.Request.Version != "test-version" {
		t.Fatal("expected version test-version got:", decoded.Request.Version)
	}
	if decoded.Request.EnvName != "test-env" {
		t.Fatal("expected env test-env got:", decoded.Request.EnvName)
	}
	if decoded.Request.Config["test-manifest-config-key"] != "test-manifest-config-value" {
		t.Fatal("expected config from manifest got:", decoded.Request.Config)
	}
	if decoded.PWD != "/release" {
		t.Fatal("expected prepare_terraform to run in /release got:", decoded.PWD)
	}

}

func Equals(a, b []string) bool {
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

	assertMatchArgs := func(t *testing.T, gotArgs, wantArgs *shell.CommandArgs) {
		t.Helper()
		if gotArgs.EnvName != wantArgs.EnvName {
			t.Errorf("EnvName: got %s want %s", gotArgs.EnvName, wantArgs.EnvName)
		}
		if gotArgs.Version != wantArgs.Version {
			t.Errorf("Version: got %s want %s", gotArgs.Version, wantArgs.Version)
		}
		if !Equals(gotArgs.ShellArgs, wantArgs.ShellArgs) {
			t.Errorf("ShellArgs: got %s want %s", gotArgs.ShellArgs, wantArgs.ShellArgs)
		}
	}

	assertError := func(t *testing.T, got, want error) {
		t.Helper()
		if got != nil && got.Error() != want.Error() {
			t.Errorf("Error: got %s want %s", got, want)
		}
	}

	t.Run("env", func(t *testing.T) {
		args := []string{"alfa"}
		gotArgs, gotError := shell.ParseArgs(args)

		var wantArgs shell.CommandArgs
		wantArgs.EnvName = "alfa"

		var wantError error = nil

		assertMatchArgs(t, gotArgs, &wantArgs)
		assertError(t, gotError, wantError)
	})

	t.Run("env and -v", func(t *testing.T) {
		args := []string{"alfa", "-v", "beta"}

		gotArgs, gotError := shell.ParseArgs(args)

		var wantArgs shell.CommandArgs
		wantArgs.EnvName = "alfa"
		wantArgs.Version = "beta"

		var wantError error = nil

		assertMatchArgs(t, gotArgs, &wantArgs)
		assertError(t, gotError, wantError)

	})

	t.Run("env and --version", func(t *testing.T) {
		args := []string{"alfa", "--version", "beta"}

		gotArgs, gotError := shell.ParseArgs(args)

		var wantArgs shell.CommandArgs
		wantArgs.EnvName = "alfa"
		wantArgs.Version = "beta"

		var wantError error = nil

		assertMatchArgs(t, gotArgs, &wantArgs)
		assertError(t, gotError, wantError)

	})

	t.Run("env and --version=", func(t *testing.T) {
		args := []string{"alfa", "--version=beta"}

		gotArgs, gotError := shell.ParseArgs(args)

		var wantArgs shell.CommandArgs
		wantArgs.EnvName = "alfa"
		wantArgs.Version = "beta"

		var wantError error = nil

		assertMatchArgs(t, gotArgs, &wantArgs)
		assertError(t, gotError, wantError)

	})

	t.Run("env and shellArgs", func(t *testing.T) {
		args := []string{"alfa", "--", "beta"}

		gotArgs, gotError := shell.ParseArgs(args)

		var wantArgs shell.CommandArgs
		wantArgs.EnvName = "alfa"
		wantArgs.ShellArgs = []string{"beta"}

		var wantError error = nil

		assertMatchArgs(t, gotArgs, &wantArgs)
		assertError(t, gotError, wantError)

	})

	t.Run("env and version and shellArgs", func(t *testing.T) {
		args := []string{"alfa", "-v", "beta", "--", "charlie", "delta"}

		gotArgs, gotError := shell.ParseArgs(args)

		var wantArgs shell.CommandArgs
		wantArgs.EnvName = "alfa"
		wantArgs.Version = "beta"
		wantArgs.ShellArgs = []string{"charlie", "delta"}

		var wantError error = nil

		assertMatchArgs(t, gotArgs, &wantArgs)
		assertError(t, gotError, wantError)

	})

	t.Run("missing env", func(t *testing.T) {
		args := []string{"-v", "beta", "--", "charlie"}

		_, gotError := shell.ParseArgs(args)

		var wantError error = errors.New("Env missing value")

		assertError(t, gotError, wantError)

	})

	t.Run("empty args", func(t *testing.T) {
		args := []string{}

		_, gotError := shell.ParseArgs(args)

		var wantError error = errors.New("Env missing value")

		assertError(t, gotError, wantError)

	})

	t.Run("missing version", func(t *testing.T) {
		args := []string{"alfa", "-v"}

		_, gotError := shell.ParseArgs(args)

		var wantError error = errors.New("missing value")

		assertError(t, gotError, wantError)

	})

}
