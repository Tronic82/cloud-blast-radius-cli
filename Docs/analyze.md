# blast-radius analyze

## Summary

The `analyze` command performs **deep transitive access analysis** for specific accounts. It traces **impersonation chains** to discover what a principal can **effectively** access through service account impersonation.

This is crucial for security assessments because a user with minimal direct permissions might have broad effective access through a chain of service account impersonations.

## Usage

```bash
blast-radius analyze [directory] [flags]
```

### Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `directory` | Path to directory containing Terraform files | Current directory (`.`) |

### Flags

| Flag | Description |
|------|-------------|
| `--account <email>` | **Required.** Account(s) to analyze (can be specified multiple times or comma-separated) |
| `--plan <path>` | Path to Terraform plan JSON file |
| `--tfvars <path>` | Path to terraform.tfvars file for variable resolution |
| `--config <path>` | Path to blast-radius.yaml configuration file (default: `blast-radius.yaml`) |
| `--output <format>` | Output format: `text` or `json` (default: `text`) |
| `--definitions <path>` | Path to custom resource definitions file |
| `--rules <path>` | Path to custom validation rules file |

## Understanding Impersonation Chains

In GCP, principals can impersonate service accounts if they have:
- `roles/iam.serviceAccountUser` - Act as the service account
- `roles/iam.serviceAccountTokenCreator` - Generate access tokens
- `roles/iam.workloadIdentityUser` - Workload identity federation

This creates **transitive access**: if Alice can impersonate SA-1, and SA-1 can impersonate SA-2, and SA-2 has `roles/owner` on a project, then Alice **effectively** has owner access.

```
Alice (user) → SA-1 (impersonate) → SA-2 (impersonate) → Project (roles/owner)
```

The `analyze` command traces these chains to show the complete picture.

## Text Output

### Example Output

```
Analyzing directory: ./terraform

--- Transitive Access Analysis ---

=== Analyzing: alice@example.com ===

Principal: user:alice@example.com

Direct Access:
  - projects/my-project/serviceAccounts/deploy-sa@project.iam.gserviceaccount.com (google_service_account_iam_member):
      roles/iam.serviceAccountUser

Hierarchical Access:
  - All resources in project 'dev-project'

Effective Grants (via impersonation):
  - production-secrets (google_storage_bucket_iam_member):
      [EFFECTIVE] roles/storage.admin
    → via chain: serviceAccount:deploy-sa@project.iam.gserviceaccount.com → serviceAccount:admin-sa@project.iam.gserviceaccount.com
  - sensitive-customer-data (google_bigquery_dataset_iam_member):
      [EFFECTIVE] roles/bigquery.admin
    → via chain: serviceAccount:deploy-sa@project.iam.gserviceaccount.com → serviceAccount:admin-sa@project.iam.gserviceaccount.com
```

### How to Read the Output

1. **Account Header**: Shows which account is being analyzed
   - `=== Analyzing: alice@example.com ===`

2. **Principal**: The fully-qualified principal identifier
   - If you provided `alice@example.com`, it resolved to `user:alice@example.com`

3. **Direct Access**: Resources the principal can access directly (without impersonation)
   - Shows resource ID, Terraform type, and roles
   - This includes impersonation permissions themselves

4. **Hierarchical Access**: Projects/folders/orgs where the principal has inherited access
   - "All resources in project 'X'" means they have project-level permissions

5. **Effective Grants (via impersonation)**: Resources accessible through impersonation chains
   - `[EFFECTIVE]` prefix (cyan) highlights these are transitive, not direct
   - Shows the impersonation chain that enables access:
     - `→ via chain: SA-1 → SA-2 → SA-3`
   - Each hop in the chain is a service account that can be impersonated

### Color Coding

- **Cyan bold**: Section headers
- **White bold**: Principal labels
- **Cyan**: `[EFFECTIVE]` prefix for transitive access

### Special Cases

- **No matching principal**: The email wasn't found in any IAM bindings
  ```
  No matching principal found for account: bob@example.com
  ```

- **No direct access**: Principal only has access through impersonation
  ```
  Direct Access: None
  ```

- **No transitive access**: Principal cannot impersonate anyone
  ```
  Effective Grants (via impersonation): None
  ```

## JSON Output

Use `--output json` for machine-readable output.

### Schema

