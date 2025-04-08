---
page_title: "Provider: AuthZed Cloud API"
description: |-
  The AuthZed Cloud API provider allows Terraform to manage access to your AuthZed environment.
---

# AuthZed Cloud API Provider

The AuthZed Cloud API provider allows you to manage access to [AuthZed Cloud API](https://www.postman.com/authzed/spicedb/collection/5fm402n/authzed-cloud-api) through Terraform. This provider is used to interact with the resources supported by AuthZed Cloud API.

Use the navigation to the left to read about the available resources and data sources.

## Example Usage

```terraform
terraform {
  required_providers {
    cloudapi = {
      source  = "authzed/cloudapi"
      version = "~> 0.1.0"
    }
  }
}

provider "cloudapi" {
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

* `endpoint` - (Optional) The host address of the AuthZed Cloud API. Default is `https://api.admin.stage.aws.authzed.net`.
* `token` - (Required) The bearer token for authentication with AuthZed.
* `api_version` - (Optional) The version of the API to use. Default is "25r1".

## Tech Preview Limitations

The AuthZed API has the following limitations that affect Terraform operations:

* **No Resource Updates**: The API does not support updating resources directly. Any change to a resource's attributes will require the resource to be deleted and recreated.
* **Token Recreation**: When a token resource is recreated, a new secret is generated, and the old token becomes invalid.
* **Resource Dependencies**: Changes to resources like service accounts will generate new IDs, requiring dependent resources to also be recreated.

Plan your changes carefully to avoid disrupting access for production systems.

## Resources and Data Sources

### Resources

* [`cloudapi_role`](resources/role.md) - Manage roles and permissions
* [`cloudapi_policy`](resources/policy.md) - Manage policies for assigning roles
* [`cloudapi_service_account`](resources/service_account.md) - Manage service accounts
* [`cloudapi_token`](resources/token.md) - Manage API tokens

### Data Sources

* [`cloudapi_permission_system`](data-sources/permission_system.md) - Get a specific permission system
* [`cloudapi_permission_systems`](data-sources/permission_systems.md) - List all permission systems
* [`cloudapi_role`](data-sources/role.md) - Get a specific role
* [`cloudapi_roles`](data-sources/roles.md) - List all roles in a permission system
* [`cloudapi_policy`](data-sources/policy.md) - Get a specific policy
* [`cloudapi_policies`](data-sources/policies.md) - List all policies in a permission system
* [`cloudapi_service_account`](data-sources/service_account.md) - Get a specific service account
* [`cloudapi_service_accounts`](data-sources/service_accounts.md) - List all service accounts in a permission system
* [`cloudapi_token`](data-sources/token.md) - Get a specific token
* [`cloudapi_tokens`](data-sources/tokens.md) - List all tokens for a service account 
