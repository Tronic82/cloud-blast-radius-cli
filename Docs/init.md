# blast-radius init

## Summary

The `init` command creates a default configuration file (`blast-radius.yaml`) in your project. This configuration file allows you to customize how blast-radius analyzes your Terraform code.

## Usage

```bash
blast-radius init [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--config <path>` | Path to create config file (default: `blast-radius.yaml`) |

## Interactive Prompts

The `init` command is interactive and will prompt you:

1. **If config exists**: Ask whether to use existing file or overwrite
2. **Cloud Provider**: Currently only GCP is supported

### Example Session

```
$ blast-radius init
Select Cloud Provider:
  1. GCP (Google Cloud Platform) [Default]
Confirm usage of GCP? [Y/n]: y
Created 'blast-radius.yaml' with provider: gcp
Ready to use! Try running 'blast-radius impact'
```

### Existing Configuration

```
$ blast-radius init
Configuration file 'blast-radius.yaml' already exists.
Use existing file? [Y/n]: n
Overwrite existing file? [y/N]: y
Created 'blast-radius.yaml' with provider: gcp
```

## Generated Configuration

The `init` command creates this default configuration:

```yaml
cloud_provider: gcp

exclusions:
  - resource: example-ignored-project
    role: .*
  - role: roles/viewer

ignored_directories:
  - .terraform
  - modules
```

---

## Configuration File Schema

### Full Schema

```yaml
# Required: Cloud provider (currently only "gcp" is supported)
cloud_provider: gcp

# Optional: Exclusion rules to filter out bindings from output
exclusions:
  - resource: "<regex>"       # Match resource ID
    resource_type: "<regex>"  # Match resource type
    role: "<regex>"           # Match role

# Optional: Directories to skip when scanning for Terraform files
ignored_directories:
  - ".terraform"
  - "modules"
  - "examples"

# Optional: Accounts to analyze with the 'analyze' command
analysis_accounts:
  - alice@example.com
  - deploy-sa@project.iam.gserviceaccount.com
```

### Field Descriptions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `cloud_provider` | string | Yes | Cloud provider (`gcp`) |
| `exclusions` | array | No | Patterns to exclude from output |
| `exclusions[].resource` | string | No | Regex for resource ID |
| `exclusions[].resource_type` | string | No | Regex for Terraform resource type |
| `exclusions[].role` | string | No | Regex for IAM role |
| `ignored_directories` | array | No | Directory names to skip |
| `analysis_accounts` | array | No | Default accounts for `analyze` command |

---

## Exclusion Rules

Exclusion rules filter bindings from the `impact` command output. A binding is excluded if it matches **all specified fields** in a rule.

### Matching Logic

- Fields are matched using **regex patterns**
- If a field is empty/not specified, it matches **anything**
- A binding must match **all non-empty fields** to be excluded

### Examples

```yaml
exclusions:
  # Exclude all bindings on "example-ignored-project" regardless of role
  - resource: example-ignored-project

  # Exclude all roles/viewer bindings everywhere
  - role: roles/viewer

  # Exclude storage roles on test buckets
  - resource: test-.*-bucket
    role: roles/storage\..*

  # Exclude all bindings on BigQuery datasets
  - resource_type: google_bigquery_dataset_iam_member
```

### Pattern Examples

| Pattern | Matches |
|---------|---------|
| `roles/viewer` | Exactly `roles/viewer` |
| `roles/.*` | Any role |
| `roles/storage\..*` | Any storage role (escape the dot) |
| `prod-.*` | Resources starting with `prod-` |
| `.*-staging-.*` | Resources containing `-staging-` |

---

## Ignored Directories

The `ignored_directories` setting prevents blast-radius from scanning certain directories for Terraform files.

### Default Values

```yaml
ignored_directories:
  - .terraform   # Terraform provider cache
  - modules      # Common module directory
```

### Common Additions

```yaml
ignored_directories:
  - .terraform
  - modules
  - examples       # Example configurations
  - test           # Test fixtures
  - vendor         # Vendored modules
  - .git           # Git directory
```

---

## Analysis Accounts

The `analysis_accounts` setting provides default accounts for the `analyze` command when `--account` is not specified.

```yaml
analysis_accounts:
  # Users
  - alice@example.com
  - bob@example.com

  # Service accounts (just the email, not the full principal)
  - deploy-sa@project.iam.gserviceaccount.com
  - github-actions@project.iam.gserviceaccount.com
```

### Usage

```bash
# Uses accounts from config
blast-radius analyze ./terraform

# Override with --account flag
blast-radius analyze --account other@example.com ./terraform
```

---

## Examples

### Minimal Configuration

```yaml
cloud_provider: gcp
```

### Development Environment

```yaml
cloud_provider: gcp

# Ignore common low-risk roles
exclusions:
  - role: roles/viewer
  - role: roles/browser

ignored_directories:
  - .terraform
  - modules
  - examples
```

### Production Security Audit

```yaml
cloud_provider: gcp

# No exclusions - see everything
exclusions: []

# Check privileged accounts
analysis_accounts:
  - admin@example.com
  - ci-cd-sa@project.iam.gserviceaccount.com
  - terraform-sa@project.iam.gserviceaccount.com

ignored_directories:
  - .terraform
```

### Multi-Environment Setup

```yaml
cloud_provider: gcp

# Exclude test/dev resources from analysis
exclusions:
  - resource: .*-dev-.*
  - resource: .*-test-.*
  - resource: sandbox-.*

analysis_accounts:
  - ops-team@example.com
  - prod-deployer@project.iam.gserviceaccount.com

ignored_directories:
  - .terraform
  - modules
  - environments/dev
  - environments/test
```

---

## Using Custom Config Path

```bash
# Create config at custom path
blast-radius init --config production-config.yaml

# Use custom config with other commands
blast-radius impact --config production-config.yaml ./terraform
blast-radius analyze --config production-config.yaml ./terraform
```
