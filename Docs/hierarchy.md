# blast-radius hierarchy

## Summary

The `hierarchy` command analyzes IAM bindings at **organization, folder, and project levels** to determine **hierarchical access**. In GCP, permissions granted at higher levels (org, folder) automatically inherit downward to resources within that scope.

This command helps identify principals who have broad access through inheritance, which is often invisible when only looking at resource-level bindings.

## Usage

```bash
blast-radius hierarchy [directory] [flags]
```

### Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `directory` | Path to directory containing Terraform files | Current directory (`.`) |

### Flags

| Flag | Description |
|------|-------------|
| `--plan <path>` | Path to Terraform plan JSON file |
| `--tfvars <path>` | Path to terraform.tfvars file for variable resolution |
| `--config <path>` | Path to blast-radius.yaml configuration file (default: `blast-radius.yaml`) |
| `--output <format>` | Output format: `text` or `json` (default: `text`) |
| `--definitions <path>` | Path to custom resource definitions file |
| `--rules <path>` | Path to custom validation rules file |

## Understanding GCP Hierarchy

```
Organization
    └── Folders
        └── Projects
            └── Resources (buckets, datasets, instances, etc.)
```

When a role is granted at the **project level**, it often grants access to **all resources of a certain type** within that project. For example:
- `roles/bigquery.dataViewer` on a project grants read access to ALL BigQuery datasets in that project
- `roles/storage.admin` on a project grants admin access to ALL Cloud Storage buckets in that project

## Text Output

### Example Output

```
Analyzing directory: ./terraform

--- Hierarchical Access Report ---

Principal: user:alice@example.com
  Hierarchical Access:
    - read access to ALL BigQuery Datasets in project 'my-project' via role roles/bigquery.dataViewer assigned on project level

Principal: serviceAccount:backup-sa@project.iam.gserviceaccount.com
  Hierarchical Access:
    - admin access to ALL Cloud Storage Buckets in project 'my-project' via role roles/storage.objectAdmin assigned on project level

Warnings:
  - [unknown_hierarchy] Hierarchy for project 'my-project' is unknown, folder and org level bindings may also apply
  - [unknown_role] Role 'roles/custom.myRole' not found in definitions, hierarchical access cannot be determined

Summary: 2 principals with hierarchical access across 2 bindings
```

### How to Read the Output

1. **Principal Block**: Each principal with hierarchical access gets a section

2. **Hierarchical Access Entry**: Each line describes inherited access
   - **Access Type**: Color-coded access level
     - Green: `read` - View/list access
     - Yellow: `write` - Create/modify access
     - Red: `admin` - Full control including delete and IAM management
     - Cyan: `impersonate` - Ability to act as another identity
   - **Resource Type**: What type of resources are affected (e.g., "BigQuery Datasets")
   - **Scope**: Where the binding is applied (organization, folder, or project)
   - **Role**: The IAM role that grants this access
   - **Level**: The hierarchy level where the role is assigned

3. **Warnings Section**: Issues detected during analysis
   - `[unknown_hierarchy]` - Parent hierarchy not defined in Terraform; there may be additional bindings at folder/org level
   - `[unknown_role]` - Role not in the built-in definitions; hierarchical impact cannot be determined

4. **Summary**: Quick stats on principals and bindings analyzed

### Color Coding

- **Cyan bold**: Section headers and "Hierarchical Access:" labels
- **White bold**: Principal labels
- **Green**: `read` access type
- **Yellow**: `write` access type
- **Red**: `admin` access type
- **Cyan**: `impersonate` access type
- **Yellow**: Warnings section

## JSON Output

Use `--output json` for machine-readable output.

### Schema

```json
{
  "version": "1.0",
  "provider": "gcp",
  "timestamp": "2024-01-15T10:30:00Z",
  "source": {
    "type": "directory",
    "path": "./terraform",
    "input_mode": "hcl"
  },
  "hierarchy": {
    "nodes": [
      {
        "id": "my-project",
        "type": "project",
        "parent_id": "folders/123456",
        "parent_type": "folder"
      }
    ],
    "unknown": [
      {
        "id": "my-folder",
        "type": "folder",
        "referenced_by": ["google_folder_iam_member.admin"]
      }
    ]
  },
  "hierarchical_access": [
    {
      "principal": "user:alice@example.com",
      "principal_type": "user",
      "role": "roles/bigquery.dataViewer",
      "scope": {
        "type": "project",
        "id": "my-project"
      },
      "grants": {
        "affected_levels": ["resource"],
        "resource_types": ["google_bigquery_dataset"],
        "display_name": "BigQuery Dataset",
        "access_type": "read"
      },
      "hierarchy_known": false,
      "source": {
        "resource_type": "google_project_iam_member",
        "resource_address": "google_project_iam_member.alice_bq_viewer"
      }
    }
  ],
  "warnings": [
    {
      "type": "unknown_hierarchy",
      "scope_id": "my-project",
      "scope_type": "project",
      "message": "Hierarchy for project 'my-project' is unknown, folder and org level bindings may also apply"
    }
  ],
  "summary": {
    "total_bindings_analyzed": 10,
    "hierarchical_bindings": 2,
    "principals_with_hierarchical_access": 2,
    "unknown_hierarchy_count": 1,
    "unknown_roles_count": 0,
    "by_scope": {
      "project": 2
    },
    "by_access_type": {
      "read": 1,
      "admin": 1
    }
  }
}
```

