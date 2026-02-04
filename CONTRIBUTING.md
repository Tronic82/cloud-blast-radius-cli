# Contributing to Blast Radius

Thank you for your interest in contributing! We welcome all contributions, from bug reports to new features.

## Development Environment

We recommend using the provided **Devcontainer** to ensure a consistent environment.

1. Open the project in VS Code.
2. Click "Reopen in Container" when prompted.
3. This will set up Go, Terraform, and all necessary tools.

## Development Workflow

### 1. Fork and Clone
Fork the repository and clone it locally:

```bash
git clone https://github.com/YOUR_USERNAME/cloud-blast-radius-cli.git
cd cloud-blast-radius-cli
```

### 2. Create a Branch
Create a new branch for your changes:

```bash
git checkout -b feat/my-new-feature
```

### 3. Make Changes & Test
Run the tests locally to ensure everything is working:

```bash
make test
```

If you are modifying output logic, you may need to update golden files:

```bash
make test-integration # or
go test ./tests/integration/... -update
```

### 4. Format & Lint
Before committing, ensure your code is formatted and linted. If you are using the Devcontainer, pre-commit hooks will handle this automatically.

```bash
make fmt
make lint
```

## Commit Messages

We use **Conventional Commits** to automate our release process. Please follow this format:

- `feat: ...` for new features (triggers Minor release)
- `fix: ...` for bug fixes (triggers Patch release)
- `docs: ...` for documentation changes
- `chore: ...` for maintenance tasks
- `refactor: ...` for code refactoring
- `test: ...` for adding tests

You can use **Commitizen** to help you format your commits:

```bash
cz commit
```

## Pull Requests

1. Push your branch to your fork.
2. Open a Pull Request against the `main` branch.
3. Ensure all CI checks pass.
