# Hierarchical Access Analysis

## Overview

This example demonstrates **hierarchical access detection** - identifying when project-level IAM roles grant access to ALL resources of a specific type within that project.

## What This Example Shows

- Project-level roles that grant broad access
- How one IAM binding can affect multiple resources
- The difference between direct resource access and hierarchical access

## Key Concept

When a principal has a role at the **project level**, they may have access to **ALL resources of certain types** in that project, even if there are no direct IAM bindings on those resources.

### Example

```hcl
# This single binding grants access to ALL BigQuery datasets in the project
resource "google_project_iam_member" "alice_bq_viewer" {
  project = "my-project"
  role    = "roles/bigquery.dataViewer"
  member  = "user:alice@example.com"
}
```

**Result:** Alice can view ALL BigQuery datasets in `my-project`, including:
- `analytics-dataset`
- `marketing-dataset`
- `sales-dataset`
- Any future datasets created in this project

## Hierarchical Roles Demonstrated

| Role | Grants Access To | Access Level |
|------|------------------|--------------|
| `roles/bigquery.dataViewer` | All BigQuery datasets | Read |
| `roles/bigquery.dataEditor` | All BigQuery datasets | Write |
| `roles/storage.objectViewer` | All Storage buckets | Read |
| `roles/storage.objectAdmin` | All Storage buckets | Admin |

## Running the Examples

### HCL Mode

```bash
cd examples/02-hierarchical-access/hcl
blast-radius hierarchy .
```

**Expected Output:**
```
--- Hierarchical Access Report ---

Project: var.project_id

Principal: user:alice@example.com
  Hierarchical Access:
    - ALL google_bigquery_dataset resources (roles/bigquery.dataViewer)

Principal: user:bob@example.com
  Hierarchical Access:
    - ALL google_storage_bucket resources (roles/storage.objectViewer)
```

### Plan Mode

```bash
cd examples/02-hierarchical-access/plan-mode
terraform init
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
blast-radius hierarchy --plan plan.json
```

**Expected Output:**
```
--- Hierarchical Access Report ---

Project: production-project

Principal: user:alice@example.com
  Hierarchical Access:
    - ALL google_bigquery_dataset resources (roles/bigquery.dataViewer)

Principal: user:bob@example.com
  Hierarchical Access:
    - ALL google_storage_bucket resources (roles/storage.objectViewer)
```

**Notice:** Plan mode shows the actual project ID (`production-project` instead of `var.project_id`)

## Why This Matters

### Security Implications

1. **Hidden Access** - Developers may not realize project-level roles grant broad access
2. **Least Privilege** - Project-level roles often violate least privilege principle
3. **Audit Complexity** - Hard to track who has access to what without hierarchical analysis

### Real-World Scenario

```hcl
# Looks innocent: giving Alice viewer access to the project
resource "google_project_iam_member" "alice_viewer" {
  project = "production"
  role    = "roles/bigquery.dataViewer"
  member  = "user:alice@example.com"
}

# But this means Alice can now view:
# - customer_data dataset (PII)
# - financial_records dataset (sensitive)
# - analytics_staging dataset (okay)
# - ALL future BigQuery datasets created in this project
```

## HCL Mode vs Plan Mode

### HCL Mode
- Shows hierarchical access patterns
- Variables may be unresolved
- Good for understanding the pattern

### Plan Mode
- Shows exact project IDs
- More accurate for production analysis
- Better for compliance reporting

## Key Takeaways

1. **Project-level roles are powerful** - One binding can grant access to many resources
2. **Use `hierarchy` command** - Specifically designed to detect this pattern
3. **Prefer resource-specific bindings** - More granular control
4. **Plan mode is clearer** - Shows actual project names, not variables

## Next Steps

- Try [`03-impersonation-chains/`](../03-impersonation-chains/) to see service account impersonation
- Try [`05-policy-role-restrictions/`](../05-policy-role-restrictions/) to enforce policies against broad access
