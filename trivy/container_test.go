package trivy_test

import (
	"bytes"
	"testing"

	"github.com/mergermarket/cdflow2/test"
	"github.com/mergermarket/cdflow2/trivy"
)

func TestNewTrivyContainer(t *testing.T) {
	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	buildVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, buildVolume)

	codeDir := test.GetConfig("TEST_ROOT") + "/test/trivy/sample-code"
	params := map[string]interface{}{
		trivy.CONFIG_ERROR_ON_FINDINGS: false,
	}

	func() { // Ensure the code directory exists
		// When
		trivyContainer, err := trivy.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TRIVY_IMAGE"),
			codeDir,
			params,
		)
		if err != nil {
			t.Fatal("error creating trivy container:", err)
		}
		defer func() {
			if err := trivyContainer.Done(); err != nil {
				t.Fatal("error cleaning up trivy container:", err)
			}
		}()
	}()
}

func TestTrivyLocalScan(t *testing.T) {
	outputBuffer := &bytes.Buffer{}
	errorBuffer := &bytes.Buffer{}

	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	buildVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, buildVolume)

	codeDir := test.GetConfig("TEST_ROOT") + "/test/trivy/sample-code"
	params := map[string]interface{}{
		trivy.CONFIG_ERROR_ON_FINDINGS: false,
	}
	func() { // Ensure the code directory exists
		// When
		trivyContainer, err := trivy.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TRIVY_IMAGE"),
			codeDir,
			params,
		)
		if err != nil {
			t.Fatal("error creating trivy container:", err)
		}
		defer func() {
			if err := trivyContainer.Done(); err != nil {
				t.Fatal("error cleaning up trivy container:", err)
			}
		}()
		if _, err := trivyContainer.ScanRepository(
			outputBuffer,
			errorBuffer,
		); err != nil {
			t.Fatalf("unexpected error during local scan: %v", err)
		}
	}()

	// Then
	if errorBuffer.String() != "" {
		t.Errorf("expected no error output, got: %s", errorBuffer.String())
	}
	expectedString := "[trivy fs --severity CRITICAL --ignore-unfixed --scanners vuln,secret --exit-code 5 /code]"
	if !bytes.Contains(outputBuffer.Bytes(), []byte(expectedString)) {
		t.Errorf("expected output to contain %s, got: %s", expectedString, outputBuffer.String())
	}
}

func TestTrivyImageScan(t *testing.T) {
	outputBuffer := &bytes.Buffer{}
	errorBuffer := &bytes.Buffer{}

	// Given
	dockerClient, debugVolume := test.GetDockerClientWithDebugVolume()
	defer test.RemoveVolume(dockerClient, debugVolume)

	buildVolume := test.CreateVolume(dockerClient)
	defer test.RemoveVolume(dockerClient, buildVolume)

	codeDir := test.GetConfig("TEST_ROOT") + "/test/trivy/sample-code"
	params := map[string]interface{}{
		trivy.CONFIG_ERROR_ON_FINDINGS: false,
	}

	func() {
		// When
		trivyContainer, err := trivy.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TRIVY_IMAGE"),
			codeDir,
			params,
		)
		if err != nil {
			t.Fatal("error creating trivy container:", err)
		}
		defer func() {
			if err := trivyContainer.Done(); err != nil {
				t.Fatal("error cleaning up trivy container:", err)
			}
		}()
		if _, err := trivyContainer.ScanImage(
			"test-image:latest", // Replace with an actual image if needed
			outputBuffer,
			errorBuffer,
		); err != nil {
			t.Fatalf("unexpected error during local scan: %v", err)
		}
	}()

	// Then
	if errorBuffer.String() != "" {
		t.Errorf("expected no error output, got: %s", errorBuffer.String())
	}
	expectedString := "[trivy image --severity CRITICAL --ignore-unfixed --scanners vuln,misconfig,secret --exit-code 5 test-image:latest]"
	if !bytes.Contains(outputBuffer.Bytes(), []byte(expectedString)) {
		t.Errorf("expected output to contain %s, got: %s", expectedString, outputBuffer.String())
	}

}
