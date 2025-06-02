# Contributing to the AuthZed Cloud API Terraform Provider

Thank you for considering contributing to the AuthZed Cloud API Terraform Provider! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## Getting Started

1. **Fork the repository** on GitHub.
2. **Clone your fork** locally:
   ```
   git clone https://github.com/YOUR-USERNAME/terraform-provider-cloudapi.git
   ```
3. **Create a branch** for your work:
   ```
   git checkout -b your-feature-branch
   ```

## Development Environment

### Requirements

- Go 1.23 or higher
- Terraform 1.0 or higher
- Access to an AuthZed Cloud account for testing

### Building and Testing

1. **Build the provider**:
   ```
   go build
   ```

2. **Run tests**:
   ```
   go test ./...
   ```

3. **Install locally for testing**:
   ```
   mkdir -p ~/.terraform.d/plugins/registry.terraform.io/authzed/cloudapi/0.1.0/$(go env GOOS)_$(go env GOARCH)
   cp terraform-provider-cloudapi ~/.terraform.d/plugins/registry.terraform.io/authzed/cloudapi/0.1.0/$(go env GOOS)_$(go env GOARCH)/
   ```

## Making Changes

### Code Style

- Follow standard Go code style and conventions
- Use `go fmt` to format your code
- Ensure your code passes `go vet` and `golint`

### Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or less
- Reference issues and pull requests liberally after the first line
- Prefix your commit messages with one of the following to help with changelog generation:
  - `feat:` for new features (e.g., "feat: add support for relationship tuples")
  - `fix:` for bug fixes (e.g., "fix: correct validation in schema resource")
  - `docs:` for documentation changes (e.g., "docs: clarify resource import process")
  - `add:` for other additions (e.g., "add: new validation function")

For example:
```
fix: correct typo in error message

Changed "permision" to "permission" in validation error.
Closes #42
```

### Pull Requests

1. Update the README.md with details of changes if appropriate
2. The PR should work against the main branch
3. Include appropriate tests

Note: While the changelog is automatically generated from commit messages, it may be curated for clarity during the release process.

## Documentation

- Update documentation to reflect any changes in functionality
- Provide examples for new features
- Keep the README and other documentation up to date

## Releasing

(For maintainers only)

1. Let GoReleaser generate the initial changelog from commits
2. Review and curate the changelog for clarity if needed
3. Create a new GitHub release with a semantic version tag
4. The release will automatically be published to the Terraform Registry

## License

By contributing, you agree that your contributions will be licensed under the project's [Apache 2.0 License](LICENSE). 