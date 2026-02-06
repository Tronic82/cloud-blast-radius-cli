# blast-radius validate

## Summary

The `validate` command checks IAM configurations against **organizational policies**. It enforces security rules such as:
- Restricting which roles certain principals can have
- Preventing forbidden role assignments
- Enforcing separation of duties
- Detecting privilege escalation through impersonation

Use this command in CI/CD pipelines to prevent non-compliant IAM changes from being deployed.

## Usage

```bash
blast-radius validate [directory] [flags]
```

### Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `directory` | Path to directory containing Terraform files | Current directory (`.`) |

### Flags

| Flag | Description |
|------|-------------|
| `--policy <path>` | **Required.** Path to policy YAML file |
| `--plan <path>` | Path to Terraform plan JSON file |
| `--tfvars <path>` | Path to terraform.tfvars file for variable resolution |
| `--strict` | Treat warnings as errors (exit code 1 for any violation) |
| `--config <path>` | Path to blast-radius.yaml configuration file (default: `blast-radius.yaml`) |
| `--output <format>` | Output format: `text` or `json` (default: `text`) |
| `--definitions <path>` | Path to custom resource definitions file |
| `--rules <path>` | Path to custom validation rules file |

## Text Output

### Example Output

```
Validating directory: ./terraform

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Policy Validation Report
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Policies Evaluated: 3
Violations Found: 2
Compliant: 1

❌ : Developer Role Restrictions
   Violation: FORBIDDEN ROLE

   Principal: group:developers@example.com
   Resource: my-project
   Role: roles/editor

   Principal has forbidden role
   Remediation: Remove role binding or update policy

✅ COMPLIANT: Production Owner Restrictions

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Status: FAILED ❌

Fix the errors above to achieve compliance.
```

### How to Read the Output

1. **Header Statistics**:
   - `Policies Evaluated`: Total number of policies checked
   - `Violations Found`: Number of policy violations
   - `Compliant`: Number of policies with no violations

2. **Violation Blocks** (marked with ❌):
   - **Policy Name**: Which policy was violated
   - **Violation Type**: Category of the violation (e.g., FORBIDDEN ROLE)
   - **Principal**: Who has the problematic binding
   - **Resource**: Which resource is affected
   - **Role**: The specific role that caused the violation
   - **Message**: Human-readable explanation
   - **Remediation**: Suggested fix

3. **Compliant Policies** (marked with ✅):
   - Policies that passed all checks

4. **Final Status**:
   - `PASSED ✅` - All policies compliant
   - `FAILED ❌` - One or more violations found

### Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | All policies passed |
| 1 | One or more violations with `error` severity |
| 1 | (with `--strict`) Any violation including warnings |

## JSON Output

Use `--output json` for machine-readable output.

### Schema

```json
{
  "command": "validate",
  "timestamp": "2024-01-15T10:30:00Z",
  "status": "failed",
  "violations": [
    {
      "policy": "Developer Role Restrictions",
      "severity": "error",
      "principal": "group:developers@example.com",
      "resource": "my-project",
      "role": "roles/editor",
      "message": "Principal has forbidden role"
    }
  ]
}
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Always `"validate"` |
| `timestamp` | string | ISO 8601 timestamp |
| `status` | string | `"passed"` or `"failed"` |
| `violations` | array | List of policy violations |
| `violations[].policy` | string | Name of the violated policy |
| `violations[].severity` | string | `error`, `warning`, or `info` |
| `violations[].principal` | string | Principal identifier |
| `violations[].resource` | string | Resource identifier |
| `violations[].role` | string | IAM role |
| `violations[].message` | string | Human-readable description |

---

## Policy File Schema

The `--policy` flag requires a YAML file defining your organizational policies.

### Basic Structure

```yaml
cloud_provider: gcp

policies:
  - name: "Policy Name"
    type: <policy_type>
    description: "Optional description"
    severity: error  # error, warning, or info
    <type_specific_config>
