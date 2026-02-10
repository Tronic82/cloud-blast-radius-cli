package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

// TerraformPlan represents the structure of terraform show -json output
type TerraformPlan struct {
	FormatVersion    string        `json:"format_version"`
	TerraformVersion string        `json:"terraform_version"`
	PlannedValues    PlannedValues `json:"planned_values"`
}

type PlannedValues struct {
	RootModule Module `json:"root_module"`
}

type Module struct {
	Resources    []Resource `json:"resources"`
	ChildModules []Module   `json:"child_modules"`
}

type Resource struct {
	Address      string                 `json:"address"`
	Mode         string                 `json:"mode"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	ProviderName string                 `json:"provider_name"`
	Values       map[string]interface{} `json:"values"`
}

// ParsePlanFile parses a Terraform plan JSON file and extracts IAM bindings
func ParsePlanFile(planPath string, definitions []ResourceDefinition) ([]IAMBinding, error) {
	// Read the plan file
	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	// Parse JSON
	var plan TerraformPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	// Create definition lookup map
	defMap := make(map[string]ResourceDefinition)
	for _, def := range definitions {
		defMap[def.Type] = def
	}

	// Extract bindings
	var bindings []IAMBinding
	bindings = append(bindings, extractBindingsFromModule(plan.PlannedValues.RootModule, defMap)...)

	return bindings, nil
}

// extractBindingsFromModule recursively extracts IAM bindings from a module and its children
func extractBindingsFromModule(module Module, defMap map[string]ResourceDefinition) []IAMBinding {
	var bindings []IAMBinding

	// Process resources in this module
	for _, resource := range module.Resources {
		// Only process managed resources
		if resource.Mode != "managed" {
			continue
		}

		// Check if this is an IAM resource we care about
		def, exists := defMap[resource.Type]
		if !exists {
			continue
		}

		// Extract binding from this resource
		bindingsFromResource, err := extractBindingFromResource(resource, def)
		if err != nil {
			fmt.Printf("Warning: failed to extract binding from %s: %v\n", resource.Address, err)
			continue
		}

		bindings = append(bindings, bindingsFromResource...)
	}

	// Process child modules recursively
	for _, child := range module.ChildModules {
		bindings = append(bindings, extractBindingsFromModule(child, defMap)...)
	}

	return bindings
}

// extractBindingFromResource extracts an IAMBinding from a Terraform resource
func extractBindingFromResource(resource Resource, def ResourceDefinition) ([]IAMBinding, error) {
	// Common fields extraction
	resourceID := ""
	// Extract ResourceID
	if def.FieldMappings.ResourceID != "" {
		if val, ok := resource.Values[def.FieldMappings.ResourceID]; ok {
			if strVal, ok := val.(string); ok {
				resourceID = strVal
			}
		}
	}

	// Extract Parent ID
	parentID := ""
	if def.FieldMappings.Parent != "" {
		parentID = GetStringFromMap(resource.Values, def.FieldMappings.Parent)
	}

	resourceLevel := def.ResourceLevel
	if resourceLevel == "" {
		resourceLevel = "resource"
	}

	// Determine parent type based on resource level
	parentType := DetermineParentType(resourceLevel, parentID)

	// --- Check for Policy Data ---
	if def.FieldMappings.PolicyData != "" {
		if val, ok := resource.Values[def.FieldMappings.PolicyData]; ok {
			if policyDataJSON, ok := val.(string); ok {
				// Unmarshal Policy Data
				var policy Policy
				if err := json.Unmarshal([]byte(policyDataJSON), &policy); err != nil {
					return nil, fmt.Errorf("failed to parse policy_data JSON: %w", err)
				}

				var bindings []IAMBinding
				for _, pb := range policy.Bindings {
					b := IAMBinding{
						ResourceID:    resourceID,
						ResourceType:  resource.Type,
						ResourceLevel: resourceLevel,
						Role:          pb.Role,
						Members:       pb.Members,
						ParentID:      parentID,
						ParentType:    parentType,
						TerraformAddr: resource.Address,
					}
					bindings = append(bindings, b)
				}
				return bindings, nil
			}
		}
	}

	// --- Standard IAM Binding/Member ---

	binding := IAMBinding{
		ResourceType:  resource.Type,
		TerraformAddr: resource.Address,
		ResourceID:    resourceID,
		ResourceLevel: resourceLevel,
		ParentID:      parentID,
		ParentType:    parentType,
	}

	// Extract Role
	if def.FieldMappings.Role != "" {
		binding.Role = GetStringFromMap(resource.Values, def.FieldMappings.Role)
	}

	// Extract Member (singular)
	if def.FieldMappings.Member != "" {
		if val := GetStringFromMap(resource.Values, def.FieldMappings.Member); val != "" {
			binding.Members = append(binding.Members, val)
		}
	}

	// Extract Members (plural)
	if def.FieldMappings.Members != "" {
		binding.Members = append(binding.Members, GetListFromMap(resource.Values, def.FieldMappings.Members)...)
	}

	// Validation
	if binding.ResourceID == "" {
		return nil, fmt.Errorf("missing resource ID: %s", resource.Address)
	}
	if binding.Role == "" {
		return nil, fmt.Errorf("missing role: %s", resource.Address)
	}
	if len(binding.Members) == 0 {
		return nil, fmt.Errorf("no members found: %s", resource.Address)
	}

	return []IAMBinding{binding}, nil
}
