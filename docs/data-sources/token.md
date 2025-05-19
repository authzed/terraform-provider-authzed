---
page_title: "Data Source: cloudapi_token"
description: |-
  Gets information about a specific token belonging to a service account.
---

# cloudapi_token

This data source retrieves information about a specific token belonging to a service account in an AuthZed permission system.

## Example Usage

```terraform
data "cloudapi_token" "example" {
  permission_system_id = "ps-123456789"
  service_account_id   = "sva-abcdef123456"
  token_id            = "atk-987654321"
}

output "token_name" {
  value = data.cloudapi_token.example.name
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system the token belongs to. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `service_account_id` - (Required) The ID of the service account this token belongs to. Must start with `sva-` followed by alphanumeric characters or hyphens.
* `token_id` - (Required) The ID of the token to look up. Must start with `atk-` followed by alphanumeric characters or hyphens.

## Attribute Reference

The following attributes are exported:

* `id` - A unique identifier for this token.
* `name` - The name of the token. Will be between 1 and 50 characters.
* `description` - The description of the token. Maximum length is 200 characters.
* `created_at` - The timestamp when the token was created (RFC 3339 format).
* `creator` - The name of the user that created this token.
* `updated_at` - The timestamp when the token was last updated (RFC 3339 format).
* `updater` - The name of the user that last updated this token.

~> **Note:** For security reasons, token secrets are not available through this data source. 
