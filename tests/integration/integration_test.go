package integration

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestIntegration_Examples(t *testing.T) {
	// Build the binary once
	binaryPath := buildBlastRadius(t)

	// List of examples to test
	examplesDir, err := filepath.Abs("../../examples")
	if err != nil {
		t.Fatalf("failed to resolve examples dir: %v", err)
	}

	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("failed to read examples dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		exampleDir := filepath.Join(examplesDir, entry.Name())

		// Test HCL mode (direct parsing)
		hclDir := filepath.Join(exampleDir, "hcl")
		if _, err := os.Stat(filepath.Join(hclDir, "main.tf")); err == nil {
			t.Run(entry.Name()+"/hcl", func(t *testing.T) {
				// Run blast-radius directly on HCL files (no Terraform needed)
				stdout, stderr, err := runCommand(hclDir, binaryPath, "impact", "--output", "json")
				if err != nil {
					t.Fatalf("blast-radius run failed: %v\nstderr: %s", err, stderr)
				}

				// Verify output against golden file
				compareWithGoldenFile(t, stdout, entry.Name()+"_hcl.json", *update)

				t.Logf("HCL mode test passed for %s", entry.Name())
			})
		}

		// Test plan-mode (Terraform JSON)
		planDir := filepath.Join(exampleDir, "plan-mode")
		if _, err := os.Stat(filepath.Join(planDir, "main.tf")); err == nil {
			t.Run(entry.Name()+"/plan-mode", func(t *testing.T) {
				// Set up mock GCP credentials for Terraform
				cleanup := setupMockGCPCredentials(t)
				defer cleanup()

				// 1. Run terraform init
				ensureTerraformInit(t, planDir)

				// 2. Run terraform plan -out=tfplan
				stdout, stderr, err := runCommand(planDir, "terraform", "plan", "-out=tfplan")
				if err != nil {
					t.Fatalf("terraform plan failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
				}

				// 3. Run terraform show -json tfplan > blast.json
				planJSON, stderr, err := runCommand(planDir, "terraform", "show", "-json", "tfplan")
				if err != nil {
					t.Fatalf("terraform show -json failed: %v\nstdout: %s\nstderr: %s", err, planJSON, stderr)
				}

				blastJSONFile := filepath.Join(planDir, "blast.json")
				if err := os.WriteFile(blastJSONFile, []byte(planJSON), 0644); err != nil {
					t.Fatalf("failed to write blast.json: %v", err)
				}
				defer func() { _ = os.Remove(blastJSONFile) }()

				// 4. Run blast-radius with the plan JSON
				stdout, stderr, err = runCommand(planDir, binaryPath, "impact", "--plan", "blast.json", "--output", "json")
				if err != nil {
					t.Fatalf("blast-radius run failed: %v\nstderr: %s", err, stderr)
				}

				// Verify output against golden file
				compareWithGoldenFile(t, stdout, entry.Name()+"_plan.json", *update)

				t.Logf("Plan-mode test passed for %s", entry.Name())
			})
		}
	}
}
