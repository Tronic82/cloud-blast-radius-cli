package policy

import (
	"regexp"
	"strings"
)

// matchPattern checks if a value matches a pattern using regex
// Patterns:
//   - "*" matches everything
//   - Regex patterns like "^user:.*" or "roles/storage\..*"
//   - Exact match as fallback if regex is invalid
func matchPattern(value, pattern string) bool {
	// Special case: "*" matches everything
	if pattern == "*" {
		return true
	}

	// Try regex matching
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Invalid regex, fall back to exact match
		return value == pattern
	}

	return re.MatchString(value)
}

// MatchesPrincipalPattern checks if a principal matches a pattern
// Supports regex patterns like "^user:.*" or "^serviceAccount:.*@.*\.iam\.gserviceaccount\.com$"
func MatchesPrincipalPattern(principal, pattern string) bool {
	return matchPattern(principal, pattern)
}

// MatchesResourcePattern checks if a resource ID matches a pattern
// Supports regex patterns like "^prod-.*" or ".*-bucket$"
func MatchesResourcePattern(resourceID, pattern string) bool {
	return matchPattern(resourceID, pattern)
}

// MatchesRolePattern checks if a role matches a pattern
// Supports regex patterns like "^roles/storage\..*" or "roles/bigquery\.(dataViewer|dataEditor)"
func MatchesRolePattern(role, pattern string) bool {
	return matchPattern(role, pattern)
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
