package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// IAMBinding represents a single IAM binding found in Terraform
type IAMBinding struct {
	ResourceID    string   // The identifier of the resource (e.g. project ID, bucket name)
	ResourceType  string   // The type of the resource (e.g. "google_project", "google_storage_bucket") - inferred from TF resource type
	ResourceLevel string   // The hierarchy level: "organization", "folder", "project", "resource"
	Role          string   // The IAM role (e.g. "roles/storage.admin")
	Members       []string // The principals granted this role
	ParentID      string   // Parent resource ID (e.g. folder ID, org ID)
	ParentType    string   // Parent resource type: "organization", "folder", "project"
	TerraformAddr string   // Full terraform address (e.g. "google_project_iam_member.alice")
}

// ParseDir scans a directory recursively for Terraform files and parses them for IAM bindings based on definitions
// It now also loads variables and resolves references.
func ParseDir(dir string, tfvarsPath string, definitions []ResourceDefinition, ignoredDirs []string) ([]IAMBinding, error) {
	// 1. Load Variables (Root level only for now)
	vars, err := LoadVariables(dir, tfvarsPath)
	if err != nil {
		fmt.Printf("Warning: failed to load variables: %v\n", err)
	}

	// 2. Load all files recursively
	parser := hclparse.NewParser()
	var parsedFiles []*hcl.File
	// var fileNames []string // Unused for now

	// Prepare ignored map for faster lookup
	ignoredMap := make(map[string]bool)
	for _, d := range ignoredDirs {
		ignoredMap[d] = true
	}
	// Always ignore .git and .terraform
	ignoredMap[".git"] = true
	ignoredMap[".terraform"] = true

	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if ignoredMap[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(d.Name(), ".tf") {
			file, diags := parser.ParseHCLFile(path)
			if diags.HasErrors() {
				fmt.Printf("Warning: failed to parse file %s: %s\n", path, diags.Error())
				return nil // Continue walking
			}
			parsedFiles = append(parsedFiles, file)
			// fileNames = append(fileNames, d.Name())
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// 3. Initialize Traverser
	traverser := NewConfigTraverser(parsedFiles, vars)

	// 4. Extract Default Project from Provider
	defaultProject := ""
	for _, file := range parsedFiles {
		content, _, _ := file.Body.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{Type: "provider", LabelNames: []string{"name"}},
			},
		})
		for _, block := range content.Blocks {
			if block.Type == "provider" && len(block.Labels) == 1 && block.Labels[0] == "google" {
				// Check for project attribute
				blockContent, _, _ := block.Body.PartialContent(&hcl.BodySchema{
					Attributes: []hcl.AttributeSchema{
						{Name: "project", Required: false},
					},
				})
				if attr, exists := blockContent.Attributes["project"]; exists {
					val, err := traverser.ResolveExpression(attr.Expr)
					if err == nil && val.Type() == cty.String {
						defaultProject = val.AsString()
						break
					}
				}
			}
		}
		if defaultProject != "" {
			break
		}
	}

	// 5. Extract Bindings
	var bindings []IAMBinding

	for i, file := range parsedFiles {
		// Use file.Body.Content to find IAM resources
		content, _, _ := file.Body.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{Type: "resource", LabelNames: []string{"type", "name"}},
			},
		})

		for _, block := range content.Blocks {
			if block.Type == "resource" {
				resourceType := block.Labels[0]
				// Check against definitions
				for _, def := range definitions {
					if resourceType == def.Type {
						// Check for for_each or count meta-arguments
						attrs, _ := block.Body.JustAttributes()

						if forEachAttr, hasForEach := attrs["for_each"]; hasForEach {
							// Expand for_each
							expanded, err := expandForEach(block, def, traverser, defaultProject, forEachAttr)
							if err != nil {
								fmt.Printf("Warning: failed to expand for_each in %s: %v\n", resourceType, err)
								continue
							}
							bindings = append(bindings, expanded...)
						} else if countAttr, hasCount := attrs["count"]; hasCount {
							// Expand count
							expanded, err := expandCount(block, def, traverser, defaultProject, countAttr)
							if err != nil {
								fmt.Printf("Warning: failed to expand count in %s: %v\n", resourceType, err)
								continue
							}
							bindings = append(bindings, expanded...)
						} else {
							// No loop - single resource
							bindingsFromResource, err := extractIAMResource(block, def, traverser, defaultProject)
							if err != nil {
								// fmt.Printf("Error extracting %s in %s: %v\n", resourceType, fileNames[i], err)
								continue
							}
							bindings = append(bindings, bindingsFromResource...)
						}
						break // Matched definition
					}
				}
			}
		}
		_ = i
	}

	return bindings, nil
}

// expandForEach expands a resource with for_each into multiple IAMBindings
func expandForEach(block *hcl.Block, def ResourceDefinition, traverser *ConfigTraverser, defaultProject string, forEachAttr *hcl.Attribute) ([]IAMBinding, error) {
	// Resolve the for_each expression to get the map/set
	forEachVal, err := traverser.ResolveExpression(forEachAttr.Expr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve for_each: %w", err)
	}

	// for_each must be a map or set (sets can be represented as tuples)
	if !forEachVal.Type().IsMapType() && !forEachVal.Type().IsSetType() && !forEachVal.Type().IsObjectType() && !forEachVal.Type().IsTupleType() {
		return nil, fmt.Errorf("for_each must be a map or set, got %s", forEachVal.Type().FriendlyName())
	}

	var bindings []IAMBinding
	it := forEachVal.ElementIterator()

	for it.Next() {
		key, val := it.Element()

		// Create scoped context with each.key and each.value
		scopedCtx := &hcl.EvalContext{
			Variables: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"key":   key,
					"value": val,
				}),
			},
		}

		// Create scoped traverser
		scopedTraverser := traverser.WithScope(scopedCtx)

		// Extract binding with scoped context
		bindingsFromResource, err := extractIAMResource(block, def, scopedTraverser, defaultProject)
		if err != nil {
			// Skip this iteration if extraction fails
			continue
		}

		bindings = append(bindings, bindingsFromResource...)
	}

	return bindings, nil
}

