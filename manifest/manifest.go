package manifest

import (
	"fmt"
	"io/ioutil"
	"path"

	"gopkg.in/yaml.v2"
)

// Manifest represents the data in the cdflow.yaml file before it is canonicalised.
type Manifest struct {
	Version   int8                       `yaml:"version"`
	Config    ImageWithParams            `yaml:"config"`
	Builds    map[string]ImageWithParams `yaml:"builds"`
	Terraform Terraform                  `yaml:"terraform"`
}

// ImageWithParams represents either the config or a build key in cdflow.yaml.
type ImageWithParams struct {
	Image  string                 `yaml:"image"`
	Params map[string]interface{} `yaml:"params"`
}

// Terraform represents the data in the terraform key in cdflow.yaml.
type Terraform struct {
	Image string `yaml:"image"`
}

// Load loads the cdflow.yaml manifest file into a Manifest struct.
func Load(dir string) (*Manifest, error) {
	data, err := ioutil.ReadFile(path.Join(dir, "cdflow.yaml"))
	if err != nil {
		return nil, fmt.Errorf("error loading cdflow.yaml: %w", err)
	}
	var result Manifest
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("error parsing cdflow.yaml: %w", err)
	}
	return &result, nil
}
