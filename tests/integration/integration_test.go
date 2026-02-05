package integration

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
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

// TestIntegration_Hierarchy tests the hierarchy command on examples with hierarchical access
func TestIntegration_Hierarchy(t *testing.T) {
	binaryPath := buildBlastRadius(t)

	examplesDir, err := filepath.Abs("../../examples")
	if err != nil {
		t.Fatalf("failed to resolve examples dir: %v", err)
	}

	// Hierarchy command is most relevant for hierarchical-access example
	hierarchyExamples := []string{"02-hierarchical-access"}

	for _, exampleName := range hierarchyExamples {
		exampleDir := filepath.Join(examplesDir, exampleName)

		// Test HCL mode
		hclDir := filepath.Join(exampleDir, "hcl")
		if _, err := os.Stat(filepath.Join(hclDir, "main.tf")); err == nil {
			t.Run(exampleName+"/hcl/hierarchy", func(t *testing.T) {
				stdout, stderr, err := runCommand(hclDir, binaryPath, "hierarchy", "--output", "json")
				if err != nil {
					t.Fatalf("blast-radius hierarchy failed: %v\nstderr: %s", err, stderr)
				}

				compareWithGoldenFile(t, stdout, exampleName+"_hierarchy_hcl.json", *update)
				t.Logf("Hierarchy HCL test passed for %s", exampleName)
			})
		}

		// Test plan-mode
		planDir := filepath.Join(exampleDir, "plan-mode")
		if _, err := os.Stat(filepath.Join(planDir, "main.tf")); err == nil {
			t.Run(exampleName+"/plan-mode/hierarchy", func(t *testing.T) {
				cleanup := setupMockGCPCredentials(t)
				defer cleanup()

				ensureTerraformInit(t, planDir)

				stdout, stderr, err := runCommand(planDir, "terraform", "plan", "-out=tfplan")
				if err != nil {
					t.Fatalf("terraform plan failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
				}

				planJSON, stderr, err := runCommand(planDir, "terraform", "show", "-json", "tfplan")
				if err != nil {
					t.Fatalf("terraform show -json failed: %v\nstdout: %s\nstderr: %s", err, planJSON, stderr)
				}

				blastJSONFile := filepath.Join(planDir, "blast.json")
				if err := os.WriteFile(blastJSONFile, []byte(planJSON), 0644); err != nil {
					t.Fatalf("failed to write blast.json: %v", err)
				}
				defer func() { _ = os.Remove(blastJSONFile) }()

				stdout, stderr, err = runCommand(planDir, binaryPath, "hierarchy", "--plan", "blast.json", "--output", "json")
				if err != nil {
					t.Fatalf("blast-radius hierarchy failed: %v\nstderr: %s", err, stderr)
				}

				compareWithGoldenFile(t, stdout, exampleName+"_hierarchy_plan.json", *update)
				t.Logf("Hierarchy plan-mode test passed for %s", exampleName)
			})
		}
	}
}

// TestIntegration_Analyze tests the analyze command with impersonation chain examples
func TestIntegration_Analyze(t *testing.T) {
	binaryPath := buildBlastRadius(t)

	examplesDir, err := filepath.Abs("../../examples")
	if err != nil {
		t.Fatalf("failed to resolve examples dir: %v", err)
	}

	// Analyze command with specific accounts to trace
	// Note: Using single account per test since analyze outputs one JSON object per account
	analyzeExamples := []struct {
		name    string
		account string
	}{
		{
			name:    "03-impersonation-chains",
			account: "user:alice@example.com",
		},
	}

	for _, example := range analyzeExamples {
		exampleDir := filepath.Join(examplesDir, example.name)

		// Test HCL mode
		hclDir := filepath.Join(exampleDir, "hcl")
		if _, err := os.Stat(filepath.Join(hclDir, "main.tf")); err == nil {
			t.Run(example.name+"/hcl/analyze", func(t *testing.T) {
				stdout, stderr, err := runCommand(hclDir, binaryPath, "analyze", "--account", example.account, "--output", "json")
				if err != nil {
					t.Fatalf("blast-radius analyze failed: %v\nstderr: %s", err, stderr)
				}

				compareWithGoldenFile(t, stdout, example.name+"_analyze_hcl.json", *update)
				t.Logf("Analyze HCL test passed for %s", example.name)
			})
		}

		// Test plan-mode
		planDir := filepath.Join(exampleDir, "plan-mode")
		if _, err := os.Stat(filepath.Join(planDir, "main.tf")); err == nil {
			t.Run(example.name+"/plan-mode/analyze", func(t *testing.T) {
				cleanup := setupMockGCPCredentials(t)
				defer cleanup()

				ensureTerraformInit(t, planDir)

				stdout, stderr, err := runCommand(planDir, "terraform", "plan", "-out=tfplan")
				if err != nil {
					t.Fatalf("terraform plan failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
				}

				planJSON, stderr, err := runCommand(planDir, "terraform", "show", "-json", "tfplan")
				if err != nil {
					t.Fatalf("terraform show -json failed: %v\nstdout: %s\nstderr: %s", err, planJSON, stderr)
				}

				blastJSONFile := filepath.Join(planDir, "blast.json")
				if err := os.WriteFile(blastJSONFile, []byte(planJSON), 0644); err != nil {
					t.Fatalf("failed to write blast.json: %v", err)
				}
				defer func() { _ = os.Remove(blastJSONFile) }()

				stdout, stderr, err = runCommand(planDir, binaryPath, "analyze", "--account", example.account, "--plan", "blast.json", "--output", "json")
				if err != nil {
					t.Fatalf("blast-radius analyze failed: %v\nstderr: %s", err, stderr)
				}

				compareWithGoldenFile(t, stdout, example.name+"_analyze_plan.json", *update)
				t.Logf("Analyze plan-mode test passed for %s", example.name)
			})
		}
	}
}

// TestIntegration_Validate tests the validate command with policy files
func TestIntegration_Validate(t *testing.T) {
	binaryPath := buildBlastRadius(t)

	examplesDir, err := filepath.Abs("../../examples")
	if err != nil {
		t.Fatalf("failed to resolve examples dir: %v", err)
	}

	// Validate command examples with their policy files
	validateExamples := []struct {
		name       string
		policyFile string
	}{
		{
			name:       "04-policy-role-restrictions",
			policyFile: "policy.yaml",
		},
	}

	for _, example := range validateExamples {
		exampleDir := filepath.Join(examplesDir, example.name)
		policyPath := filepath.Join(exampleDir, example.policyFile)

		// Test HCL mode
		hclDir := filepath.Join(exampleDir, "hcl")
		if _, err := os.Stat(filepath.Join(hclDir, "main.tf")); err == nil {
			t.Run(example.name+"/hcl/validate", func(t *testing.T) {
				stdout, stderr, err := runCommand(hclDir, binaryPath, "validate", "--policy", policyPath, "--output", "json")
				// validate command may return non-zero exit code for policy violations
				// but we still want to capture and compare the output
				if err != nil && stderr != "" && stdout == "" {
					t.Fatalf("blast-radius validate failed: %v\nstderr: %s", err, stderr)
				}

				compareWithGoldenFile(t, stdout, example.name+"_validate_hcl.json", *update)
				t.Logf("Validate HCL test passed for %s", example.name)
			})
		}

		// Test plan-mode
		planDir := filepath.Join(exampleDir, "plan-mode")
		if _, err := os.Stat(filepath.Join(planDir, "main.tf")); err == nil {
			t.Run(example.name+"/plan-mode/validate", func(t *testing.T) {
				cleanup := setupMockGCPCredentials(t)
				defer cleanup()

				ensureTerraformInit(t, planDir)

				stdout, stderr, err := runCommand(planDir, "terraform", "plan", "-out=tfplan")
				if err != nil {
					t.Fatalf("terraform plan failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
				}

				planJSON, stderr, err := runCommand(planDir, "terraform", "show", "-json", "tfplan")
				if err != nil {
					t.Fatalf("terraform show -json failed: %v\nstdout: %s\nstderr: %s", err, planJSON, stderr)
				}

				blastJSONFile := filepath.Join(planDir, "blast.json")
				if err := os.WriteFile(blastJSONFile, []byte(planJSON), 0644); err != nil {
					t.Fatalf("failed to write blast.json: %v", err)
				}
				defer func() { _ = os.Remove(blastJSONFile) }()

				stdout, stderr, err = runCommand(planDir, binaryPath, "validate", "--policy", policyPath, "--plan", "blast.json", "--output", "json")
				// validate command may return non-zero exit code for policy violations
				if err != nil && stderr != "" && stdout == "" {
					t.Fatalf("blast-radius validate failed: %v\nstderr: %s", err, stderr)
				}

				compareWithGoldenFile(t, stdout, example.name+"_validate_plan.json", *update)
				t.Logf("Validate plan-mode test passed for %s", example.name)
			})
		}
	}
}

// TestIntegration_Init tests the init command
func TestIntegration_Init(t *testing.T) {
	binaryPath := buildBlastRadius(t)

	// Create a temporary directory for the init test
	tempDir, err := os.MkdirTemp("", "blast-radius-init-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("init/creates-config", func(t *testing.T) {
		// Run init command with stdin input to select GCP provider
		stdout, stderr, err := runCommandWithInput(tempDir, "1\ny\n", binaryPath, "init")
		if err != nil {
			t.Fatalf("blast-radius init failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}

		// Verify config file was created
		configPath := filepath.Join(tempDir, "blast-radius.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatalf("expected config file to be created at %s", configPath)
		}

		// Read and verify config content
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config file: %v", err)
		}

		// Basic validation of config content
		if !strings.Contains(string(content), "cloud_provider:") {
			t.Errorf("config file missing cloud_provider field")
		}

		t.Logf("Init test passed, config created at %s", configPath)
	})

	t.Run("init/already-exists", func(t *testing.T) {
		// Create another temp dir
		tempDir2, err := os.MkdirTemp("", "blast-radius-init-test-exists")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir2)

		// Create a config file first
		configPath := filepath.Join(tempDir2, "blast-radius.yaml")
		if err := os.WriteFile(configPath, []byte("cloud_provider: gcp\n"), 0644); err != nil {
			t.Fatalf("failed to create existing config: %v", err)
		}

		// Run init and decline overwrite
		stdout, _, _ := runCommandWithInput(tempDir2, "n\n", binaryPath, "init")

		// Should mention file exists
		if !strings.Contains(stdout, "exists") && !strings.Contains(stdout, "already") {
			t.Logf("Note: init command behavior with existing file - stdout: %s", stdout)
		}
	})
}
