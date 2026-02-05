package parser

// ResourceDefinition defines how to extract IAM information from a Terraform resource
type ResourceDefinition struct {
	Type          string       `yaml:"type"`           // e.g. "google_project_iam_member"
	DisplayName   string       `yaml:"display_name"`   // e.g. "BigQuery Datasets" - human readable name
	ResourceLevel string       `yaml:"resource_level"` // e.g. "project", "folder", "organization", "resource"
	FieldMappings FieldMapping `yaml:"field_mappings"` // Structured mapping of fields
}

// FieldMapping defines the HCL attribute names for specific IAM concepts
type FieldMapping struct {
	ResourceID string `yaml:"resource_id"` // e.g. "project", "dataset_id", "bucket"
	Role       string `yaml:"role"`        // e.g. "role"
	Member     string `yaml:"member"`      // e.g. "member"
	Members    string `yaml:"members"`     // e.g. "members"
	Parent     string `yaml:"parent"`      // e.g. "folder_id", "org_id" - parent resource reference
}
