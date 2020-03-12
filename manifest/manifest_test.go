package manifest_test

import (
	"log"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/manifest"
	"github.com/mergermarket/cdflow2/test"
)

func TestLoad(t *testing.T) {
	loadedManifest, err := manifest.Load(test.GetConfig("TEST_ROOT") + "/test/config/sample-code")
	if err != nil {
		log.Fatalln("error loading manifest:", err)
	}
	if loadedManifest.Version != 2 {
		log.Fatalln("unexpected version:", loadedManifest.Version)
	}
	if !reflect.DeepEqual(loadedManifest.Builds, map[string]manifest.ImageWithParams{
		"release": {Image: "test-release-image"},
	}) {
		log.Fatalln("unexpected release data from manifest:", loadedManifest.Builds)
	}
	if loadedManifest.Config.Image != "test-config-image" {
		log.Fatalln("unexpected config image from manifest:", loadedManifest.Config.Image)
	}
	if loadedManifest.Terraform.Image != "test-terraform-image" {
		log.Fatalln("unexpected terraform image from manifest:", loadedManifest.Terraform.Image)
	}
	if loadedManifest.Team != "test-team" {
		log.Fatalln("unexpected team from manifest:", loadedManifest.Team)
	}
	if loadedManifest.Config.Params["config-key"] != "config-value" {
		log.Fatalln("unexpected config params from manifest:", loadedManifest.Config.Params)
	}
}
