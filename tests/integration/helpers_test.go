package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runCommand runs a command and returns stdout, stderr, and error.
func runCommand(dir string, name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// ensureTerraformInit runs terraform init in the given directory if not already initialized.
func ensureTerraformInit(t *testing.T, dir string) {
	// We'll rely on "terraform init" being idempotent.
	// Use -backend=false since we don't need remote state for testing
	stdout, stderr, err := runCommand(dir, "terraform", "init", "-backend=false", "-upgrade")
	if err != nil {
		t.Fatalf("terraform init failed in %s: %v\nstdout: %s\nstderr: %s", dir, err, stdout, stderr)
	}
}

// setupMockGCPCredentials sets up mock GCP credentials for Terraform to use.
// This allows terraform plan to run without real credentials.
func setupMockGCPCredentials(t *testing.T) func() {
	mockCredsPath, err := filepath.Abs("mock-gcp-credentials.json")
	if err != nil {
		t.Fatalf("failed to get mock credentials path: %v", err)
	}

	// Save original value
	originalCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	// Set mock credentials
	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", mockCredsPath); err != nil {
		t.Fatalf("failed to set mock credentials env: %v", err)
	}

	// Return cleanup function
	return func() {
		if originalCreds != "" {
			if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", originalCreds); err != nil {
				t.Errorf("failed to restore original credentials env: %v", err)
			}
		} else {
			if err := os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS"); err != nil {
				t.Errorf("failed to unset mock credentials env: %v", err)
			}
		}
	}
}

// buildBlastRadius builds the blast-radius binary for testing.
// Returns the absolute path to the binary.
func buildBlastRadius(t *testing.T) string {
	rootDir, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("failed to get absolute path to root: %v", err)
	}

	outputPath := filepath.Join(rootDir, "bin", "blast-radius-test.exe")
	// Note: Assuming Windows environment based on user metadata, adding .exe extension.
	// Ideally this should detect OS, but for this context Windows is specified.
	if !isWindows() {
		outputPath = filepath.Join(rootDir, "bin", "blast-radius-test")
	}

	cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/blast-radius")
	cmd.Dir = rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build blast-radius: %v\noutput: %s", err, out)
	}

	return outputPath
}

func isWindows() bool {
	return filepath.Separator == '\\'
}