```

### Pattern Matching (Regex)

All pattern fields (`principal_pattern`, `resource_pattern`, `role_pattern`, etc.) use **regular expressions** (regex). The special pattern `"*"` matches everything.

#### Common Regex Patterns

| Pattern | Matches | Description |
|---------|---------|-------------|
| `*` | Everything | Special wildcard |
| `^user:.*` | `user:alice@example.com` | All users |
| `^serviceAccount:.*` | `serviceAccount:sa@project.iam.gserviceaccount.com` | All service accounts |
| `^group:.*@example\.com$` | `group:developers@example.com` | Groups in example.com domain |
| `^roles/owner$` | `roles/owner` | Exact match for owner role |
| `^roles/storage\..*` | `roles/storage.admin`, `roles/storage.objectViewer` | All storage roles |
| `^prod-.*` | `prod-project`, `prod-database` | Resources starting with prod- |
| `.*-staging$` | `app-staging`, `db-staging` | Resources ending with -staging |

#### Regex Tips

- Use `^` to anchor at the start of the string
- Use `$` to anchor at the end of the string
- Escape special characters with `\` (e.g., `\.` for a literal dot)
- Use `.*` to match any characters
- Use `[a-z]+` for one or more lowercase letters
- If your pattern is invalid regex, it falls back to exact string matching

### Policy Types

#### 1. Role Restriction (`role_restriction`)

Restrict which roles principals can have.

```yaml
policies:
  - name: "Developer Role Restrictions"
    type: role_restriction
    severity: error
    role_restriction:
      selector:
        principal_pattern: "^group:developers@.*"  # Regex: starts with group:developers@
      allowed_roles:
        - "roles/viewer"
        - "roles/bigquery.dataViewer"
      denied_roles:
        - "roles/owner"
        - "roles/editor"
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `selector.principal_pattern` | string | Regex pattern to match principals (e.g., `^user:.*@example\.com$`) |
| `allowed_roles` | array | Only these roles are permitted (if specified) |
| `denied_roles` | array | These roles are forbidden |

---

#### 2. Persona (`persona`)

Define expected access for specific personas/identities.

```yaml
policies:
  - name: "Developer Persona"
    type: persona
    severity: warning
    persona:
      persona_name: "Developer"
      principals:
        - "group:developers@example.com"
      required_bindings:
        - resource_pattern: "dev-project"
          role: "roles/viewer"
      forbidden_bindings:
        - resource_pattern: "^prod-.*"
          role: ".*"  # No access to prod (any role)
      allow_additional_access: true
      validate_transitive_access: true
      transitive_constraints:
        max_impersonation_depth: 2
        forbidden_transitive_roles:
          - "roles/owner"
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `persona_name` | string | Name for this persona |
| `principals` | array | Principals belonging to this persona |
| `required_bindings` | array | Bindings that must exist |
| `forbidden_bindings` | array | Bindings that must not exist |
| `allow_additional_access` | boolean | Allow bindings beyond required |
| `validate_transitive_access` | boolean | Check impersonation chains |
| `transitive_constraints` | object | Limits on transitive access |

---

#### 3. Resource Access (`resource_access`)

Control who can access specific resources.

```yaml
policies:
  - name: "Production Database Access"
    type: resource_access
    severity: error
    resource_access:
      selector:
        resource_pattern: "^prod-.*"  # Regex: resources starting with prod-
        resource_type: "google_sql_database_instance"
      allowed_principals:
        - "group:dba@example.com"
        - "serviceAccount:backup-sa@project.iam.gserviceaccount.com"
      validate_effective_access: true
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `selector.resource_pattern` | string | Regex to match resource IDs |
| `selector.resource_type` | string | Terraform resource type |
| `allowed_principals` | array | Only these principals can access |
| `allowed_roles_per_principal` | object | Map of principal to allowed roles |
| `validate_effective_access` | boolean | Check transitive access too |

---

#### 4. Separation of Duty (`separation_of_duty`)

Prevent conflicting role combinations.

