package definitions

import (
	"blast-radius/internal/parser"
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed resources.yaml
var embeddedResources []byte

// ResourceConfig matches the structure of resources.yaml
type ResourceConfig struct {
	Resources []parser.ResourceDefinition `yaml:"definitions"`
}

// LoadResourceDefinitions loads definitions from embedded YAML or a custom file
func LoadResourceDefinitions(customPath string) ([]parser.ResourceDefinition, error) {
	var data []byte
	var err error

	if customPath != "" {
		data, err = os.ReadFile(customPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read custom definitions file: %w", err)
		}
	} else {
		data = embeddedResources
	}

	var config ResourceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse resource definitions: %w", err)
	}

	return config.Resources, nil
}
