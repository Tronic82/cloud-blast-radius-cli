package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// LoadVariables loads variables from .tf files (defaults) and a .tfvars file (overrides).
func LoadVariables(dir string, tfvarsPath string) (map[string]cty.Value, error) {
	vars := make(map[string]cty.Value)
	parser := hclparse.NewParser()

	// 1. Scan .tf files for variable defaults
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".tf") {
			path := filepath.Join(dir, f.Name())
			file, diags := parser.ParseHCLFile(path)
			if diags.HasErrors() {
				return nil, diags
			}

			// Correct approach for partial parsing:
			rootContent, _, _ := file.Body.PartialContent(&hcl.BodySchema{
				Blocks: []hcl.BlockHeaderSchema{
					{Type: "variable", LabelNames: []string{"name"}},
				},
			})

			for _, block := range rootContent.Blocks {
				if block.Type == "variable" {
					name := block.Labels[0]

					// Extract default value
					blockContent, _, _ := block.Body.PartialContent(&hcl.BodySchema{
						Attributes: []hcl.AttributeSchema{
							{Name: "default", Required: false},
						},
					})

					if attr, exists := blockContent.Attributes["default"]; exists {
						val, diags := attr.Expr.Value(nil) // Evaluate constant expression
						if !diags.HasErrors() {
							vars[name] = val
						}
					}
				}
			}
		}
	}

	// 2. Load .tfvars if provided or default exists
	loadTFVars := func(path string) error {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil // Not an error if file misses
		}

		f, diags := parser.ParseHCLFile(path)
		if diags.HasErrors() {
			return diags
		}

		attrs, diags := f.Body.JustAttributes()
		if diags.HasErrors() {
			return diags
		}

		for name, attr := range attrs {
			val, diags := attr.Expr.Value(nil)
			if !diags.HasErrors() {
				vars[name] = val
			}
		}
		return nil
	}

	// Prioritize passed tfvars > terraform.tfvars
	if tfvarsPath != "" {
		if err := loadTFVars(tfvarsPath); err != nil {
			return nil, err
		}
	} else {
		// Try default
		_ = loadTFVars(filepath.Join(dir, "terraform.tfvars"))
	}

	return vars, nil
}
