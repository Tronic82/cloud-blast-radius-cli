# Policy: Role Restrictions

## Overview

This example demonstrates **role restriction policies** - enforcing which roles specific principals can or cannot have.

## What This Example Shows

- Allowed roles for specific principals
- Denied roles for specific principals
- Pattern matching for principals and resources
- Policy validation with the `validate` command

## Policy Configuration

```yaml
- name: "Developer Role Restrictions"
  type: role_restriction
  role_restriction:
    selector:
      principal_pattern: "group:developers@*"
    allowed_roles:
      - "roles/viewer"
      - "roles/bigquery.dataViewer"
    denied_roles:
      - "roles/owner"
      - "roles/editor"
```

## Running the Example

### HCL Mode

```bash
cd examples/05-policy-role-restrictions/hcl
blast-radius validate . --policy ../policy.yaml
```

### Plan Mode

```bash
cd examples/05-policy-role-restrictions/plan-mode
terraform init
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
blast-radius validate --plan plan.json --policy ../policy.yaml
```

## Expected Output

```
Validating against policy: ../policy.yaml

✅ PASS: Developer Role Restrictions
   - group:developers@example.com has allowed role roles/viewer

❌ FAIL: Developer Role Restrictions
   - group:developers@example.com has denied role roles/editor
   - Violation: Developers should not have editor access

Policy Validation: FAILED (1 violation)
```

## Use Cases

- Enforce least privilege
- Prevent accidental over-permissioning
- Compliance requirements (SOC2, ISO27001)
- Organizational security policies

## Key Takeaway

Role restriction policies are the foundation of IAM governance.
