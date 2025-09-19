# Terraform Provider for AuthZed

A Terraform provider for managing [AuthZed](https://authzed.com/) resources through its Cloud API.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Terraform Registry](https://img.shields.io/badge/Terraform%20Registry-AuthZed-blue.svg)](https://registry.terraform.io/providers/authzed/authzed/latest/docs)

## Overview

This provider automates the management of resources in AuthZed Dedicated environments:

- **Service accounts** for programmatic access to permission systems
- **API tokens** for secure authentication
- **Roles and policies** for fine-grained access control
- **Permission system monitoring** and configuration

> **Note:** This provider manages platform administration only. For managing permissions data (relationships between users and resources), use the [AuthZed Permissions API](https://docs.authzed.com/reference/api) directly.

## Documentation

Full provider documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/authzed/authzed/latest/docs).

API documentation is available on [Postman](https://www.postman.com/authzed/spicedb/collection/5fm402n/authzed-cloud-api).

## Development

### Building Locally

```bash
# Clone the repository
git clone https://github.com/authzed/terraform-provider-authzed.git
cd terraform-provider-authzed

# Build the provider
go build

# Install locally
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/authzed/authzed/0.1.0/$(go env GOOS)_$(go env GOARCH)
cp terraform-provider-authzed ~/.terraform.d/plugins/registry.terraform.io/authzed/authzed/0.1.0/$(go env GOOS)_$(go env GOARCH)/
```

## Working with Development Containers

This repository includes a `.devcontainer` directory for use with [Visual Studio Code](https://code.visualstudio.com/) and the [Remote - Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers).

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/authzed/terraform-provider-authzed?quickstart=1)

### Testing Changes

To use a local build with Terraform, configure your `.terraformrc` file:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/authzed/authzed" = "/path/to/terraform-provider-authzed"
  }
  direct {}
}
```

## Maintenance

This provider is actively maintained by AuthZed, however, many issues are tracked outside of GitHub on internal AuthZed systems. For urgent issues please file a GH Issue or contact your account team.

## Contributing

Contributions are welcome! Please see the [contribution guidelines](CONTRIBUTING.md) for more information.

## License

[Apache 2.0 License](LICENSE)

## Troubleshooting

### Common Issues

**Provider Installation**: If you encounter "Failed to query available provider packages" or "no available releases match", this is often caused by locally cached provider files. 

**Quick fix:**
```bash
# Remove cached files and reinitialize
rm -rf ~/.terraform.d/plugins/*/authzed
terraform init
```

**Large Deployments**: For deployments with 8+ mixed resources or 5+ service accounts, use `terraform apply -parallelism=1` to avoid FGAM conflicts and state management issues due to temporary API limitation. The provider includes optimizations to minimize performance impact. 

For more detailed troubleshooting information, see our [Troubleshooting Guide](https://registry.terraform.io/providers/authzed/authzed/latest/docs/guides/troubleshooting).
