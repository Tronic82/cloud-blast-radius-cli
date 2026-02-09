package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDir_PolicyResource(t *testing.T) {
	// 1. Setup Temp Dir
	tmpDir, err := os.MkdirTemp("", "parser-policy-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// 2. Create Terraform Files
	// main.tf
	mainTF := `
resource "google_project_iam_policy" "project_policy" {
  project = "my-project-id"
  policy_data = <<EOF
{
  "bindings": [
    {
      "role": "roles/owner",
      "members": ["user:owner@example.com"]
    },
    {
      "role": "roles/viewer",
      "members": ["user:viewer1@example.com", "group:viewers@example.com"]
    }
  ]
}
EOF
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainTF), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Define Resource Definitions
	defs := []ResourceDefinition{
		{
			Type: "google_project_iam_policy",
			FieldMappings: FieldMapping{
				ResourceID: "project",
				PolicyData: "policy_data",
			},
		},
	}

	// 4. Run ParseDir
	bindings, err := ParseDir(tmpDir, "", defs, nil)
	if err != nil {
		t.Fatalf("ParseDir failed: %v", err)
	}

	// 5. Verify Results
	// We expect 2 bindings:
	// 1. roles/owner -> [user:owner@example.com]
	// 2. roles/viewer -> [user:viewer1@example.com, group:viewers@example.com]
	if len(bindings) != 2 {
		t.Fatalf("Expected 2 bindings, got %d", len(bindings))
	}

	// Helper to find binding
	findBinding := func(role string) *IAMBinding {
		for _, b := range bindings {
			if b.Role == role {
				return &b
			}
		}
		return nil
	}

	// Check Owner Binding
	b1 := findBinding("roles/owner")
	if b1 == nil {
		t.Error("Binding for roles/owner not found")
	} else {
		if b1.ResourceID != "my-project-id" {
			t.Errorf("Expected ResourceID my-project-id, got %s", b1.ResourceID)
		}
		if len(b1.Members) != 1 || b1.Members[0] != "user:owner@example.com" {
			t.Errorf("Unexpected members for owner: %v", b1.Members)
		}
	}

	// Check Viewer Binding
	b2 := findBinding("roles/viewer")
	if b2 == nil {
		t.Error("Binding for roles/viewer not found")
	} else {
		if b2.ResourceID != "my-project-id" {
			t.Errorf("Expected ResourceID my-project-id, got %s", b2.ResourceID)
		}
		if len(b2.Members) != 2 {
			t.Errorf("Expected 2 members for viewer, got %d: %v", len(b2.Members), b2.Members)
		}
	}
}
