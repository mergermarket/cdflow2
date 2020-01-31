package config

import (
	"io/ioutil"
	"log"
	"path"

	"gopkg.in/yaml.v2"
)

// Manifest represents the data in the cdflow.yaml file.
type Manifest struct {
	Version        int8   `yaml:"version"`
	ConfigImage    string `yaml:"config_image"`
	ReleaseImage   string `yaml:"release_image"`
	TerraformImage string `yaml:"terraform_image"`
	Team           string `yaml:"team"`
}

func parseManifest(content []byte) (*Manifest, error) {
	var result Manifest
	if err := yaml.Unmarshal(content, &result); err != nil {
		log.Fatalf("invalid terraflow.yaml: %v", err)
	}
	return &result, nil
}

// LoadManifest loads the cdflow.yaml manifest file into a Manifest struct.
func LoadManifest(dir string) (*Manifest, error) {
	data, err := ioutil.ReadFile(path.Join(dir, "cdflow.yaml"))
	if err != nil {
		return nil, err
	}
	return parseManifest(data)
}
