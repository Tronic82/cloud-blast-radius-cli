package definitions

import (
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed rules.yaml
var embeddedRules []byte

// RoleHierarchy defines access granted by a role
type RoleHierarchy struct {
	ResourceTypes []string `yaml:"resource_types"`
	AccessLevel   string   `yaml:"access_level"`
}

// ImpersonationRule defines a valid impersonation path
type ImpersonationRule struct {
	SourceType string `yaml:"source"`
	TargetType string `yaml:"target"`
}

// RulesConfig matches the structure of rules.yaml
type RulesConfig struct {
	HierarchicalRoles  map[string]RoleHierarchy `yaml:"hierarchical_roles"`
	ImpersonationRoles []string                 `yaml:"impersonation_roles"`
	ImpersonationRules []ImpersonationRule      `yaml:"impersonation_rules"`
}

// Global caches
var (
	hierarchicalRolesCache  map[string]RoleHierarchy
	impersonationRolesCache []string
	impersonationRulesCache []ImpersonationRule
)

// LoadRules loads definitions from embedded YAML or a custom file
func LoadRules(customPath string) error {
	var data []byte
	var err error

	if customPath != "" {
		data, err = os.ReadFile(customPath)
		if err != nil {
			return fmt.Errorf("failed to read custom rules file: %w", err)
		}
	} else {
		data = embeddedRules
	}

	var config RulesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse rules: %w", err)
	}

	hierarchicalRolesCache = config.HierarchicalRoles
	impersonationRolesCache = config.ImpersonationRoles
	impersonationRulesCache = config.ImpersonationRules

	return nil
}

// GetResourceTypesForRole returns the resource types that a role grants access to
func GetResourceTypesForRole(role string) []string {
	if hierarchicalRolesCache == nil {
		return nil
	}
	if hierarchy, exists := hierarchicalRolesCache[role]; exists {
		return hierarchy.ResourceTypes
	}
	return nil
}

// IsImpersonationRole checks if the role grants impersonation capabilities
func IsImpersonationRole(role string) bool {
	if impersonationRolesCache == nil {
		return false
	}
	for _, r := range impersonationRolesCache {
		if r == role {
			return true
		}
	}
	return false
}

// GetCanImpersonateFunc returns a function to check impersonation validity
func GetCanImpersonateFunc() func(string, string) bool {
	// Create lookup map for fast checking
	allowedMap := make(map[string]map[string]bool)
	for _, rule := range impersonationRulesCache {
		if _, exists := allowedMap[rule.SourceType]; !exists {
			allowedMap[rule.SourceType] = make(map[string]bool)
		}
		allowedMap[rule.SourceType][rule.TargetType] = true
	}

	return func(sourceType, targetType string) bool {
		if targets, exists := allowedMap[sourceType]; exists {
			return targets[targetType]
		}
		return false
	}
}
