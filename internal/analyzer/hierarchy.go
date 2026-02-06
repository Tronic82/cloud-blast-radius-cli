package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/definitions"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/parser"
)

// HierarchyNode represents a node in the resource hierarchy
type HierarchyNode struct {
	ID         string `json:"id"`
	Type       string `json:"type"`                  // organization, folder, project
	ParentID   string `json:"parent_id,omitempty"`   // parent resource ID
	ParentType string `json:"parent_type,omitempty"` // parent resource type
}

// UnknownHierarchy represents a resource with unknown parent
type UnknownHierarchy struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	ReferencedBy []string `json:"referenced_by"` // terraform addresses that reference it
}

// Scope represents where the role binding is applied
type Scope struct {
	Type string `json:"type"` // organization, folder, project
	ID   string `json:"id"`
}

// Grants represents what access the role binding provides
type Grants struct {
	AffectedLevels []string `json:"affected_levels"` // levels below the scope
	ResourceTypes  []string `json:"resource_types"`  // terraform resource types or ["*"] for all
	DisplayName    string   `json:"display_name"`    // human-readable name
	AccessType     string   `json:"access_type"`     // read, write, admin, impersonate
}

// Source represents where this binding came from in terraform
type Source struct {
	ResourceType    string `json:"resource_type"`
	ResourceAddress string `json:"resource_address"`
}

// HierarchicalAccessEntry represents a single hierarchical access grant
type HierarchicalAccessEntry struct {
	Principal      string `json:"principal"`
	PrincipalType  string `json:"principal_type"`
	Role           string `json:"role"`
	Scope          Scope  `json:"scope"`
	Grants         Grants `json:"grants"`
	HierarchyKnown bool   `json:"hierarchy_known"`
	Source         Source `json:"source"`
}

// Warning represents an issue found during analysis
type Warning struct {
	Type            string `json:"type"`
	Role            string `json:"role,omitempty"`
	ScopeID         string `json:"scope_id,omitempty"`
	ScopeType       string `json:"scope_type,omitempty"`
	ResourceAddress string `json:"resource_address,omitempty"`
	Message         string `json:"message"`
}

// HierarchyAnalysisResult is the complete hierarchy analysis
type HierarchyAnalysisResult struct {
	Nodes              []HierarchyNode           `json:"nodes"`
	Unknown            []UnknownHierarchy        `json:"unknown"`
	HierarchicalAccess []HierarchicalAccessEntry `json:"hierarchical_access"`
	Warnings           []Warning                 `json:"warnings"`
}

// hierarchyOrder defines the static GCP hierarchy ordering
var hierarchyOrder = map[string]int{
	"organization": 0,
	"folder":       1,
	"project":      2,
	"resource":     3,
}