```json
{
  "command": "analyze",
  "timestamp": "2024-01-15T10:30:00Z",
  "account": "alice@example.com",
  "direct_access": [
    {
      "resource_id": "projects/my-project/serviceAccounts/deploy-sa@project.iam.gserviceaccount.com",
      "resource_type": "google_service_account_iam_member",
      "roles": ["roles/iam.serviceAccountUser"],
      "terraform_addresses": {
        "roles/iam.serviceAccountUser": "google_service_account_iam_member.alice_deploy"
      }
    }
  ],
  "hierarchical_access": [
    {
      "principal": "user:alice@example.com",
      "project": "dev-project",
      "resource_types": ["google_bigquery_dataset", "google_storage_bucket"]
    }
  ],
  "transitive_access": [
    {
      "resource_id": "production-secrets",
      "resource_type": "google_storage_bucket_iam_member",
      "roles": ["roles/storage.admin"],
      "via_chain": [
        "serviceAccount:deploy-sa@project.iam.gserviceaccount.com",
        "serviceAccount:admin-sa@project.iam.gserviceaccount.com"
      ],
      "terraform_addresses": {
        "roles/storage.admin": "google_storage_bucket_iam_member.admin_storage"
      }
    }
  ]
}
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Always `"analyze"` |
| `timestamp` | string | ISO 8601 timestamp |
| `account` | string | The email that was analyzed |
| `direct_access` | array | Resources directly accessible |
| `direct_access[].resource_id` | string | Resource identifier |
| `direct_access[].resource_type` | string | Terraform resource type |
| `direct_access[].roles` | array | IAM roles on this resource |
| `direct_access[].terraform_addresses` | object | (Optional) Role to Terraform address mapping |
| `hierarchical_access` | array | Project-level inherited access |
| `hierarchical_access[].principal` | string | Full principal identifier |
| `hierarchical_access[].project` | string | Project ID with inherited access |
| `hierarchical_access[].resource_types` | array | Resource types accessible in this project |
| `transitive_access` | array | Access gained through impersonation |
| `transitive_access[].resource_id` | string | Resource identifier |
| `transitive_access[].resource_type` | string | Terraform resource type |
| `transitive_access[].roles` | array | Effective roles on this resource |
| `transitive_access[].via_chain` | array | Ordered list of service accounts in the impersonation chain |
| `transitive_access[].terraform_addresses` | object | (Optional) Role to Terraform address mapping |

## Configuration File

You can specify accounts to analyze in `blast-radius.yaml`:

```yaml
cloud_provider: gcp

# Accounts to analyze when --account is not specified
analysis_accounts:
  - alice@example.com
  - bob@example.com
  - deploy-sa@project.iam.gserviceaccount.com
```

## Examples

### Basic Usage

```bash
# Analyze a single user
blast-radius analyze --account alice@example.com ./terraform

# Analyze multiple accounts
blast-radius analyze --account alice@example.com --account bob@example.com ./terraform

# Or comma-separated
blast-radius analyze --account alice@example.com,bob@example.com ./terraform
```

### Using Config for Accounts

```bash
# With analysis_accounts in blast-radius.yaml
blast-radius analyze ./terraform
```

### Using Terraform Plan

```bash
blast-radius analyze --account deploy-sa@project.iam.gserviceaccount.com --plan plan.json
```

### JSON Output for Security Audits

```bash
# Find all accounts with effective admin access
blast-radius analyze --account alice@example.com --output json | jq '
  .transitive_access[] |
  select(.roles[] | contains("admin")) |
  {resource: .resource_id, role: .roles[], chain_length: (.via_chain | length)}
'

# Get the deepest impersonation chain
blast-radius analyze --output json | jq '.transitive_access | max_by(.via_chain | length) | .via_chain'
```

### Analyzing Service Accounts

```bash
# Check what a CI/CD service account can reach
blast-radius analyze --account github-actions-sa@project.iam.gserviceaccount.com ./terraform
```

## Security Implications

The `analyze` command helps identify:

1. **Privilege Escalation Paths**: Users with seemingly limited access who can escalate through impersonation

2. **Excessive Impersonation Rights**: Service accounts that can be impersonated by too many principals

3. **Circular Impersonation**: The tool detects and prevents infinite loops in circular chains

4. **Hidden Admin Access**: Effective access to sensitive resources that isn't visible from direct bindings

## Impersonation Roles

The following roles enable impersonation:
- `roles/iam.serviceAccountUser` - Run operations as the service account
- `roles/iam.serviceAccountTokenCreator` - Generate OAuth2/OIDC tokens
- `roles/iam.workloadIdentityUser` - Workload identity for external identities

The tool uses BFS (breadth-first search) to efficiently trace all reachable service accounts from a starting principal.
