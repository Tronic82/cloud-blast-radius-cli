# Blast Radius CLI Documentation

Blast Radius is a CLI tool for analyzing the impact of IAM changes in Terraform configurations for Google Cloud Platform (GCP).

## Quick Start

```bash
# Initialize configuration
blast-radius init

# See who has access to what
blast-radius impact ./terraform

# Find inherited access from project/folder/org level
blast-radius hierarchy ./terraform

# Trace impersonation chains for a specific account
blast-radius analyze --account alice@example.com ./terraform

# Validate against organizational policies
blast-radius validate --policy policy.yaml ./terraform
```

## Commands

| Command | Description | Documentation |
|---------|-------------|---------------|
| [`init`](init.md) | Create default configuration file | [init.md](init.md) |
| [`impact`](impact.md) | Calculate blast radius of IAM principals | [impact.md](impact.md) |
| [`hierarchy`](hierarchy.md) | Analyze hierarchical access inheritance | [hierarchy.md](hierarchy.md) |
| [`analyze`](analyze.md) | Trace impersonation chains for accounts | [analyze.md](analyze.md) |
| [`validate`](validate.md) | Validate IAM against policies | [validate.md](validate.md) |

## Global Flags

These flags work with all commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--config <path>` | Path to configuration file | `blast-radius.yaml` |
| `--output <format>` | Output format: `text` or `json` | `text` |
| `--definitions <path>` | Path to custom resource definitions | Built-in |
| `--rules <path>` | Path to custom role rules | Built-in |

## Input Modes

Blast Radius supports two input modes:

### HCL Mode (Default)

Parses `.tf` files directly from a directory.

```bash
blast-radius impact ./terraform
```

### Plan Mode

Parses Terraform plan JSON for accurate values.

```bash
# Generate plan
terraform plan -out=tfplan
terraform show -json tfplan > plan.json

# Analyze
blast-radius impact --plan plan.json
```

**When to use Plan Mode:**
- Variables have complex expressions
- You want to see the exact values that will be applied
- You need 100% accurate resource IDs

## Configuration Files

### blast-radius.yaml

Main configuration file. See [init.md](init.md) for schema.

```yaml
cloud_provider: gcp
exclusions:
  - role: roles/viewer
ignored_directories:
  - .terraform
analysis_accounts:
  - alice@example.com
```

### Policy Files

Policy files for the `validate` command. See [validate.md](validate.md) for schema.

```yaml
cloud_provider: gcp
policies:
  - name: "No Owner Role for Service Accounts"
    type: role_restriction
    severity: error
    role_restriction:
      selector:
        principal_pattern: "serviceAccount:.*"
      denied_roles:
        - "roles/owner"
```

## Output Formats

### Text (Default)

Human-readable output with color coding:
- **Green**: Read access
- **Yellow**: Write access
- **Red**: Admin access
- **Cyan**: Impersonate access / Effective grants

### JSON

Machine-readable output for automation. Use `--output json`:

```bash
blast-radius impact --output json | jq '.principals[].principal'
```

Each command has a documented JSON schema in its respective documentation file.

## Supported GCP Resources

Blast Radius recognizes 110+ GCP IAM roles across these services:

- BigQuery
- Cloud Storage
- Compute Engine
- Cloud Run
- Cloud Functions
- Secret Manager
- Cloud SQL
- Spanner
- GKE (Kubernetes Engine)
- Pub/Sub
- Cloud KMS
- Artifact Registry
- Cloud Build
- Logging & Monitoring
- Dataflow & Dataproc
- And more...

## CI/CD Integration

### GitHub Actions

```yaml
- name: Validate IAM Policies
  run: |
    blast-radius validate --policy policy.yaml --output json > report.json
    if [ $? -ne 0 ]; then
      echo "::error::IAM policy violations detected"
      cat report.json | jq '.violations[]'
      exit 1
    fi
```

### GitLab CI

```yaml
validate-iam:
  script:
    - blast-radius validate --policy policy.yaml ./terraform
  allow_failure: false
```

### Pre-commit Hook

```bash
#!/bin/bash
blast-radius validate --policy policy.yaml .
```

## Troubleshooting

### "No matching principal found"

The email doesn't exist in any IAM binding. Check:
- Is the email correct?
- Is the Terraform code in the scanned directory?
- Are you using Plan mode if variables are complex?

### "Role not found in definitions"

Custom roles or new GCP roles may not be in the built-in definitions. The tool will show a warning but continue. Hierarchical access for unknown roles cannot be determined.

### "Hierarchy unknown"

The parent folder/organization isn't defined in the Terraform code. This is common when projects reference external folders. The tool warns that additional inherited permissions may exist.

## Examples

See the `examples/` directory in the repository for complete working examples:

- `01-basic-impact/` - Simple IAM bindings
- `02-hierarchical-access/` - Project-level permissions
- `03-impersonation-chains/` - Service account impersonation
- `04-policy-role-restrictions/` - Policy validation
- `05-dynamic-blocks/` - Dynamic Terraform blocks
- `06-variable-resolution/` - Variable handling
