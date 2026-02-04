# Blast Radius

[![CI](https://github.com/Tronic82/cloud-blast-radius-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/Tronic82/cloud-blast-radius-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Tronic82/cloud-blast-radius-cli)](https://goreportcard.com/report/github.com/Tronic82/cloud-blast-radius-cli)
[![Release](https://img.shields.io/github/v/release/Tronic82/cloud-blast-radius-cli?style=flat-square)](https://github.com/Tronic82/cloud-blast-radius-cli/releases)

**Blast Radius** is an open-source IAM permissions analyzer for Terraform. It helps you understand who has access to what resources in your cloud environment, identify potential security risks, and detect hidden access paths through service account impersonation.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to get started.

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

Distributed under the Apache 2.0 License. See `LICENSE` for more information.

## Contributor License Agreement

We require all contributors to sign a Contributor License Agreement (CLA). This will be handled automatically by a bot when you open a Pull Request. See [CLA.md](CLA.md) for more details.
