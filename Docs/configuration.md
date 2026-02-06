# Configuration Reference

This document describes the complete schema for the `blast-radius.yaml` configuration file.

## File Location

By default, blast-radius looks for `blast-radius.yaml` in the current directory. Override with `--config`:

```bash
blast-radius impact --config /path/to/config.yaml
```

## Complete Schema

```yaml
# Required: Cloud provider identifier
# Currently only "gcp" is supported
cloud_provider: gcp

# Optional: Rules to exclude bindings from output
# Exclusions apply to the 'impact' command
exclusions:
  - resource: "<regex>"       # Match against resource ID
    resource_type: "<regex>"  # Match against Terraform resource type
    role: "<regex>"           # Match against IAM role

# Optional: Directory names to skip when scanning for .tf files
ignored_directories:
  - "<directory_name>"

# Optional: Default accounts for the 'analyze' command
# Used when --account flag is not provided
analysis_accounts:
  - "<email>"
```

---

## Field Reference

### cloud_provider

**Type:** `string`
**Required:** Yes
**Values:** `gcp`

The cloud provider for this configuration. Currently only Google Cloud Platform is supported.

```yaml
cloud_provider: gcp
```

---

### exclusions

**Type:** `array[ExclusionRule]`
**Required:** No
**Default:** `[]`

A list of rules to exclude IAM bindings from the `impact` command output. Useful for filtering out noise like viewer roles or known-safe resources.

#### ExclusionRule Schema

| Field | Type | Description |
|-------|------|-------------|
| `resource` | string | Regex pattern to match resource ID |
| `resource_type` | string | Regex pattern to match Terraform resource type |
| `role` | string | Regex pattern to match IAM role |

#### Matching Behavior

- All specified fields must match for a binding to be excluded
- Empty/unspecified fields match anything
- Patterns are **Go regex** (similar to Perl/PCRE)

#### Examples

```yaml
exclusions:
  # Exclude all viewer roles everywhere
  - role: roles/viewer

  # Exclude any role on resources matching pattern
  - resource: ".*-sandbox-.*"

  # Exclude specific role on specific resource type
  - resource_type: google_project_iam_member
    role: roles/browser

  # Exclude storage viewer on test buckets
  - resource: "test-.*"
    role: roles/storage\.objectViewer

  # Exclude all BigQuery dataset bindings
  - resource_type: google_bigquery_dataset_iam_member
```

#### Regex Tips

| Pattern | Matches |
|---------|---------|
| `.*` | Anything |
| `roles/viewer` | Exactly "roles/viewer" |
| `roles/storage\..*` | Any storage role (escape the dot) |
| `^prod-` | Starts with "prod-" |
| `-staging$` | Ends with "-staging" |
| `(dev\|test)` | Contains "dev" or "test" |

---

### ignored_directories

**Type:** `array[string]`
**Required:** No
**Default:** `[]`

Directory names to skip when recursively scanning for Terraform files. Matched against directory basename, not full path.

```yaml
ignored_directories:
  - .terraform      # Terraform provider cache
  - modules         # Shared modules
  - examples        # Example configurations
  - test            # Test fixtures
  - vendor          # Vendored dependencies
  - node_modules    # If mixed with other tools
```

**Note:** This affects HCL parsing mode only. Plan mode parses a single JSON file.

---

### analysis_accounts

**Type:** `array[string]`
**Required:** No
**Default:** `[]`

Email addresses of accounts to analyze with the `analyze` command when `--account` is not provided.

```yaml
analysis_accounts:
  # User accounts
  - alice@example.com
  - bob@example.com

  # Service accounts (email only, not full principal)
  - my-sa@project.iam.gserviceaccount.com
  - github-actions@project.iam.gserviceaccount.com
```

**Note:** Use email addresses only, not full principal identifiers. The tool will automatically resolve `alice@example.com` to `user:alice@example.com`.

---

## Example Configurations

### Minimal

```yaml
cloud_provider: gcp
```

### Development Environment

```yaml
cloud_provider: gcp

exclusions:
  # Ignore low-risk viewer roles
  - role: roles/viewer
  - role: roles/browser

  # Ignore sandbox/test resources
  - resource: ".*sandbox.*"
  - resource: ".*test.*"

ignored_directories:
  - .terraform
  - modules
  - examples
```

### Security Audit

```yaml
cloud_provider: gcp

# No exclusions - see everything
exclusions: []

# Key accounts to check
analysis_accounts:
  - admin@example.com
  - break-glass@example.com
  - terraform-automation@project.iam.gserviceaccount.com
  - ci-cd-pipeline@project.iam.gserviceaccount.com

ignored_directories:
  - .terraform
```

### Production Focus

```yaml
cloud_provider: gcp

exclusions:
  # Filter out dev/test from analysis
  - resource: ".*-dev-.*"
  - resource: ".*-test-.*"
  - resource: ".*-staging-.*"

  # Ignore viewer roles on non-sensitive resources
  - resource_type: google_project_iam_member
    role: roles/viewer

analysis_accounts:
  - sre-team@example.com
  - prod-deployer@project.iam.gserviceaccount.com

ignored_directories:
  - .terraform
  - modules
  - environments/dev
  - environments/staging
```

### Multi-Team Setup

```yaml
cloud_provider: gcp

exclusions:
  # Each team manages their own viewer bindings
  - role: roles/viewer
  - role: roles/browser

  # Ignore shared infrastructure
  - resource: "shared-.*"

# Key service accounts across teams
analysis_accounts:
  - team-a-deployer@project-a.iam.gserviceaccount.com
  - team-b-deployer@project-b.iam.gserviceaccount.com
  - platform-automation@platform.iam.gserviceaccount.com

ignored_directories:
  - .terraform
  - modules
  - vendor
```

---

## Creating Configuration

Use `blast-radius init` to create a default configuration:

```bash
blast-radius init
```

This creates:

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

## Validation

The configuration file is validated when loaded. Common errors:

| Error | Cause |
|-------|-------|
| `invalid cloud_provider` | Provider is not "gcp" |
| `invalid regex pattern` | Malformed regex in exclusion rule |
| `yaml parse error` | Invalid YAML syntax |

Test your configuration:

```bash
blast-radius impact --config myconfig.yaml .
```
