---
page_title: "Provider: AuthZed"
description: |-
  The AuthZed provider allows Terraform to manage resources in your AuthZed environment.
---

# AuthZed Provider

The AuthZed provider allows you to manage resources via the [AuthZed Cloud API](https://www.postman.com/authzed/spicedb/collection/5fm402n/authzed-cloud-api) through Terraform. This provider is used to interact with the resources supported by AuthZed Cloud API.

Use the navigation to the left to read about the available resources and data sources.

## Example Usage

```terraform
terraform {
  required_providers {
    authzed = {
      source  = "authzed/authzed"
      version = "~> 0.1"
    }
  }
}

provider "authzed" {
  endpoint    = "https://api.admin.stage.aws.authzed.net"
  token       = var.authzed_api_token
  # Uncomment to specify a different API version
  # api_version = "25r1"
}
```

## Authentication

The provider needs to be configured with a valid API token. This can be provided in the following ways:

1. Using a provider configuration block, as shown in the example above.
2. Via the environment variable `AUTHZED_API_TOKEN`.

To obtain a token, contact your AuthZed account team. They will provide you with a unique token specific to your organization.

-> **Note:** API tokens should be treated as sensitive values and never hardcoded in your Terraform configuration files.

## Provider Arguments

* `endpoint` - (Required) The host address of the AuthZed Cloud API. Default is `https://api.admin.stage.aws.authzed.net`.
* `token` - (Required) The bearer token for authentication with AuthZed.
* `api_version` - (Optional) The version of the API to use. Default is "25r1".

## Important Notes

* **Token Recreation**: When a token resource is recreated, a new secret is generated, and the old token becomes invalid.
* **Resource Dependencies**: Consider the impact on dependent resources when making changes.

Plan your changes carefully to avoid disrupting access for production systems.

## Guides

* [`Getting Started`](guides/getting-started.md) - Get started with the AuthZed provider
* [`Troubleshooting`](guides/troubleshooting.md) - Common issues and solutions

## Resources and Data Sources

### Resources

* [`authzed_role`](resources/role.md) - Manage roles and permissions
* [`authzed_policy`](resources/policy.md) - Manage policies for assigning roles
* [`authzed_service_account`](resources/service_account.md) - Manage service accounts
* [`authzed_token`](resources/token.md) - Manage tokens for service accounts

### Data Sources

* [`authzed_permission_system`](data-sources/permission_system.md) - Get a specific permission system
* [`authzed_permission_systems`](data-sources/permission_systems.md) - List all permission systems
* [`authzed_role`](data-sources/role.md) - Get a specific role
* [`authzed_roles`](data-sources/roles.md) - List all roles in a permission system
* [`authzed_policy`](data-sources/policy.md) - Get a specific policy
* [`authzed_policies`](data-sources/policies.md) - List all policies in a permission system
* [`authzed_service_account`](data-sources/service_account.md) - Get a specific service account
* [`authzed_service_accounts`](data-sources/service_accounts.md) - List all service accounts in a permission system
* [`authzed_token`](data-sources/token.md) - Get a specific token
* [`authzed_tokens`](data-sources/tokens.md) - List all tokens for a service account 
