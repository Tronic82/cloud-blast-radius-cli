package parser

import "strings"

// DetermineParentType infers the parent resource type based on the resource level
// and presence of a parent ID.
func DetermineParentType(resourceLevel, parentID string) string {
	// Check for explicit prefixes
	if strings.HasPrefix(parentID, "folders/") {
		return "folder"
	}
	if strings.HasPrefix(parentID, "organizations/") {
		return "organization"
	}

	switch resourceLevel {
	case "folder":
		return "organization"
	case "project":
		if parentID != "" {
			return "folder"
		}
	case "resource":
		return "project"
	}
	return ""
}

// PolicyBinding matches the structure of a binding in policy_data JSON
type PolicyBinding struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

// Policy matches the structure of policy_data JSON
type Policy struct {
	Bindings []PolicyBinding `json:"bindings"`
}

// GetStringFromMap safely extracts a string value from a map[string]interface{}.
// Returns empty string if key not found or value is not a string.
func GetStringFromMap(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetListFromMap safely extracts a list of strings from a map[string]interface{}.
// Handles []interface{} where elements are strings.
func GetListFromMap(data map[string]interface{}, key string) []string {
	var result []string
	if val, ok := data[key]; ok {
		if list, ok := val.([]interface{}); ok {
			for _, item := range list {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
		}
	}
	return result
}
