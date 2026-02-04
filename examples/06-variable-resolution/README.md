# Variable Resolution

## Overview

This example demonstrates the **critical difference** between HCL mode and plan mode when handling Terraform variables.

## The Problem: Unresolved Variables in HCL Mode

HCL mode parses `.tf` files directly but cannot resolve variables to their actual values. This leads to:
- Resource IDs showing as `var.project_id` instead of actual project names
- Service account emails incomplete
- Unclear blast radius analysis

## The Solution: Plan Mode

Plan mode uses `terraform show -json` which contains all variables resolved to their actual values.

## Side-by-Side Comparison

### Configuration

```hcl
variable "project_id" {
  type = string
}

variable "environment" {
  type = string
}

resource "google_project_iam_member" "viewer" {
  project = var.project_id
  role    = "roles/viewer"
  member  = "user:alice@example.com"
}

resource "google_storage_bucket_iam_member" "bucket_access" {
  bucket = "${var.environment}-data-bucket"
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:app-sa@${var.project_id}.iam.gserviceaccount.com"
}
```

### HCL Mode Output

```
Principal: user:alice@example.com
  Resources (1):
    - var.project_id (google_project_iam_member):
      - roles/viewer

Principal: serviceAccount:app-sa@${var.project_id}.iam.gserviceaccount.com
  Resources (1):
    - ${var.environment}-data-bucket (google_storage_bucket_iam_member):
      - roles/storage.objectViewer
```

❌ **Problems:**
- Can't tell which actual project
- Service account email is incomplete
- Bucket name is unclear

### Plan Mode Output

```
Principal: user:alice@example.com
  Resources (1):
    - production-project (google_project_iam_member):
      - roles/viewer

Principal: serviceAccount:app-sa@production-project.iam.gserviceaccount.com
  Resources (1):
    - prod-data-bucket (google_storage_bucket_iam_member):
      - roles/storage.objectViewer
```

✅ **Benefits:**
- Clear project name: `production-project`
- Complete service account: `app-sa@production-project.iam.gserviceaccount.com`
- Resolved bucket name: `prod-data-bucket`

## Running the Examples

### HCL Mode (with tfvars)

```bash
cd examples/10-variable-resolution/hcl
blast-radius impact . --tfvars terraform.tfvars
```

**Note:** Even with `--tfvars`, HCL mode has limited variable resolution

### Plan Mode (Full Resolution)

```bash
cd examples/10-variable-resolution/plan-mode
terraform init
terraform plan -out=plan.tfplan -var-file=terraform.tfvars
terraform show -json plan.tfplan > plan.json
blast-radius impact --plan plan.json
```

## When Plan Mode is Critical

- ✅ **Production deployments** - Need exact resource IDs
- ✅ **Compliance reporting** - Must show actual values
- ✅ **Security audits** - Accuracy is critical
- ✅ **Complex variable usage** - Multiple variables, interpolations
- ✅ **CI/CD pipelines** - Validate before apply

## Key Takeaway

**For production analysis, plan mode is essential for accurate variable resolution.**
