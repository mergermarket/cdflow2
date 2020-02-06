package manifest

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"gopkg.in/yaml.v2"
)

// Manifest represents the data in the cdflow.yaml file before it is canonicalised.
type Manifest struct {
	Version   int8        `yaml:"version"`
	Team      string      `yaml:"team"`
	Config    interface{} `yaml:"config"`
	Release   interface{} `yaml:"release"`
	Terraform interface{} `yaml:"terraform"`
}

// Canonical represents the data in the cdflow.yaml file after it is canonicalised.
type Canonical struct {
	Version        int8
	Team           string
	Config         map[string]interface{}
	ConfigImage    string
	Builds         map[string]string
	TerraformImage string
}

func parse(content []byte) (*Manifest, error) {
	var result Manifest
	if err := yaml.Unmarshal(content, &result); err != nil {
		log.Fatalf("invalid terraflow.yaml: %v", err)
	}
	return &result, nil
}

func canonicaliseConfig(manifest *Manifest, canonical *Canonical) error {
	if image, ok := manifest.Config.(string); ok {
		canonical.ConfigImage = image
	} else if config, ok := manifest.Config.(map[interface{}]interface{}); ok {
		if image, ok := config["image"].(string); ok {
			canonical.ConfigImage = image
		} else {
			return fmt.Errorf("cdflow.yaml error - invalid type for config.image: %T", config["image"])
		}
		for key, value := range config {
			if key, ok := key.(string); ok && key != "image" {
				canonical.Config[key] = value
			}
		}
	} else {
		return fmt.Errorf("cdflow.yaml error - invalid type for config: %T", manifest.Config)
	}
	return nil
}

func canonicaliseRelease(manifest *Manifest, canonical *Canonical) error {
	if image, ok := manifest.Release.(string); ok {
		canonical.Builds["release"] = image
	} else if builds, ok := manifest.Release.(map[interface{}]interface{}); ok {
		for key, value := range builds {
			if key, ok := key.(string); ok {
				if image, ok := value.(string); ok {
					canonical.Builds[key] = image
				} else if buildMap, ok := value.(map[interface{}]interface{}); ok {
					if image, ok := buildMap["image"].(string); ok {
						canonical.Builds[key] = image
					} else {
						return fmt.Errorf("cdflow.yaml error - invalid type for release.%v.image: %T", key, manifest.Release)
					}
				} else {
					return fmt.Errorf("cdflow.yaml error - invalid type for release.%v: %T", key, value)
				}
			} else {
				return fmt.Errorf("cdflow.yaml error - unexpected non-string key in release: %T", key)
			}
		}
	} else {
		return fmt.Errorf("cdflow.yaml error - invalid type for release: %T", manifest.Release)
	}
	return nil
}

func canonicaliseTerraform(manifest *Manifest, canonical *Canonical) error {
	if image, ok := manifest.Terraform.(string); ok {
		canonical.TerraformImage = image
	} else if terraform, ok := manifest.Terraform.(map[interface{}]interface{}); ok {
		if image, ok := terraform["image"].(string); ok {
			canonical.TerraformImage = image
		} else {
			return fmt.Errorf("cdflow.yaml error - invalid type for terraform.image: %T", terraform["image"])
		}
	} else {
		return fmt.Errorf("cdflow.yaml error - invalid type for terraform: %T", manifest.Terraform)
	}
	return nil
}

// Canonicalise transforms the raw Manifest that's loaded in with loose typing into a strongly typed manifest.Canonical.
func Canonicalise(manifest *Manifest) (*Canonical, error) {
	var canonical Canonical
	canonical.Config = make(map[string]interface{})
	canonical.Builds = make(map[string]string)
	canonical.Version = manifest.Version
	canonical.Team = manifest.Team
	if err := canonicaliseConfig(manifest, &canonical); err != nil {
		return nil, err
	}
	if err := canonicaliseRelease(manifest, &canonical); err != nil {
		return nil, err
	}
	if err := canonicaliseTerraform(manifest, &canonical); err != nil {
		return nil, err
	}
	return &canonical, nil
}

// Load loads the cdflow.yaml manifest file into a Manifest struct.
func Load(dir string) (*Canonical, error) {
	data, err := ioutil.ReadFile(path.Join(dir, "cdflow.yaml"))
	if err != nil {
		return nil, err
	}
	manifest, err := parse(data)
	if err != nil {
		return nil, err
	}
	return Canonicalise(manifest)
}
