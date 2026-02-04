# Impersonation Chains

## Overview

This example demonstrates **service account impersonation chain analysis** - the unique capability that sets blast-radius apart from other IAM analysis tools.

## What This Example Shows

- Simple impersonation chains (A → B)
- Multi-hop impersonation chains (A → B → C)
- How impersonation enables privilege escalation
- Transitive access through impersonation

## Key Concept: Impersonation

**Impersonation** allows one principal to act as another service account. This is done through roles like:
- `roles/iam.serviceAccountUser`
- `roles/iam.serviceAccountTokenCreator`

### Why This Matters

A service account with minimal direct permissions can gain powerful access by impersonating other service accounts.

## Examples in This Folder

### 1. Simple Chain (2-hop)

```
dev-sa → can impersonate → prod-sa
```

- `dev-sa` has only viewer access directly
- `dev-sa` can impersonate `prod-sa`
- `prod-sa` has owner access to production
- **Result:** `dev-sa` effectively has owner access via impersonation

### 2. Multi-Hop Chain (3-hop)

```
user:alice → impersonates → deploy-sa
                          → impersonates → admin-sa
```

- Alice has no direct resource access
- Alice can impersonate `deploy-sa`
- `deploy-sa` can impersonate `admin-sa`
- `admin-sa` has admin access to sensitive data
- **Result:** Alice can access sensitive data through 2 hops of impersonation

## Running the Examples

### HCL Mode

```bash
cd examples/03-impersonation-chains/hcl

# Analyze impersonation chains
blast-radius analyze . --account serviceAccount:dev-sa@project.iam.gserviceaccount.com
```

**Expected Output:**
```
Analyzing account: serviceAccount:dev-sa@project.iam.gserviceaccount.com

Direct Access:
  - dev-project (google_project_iam_member):
    - roles/viewer

Transitive Access (via impersonation):
  - prod-project (google_project_iam_member):
    - roles/owner
    → via: serviceAccount:prod-sa@project.iam.gserviceaccount.com

Maximum Impersonation Depth: 1 hop
Privilege Escalation: Yes (viewer → owner)
```

### Plan Mode

```bash
cd examples/03-impersonation-chains/plan-mode
terraform init
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json

blast-radius analyze --plan plan.json --account serviceAccount:dev-sa@production-project.iam.gserviceaccount.com
```

**Expected Output:**
```
Analyzing account: serviceAccount:dev-sa@production-project.iam.gserviceaccount.com

Direct Access:
  - dev-project (google_project_iam_member):
    - roles/viewer

Transitive Access (via impersonation):
  - production-project (google_project_iam_member):
    - roles/owner
    → via: serviceAccount:prod-sa@production-project.iam.gserviceaccount.com

Maximum Impersonation Depth: 1 hop
Privilege Escalation: Yes (viewer → owner)
```

## Real-World Security Scenario

### The Hidden Danger

```hcl
# Looks safe: dev-sa only has viewer on dev resources
resource "google_project_iam_member" "dev_sa_viewer" {
  project = "dev-project"
  role    = "roles/viewer"
  member  = "serviceAccount:dev-sa@project.iam.gserviceaccount.com"
}

# Hidden danger: dev-sa can impersonate prod-sa
resource "google_service_account_iam_member" "dev_can_impersonate_prod" {
  service_account_id = "prod-sa@project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:dev-sa@project.iam.gserviceaccount.com"
}

# prod-sa has owner on production
resource "google_project_iam_member" "prod_sa_owner" {
  project = "prod-project"
  role    = "roles/owner"
  member  = "serviceAccount:prod-sa@project.iam.gserviceaccount.com"
}

# RESULT: dev-sa effectively has owner on prod via impersonation!
# Traditional tools miss this. Blast Radius detects it.
```

## Why Blast Radius is Unique

**Other tools (Checkov, tfsec, OPA):**
- ❌ Don't analyze impersonation chains
- ❌ Don't calculate transitive access
- ❌ Can't show the full blast radius

**Blast Radius:**
- ✅ Detects multi-hop impersonation chains
- ✅ Calculates transitive access
- ✅ Shows complete blast radius including impersonation paths
- ✅ Detects privilege escalation via impersonation

## HCL Mode vs Plan Mode

### HCL Mode
- Detects impersonation patterns
- May have unresolved service account IDs
- Good for understanding the structure

### Plan Mode
- Fully resolved service account emails
- Exact impersonation paths
- Better for production analysis
- Recommended for security audits

## Key Takeaways

1. **Impersonation is powerful** - Can grant indirect access to sensitive resources
2. **Multi-hop chains are dangerous** - Escalation through multiple service accounts
3. **Traditional tools miss this** - Blast Radius is unique in detecting this
4. **Use `analyze` command** - Specifically designed for impersonation analysis
5. **Plan mode is clearer** - Shows exact service account identities

## Next Steps

- Try [`04-transitive-access/`](../04-transitive-access/) for complete blast radius analysis
- Try [`07-policy-privilege-escalation/`](../07-policy-privilege-escalation/) to enforce policies against escalation
