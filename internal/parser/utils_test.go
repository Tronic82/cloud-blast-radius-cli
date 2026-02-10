package parser

import "testing"

func TestDetermineParentType(t *testing.T) {
	tests := []struct {
		name          string
		resourceLevel string
		parentID      string
		want          string
	}{
		{
			name:          "Folder Level - Org Parent (Prefix)",
			resourceLevel: "folder",
			parentID:      "organizations/123",
			want:          "organization",
		},
		{
			name:          "Folder Level - Folder Parent (Prefix)",
			resourceLevel: "folder",
			parentID:      "folders/456",
			want:          "folder",
		},
		{
			name:          "Folder Level - Ambiguous (Legacy Behavior)",
			resourceLevel: "folder",
			parentID:      "123",
			want:          "organization", // Preserving existing behavior for now
		},
		{
			name:          "Project Level - Folder Parent (Prefix)",
			resourceLevel: "project",
			parentID:      "folders/789",
			want:          "folder",
		},
		{
			name:          "Project Level - Org Parent (Prefix)",
			resourceLevel: "project",
			parentID:      "organizations/123",
			want:          "organization",
		},
		{
			name:          "Project Level - Ambiguous (Legacy Behavior)",
			resourceLevel: "project",
			parentID:      "789",
			want:          "folder", // Preserving existing behavior for now
		},
		{
			name:          "Resource Level - Always Project",
			resourceLevel: "resource",
			parentID:      "",
			want:          "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetermineParentType(tt.resourceLevel, tt.parentID); got != tt.want {
				t.Errorf("DetermineParentType(%q, %q) = %q, want %q", tt.resourceLevel, tt.parentID, got, tt.want)
			}
		})
	}
}
