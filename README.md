# Terraform Provider for AuthZed Cloud API

The AuthZed Cloud API Terraform Provider allows you to manage AuthZed Cloud resources through Terraform. AuthZed is a fine-grained permissions database that enables you to implement advanced access control models.

## Documentation

Full documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/authzed/cloudapi/latest/docs).

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23
- AuthZed Cloud account

## Using the Provider

To use the provider in your Terraform configuration:

```hcl
terraform {
  required_providers {
    cloudapi = {
      source  = "authzed/cloudapi"
      version = "~> 0.1.0"
    }
  }
}

provider "cloudapi" {
  endpoint = "https://cloud.authzed.com/api"
  token    = "your-token"
}

# Example: Creating a service account
resource "cloudapi_service_account" "api_service" {
  name                 = "api-service"
  description          = "Service account for the API service"
  permission_system_id = "sys_123456789"
}

# Example: Creating a role
resource "cloudapi_role" "reader" {
  name                 = "reader"
  description          = "Role for read-only operations"
  permission_system_id = "sys_123456789"
  permissions = {
    "authzed.v1/ReadSchema"        = "true"
    "authzed.v1/ReadRelationships" = "true"
    "authzed.v1/CheckPermission"   = "true"
  }
}
```

## Resources and Data Sources

This provider offers resources for managing:

* Permission systems
* Roles and permissions
* Service accounts
* Tokens
* Policies

And data sources for retrieving:

* Permission systems
* Roles
* Service accounts
* Tokens
* Policies

## Development

### Building the Provider

Clone the repository:
```bash
git clone https://github.com/authzed/terraform-provider-cloudapi.git
cd terraform-provider-cloudapi
go build
```

### Local Installation

To install the provider locally for development:

```bash
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/authzed/cloudapi/0.1.0/$(go env GOOS)_$(go env GOARCH)
cp terraform-provider-cloudapi ~/.terraform.d/plugins/registry.terraform.io/authzed/cloudapi/0.1.0/$(go env GOOS)_$(go env GOARCH)/
```

### Using a Local Build in Terraform

To use the locally built provider, add a dev_overrides section to your ~/.terraformrc:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/authzed/cloudapi" = "/path/to/terraform-provider-cloudapi"
  }
  direct {}
}
```

## Contributing

Contributions are welcome! Please see the [contribution guidelines](CONTRIBUTING.md) for more information.

## License

This provider is licensed under the [Apache 2.0 License](LICENSE).
