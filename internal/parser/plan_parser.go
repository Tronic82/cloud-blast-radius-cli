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
		binding, err := extractBindingFromResource(resource, def)
		if err != nil {
			fmt.Printf("Warning: failed to extract binding from %s: %v\n", resource.Address, err)
			continue
		}

		bindings = append(bindings, binding)
	}

	// Process child modules recursively
	for _, child := range module.ChildModules {
		bindings = append(bindings, extractBindingsFromModule(child, defMap)...)
	}

	return bindings
}

// extractBindingFromResource extracts an IAMBinding from a Terraform resource
func extractBindingFromResource(resource Resource, def ResourceDefinition) (IAMBinding, error) {
	binding := IAMBinding{
		ResourceType:  resource.Type,
		TerraformAddr: resource.Address,
	}

	// Extract ResourceID
	if def.FieldMappings.ResourceID != "" {
		if val, ok := resource.Values[def.FieldMappings.ResourceID]; ok {
			if strVal, ok := val.(string); ok {
				binding.ResourceID = strVal
			}
		}
	}

	// Extract Role
	if def.FieldMappings.Role != "" {
		if val, ok := resource.Values[def.FieldMappings.Role]; ok {
			if strVal, ok := val.(string); ok {
				binding.Role = strVal
			}
		}
	}

	// Extract Member (singular)
	if def.FieldMappings.Member != "" {
		if val, ok := resource.Values[def.FieldMappings.Member]; ok {
			if strVal, ok := val.(string); ok {
				binding.Members = append(binding.Members, strVal)
			}
		}
	}

	// Extract Members (plural)
	if def.FieldMappings.Members != "" {
		if val, ok := resource.Values[def.FieldMappings.Members]; ok {
			if members, ok := val.([]interface{}); ok {
				for _, member := range members {
					if strVal, ok := member.(string); ok {
						binding.Members = append(binding.Members, strVal)
					}
				}
			}
		}
	}

	// Set ResourceLevel from definition (defaults to "resource" if not set)
	binding.ResourceLevel = def.ResourceLevel
	if binding.ResourceLevel == "" {
		binding.ResourceLevel = "resource"
	}

	// Extract parent ID if defined
	if def.FieldMappings.Parent != "" {
		if val, ok := resource.Values[def.FieldMappings.Parent]; ok {
			if strVal, ok := val.(string); ok {
				binding.ParentID = strVal
			}
		}
	}

	// Determine parent type based on resource level
	switch binding.ResourceLevel {
	case "folder":
		binding.ParentType = "organization"
	case "project":
		if binding.ParentID != "" {
			binding.ParentType = "folder"
		}
	case "resource":
		binding.ParentType = "project"
	}

	// Validation
	if binding.ResourceID == "" {
		return binding, fmt.Errorf("missing resource ID")
	}
	if binding.Role == "" {
		return binding, fmt.Errorf("missing role")
	}
	if len(binding.Members) == 0 {
		return binding, fmt.Errorf("no members found")
	}

	return binding, nil
}