### Field Descriptions

#### Root Level

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Schema version |
| `provider` | string | Cloud provider (`gcp`) |
| `timestamp` | string | ISO 8601 timestamp |
| `source` | object | Information about the input source |
| `hierarchy` | object | Discovered hierarchy structure |
| `hierarchical_access` | array | List of hierarchical access grants |
| `warnings` | array | Issues found during analysis |
| `summary` | object | Aggregated statistics |

#### Source Object

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `"directory"` or `"plan_file"` |
| `path` | string | Path to the source |
| `input_mode` | string | `"hcl"` or `"plan_json"` |

#### Hierarchy Object

| Field | Type | Description |
|-------|------|-------------|
| `nodes` | array | Known hierarchy nodes (orgs, folders, projects) |
| `nodes[].id` | string | Resource identifier |
| `nodes[].type` | string | `organization`, `folder`, or `project` |
| `nodes[].parent_id` | string | Parent resource ID (if known) |
| `nodes[].parent_type` | string | Parent resource type |
| `unknown` | array | Resources with unknown parent hierarchy |

#### Hierarchical Access Entry

| Field | Type | Description |
|-------|------|-------------|
| `principal` | string | Full principal identifier |
| `principal_type` | string | `user`, `group`, `serviceAccount`, `domain` |
| `role` | string | IAM role |
| `scope.type` | string | `organization`, `folder`, or `project` |
| `scope.id` | string | Resource ID where binding is applied |
| `grants.affected_levels` | array | Levels below scope that inherit access |
| `grants.resource_types` | array | Terraform resource types affected (or `["*"]` for all) |
| `grants.display_name` | string | Human-readable resource name |
| `grants.access_type` | string | `read`, `write`, `admin`, or `impersonate` |
| `hierarchy_known` | boolean | Whether the full hierarchy is known |
| `source.resource_type` | string | Terraform resource type of the binding |
| `source.resource_address` | string | Terraform resource address |

#### Warning Object

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `unknown_hierarchy` or `unknown_role` |
| `role` | string | (For unknown_role) The unrecognized role |
| `scope_id` | string | (For unknown_hierarchy) The resource ID |
| `scope_type` | string | (For unknown_hierarchy) The resource type |
| `resource_address` | string | Terraform address for reference |
| `message` | string | Human-readable description |

#### Summary Object

| Field | Type | Description |
|-------|------|-------------|
| `total_bindings_analyzed` | number | Total IAM bindings processed |
| `hierarchical_bindings` | number | Bindings that create hierarchical access |
| `principals_with_hierarchical_access` | number | Unique principals with inherited access |
| `unknown_hierarchy_count` | number | Resources without known parent |
| `unknown_roles_count` | number | Roles not in definitions |
| `by_scope` | object | Binding count by scope type |
| `by_access_type` | object | Binding count by access level |

## Examples

### Basic Usage

```bash
# Analyze current directory
blast-radius hierarchy

# Analyze specific directory
blast-radius hierarchy ./terraform/infrastructure
```

### Using Terraform Plan

```bash
# Analyze a plan file
blast-radius hierarchy --plan plan.json
```

### JSON Output for Automation

```bash
# Find all principals with admin access at project level
blast-radius hierarchy --output json | jq '
  .hierarchical_access[] |
  select(.grants.access_type == "admin") |
  {principal: .principal, project: .scope.id, role: .role}
'

# Count hierarchical bindings by access type
blast-radius hierarchy --output json | jq '.summary.by_access_type'
```

## Supported Roles

The hierarchy command recognizes 110+ GCP roles including:
- **Primitive roles**: `roles/viewer`, `roles/editor`, `roles/owner`
- **BigQuery**: `roles/bigquery.dataViewer`, `roles/bigquery.dataEditor`, `roles/bigquery.admin`
- **Storage**: `roles/storage.objectViewer`, `roles/storage.objectAdmin`, `roles/storage.admin`
- **Compute**: `roles/compute.viewer`, `roles/compute.admin`, `roles/compute.instanceAdmin`
- **Cloud Run**: `roles/run.viewer`, `roles/run.admin`, `roles/run.invoker`
- **Secret Manager**: `roles/secretmanager.secretAccessor`, `roles/secretmanager.admin`
- And many more...

For roles not in the built-in definitions, a warning is generated indicating hierarchical access cannot be determined.
