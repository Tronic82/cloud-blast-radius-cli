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
	principal := findMatchingPrincipal(accountEmail, directAccess)
	if principal == "" {
		return nil
	}

	result := &TransitiveAccess{
		Principal:        principal,
		DirectAccess:     directAccess[principal],
		TransitiveAccess: make(map[string]*AccessVia),
	}

	queue := []struct {
		principal string
		chain     []string
	}{
		{principal: principal, chain: []string{}},
	}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.principal] {
			continue
		}
		visited[current.principal] = true

		targets, exists := graph.Graph[current.principal]
		if !exists {
			continue
		}

		for _, target := range targets {
			if isCircularReference(current.chain, target) {
				continue
			}

			newChain := append(append([]string{}, current.chain...), target)
			mergeTransitiveAccess(result, target, directAccess, newChain)

			queue = append(queue, struct {
				principal string
				chain     []string
			}{principal: target, chain: newChain})
		}
	}

	return result
}

func findMatchingPrincipal(email string, directAccess map[string]*PrincipalData) string {
	for principal := range directAccess {
		if MatchesPrincipalEmail(principal, email) {
			return principal
		}
	}
	return ""
}

func isCircularReference(chain []string, target string) bool {
	for _, p := range chain {
		if p == target {
			return true
		}
	}
	return false
}

func mergeTransitiveAccess(result *TransitiveAccess, target string, directAccess map[string]*PrincipalData, chain []string) {
	targetAccess, ok := directAccess[target]
	if !ok {
		return
	}

	for resID, resMeta := range targetAccess.ResourceAccess {
		newRoles := make(map[string]bool)
		for role := range resMeta.Roles {
			if !hasDirectRole(result, resID, role) {
				newRoles[role] = true
			}
		}

		if len(newRoles) > 0 {
			result.TransitiveAccess[resID] = &AccessVia{
				Resource: &ResourceMetadata{
					Type:  resMeta.Type,
					Roles: newRoles,
				},
				ViaChain: chain,
			}
		}
	}
}

func hasDirectRole(result *TransitiveAccess, resID, role string) bool {
	if result.DirectAccess != nil && result.DirectAccess.ResourceAccess != nil {
		if directMeta, exists := result.DirectAccess.ResourceAccess[resID]; exists {
			return directMeta.Roles[role]
		}
	}
	return false
}

// MatchesPrincipalEmail checks if a principal matches an email (without principal type prefix)
func MatchesPrincipalEmail(principal, email string) bool {
	parts := strings.SplitN(principal, ":", 2)
	if len(parts) != 2 {
		return false
	}
	return parts[1] == email || strings.HasSuffix(parts[1], email)
}
