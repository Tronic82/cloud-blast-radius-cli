package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDir_EndToEnd(t *testing.T) {
	// 1. Setup Temp Dir
	tmpDir, err := os.MkdirTemp("", "parser-e2e")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. Create Terraform Files
	// variables.tf
	if err := os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte(`
variable "env" { default = "dev" }
variable "owner" { default = "default-owner" }
`), 0644); err != nil {
		t.Fatal(err)
	}

	// main.tf
	mainTF := `
resource "google_project" "my_project" {
  project_id = "test-project-${var.env}" // "test-project-prod" (overridden)
}

resource "google_project_iam_member" "binding_1" {
  project = google_project.my_project.project_id
  role    = "roles/editor"
  member  = "user:${var.owner}@example.com" // "user:custom-owner@..."
}

resource "google_storage_bucket" "my_bucket" {
  name = "bucket-${var.env}"
}

resource "google_storage_bucket_iam_member" "binding_2" {
  bucket = google_storage_bucket.my_bucket.name
  role   = "roles/storage.admin"
  member = "serviceAccount:sa@test.com"
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainTF), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Create tfvars
	tfvarsContent := `
env = "prod"
owner = "custom-owner"
`
	tfvarsPath := filepath.Join(tmpDir, "prod.tfvars")
	if err := os.WriteFile(tfvarsPath, []byte(tfvarsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 4. Define Resource Definitions (Mock)
	defs := []ResourceDefinition{
		{
			Type: "google_project_iam_member",
			FieldMappings: FieldMapping{
				ResourceID: "project",
				Role:       "role",
				Member:     "member",
			},
		},
		{
			Type: "google_storage_bucket_iam_member",
			FieldMappings: FieldMapping{
				ResourceID: "bucket",
				Role:       "role",
				Member:     "member",
			},
		},
	}

	// 5. Run ParseDir
	bindings, err := ParseDir(tmpDir, tfvarsPath, defs, nil)
	if err != nil {
		t.Fatalf("ParseDir failed: %v", err)
	}

	// 6. Verify Results
	if len(bindings) != 2 {
		t.Fatalf("Expected 2 bindings, got %d", len(bindings))
	}

	// Helper to check binding presence
	findBinding := func(role, resourceID string) *IAMBinding {
		for _, b := range bindings {
			if b.Role == role && b.ResourceID == resourceID {
				return &b
			}
		}
		return nil
	}

	// Check Binding 1 (Project)
	// Resolution: project_id = "test-project-prod", role = "roles/editor", member = "user:custom-owner@..."
	b1 := findBinding("roles/editor", "test-project-prod")
	if b1 == nil {
		t.Errorf("Binding 1 not found (Role: roles/editor, Resource: test-project-prod)")
	} else {
		if len(b1.Members) != 1 || b1.Members[0] != "user:custom-owner@example.com" {
			t.Errorf("Binding 1 Member mismatch: got %v", b1.Members)
		}
	}

	// Check Binding 2 (Bucket)
	// Resolution: bucket = "bucket-prod", role = "roles/storage.admin"
	b2 := findBinding("roles/storage.admin", "bucket-prod")
	if b2 == nil {
		t.Errorf("Binding 2 not found (Role: roles/storage.admin, Resource: bucket-prod)")
	} else {
		if len(b2.Members) != 1 || b2.Members[0] != "serviceAccount:sa@test.com" {
			t.Errorf("Binding 2 Member mismatch: got %v", b2.Members)
		}
	}
}
