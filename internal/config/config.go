package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	CloudProvider      string          `yaml:"cloud_provider"`
	Exclusions         []ExclusionRule `yaml:"exclusions"`
	IgnoredDirectories []string        `yaml:"ignored_directories"`
	AnalysisAccounts   []string        `yaml:"analysis_accounts"` // Accounts to analyze (emails without principal type)
}

type ExclusionRule struct {
	Resource     string `yaml:"resource"`      // Regex pattern for Resource ID
	ResourceType string `yaml:"resource_type"` // Regex pattern for Resource Type
	Role         string `yaml:"role"`          // Regex pattern for Role
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func CreateDefault(path string, provider string) error {
	defaultConfig := Config{
		CloudProvider: provider,
		Exclusions: []ExclusionRule{
			{Resource: "example-ignored-project", Role: ".*"},
			{Role: "roles/viewer"},
		},
		IgnoredDirectories: []string{".terraform", "modules"},
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetAnalysisAccounts returns the list of accounts to analyze from config
func (c *Config) GetAnalysisAccounts() []string {
	return c.AnalysisAccounts
}