// expandCount expands a resource with count into multiple IAMBindings
func expandCount(block *hcl.Block, def ResourceDefinition, traverser *ConfigTraverser, defaultProject string, countAttr *hcl.Attribute) ([]IAMBinding, error) {
	// Resolve the count expression to get the number
	countVal, err := traverser.ResolveExpression(countAttr.Expr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve count: %w", err)
	}

	// count must be a number
	if countVal.Type() != cty.Number {
		return nil, fmt.Errorf("count must be a number, got %s", countVal.Type().FriendlyName())
	}

	// Convert to int using BigFloat
	bf := countVal.AsBigFloat()
	countInt64, _ := bf.Int64()
	count := int(countInt64)

	var bindings []IAMBinding

	for i := 0; i < count; i++ {
		// Create scoped context with count.index
		scopedCtx := &hcl.EvalContext{
			Variables: map[string]cty.Value{
				"count": cty.ObjectVal(map[string]cty.Value{
					"index": cty.NumberIntVal(int64(i)),
				}),
			},
		}

		// Create scoped traverser
		scopedTraverser := traverser.WithScope(scopedCtx)

		// Extract binding with scoped context
		bindingsFromResource, err := extractIAMResource(block, def, scopedTraverser, defaultProject)
		if err != nil {
			// Skip this iteration if extraction fails
			continue
		}

		bindings = append(bindings, bindingsFromResource...)
	}

	return bindings, nil
}

func extractIAMResource(block *hcl.Block, def ResourceDefinition, traverser *ConfigTraverser, defaultProject string) ([]IAMBinding, error) {
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}

	// Common fields extraction
	resourceID := defaultProject
	parentID := ""
	parentType := ""
	terraformAddr := ""
	if len(block.Labels) >= 2 {
		terraformAddr = fmt.Sprintf("%s.%s", block.Labels[0], block.Labels[1])
	}

	// Helper to get string value resolved
	getString := func(attrName string) string {
		if attr, ok := attrs[attrName]; ok {
			val, err := traverser.ResolveExpression(attr.Expr)
			if err == nil && val.Type() == cty.String {
				return val.AsString()
			}
		}
		return ""
	}

	// Extract Resource ID
	if hclName := def.FieldMappings.ResourceID; hclName != "" {
		if val := getString(hclName); val != "" {
			resourceID = val
		}
	}

	// Parent ID
	if hclName := def.FieldMappings.Parent; hclName != "" {
		parentID = getString(hclName)
	}

	// Resource Type & Level
	resourceType := def.Type
	resourceLevel := def.ResourceLevel
	if resourceLevel == "" {
		resourceLevel = "resource"
	}

	// Determine parent type
	parentType = DetermineParentType(resourceLevel, parentID)

	// --- Check for Policy Data ---
	if hclName := def.FieldMappings.PolicyData; hclName != "" {
		policyDataJSON := getString(hclName)
		if policyDataJSON != "" {
			// Unmarshal Policy Data
			var policy Policy
			if err := json.Unmarshal([]byte(policyDataJSON), &policy); err != nil {
				return nil, fmt.Errorf("failed to parse policy_data JSON: %w", err)
			}

			var bindings []IAMBinding
			for _, pb := range policy.Bindings {
				b := IAMBinding{
					ResourceID:    resourceID,
					ResourceType:  resourceType,
					ResourceLevel: resourceLevel,
					Role:          pb.Role,
					Members:       pb.Members,
					ParentID:      parentID,
					ParentType:    parentType,
					TerraformAddr: terraformAddr,
				}
				bindings = append(bindings, b)
			}
			return bindings, nil
		}
	}

	// --- Standard IAM Binding/Member ---
	binding := IAMBinding{
		ResourceID:    resourceID,
		ResourceType:  resourceType,
		ResourceLevel: resourceLevel,
		ParentID:      parentID,
		ParentType:    parentType,
		TerraformAddr: terraformAddr,
	}

	// Role
	if hclName := def.FieldMappings.Role; hclName != "" {
		binding.Role = getString(hclName)
	}
	// Member
	if hclName := def.FieldMappings.Member; hclName != "" {
		m := getString(hclName)
		if m != "" {
			binding.Members = []string{m}
		}
	}
	// Members
	if hclName := def.FieldMappings.Members; hclName != "" {
		if attr, ok := attrs[hclName]; ok {
			val, err := traverser.ResolveExpression(attr.Expr)
			if err == nil {
				if val.Type().IsTupleType() || val.Type().IsListType() {
					it := val.ElementIterator()
					for it.Next() {
						_, v := it.Element()
						if v.Type() == cty.String {
							binding.Members = append(binding.Members, v.AsString())
						}
					}
				}
			}
		}
	}

	// Validation
	if binding.Role == "" || len(binding.Members) == 0 {
		return nil, fmt.Errorf("missing role or members")
	}

	return []IAMBinding{binding}, nil
}
