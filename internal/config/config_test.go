package config

import (
	"testing"
)

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name         string
		exclusions   []ExclusionRule
		resourceID   string
		resourceType string
		role         string
		want         bool
	}{
		{
			name: "Exact Match",
			exclusions: []ExclusionRule{
				{Resource: "^my-project$", Role: "^roles/owner$"},
			},
			resourceID:   "my-project",
			resourceType: "google_project",
			role:         "roles/owner",
			want:         true,
		},
		{
			name: "Regex Match",
			exclusions: []ExclusionRule{
				{Resource: "^dev-.*", Role: ".*"},
			},
			resourceID:   "dev-test-1",
			resourceType: "google_storage_bucket",
			role:         "roles/viewer",
			want:         true,
		},
		{
			name: "No Match - Different Project",
			exclusions: []ExclusionRule{
				{Resource: "^dev-.*"},
			},
			resourceID:   "prod-app",
			resourceType: "google_project",
			role:         "roles/owner",
			want:         false,
		},
		{
			name: "No Match - Different Role",
			exclusions: []ExclusionRule{
				{Resource: ".*", Role: "^roles/viewer$"},
			},
			resourceID:   "any-project",
			resourceType: "google_project",
			role:         "roles/owner",
			want:         false,
		},
		{
			name: "Match by ResourceType",
			exclusions: []ExclusionRule{
				{ResourceType: "^google_storage_bucket$"},
			},
			resourceID:   "my-bucket",
			resourceType: "google_storage_bucket",
			role:         "roles/storage.admin",
			want:         true,
		},
		{
			name: "Mixed Criterion",
			exclusions: []ExclusionRule{
				{ResourceType: "google_project", Role: ".*viewer.*"},
			},
			resourceID:   "my-project",
			resourceType: "google_project",
			role:         "roles/viewer",
			want:         true,
		},
		{
			name: "Empty Field acts as Wildcard",
			exclusions: []ExclusionRule{
				{Role: "roles/owner"}, // Resource & Type empty = match all
			},
			resourceID:   "any-res",
			resourceType: "any-type",
			role:         "roles/owner",
			want:         true,
		},
		{
			name: "Multiple Rules - Second Matches",
			exclusions: []ExclusionRule{
				{Resource: "nomatch"},
				{Role: "roles/owner"},
			},
			resourceID:   "project-x",
			resourceType: "t",
			role:         "roles/owner",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Exclusions: tt.exclusions,
			}

			if got := cfg.IsExcluded(tt.resourceID, tt.resourceType, tt.role); got != tt.want {
				t.Errorf("IsExcluded() = %v, want %v", got, tt.want)
			}
		})
	}
}
