package config_test

import (
	"log"
	"testing"

	"github.com/mergermarket/cdflow2/config"
	"github.com/mergermarket/cdflow2/test"
)

func TestLoadManifest(t *testing.T) {
	manifest, err := config.LoadManifest(test.GetConfig("TEST_ROOT") + "/test/config/sample-code")
	if err != nil {
		log.Fatalln("error loading manifest:", manifest)
	}
	if manifest.Version != 2 {
		log.Fatalln("unexpected version:", manifest.Version)
	}
	if manifest.ConfigImage != "test-config-image" {
		log.Fatalln("unexpected config image:", manifest.ConfigImage)
	}
	if manifest.ReleaseImage != "test-release-image" {
		log.Fatalln("unexpected release image:", manifest.ReleaseImage)
	}
	if manifest.TerraformImage != "test-terraform-image" {
		log.Fatalln("unexpected terraform image:", manifest.TerraformImage)
	}
	if manifest.Team != "test-team" {
		log.Fatalln("unexpected team:", manifest.Team)
	}
}
