# Blast Radius

**Blast Radius** is an open-source IAM permissions analyzer for Terraform. It helps you understand who has access to what resources in your cloud environment, identify potential security risks, and detect hidden access paths through service account impersonation.

## Key Features

- **Impact Analysis**: Map direct access from principals to resources.
- **Hierarchical Analysis**: Identify project-level roles that grant broad access.
- **Impersonation Analysis**: Trace transitive access through service account impersonation chains.
- **Policy Validation**: Enforce custom IAM policies (e.g., role restrictions, separation of duties).
- **Standalone**: Runs entirely locally. No API keys or external servers required.

## Installation

### From Source

```bash
git clone https://github.com/your-org/blast-radius.git
cd blast-radius
go build -o blast-radius ./cmd/blast-radius
```

## Quick Start

### Initialize

Navigate to your Terraform project directory:

```bash
blast-radius init
```

### Analyze Impact

```bash
blast-radius impact .
```

### Advanced Analysis (Plan Mode)

For the most accurate results, use a Terraform plan converted to JSON:

```bash
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
blast-radius impact --plan plan.json
```

## Documentation

See the [USER_GUIDE.md](USER_GUIDE.md) for detailed usage instructions and examples.

## License

MIT
