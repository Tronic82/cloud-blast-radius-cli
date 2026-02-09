package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParsePlanFile_PolicyResource(t *testing.T) {
	// Create a temporary plan file
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "test-plan-policy.json")

	// Create test plan JSON with policy_data
	// mimicking what we see in example_policy.json
	policyData := `{"bindings":[{"members":["user:owner@example.com"],"role":"roles/owner"},{"members":["user:viewer@example.com"],"role":"roles/viewer"}]}`

	plan := TerraformPlan{
		FormatVersion:    "1.2",
		TerraformVersion: "1.5.4",
		PlannedValues: PlannedValues{
			RootModule: Module{
				Resources: []Resource{
					{
						Address:      "google_project_iam_policy.policy",
						Mode:         "managed",
						Type:         "google_project_iam_policy",
						Name:         "policy",
						ProviderName: "google",
						Values: map[string]interface{}{
							"project":     "my-br-test-project",
							"policy_data": policyData,
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
			Type: "google_project_iam_policy",
			FieldMappings: FieldMapping{
				ResourceID: "project",
				PolicyData: "policy_data",
			},
		},
	}

	// Parse plan file
	bindings, err := ParsePlanFile(planFile, definitions)
	if err != nil {
		t.Fatalf("ParsePlanFile failed: %v", err)
	}

	// Verify bindings
	// We expect 2 bindings from the policy_data
	if len(bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(bindings))
	}

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
		if b1.ResourceID != "my-br-test-project" {
			t.Errorf("Expected ResourceID my-br-test-project, got %s", b1.ResourceID)
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
		if b2.ResourceID != "my-br-test-project" {
			t.Errorf("Expected ResourceID my-br-test-project, got %s", b2.ResourceID)
		}
		if len(b2.Members) != 1 || b2.Members[0] != "user:viewer@example.com" {
			t.Errorf("Unexpected members for viewer: %v", b2.Members)
		}
	}
}
