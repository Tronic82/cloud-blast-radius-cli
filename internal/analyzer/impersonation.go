package analyzer

import (
	"strings"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/definitions"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/parser"
)

// ImpersonationGraph represents the impersonation relationships between principals
type ImpersonationGraph struct {
	Graph map[string][]string // principal → service accounts they can impersonate
}

// TransitiveAccess represents the complete access analysis for a principal including impersonation
type TransitiveAccess struct {
	Principal        string
	DirectAccess     *PrincipalData
	TransitiveAccess map[string]*AccessVia // resourceID → how it was accessed
}

// AccessVia represents access obtained through impersonation
type AccessVia struct {
	Resource *ResourceMetadata
	ViaChain []string // Chain of impersonation (e.g., ["sa-b", "sa-c"])
}

// GetPrincipalType extracts the principal type from a full principal string
func GetPrincipalType(principal string) string {
	parts := strings.SplitN(principal, ":", 2)
	if len(parts) != 2 {
		return "unknown"
	}
	return parts[0]
}

// CanImpersonate checks if a source principal type can impersonate a target principal type per GCP rules
func CanImpersonate(sourcePrincipalType, targetPrincipalType string) bool {
	// PrincipalSets cannot be impersonated by anyone
	if targetPrincipalType == "principalSet" {
		return false
	}

	// Users and groups cannot be impersonated
	if targetPrincipalType == "user" || targetPrincipalType == "group" {
		return false
	}

	// Service accounts can be impersonated by:
	// - other service accounts
	// - users
	// - groups
	// - principalSets
	if targetPrincipalType == "serviceAccount" {
		return sourcePrincipalType == "serviceAccount" ||
			sourcePrincipalType == "user" ||
			sourcePrincipalType == "group" ||
			sourcePrincipalType == "principalSet"
	}

	return false
}

// BuildImpersonationGraph scans bindings for impersonation relationships
func BuildImpersonationGraph(bindings []parser.IAMBinding) *ImpersonationGraph {
	return BuildImpersonationGraphWithFunc(bindings, CanImpersonate)
}

// BuildImpersonationGraphWithFunc scans bindings for impersonation relationships using a custom canImpersonate function
func BuildImpersonationGraphWithFunc(bindings []parser.IAMBinding, canImpersonate func(string, string) bool) *ImpersonationGraph {
	graph := &ImpersonationGraph{
		Graph: make(map[string][]string),
	}

	for _, b := range bindings {
		// Check if this is an impersonation role
		if !definitions.IsImpersonationRole(b.Role) {
			continue
		}

		// The resourceID for service account IAM is the full path: projects/.../serviceAccounts/email
		// Extract just the service account email
		targetSA := extractServiceAccountEmail(b.ResourceID)
		if targetSA == "" {
			continue // Not a valid service account resource
		}

		// Build full principal for target (serviceAccount:email)
		targetPrincipal := "serviceAccount:" + targetSA
		targetType := GetPrincipalType(targetPrincipal)

		// For each member, check if they can impersonate this target
		for _, member := range b.Members {
			sourceType := GetPrincipalType(member)

			// Validate impersonation is allowed per GCP rules
			if !canImpersonate(sourceType, targetType) {
				continue
			}

			// Skip self-impersonation
			if member == targetPrincipal {
				continue
			}

			// Add to graph
			if _, exists := graph.Graph[member]; !exists {
				graph.Graph[member] = []string{}
			}
			graph.Graph[member] = append(graph.Graph[member], targetPrincipal)
		}
	}

	return graph
}

// extractServiceAccountEmail extracts the email from a service_account_id resource path
// Example: "projects/my-project/serviceAccounts/sa-b@my-project.iam.gserviceaccount.com" → "sa-b@my-project.iam.gserviceaccount.com"
func extractServiceAccountEmail(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	if len(parts) >= 4 && parts[0] == "projects" && parts[2] == "serviceAccounts" {
		return parts[3]
	}
	// If it's already just an email, return it
	if strings.Contains(resourceID, "@") {
		return resourceID
	}
	return ""
}

// AnalyzeTransitiveAccess calculates transitive access for a specific account via impersonation
func AnalyzeTransitiveAccess(accountEmail string, directAccess map[string]*PrincipalData, graph *ImpersonationGraph) *TransitiveAccess {
	// Find matching principals for this account email
	var matchingPrincipals []string
	for principal := range directAccess {
		if MatchesPrincipalEmail(principal, accountEmail) {
			matchingPrincipals = append(matchingPrincipals, principal)
		}
	}

	if len(matchingPrincipals) == 0 {
		return nil
	}

	// Use the first matching principal (typically there should only be one)
	principal := matchingPrincipals[0]

	result := &TransitiveAccess{
		Principal:        principal,
		DirectAccess:     directAccess[principal],
		TransitiveAccess: make(map[string]*AccessVia),
	}

	// Traverse impersonation graph using BFS
	visited := make(map[string]bool)
	queue := []struct {
		principal string
		chain     []string
	}{
		{principal: principal, chain: []string{}},
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Skip if already visited (prevents infinite loops)
		if visited[current.principal] {
			continue
		}
		visited[current.principal] = true

		// Get impersonation targets for this principal
		targets, exists := graph.Graph[current.principal]
		if !exists {
			continue
		}

		for _, target := range targets {
			// Check for circular impersonation
			isCircular := false
			for _, p := range current.chain {
				if p == target {
					isCircular = true
					break
				}
			}
			if isCircular {
				continue
			}

			// Build new chain
			newChain := append([]string{}, current.chain...)
			newChain = append(newChain, target)

			// Add target's access to transitive access
			if targetAccess, ok := directAccess[target]; ok {
				for resID, resMeta := range targetAccess.ResourceAccess {
					// Check which roles are new (not in direct access)
					newRoles := make(map[string]bool)
					for role := range resMeta.Roles {
						hasRole := false
						if result.DirectAccess != nil && result.DirectAccess.ResourceAccess != nil {
							if directMeta, exists := result.DirectAccess.ResourceAccess[resID]; exists {
								hasRole = directMeta.Roles[role]
							}
						}
						if !hasRole {
							newRoles[role] = true
						}
					}

					// Only add if there are new roles
					if len(newRoles) > 0 {
						// Add to transitive access with only the new roles
						result.TransitiveAccess[resID] = &AccessVia{
							Resource: &ResourceMetadata{
								Type:  resMeta.Type,
								Roles: newRoles,
							},
							ViaChain: newChain,
						}
					}
				}
			}

			// Add to queue for further traversal
			queue = append(queue, struct {
				principal string
				chain     []string
			}{principal: target, chain: newChain})
		}
	}

	return result
}

// MatchesPrincipalEmail checks if a principal matches an email (without principal type prefix)
func MatchesPrincipalEmail(principal, email string) bool {
	parts := strings.SplitN(principal, ":", 2)
	if len(parts) != 2 {
		return false
	}
	return parts[1] == email || strings.HasSuffix(parts[1], email)
}
