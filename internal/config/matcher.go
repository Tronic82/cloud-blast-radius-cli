package config

import (
	"regexp"
)

// IsExcluded checks if a given binding (ResourceID, Type, Role) matches any exclusion rule
func (c *Config) IsExcluded(resourceID, resourceType, role string) bool {
	for _, rule := range c.Exclusions {
		if matchRule(rule, resourceID, resourceType, role) {
			return true
		}
	}
	return false
}

func matchRule(rule ExclusionRule, resourceID, resourceType, role string) bool {
	if rule.Resource != "" {
		if matched, _ := regexp.MatchString(rule.Resource, resourceID); !matched {
			return false
		}
	}
	if rule.ResourceType != "" {
		if matched, _ := regexp.MatchString(rule.ResourceType, resourceType); !matched {
			return false
		}
	}
	if rule.Role != "" {
		if matched, _ := regexp.MatchString(rule.Role, role); !matched {
			return false
		}
	}
	// All specified fields matched
	return true
}
