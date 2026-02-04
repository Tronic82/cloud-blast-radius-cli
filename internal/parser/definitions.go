package parser

// ResourceDefinition defines how to extract IAM information from a Terraform resource
type ResourceDefinition struct {
	Type          string       `yaml:"type"`           // e.g. "google_project_iam_member"
	FieldMappings FieldMapping `yaml:"field_mappings"` // Structured mapping of fields
}

// FieldMapping defines the HCL attribute names for specific IAM concepts
type FieldMapping struct {
	ResourceID string `yaml:"resource_id"` // e.g. "project", "dataset_id", "bucket"
	Role       string `yaml:"role"`        // e.g. "role"
	Member     string `yaml:"member"`      // e.g. "member"
	Members    string `yaml:"members"`     // e.g. "members"
}
