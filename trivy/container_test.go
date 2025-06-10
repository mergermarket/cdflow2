package trivy_test

import (
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

	func() { // Ensure the code directory exists
		// When
		trivyContainer, err := trivy.NewContainer(
			dockerClient,
			test.GetConfig("TEST_TRIVY_IMAGE"),
			codeDir,
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

// func TestTrivyLocalScan(t *testing.T) {

// }

// func TestTrivyImageScan(t *testing.T) {

// }
