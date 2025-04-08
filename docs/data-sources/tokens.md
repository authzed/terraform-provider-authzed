---
page_title: "Tokens Data Source - cloudapi"
subcategory: ""
description: |-
  Lists all tokens for a service account in a permission system.
---

# cloudapi_tokens (Data Source)

This data source retrieves a list of all tokens belonging to a service account in an AuthZed Cloud permission system.

## Example Usage

```terraform
data "cloudapi_tokens" "all" {
  permission_system_id = "sys_123456789"
  service_account_id   = "sa_abcdef123456"
}

output "token_names" {
  value = [for token in data.cloudapi_tokens.all.tokens : token.name]
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system containing the tokens.
* `service_account_id` - (Required) The ID of the service account to list tokens for.

## Attributes Reference

The following attributes are exported:

* `id` - A unique identifier for this data source in the Terraform state.
* `tokens` - A list of tokens. Each token contains the following attributes:
  * `id` - The ID of the token.
  * `name` - The name of the token.
  * `description` - The description of the token.
  * `created_at` - The timestamp when the token was created.
  * `creator` - The identifier of the creator of the token.
* `tokens_count` - The total number of tokens found.

~> **Note:** Token secrets are not available through this data source. Please contact your account manager to get your unique API token. 