```yaml
policies:
  - name: "Approver/Deployer Separation"
    type: separation_of_duty
    severity: error
    separation_of_duty:
      conflicting_roles:
        - ["roles/iam.securityAdmin", "roles/owner"]
        - ["roles/cloudbuild.builds.editor", "roles/iam.serviceAccountUser"]
      scope: "per_principal"  # or "per_resource"
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `conflicting_roles` | array | Arrays of roles that cannot coexist |
| `scope` | string | `per_principal` or `per_resource` |

---

#### 5. Impersonation Escalation (`impersonation_escalation`)

Prevent privilege escalation through impersonation.

```yaml
policies:
  - name: "Prevent Escalation to Owner"
    type: impersonation_escalation
    severity: error
    impersonation_escalation:
      forbidden_escalations:
        - from_role_pattern: "^roles/viewer$"
          to_role_pattern: "^roles/owner$"
          via: impersonation
        - from_principal_pattern: "^group:developers@.*"
          to_resource_pattern: "^prod-.*"
          via: impersonation
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `forbidden_escalations` | array | Escalation patterns to block |
| `forbidden_escalations[].from_role_pattern` | string | Starting role regex |
| `forbidden_escalations[].to_role_pattern` | string | Target role regex |
| `forbidden_escalations[].from_principal_pattern` | string | Starting principal regex |
| `forbidden_escalations[].to_principal_pattern` | string | Target principal regex |
| `forbidden_escalations[].to_resource_pattern` | string | Target resource regex |
| `forbidden_escalations[].via` | string | Currently only `impersonation` |

---

#### 6. Effective Access (`effective_access`)

Validate who has effective access (including transitive) to resources.

```yaml
policies:
  - name: "Production Secrets Access"
    type: effective_access
    severity: error
    effective_access:
      selector:
        resource_pattern: "^prod-secrets-.*"  # Regex: resources starting with prod-secrets-
        resource_type: "google_secret_manager_secret"
      validate_effective_access: true
      allowed_effective_principals:
        - "serviceAccount:prod-app@project.iam.gserviceaccount.com"
      forbidden_effective_principals:
        - "^user:.*"  # No users should have direct access
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `selector.resource_pattern` | string | Regex for resource IDs |
| `selector.resource_type` | string | Terraform resource type |
| `validate_effective_access` | boolean | Include transitive access |
| `allowed_effective_principals` | array | Principals allowed effective access |
| `forbidden_effective_principals` | array | Principals forbidden from effective access |

---

## Complete Policy Example

```yaml
cloud_provider: gcp

policies:
  # Restrict developer group roles
  - name: "Developer Role Restrictions"
    type: role_restriction
    severity: error
    description: "Developers should only have viewer access"
    role_restriction:
      selector:
        principal_pattern: "^group:developers@.*"  # Regex: starts with group:developers@
      allowed_roles:
        - "roles/viewer"
        - "roles/bigquery.dataViewer"
        - "roles/storage.objectViewer"
      denied_roles:
        - "roles/owner"
        - "roles/editor"
        - "roles/iam.serviceAccountAdmin"

  # Service accounts should never be owners
  - name: "Service Account Role Restrictions"
    type: role_restriction
    severity: error
    role_restriction:
      selector:
        principal_pattern: "^serviceAccount:.*"  # Regex: all service accounts
      denied_roles:
        - "roles/owner"

  # Prevent privilege escalation
  - name: "Block Escalation to Owner"
    type: impersonation_escalation
    severity: error
    impersonation_escalation:
      forbidden_escalations:
        - from_role_pattern: "^roles/viewer$"  # Exact match for roles/viewer
          to_role_pattern: "^roles/owner$"     # Exact match for roles/owner
          via: impersonation
```

---

## Examples

### Basic Validation

```bash
# Validate against policy file
blast-radius validate --policy policy.yaml ./terraform
```

### CI/CD Integration

```bash
# Fail on any violation (strict mode)
blast-radius validate --policy policy.yaml --strict ./terraform
if [ $? -ne 0 ]; then
  echo "IAM policy violation detected!"
  exit 1
fi
```

### Using Terraform Plan

```bash
# Validate planned changes before apply
terraform plan -out=tfplan
terraform show -json tfplan > plan.json
blast-radius validate --policy policy.yaml --plan plan.json
```

### JSON Output for Reporting

```bash
# Generate compliance report
blast-radius validate --policy policy.yaml --output json > compliance-report.json

# Count violations by severity
cat compliance-report.json | jq '[.violations[].severity] | group_by(.) | map({(.[0]): length}) | add'
```

### Multiple Policy Files

Currently, blast-radius supports one policy file. For multiple policies, combine them into a single file:

```yaml
cloud_provider: gcp

policies:
  # Include all your policies here
  - name: "Policy 1"
    # ...
  - name: "Policy 2"
    # ...
```
