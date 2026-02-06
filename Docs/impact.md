# blast-radius impact

## Summary

The `impact` command analyzes Terraform IAM configurations to determine the **blast radius** of IAM principals. It shows which resources each principal (user, group, or service account) has access to and what roles they hold.

This is the primary command for understanding "who can access what" in your infrastructure.

## Usage

```bash
blast-radius impact [directory] [flags]
```

### Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `directory` | Path to directory containing Terraform files | Current directory (`.`) |

### Flags

| Flag | Description |
|------|-------------|
| `--plan <path>` | Path to Terraform plan JSON file. When specified, analyzes the plan instead of HCL files |
| `--tfvars <path>` | Path to terraform.tfvars file for variable resolution |
| `--config <path>` | Path to blast-radius.yaml configuration file (default: `blast-radius.yaml`) |
| `--output <format>` | Output format: `text` or `json` (default: `text`) |
| `--definitions <path>` | Path to custom resource definitions file |
| `--rules <path>` | Path to custom validation rules file |
| `-v, --visual` | Enable visual output (placeholder for future feature) |

## Text Output

### Example Output

```
Analyzing directory: ./terraform
Loaded 300 resource definitions

--- Analysis Results ---

Principal: user:alice@example.com
  Resources (2):
    - analytics-dataset (google_bigquery_dataset_iam_member):
      - roles/bigquery.dataViewer
    - my-project (google_project_iam_member):
      - roles/viewer

Principal: serviceAccount:app-sa@project.iam.gserviceaccount.com
  Resources (1):
    - production-secrets (google_storage_bucket_iam_member):
      - roles/storage.admin
```

### How to Read the Output

1. **Header**: Shows the directory or plan file being analyzed and how many resource definitions were loaded

2. **Principal Block**: Each principal (user, group, service account) gets its own section
   - Format: `Principal: <type>:<identifier>`
   - Types: `user:`, `group:`, `serviceAccount:`, `domain:`

3. **Resources**: Lists all resources the principal has access to
   - `Resources (N):` - Total count of resources
   - Each resource shows:
     - **Resource ID** - The identifier of the resource (project ID, bucket name, dataset ID, etc.)
     - **Resource Type** - The Terraform resource type in parentheses
     - **Roles** - List of IAM roles granted on this resource

### Color Coding

When running in a terminal that supports colors:
- **Cyan bold**: Section headers ("--- Analysis Results ---")
- **White bold**: Principal labels

## JSON Output

Use `--output json` for machine-readable output.

### Schema

```json
{
  "command": "impact",
  "timestamp": "2024-01-15T10:30:00Z",
  "principals": [
    {
      "principal": "user:alice@example.com",
      "resources": [
        {
          "resource_id": "my-project",
          "resource_type": "google_project_iam_member",
          "roles": ["roles/viewer", "roles/editor"],
          "terraform_addresses": {
            "roles/viewer": "google_project_iam_member.alice_viewer",
            "roles/editor": "google_project_iam_member.alice_editor"
          }
        }
      ]
    }
  ]
}
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Always `"impact"` |
| `timestamp` | string | ISO 8601 timestamp of when the analysis ran |
| `principals` | array | List of principals with their access |
| `principals[].principal` | string | Full principal identifier (e.g., `user:alice@example.com`) |
| `principals[].resources` | array | List of resources this principal can access |
| `principals[].resources[].resource_id` | string | The resource identifier |
| `principals[].resources[].resource_type` | string | Terraform resource type |
| `principals[].resources[].roles` | array | List of IAM roles on this resource |
| `principals[].resources[].terraform_addresses` | object | (Optional) Map of role to Terraform resource address |

## Configuration File

The `impact` command respects settings in `blast-radius.yaml`:

```yaml
cloud_provider: gcp

# Exclude certain bindings from the output
exclusions:
  - resource: "example-ignored-project"  # Regex pattern
    role: ".*"                            # Match any role
  - role: "roles/viewer"                  # Exclude all viewer roles

# Directories to skip when scanning
ignored_directories:
  - ".terraform"
  - "modules"
```

See [configuration.md](configuration.md) for the full schema.

## Examples

### Basic Usage

```bash
# Analyze current directory
blast-radius impact

# Analyze specific directory
blast-radius impact ./terraform/production

# Analyze with custom config
blast-radius impact --config my-config.yaml ./terraform
```

### Using Terraform Plan

```bash
# Generate plan JSON
terraform plan -out=tfplan
terraform show -json tfplan > plan.json

# Analyze the plan
blast-radius impact --plan plan.json
```

### JSON Output for CI/CD

```bash
# Get JSON output and save to file
blast-radius impact --output json ./terraform > impact-report.json

# Parse with jq to find all principals with owner role
blast-radius impact --output json | jq '.principals[] | select(.resources[].roles[] | contains("owner"))'
```

### With Variable Resolution

```bash
# Use tfvars for variable values
blast-radius impact --tfvars production.tfvars ./terraform
```
