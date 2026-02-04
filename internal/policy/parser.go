package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadPolicies loads policy configuration from a YAML file
func LoadPolicies(path string) (*PolicyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var config PolicyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse policy file: %w", err)
	}

	// Validate policies
	if err := validatePolicyConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid policy configuration: %w", err)
	}

	// Normalize policies (populate type-specific fields)
	if err := normalizePolicies(&config); err != nil {
		return nil, fmt.Errorf("failed to normalize policies: %w", err)
	}

	return &config, nil
}

// validatePolicyConfig performs basic validation on the policy configuration
func validatePolicyConfig(config *PolicyConfig) error {
	if config.CloudProvider == "" {
		return fmt.Errorf("cloud_provider is required")
	}

	if len(config.Policies) == 0 {
		return fmt.Errorf("at least one policy is required")
	}

	for i, policy := range config.Policies {
		if policy.Name == "" {
			return fmt.Errorf("policy %d: name is required", i)
		}

		if policy.Type == "" {
			return fmt.Errorf("policy %s: type is required", policy.Name)
		}

		if policy.Severity == "" {
			policy.Severity = SeverityError // Default to error
		}

		// Validate severity
		if policy.Severity != SeverityError && policy.Severity != SeverityWarning && policy.Severity != SeverityInfo {
			return fmt.Errorf("policy %s: invalid severity '%s'", policy.Name, policy.Severity)
		}
	}

	return nil
}

// normalizePolicies ensures policy type-specific fields are properly set
func normalizePolicies(config *PolicyConfig) error {
	for i := range config.Policies {
		policy := &config.Policies[i]

		// Ensure exactly one type-specific field is set
		typeFieldCount := 0
		if policy.RoleRestriction != nil {
			typeFieldCount++
		}
		if policy.Persona != nil {
			typeFieldCount++
		}
		if policy.ResourceAccess != nil {
			typeFieldCount++
		}
		if policy.SeparationOfDuty != nil {
			typeFieldCount++
		}
		if policy.ImpersonationEscalation != nil {
			typeFieldCount++
		}
		if policy.EffectiveAccess != nil {
			typeFieldCount++
		}

		if typeFieldCount == 0 {
			return fmt.Errorf("policy %s: no type-specific configuration found", policy.Name)
		}
		if typeFieldCount > 1 {
			return fmt.Errorf("policy %s: multiple type-specific configurations found", policy.Name)
		}
	}

	return nil
}