// AnalyzeHierarchy performs comprehensive hierarchy analysis
func AnalyzeHierarchy(bindings []parser.IAMBinding) *HierarchyAnalysisResult {
	result := &HierarchyAnalysisResult{
		Nodes:              []HierarchyNode{},
		Unknown:            []UnknownHierarchy{},
		HierarchicalAccess: []HierarchicalAccessEntry{},
		Warnings:           []Warning{},
	}

	// Track known hierarchy nodes to avoid duplicates
	knownNodes := make(map[string]bool) // "type:id" -> true

	// Track unknown references for grouping
	unknownRefs := make(map[string][]string) // "type:id" -> []terraform addresses

	// 1. Build hierarchy nodes from bindings
	for _, binding := range bindings {
		if isHierarchyLevel(binding.ResourceLevel) {
			nodeKey := binding.ResourceLevel + ":" + binding.ResourceID
			if !knownNodes[nodeKey] {
				knownNodes[nodeKey] = true
				node := HierarchyNode{
					ID:         binding.ResourceID,
					Type:       binding.ResourceLevel,
					ParentID:   binding.ParentID,
					ParentType: binding.ParentType,
				}
				result.Nodes = append(result.Nodes, node)

				// Track if hierarchy is unknown
				if binding.ParentID == "" && binding.ResourceLevel != "organization" {
					unknownKey := binding.ResourceLevel + ":" + binding.ResourceID
					unknownRefs[unknownKey] = append(unknownRefs[unknownKey], binding.TerraformAddr)
				}
			}
		}
	}

	// 2. Build unknown hierarchy list
	for key, refs := range unknownRefs {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			result.Unknown = append(result.Unknown, UnknownHierarchy{
				ID:           parts[1],
				Type:         parts[0],
				ReferencedBy: refs,
			})
		}
	}

	// Track warnings to avoid duplicates
	warnedScopes := make(map[string]bool) // "type:id" -> true for unknown hierarchy warnings
	warnedRoles := make(map[string]bool)  // role -> true for unknown role warnings

	// 3. Analyze hierarchical access for each binding
	for _, binding := range bindings {
		if !isHierarchyLevel(binding.ResourceLevel) {
			continue // Only process org/folder/project level bindings
		}

		roleHierarchy := definitions.GetRoleHierarchy(binding.Role)
		if roleHierarchy == nil {
			// Role not in definitions - add warning (only once per role)
			if !warnedRoles[binding.Role] {
				warnedRoles[binding.Role] = true
				result.Warnings = append(result.Warnings, Warning{
					Type:            "unknown_role",
					Role:            binding.Role,
					ResourceAddress: binding.TerraformAddr,
					Message:         fmt.Sprintf("Role '%s' not found in definitions, hierarchical access cannot be determined", binding.Role),
				})
			}
			continue
		}

		// Check if this creates hierarchical access
		// (binding level is higher/above the target level)
		if !isHigherLevel(binding.ResourceLevel, roleHierarchy.TargetLevel) {
			continue
		}

		// Determine if hierarchy is known
		hierarchyKnown := binding.ParentID != "" || binding.ResourceLevel == "organization"

		// Add warning for unknown hierarchy (only once per scope)
		scopeKey := binding.ResourceLevel + ":" + binding.ResourceID
		if !hierarchyKnown && !warnedScopes[scopeKey] {
			warnedScopes[scopeKey] = true
			result.Warnings = append(result.Warnings, Warning{
				Type:      "unknown_hierarchy",
				ScopeID:   binding.ResourceID,
				ScopeType: binding.ResourceLevel,
				Message:   fmt.Sprintf("Hierarchy for %s '%s' is unknown, folder and org level bindings may also apply", binding.ResourceLevel, binding.ResourceID),
			})
		}

		// Create entry for each member
		for _, member := range binding.Members {
			entry := HierarchicalAccessEntry{
				Principal:     member,
				PrincipalType: GetPrincipalType(member),
				Role:          binding.Role,
				Scope: Scope{
					Type: binding.ResourceLevel,
					ID:   binding.ResourceID,
				},
				Grants: Grants{
					AffectedLevels: getAffectedLevels(binding.ResourceLevel),
					ResourceTypes:  roleHierarchy.ResourceTypes,
					DisplayName:    roleHierarchy.DisplayName,
					AccessType:     roleHierarchy.AccessLevel,
				},
				HierarchyKnown: hierarchyKnown,
				Source: Source{
					ResourceType:    binding.ResourceType,
					ResourceAddress: binding.TerraformAddr,
				},
			}
			result.HierarchicalAccess = append(result.HierarchicalAccess, entry)
		}
	}

	// Sort results for deterministic output
	sort.Slice(result.Nodes, func(i, j int) bool {
		if result.Nodes[i].Type != result.Nodes[j].Type {
			return result.Nodes[i].Type < result.Nodes[j].Type
		}
		return result.Nodes[i].ID < result.Nodes[j].ID
	})

	sort.Slice(result.Unknown, func(i, j int) bool {
		if result.Unknown[i].Type != result.Unknown[j].Type {
			return result.Unknown[i].Type < result.Unknown[j].Type
		}
		return result.Unknown[i].ID < result.Unknown[j].ID
	})

	return result
}

// isHierarchyLevel checks if the level is a hierarchy level (not a resource)
func isHierarchyLevel(level string) bool {
	return level == "organization" || level == "folder" || level == "project"
}

// isHigherLevel returns true if bindingLevel is higher (closer to root) than targetLevel
func isHigherLevel(bindingLevel, targetLevel string) bool {
	bindingOrder, bindingOk := hierarchyOrder[bindingLevel]
	targetOrder, targetOk := hierarchyOrder[targetLevel]
	if !bindingOk || !targetOk {
		return false
	}
	return bindingOrder < targetOrder
}

// getAffectedLevels returns the levels that are affected by a binding at the given scope
func getAffectedLevels(scopeLevel string) []string {
	switch scopeLevel {
	case "organization":
		return []string{"folder", "project", "resource"}
	case "folder":
		return []string{"project", "resource"}
	case "project":
		return []string{"resource"}
	default:
		return []string{}
	}
}

// Note: GetPrincipalType is defined in impersonation.go and reused here
