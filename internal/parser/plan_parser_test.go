package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParsePlanFile(t *testing.T) {
	// Create a temporary plan file
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "test-plan.json")

	// Create test plan JSON
	plan := TerraformPlan{
		FormatVersion:    "1.0",
		TerraformVersion: "1.5.0",
		PlannedValues: PlannedValues{
			RootModule: Module{
				Resources: []Resource{
					{
						Address:      "google_project_iam_binding.test",
						Mode:         "managed",
						Type:         "google_project_iam_binding",
						Name:         "test",
						ProviderName: "google",
						Values: map[string]interface{}{
							"project": "test-project-123",
							"role":    "roles/viewer",
							"members": []interface{}{
								"user:test@example.com",
								"serviceAccount:sa@test-project-123.iam.gserviceaccount.com",
							},
						},
					},
					{
						Address:      "google_service_account_iam_member.test",
						Mode:         "managed",
						Type:         "google_service_account_iam_member",
						Name:         "test",
						ProviderName: "google",
						Values: map[string]interface{}{
							"service_account_id": "projects/test-project/serviceAccounts/target@test-project.iam.gserviceaccount.com",
							"role":               "roles/iam.serviceAccountTokenCreator",
							"member":             "serviceAccount:impersonator@test-project.iam.gserviceaccount.com",
						},
					},
				},
			},
		},
	}

	// Write plan to file
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal plan: %v", err)
	}

	if err := os.WriteFile(planFile, data, 0644); err != nil {
		t.Fatalf("Failed to write plan file: %v", err)
	}

	// Create definitions
	definitions := []ResourceDefinition{
		{
			Type: "google_project_iam_binding",
			FieldMappings: FieldMapping{
				ResourceID: "project",
				Role:       "role",
				Members:    "members",
			},
		},
		{
			Type: "google_service_account_iam_member",
			FieldMappings: FieldMapping{
				ResourceID: "service_account_id",
				Role:       "role",
				Member:     "member",
			},
		},
	}

	// Parse plan file
	bindings, err := ParsePlanFile(planFile, definitions)
	if err != nil {
		t.Fatalf("ParsePlanFile failed: %v", err)
	}

	// Verify bindings
	if len(bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(bindings))
	}

	// Verify first binding (google_project_iam_binding)
	if bindings[0].ResourceType != "google_project_iam_binding" {
		t.Errorf("Expected ResourceType 'google_project_iam_binding', got '%s'", bindings[0].ResourceType)
	}
	if bindings[0].ResourceID != "test-project-123" {
		t.Errorf("Expected ResourceID 'test-project-123', got '%s'", bindings[0].ResourceID)
	}
	if bindings[0].Role != "roles/viewer" {
		t.Errorf("Expected Role 'roles/viewer', got '%s'", bindings[0].Role)
	}
	if len(bindings[0].Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(bindings[0].Members))
	}

	// Verify second binding (google_service_account_iam_member)
	if bindings[1].ResourceType != "google_service_account_iam_member" {
		t.Errorf("Expected ResourceType 'google_service_account_iam_member', got '%s'", bindings[1].ResourceType)
	}
	if bindings[1].ResourceID != "projects/test-project/serviceAccounts/target@test-project.iam.gserviceaccount.com" {
		t.Errorf("Expected ResourceID 'projects/test-project/serviceAccounts/target@test-project.iam.gserviceaccount.com', got '%s'", bindings[1].ResourceID)
	}
	if bindings[1].Role != "roles/iam.serviceAccountTokenCreator" {
		t.Errorf("Expected Role 'roles/iam.serviceAccountTokenCreator', got '%s'", bindings[1].Role)
	}
	if len(bindings[1].Members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(bindings[1].Members))
	}
	if bindings[1].Members[0] != "serviceAccount:impersonator@test-project.iam.gserviceaccount.com" {
		t.Errorf("Expected member 'serviceAccount:impersonator@test-project.iam.gserviceaccount.com', got '%s'", bindings[1].Members[0])
	}
}

func TestParsePlanFileWithNestedModules(t *testing.T) {
	// Create a temporary plan file
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "test-plan-nested.json")

	// Create test plan JSON with nested modules
	plan := TerraformPlan{
		FormatVersion:    "1.0",
		TerraformVersion: "1.5.0",
		PlannedValues: PlannedValues{
			RootModule: Module{
				Resources: []Resource{
					{
						Address:      "google_project_iam_binding.root",
						Mode:         "managed",
						Type:         "google_project_iam_binding",
						Name:         "root",
						ProviderName: "google",
						Values: map[string]interface{}{
							"project": "root-project",
							"role":    "roles/viewer",
							"members": []interface{}{
								"user:root@example.com",
							},
						},
					},
				},
				ChildModules: []Module{
					{
						Resources: []Resource{
							{
								Address:      "module.child.google_project_iam_binding.child",
								Mode:         "managed",
								Type:         "google_project_iam_binding",
								Name:         "child",
								ProviderName: "google",
								Values: map[string]interface{}{
									"project": "child-project",
									"role":    "roles/editor",
									"members": []interface{}{
										"user:child@example.com",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Write plan to file
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal plan: %v", err)
	}

	if err := os.WriteFile(planFile, data, 0644); err != nil {
		t.Fatalf("Failed to write plan file: %v", err)
	}

	// Create definitions
	definitions := []ResourceDefinition{
		{
			Type: "google_project_iam_binding",
			FieldMappings: FieldMapping{
				ResourceID: "project",
				Role:       "role",
				Members:    "members",
			},
		},
	}

	// Parse plan file
	bindings, err := ParsePlanFile(planFile, definitions)
	if err != nil {
		t.Fatalf("ParsePlanFile failed: %v", err)
	}

	// Verify bindings from both root and child modules
	if len(bindings) != 2 {
		t.Errorf("Expected 2 bindings (root + child), got %d", len(bindings))
	}

	// Verify root binding
	if bindings[0].ResourceID != "root-project" {
		t.Errorf("Expected first binding from root-project, got '%s'", bindings[0].ResourceID)
	}

	// Verify child binding
	if bindings[1].ResourceID != "child-project" {
		t.Errorf("Expected second binding from child-project, got '%s'", bindings[1].ResourceID)
	}
}
