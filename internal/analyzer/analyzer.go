package analyzer

import (
	"github.com/Tronic82/cloud-blast-radius-cli/internal/definitions"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/parser"
)

// PrincipalData holds all access data for a single principal
type PrincipalData struct {
	ResourceAccess     map[string]*ResourceMetadata // resourceID -> metadata
	HierarchicalAccess map[string]map[string]bool   // projectID -> resourceType -> true
}

// ResourceMetadata holds access details for a specific resource
type ResourceMetadata struct {
	Type           string
	Roles          map[string]bool
	TerraformAddrs map[string]string // role -> terraform address
}

// Analyze processes IAM bindings and groups them by principal
func Analyze(bindings []parser.IAMBinding) map[string]*PrincipalData {
	results := make(map[string]*PrincipalData)

	for _, binding := range bindings {
		for _, member := range binding.Members {
			// Ensure principal exists in results
			if _, exists := results[member]; !exists {
				results[member] = &PrincipalData{
					ResourceAccess:     make(map[string]*ResourceMetadata),
					HierarchicalAccess: make(map[string]map[string]bool),
				}
			}

			// Process Direct Access
			processDirectAccess(results[member], binding)

			// Process Hierarchical Access
			processHierarchicalAccess(results[member], binding)
		}
	}

	return results
}

func processDirectAccess(data *PrincipalData, binding parser.IAMBinding) {
	if _, exists := data.ResourceAccess[binding.ResourceID]; !exists {
		data.ResourceAccess[binding.ResourceID] = &ResourceMetadata{
			Type:           binding.ResourceType,
			Roles:          make(map[string]bool),
			TerraformAddrs: make(map[string]string),
		}
	}
	data.ResourceAccess[binding.ResourceID].Roles[binding.Role] = true
	if binding.TerraformAddr != "" {
		data.ResourceAccess[binding.ResourceID].TerraformAddrs[binding.Role] = binding.TerraformAddr
	}
}

func processHierarchicalAccess(data *PrincipalData, binding parser.IAMBinding) {
	// Check if this binding is on a project
	if binding.ResourceType == "google_project_iam_member" || binding.ResourceType == "google_project_iam_binding" {
		projectID := binding.ResourceID

		// Check if the role grants access to resources of this type
		// We look up the hierarchy for the role
		resourceTypes := definitions.GetResourceTypesForRole(binding.Role)

		for _, rt := range resourceTypes {
			if _, exists := data.HierarchicalAccess[projectID]; !exists {
				data.HierarchicalAccess[projectID] = make(map[string]bool)
			}
			data.HierarchicalAccess[projectID][rt] = true
		}
	}
}
