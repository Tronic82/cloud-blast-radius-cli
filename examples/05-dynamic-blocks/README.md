# Dynamic Blocks (for_each and count)

## Overview

This example demonstrates how **plan mode excels** at handling dynamic Terraform blocks (`for_each` and `count`) compared to HCL mode.

## The Problem with HCL Mode

HCL mode cannot expand dynamic blocks because it doesn't evaluate Terraform expressions. Variables and loops remain unexpanded.

## The Plan Mode Solution

Plan mode uses `terraform show -json` which contains the **fully expanded** resources after Terraform has evaluated all `for_each` and `count` expressions.

## Examples

### for_each Example

```hcl
variable "developers" {
  default = {
    "alice" = "alice@example.com"
    "bob"   = "bob@example.com"
    "charlie" = "charlie@example.com"
  }
}

resource "google_project_iam_member" "dev_access" {
  for_each = var.developers
  
  project = "dev-project"
  role    = "roles/viewer"
  member  = "user:${each.value}"
}
```

**HCL Mode:** Cannot expand - sees only the template
**Plan Mode:** Expands to 3 separate bindings ✅

### count Example

```hcl
variable "environments" {
  default = ["dev", "staging", "prod"]
}

resource "google_project_iam_member" "env_access" {
  count = length(var.environments)
  
  project = var.environments[count.index]
  role    = "roles/viewer"
  member  = "serviceAccount:deployer@project.iam.gserviceaccount.com"
}
```

**HCL Mode:** Cannot expand - sees only the template
**Plan Mode:** Expands to 3 separate bindings ✅

## Running the Examples

### HCL Mode (Limited)

```bash
cd examples/08-dynamic-blocks/hcl
blast-radius impact .
```

**Result:** Will not show expanded resources ❌

### Plan Mode (Full Expansion)

```bash
cd examples/08-dynamic-blocks/plan-mode
terraform init
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
blast-radius impact --plan plan.json
```

**Result:** Shows all expanded resources ✅

## When Plan Mode is Essential

- ✅ Using `for_each` to create multiple IAM bindings
- ✅ Using `count` to create multiple resources
- ✅ Dynamic resource creation based on variables
- ✅ Production deployments with complex configurations

## Key Takeaway

**For dynamic blocks, plan mode is not optional - it's required for accurate analysis.**
