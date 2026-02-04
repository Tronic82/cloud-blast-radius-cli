# Blast Radius - User Guide

## Table of Contents

1. [Introduction](#introduction)
2. [What is Blast Radius?](#what-is-blast-radius)
3. [Installation](#installation)
4. [Getting Started](#getting-started)
5. [Configuration](#configuration)
6. [Commands](#commands)
7. [Common Use Cases](#common-use-cases)
8. [Troubleshooting](#troubleshooting)
9. [Best Practices](#best-practices)

---

## Introduction

Blast Radius is a powerful IAM permissions analyzer for your Terraform infrastructure. It helps you understand who has access to what resources in your cloud environment, identify potential security risks, and maintain the principle of least privilege.

### Key Benefits

- **Visibility**: See all IAM permissions at a glance
- **Security**: Identify over-privileged accounts
- **Compliance**: Maintain audit trails of permissions
- **Risk Management**: Understand the "blast radius" of each principal
- **Impersonation Detection**: Discover hidden access paths through service account impersonation

---

## What is Blast Radius?

### The Problem

In modern cloud infrastructure:
- IAM permissions can be granted at multiple levels (project, resource, etc.)
- Service accounts can impersonate other service accounts
- It's difficult to know the full scope of access for a single principal
- Over-privileged accounts create security risks

### The Solution

Blast Radius analyzes your Terraform code to:
1. **Map Direct Access**: Show which principals have access to which resources
2. **Detect Hierarchical Access**: Identify project-level roles that grant access to all resources of a type
3. **Trace Impersonation Chains**: Reveal transitive access through service account impersonation
4. **Calculate Blast Radius**: Determine the full scope of impact if a principal is compromised

---

## Installation

### Prerequisites

- Access to your Terraform infrastructure code
- Go 1.21+ (if building from source)

### From Source

```bash
git clone https://github.com/your-org/blast-radius.git
cd blast-radius
go build -o blast-radius ./cmd/blast-radius
```

---

## Getting Started

### Step 1: Initialize Your Project

Navigate to your Terraform project directory:
```bash
cd /path/to/your/terraform/project
blast-radius init
```

This creates a `blast-radius.yaml` configuration file.

### Step 2: Run Your First Analysis

```bash
blast-radius impact .
```

This analyzes the current directory and shows who has access to what.

---

## Parsing Modes

Blast Radius supports two modes for analyzing your Terraform infrastructure:

### HCL Mode (Default)

Parses `.tf` files directly from your Terraform directory.

**Usage:**
```bash
blast-radius impact ./infrastructure
```

**Pros:**
- No terraform plan required
- Fast analysis
- Works without `terraform init`

**Cons:**
- Variables may not resolve
- Resource references may not resolve
- `for_each` and `count` blocks are not expanded

### Plan Mode (Recommended)

Parses JSON output from `terraform show -json` instead of HCL files.

**Usage:**
```bash
# Step 1: Create a terraform plan
terraform plan -out=plan.tfplan

# Step 2: Convert plan to JSON
terraform show -json plan.tfplan > plan.json

# Step 3: Analyze with plan mode
blast-radius impact --plan plan.json
```

**Pros:**
- ✅ All variables and references resolved
- ✅ Exact representation of what Terraform will apply

---

## Configuration

### Configuration File

The `blast-radius.yaml` file controls how Blast Radius operates.

**Example:**
```yaml
# Cloud provider (currently only 'gcp' supported)
cloud_provider: gcp

# Optional: Exclude specific permissions from reports
exclusions:
  - resource_id: "dev-*"
    resource_type: "google_project_iam_member"
    role: "roles/viewer"

# Optional: Directories to ignore during parsing
ignored_directories:
  - ".terraform"
  - "node_modules"
  - ".git"
```

### Configuration Options

#### cloud_provider
- **Required**: Yes
- **Values**: `gcp`
- **Description**: Which cloud provider your infrastructure uses

#### exclusions
- **Description**: Filter specific resource/role combinations from reports

#### ignored_directories
- **Description**: Directories to skip during Terraform file parsing

#### analysis_accounts
- **Description**: Default accounts for the `analyze` command

---

## Commands

### init

**Purpose:** Initialize a new Blast Radius project

```bash
blast-radius init [--config path/to/config.yaml]
```

### impact

**Purpose:** Calculate the blast radius of all IAM principals

```bash
blast-radius impact [directory] [flags]
```

**Flags:**
- `--plan`: Path to terraform plan JSON file (recommended)

**What it shows:**
- Which principals (users, service accounts, groups) have access
- Which resources they can access
- Which roles they have on each resource

### hierarchy

**Purpose:** Show hierarchical access from project-level roles

```bash
blast-radius hierarchy [directory] [flags]
```

**What it shows:**
- Project-level roles that grant access to ALL resources of a type
- Which principals have these powerful permissions

### analyze

**Purpose:** Analyze transitive access via service account impersonation

```bash
blast-radius analyze [directory] --account email@example.com [flags]
```

**What it shows:**
- Direct access
- Hierarchical access
- Transitive access (via service account impersonation chains)

### validate

**Purpose:** Validate IAM configuration against custom organizational policies

```bash
blast-radius validate [directory] --policy path/to/policy.yaml [flags]
```

**What it validates:**
- **Role Restrictions**
- **Persona Policies**
- **Resource Access**
- **Separation of Duties**
- **Privilege Escalation**

---

## Troubleshooting

### "No cloud provider specified"
Ensure your `blast-radius.yaml` contains `cloud_provider: gcp`.

### "No matching principal found"
When using `analyze`, ensure the email address exactly matches the principal in your Terraform code (e.g., `user:alice@example.com` or just `alice@example.com` depending on your tfvars).

---

## Best Practices

1. **Use Plan Mode**: For production environments, always use `--plan` to ensure accuracy.
2. **Validate in CI/CD**: Run `blast-radius validate` in your CI pipeline to catch policy violations before they are applied.
3. **Review Impersonation**: Regularly check `analyze` output for sensitive service accounts to ensure no unexpected impersonation paths exist.
