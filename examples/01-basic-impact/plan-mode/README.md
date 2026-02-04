# Plan Mode Example

## Generating the Plan

```bash
# Initialize Terraform
terraform init

# Create a plan
terraform plan -out=plan.tfplan

# Convert plan to JSON
terraform show -json plan.tfplan > plan.json
```

## Analyzing with Plan Mode

```bash
# Run blast-radius with plan mode
blast-radius impact --plan plan.json
```

## Key Differences from HCL Mode

### Variable Resolution

**HCL Mode Output:**
```
- var.project_id (google_project_iam_member)
```

**Plan Mode Output:**
```
- my-production-project (google_project_iam_member)
```

### Service Account References

**HCL Mode:**
```
serviceAccount:backup-sa@project.iam.gserviceaccount.com
```

**Plan Mode:**
```
serviceAccount:backup-sa@my-production-project.iam.gserviceaccount.com
```

## Why Plan Mode is Better Here

1. ✅ **Exact Project ID** - Shows actual project, not variable name
2. ✅ **Resolved Service Accounts** - Full service account emails
3. ✅ **Production Accuracy** - Exactly what will be deployed

## When to Use Plan Mode

- ✅ Before deploying to production
- ✅ In CI/CD pipelines
- ✅ When variables are extensively used
- ✅ For compliance reporting
