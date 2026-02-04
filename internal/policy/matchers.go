package policy

import (
	"path/filepath"
	"strings"
)

// MatchesPrincipalPattern checks if a principal matches a pattern
func MatchesPrincipalPattern(principal, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Try glob matching
	matched, err := filepath.Match(pattern, principal)
	if err == nil && matched {
		return true
	}

	// Try exact match
	return principal == pattern
}

// MatchesResourcePattern checks if a resource ID matches a pattern
func MatchesResourcePattern(resourceID, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Try glob matching
	matched, err := filepath.Match(pattern, resourceID)
	if err == nil && matched {
		return true
	}

	// Try exact match
	return resourceID == pattern
}

// MatchesRolePattern checks if a role matches a pattern
func MatchesRolePattern(role, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Try glob matching
	matched, err := filepath.Match(pattern, role)
	if err == nil && matched {
		return true
	}

	// Try exact match
	return role == pattern
}

// ExtractPrincipalEmail extracts email from principal string
// e.g., "user:alice@example.com" -> "alice@example.com"
func ExtractPrincipalEmail(principal string) string {
	parts := strings.SplitN(principal, ":", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return principal
}

// GetPrincipalType extracts type from principal string
// e.g., "user:alice@example.com" -> "user"
func GetPrincipalType(principal string) string {
	parts := strings.SplitN(principal, ":", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return "unknown"
}

// IsRoleIn checks if a role is in a list of roles
func IsRoleIn(role string, roles []string) bool {
	for _, r := range roles {
		if role == r || MatchesRolePattern(role, r) {
			return true
		}
	}
	return false
}

// IsPrincipalIn checks if a principal matches any pattern in a list
func IsPrincipalIn(principal string, patterns []string) bool {
	for _, pattern := range patterns {
		if MatchesPrincipalPattern(principal, pattern) {
			return true
		}
	}
	return false
}
