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
  permission_system_id = "ps-123456789"
  service_account_id   = "asa-abcdef123456"
}

output "token_names" {
  value = [for token in data.cloudapi_tokens.all.tokens : token.name]
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system containing the tokens. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `service_account_id` - (Required) The ID of the service account to list tokens for. Must start with `asa-` followed by alphanumeric characters or hyphens.

## Attributes Reference

The following attributes are exported:

* `id` - A unique identifier for this data source in the Terraform state.
* `tokens` - A list of tokens. Each token contains the following attributes:
  * `id` - The ID of the token. Will start with `atk-` followed by alphanumeric characters or hyphens.
  * `name` - The name of the token. Will be between 1 and 50 characters.
  * `description` - The description of the token. Maximum length is 200 characters.
  * `hash` - The SHA256 hash of the secret part of the token, without the prefix.
  * `created_at` - The timestamp when the token was created (RFC 3339 format).
  * `creator` - The name of the user that created this token.
  * `updated_at` - The timestamp when the token was last updated (RFC 3339 format).
  * `updater` - The name of the user that last updated this token.
* `tokens_count` - The total number of tokens found.

~> **Note:** For security reasons, token secrets are not available through this data source.  
