package manifest_test

import (
	"log"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/manifest"
	"github.com/mergermarket/cdflow2/test"
)

func TestLoad(t *testing.T) {
	manifest, err := manifest.Load(test.GetConfig("TEST_ROOT") + "/test/config/sample-code")
	if err != nil {
		log.Fatalln("error loading manifest:", err)
	}
	if manifest.Version != 2 {
		log.Fatalln("unexpected version:", manifest.Version)
	}
	if !reflect.DeepEqual(manifest.Builds, map[string]string{
		"release": "release-image",
	}) {
		log.Fatalln("unexpected release data:", manifest.Builds)
	}
	if manifest.ConfigImage != "test-config-image" {
		log.Fatalln("unexpected config image:", manifest.ConfigImage)
	}
	if manifest.TerraformImage != "test-terraform-image" {
		log.Fatalln("unexpected terraform image:", manifest.TerraformImage)
	}
	if manifest.Team != "test-team" {
		log.Fatalln("unexpected team:", manifest.Team)
	}
	if manifest.Config["config-key"] != "config-value" {
		log.Fatalln("unexpected config from manifest:", manifest.Config)
	}
}

func TestCanonicaliseSimple(t *testing.T) {
	result, err := manifest.Canonicalise(&manifest.Manifest{
		Version:   2,
		Team:      "test-team",
		Config:    "config-image",
		Release:   "release-image",
		Terraform: "terraform-image",
	})
	if err != nil {
		log.Fatalln("unexpected error in canonicalisation:", err)
	}
	if !reflect.DeepEqual(result, &manifest.Canonical{
		Version:        2,
		Team:           "test-team",
		Config:         map[string]interface{}{},
		ConfigImage:    "config-image",
		Builds:         map[string]string{"release": "release-image"},
		TerraformImage: "terraform-image",
	}) {
		log.Fatalln("unexpected canonicalisation:", result)
	}
}

func TestCanonicaliseConfigValues(t *testing.T) {
	result, err := manifest.Canonicalise(&manifest.Manifest{
		Version: 2,
		Team:    "test-team",
		Config: map[interface{}]interface{}{
			"image":      "config-image",
			"config-key": "config-value",
		},
		Release:   "release-image",
		Terraform: "terraform-image",
	})
	if err != nil {
		log.Fatalln("unexpected error in canonicalisation:", err)
	}
	if !reflect.DeepEqual(result, &manifest.Canonical{
		Version:        2,
		Team:           "test-team",
		Config:         map[string]interface{}{"config-key": "config-value"},
		ConfigImage:    "config-image",
		Builds:         map[string]string{"release": "release-image"},
		TerraformImage: "terraform-image",
	}) {
		log.Fatalln("unexpected canonicalisation:", result)
	}
}

func TestCanonicaliseMultipleReleases(t *testing.T) {
	result, err := manifest.Canonicalise(&manifest.Manifest{
		Version: 2,
		Team:    "test-team",
		Config:  "config-image",
		Release: map[interface{}]interface{}{
			"build-a": "image-a",
			"build-b": map[interface{}]interface{}{
				"image": "image-b",
			},
		},
		Terraform: "terraform-image",
	})
	if err != nil {
		log.Fatalln("unexpected error in canonicalisation:", err)
	}
	if !reflect.DeepEqual(result, &manifest.Canonical{
		Version:     2,
		Team:        "test-team",
		Config:      map[string]interface{}{},
		ConfigImage: "config-image",
		Builds: map[string]string{
			"build-a": "image-a",
			"build-b": "image-b",
		},
		TerraformImage: "terraform-image",
	}) {
		log.Fatalln("unexpected canonicalisation:", result)
	}
}

func TestCanonicaliseTerraformExplicitImage(t *testing.T) {
	result, err := manifest.Canonicalise(&manifest.Manifest{
		Version: 2,
		Team:    "test-team",
		Config:  "config-image",
		Release: "release-image",
		Terraform: map[interface{}]interface{}{
			"image": "terraform-image",
		},
	})
	if err != nil {
		log.Fatalln("unexpected error in canonicalisation:", err)
	}
	if !reflect.DeepEqual(result, &manifest.Canonical{
		Version:        2,
		Team:           "test-team",
		Config:         map[string]interface{}{},
		ConfigImage:    "config-image",
		Builds:         map[string]string{"release": "release-image"},
		TerraformImage: "terraform-image",
	}) {
		log.Fatalln("unexpected canonicalisation:", result)
	}
}
