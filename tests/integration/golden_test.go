package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// normalizeJSON removes timestamp field from JSON for comparison
func normalizeJSON(jsonStr string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	// Remove timestamp as it changes on every run
	delete(data, "timestamp")

	return data, nil
}

// compareWithGoldenFile compares output with golden file, or creates it if -update flag is set
func compareWithGoldenFile(t *testing.T, output string, goldenPath string, update bool) {
	goldenFile := filepath.Join("testdata", goldenPath)

	if update {
		// Create golden file
		if err := os.MkdirAll(filepath.Dir(goldenFile), 0755); err != nil {
			t.Fatalf("failed to create testdata dir: %v", err)
		}

		// Normalize before saving
		normalized, err := normalizeJSON(output)
		if err != nil {
			t.Fatalf("failed to normalize JSON for golden file: %v", err)
		}

		prettyJSON, err := json.MarshalIndent(normalized, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal normalized JSON: %v", err)
		}

		if err := os.WriteFile(goldenFile, prettyJSON, 0644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenFile)
		return
	}

	// Read golden file
	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file %s (run with -update to create): %v", goldenFile, err)
	}

	// Normalize both for comparison
	actualNormalized, err := normalizeJSON(output)
	if err != nil {
		t.Fatalf("failed to normalize actual output: %v", err)
	}

	expectedNormalized, err := normalizeJSON(string(expected))
	if err != nil {
		t.Fatalf("failed to normalize expected output: %v", err)
	}

	// Compare
	if diff := cmp.Diff(expectedNormalized, actualNormalized); diff != "" {
		t.Errorf("output mismatch (-expected +actual):\n%s", diff)
	}
}
