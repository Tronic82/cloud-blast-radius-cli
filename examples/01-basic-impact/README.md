# Basic Impact Analysis

## Overview

This example demonstrates the most fundamental feature of blast-radius: analyzing **direct IAM access** to cloud resources.

## What This Example Shows

- Direct IAM bindings on various GCP resources
- Principal → Resource → Role mapping
- Basic blast radius calculation (who has access to what)

## Resources Demonstrated

1. **Project IAM** - Project-level access
2. **BigQuery Dataset IAM** - Dataset-level access
3. **Storage Bucket IAM** - Bucket-level access
4. **Service Account IAM** - Service account access

## HCL Mode vs Plan Mode

### HCL Mode

**Pros:**
- Fast analysis
- No terraform plan required
- Good for quick local checks

**Cons:**
- Variables remain unresolved (e.g., `var.project_id`)
- May show incomplete resource IDs

### Plan Mode

**Pros:**
- ✅ All variables resolved to actual values
- ✅ Exact resource IDs shown
- ✅ Accurate representation of what will be deployed

**Cons:**
- Requires terraform plan step

## Running the Examples

### HCL Mode

```bash
cd examples/01-basic-impact/hcl

# Analyze with HCL parsing
blast-radius impact .
```

**Expected Output:**
```
Principal: user:alice@example.com
  Resources (2):
    - var.project_id (google_project_iam_member):
      - roles/viewer
    - analytics-dataset (google_bigquery_dataset_iam_member):
      - roles/bigquery.dataViewer

Principal: serviceAccount:backup-sa@project.iam.gserviceaccount.com
  Resources (1):
    - data-bucket (google_storage_bucket_iam_member):
      - roles/storage.objectViewer
```

### Plan Mode

```bash
cd examples/01-basic-impact/plan-mode

# Initialize Terraform
terraform init

# Create plan
terraform plan -out=plan.tfplan

# Convert to JSON
terraform show -json plan.tfplan > plan.json

# Analyze with plan mode
blast-radius impact --plan plan.json
```

**Expected Output:**
```
Principal: user:alice@example.com
  Resources (2):
    - my-production-project (google_project_iam_member):
      - roles/viewer
    - analytics-dataset (google_bigquery_dataset_iam_member):
      - roles/bigquery.dataViewer

Principal: serviceAccount:backup-sa@my-production-project.iam.gserviceaccount.com
  Resources (1):
    - data-bucket (google_storage_bucket_iam_member):
      - roles/storage.objectViewer
```

**Notice:** In plan mode, `var.project_id` is resolved to `my-production-project` ✅

## Key Takeaways

1. **HCL mode** is great for quick checks during development
2. **Plan mode** provides accurate analysis for production deployments
3. Even basic examples show the value of variable resolution in plan mode
4. This is the foundation for more advanced features (hierarchical access, impersonation)

## Next Steps

- Try [`02-hierarchical-access/`](../02-hierarchical-access/) to see project-level roles
- Try [`03-impersonation-chains/`](../03-impersonation-chains/) to see service account impersonation
