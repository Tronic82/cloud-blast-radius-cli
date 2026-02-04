package parser

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestResolveExpression(t *testing.T) {
	// 1. Setup Mock HCL Files
	parser := hclparse.NewParser()

	// File 1: Defines resources to be referenced
	src1 := `
resource "google_bigquery_dataset" "my_ds" {
  dataset_id = "target_dataset_id"
}

resource "google_project" "my_project" {
  project_id = "target_project_id"
  name       = google_bigquery_dataset.my_ds.dataset_id // Recursive ref
}
`
	file1, diags := parser.ParseHCL([]byte(src1), "file1.tf")
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	// Setup Variables
	vars := map[string]cty.Value{
		"env": cty.StringVal("prod"),
	}

	traverser := NewConfigTraverser([]*hcl.File{file1}, vars)

	tests := []struct {
		name    string
		exprStr string
		wantStr string
		wantErr bool
	}{
		{
			name:    "Literal String",
			exprStr: `"static_value"`,
			wantStr: "static_value",
		},
		{
			name:    "Variable Lookup",
			exprStr: `var.env`,
			wantStr: "prod",
		},
		{
			name:    "Direct Resource Reference",
			exprStr: `google_bigquery_dataset.my_ds.dataset_id`,
			wantStr: "target_dataset_id",
		},
		{
			name:    "Recursive Resource Reference",
			exprStr: `google_project.my_project.name`,
			// Should resolve to my_project.name -> my_ds.dataset_id -> "target_dataset_id"
			wantStr: "target_dataset_id",
		},
		{
			name:    "Resource Not Found",
			exprStr: `google_storage_bucket.missing.name`,
			wantErr: true,
		},
		{
			name:    "Attribute Not Found",
			exprStr: `google_bigquery_dataset.my_ds.missing_attr`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the expression string into an HCL expression
			expr, diags := hclsyntax.ParseExpression([]byte(tt.exprStr), "test.tf", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf("Failed to parse test expression: %v", diags)
			}

			val, err := traverser.ResolveExpression(expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if val.Type() != cty.String || val.AsString() != tt.wantStr {
					t.Errorf("ResolveExpression() = %v, want %v", val, tt.wantStr)
				}
			}
		})
	}
}
