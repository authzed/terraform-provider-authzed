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
  permission_system_id = "sys_123456789"
  service_account_id   = "sva_abcdef123456"
  token_id             = "tok_987654321"
}

output "token_name" {
  value = data.cloudapi_token.example.name
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system the token belongs to.
* `service_account_id` - (Required) The ID of the service account this token belongs to.
* `token_id` - (Required) The ID of the token to look up.

## Attribute Reference

The following attributes are exported:

* `id` - A unique identifier for this token.
* `name` - The name of the token.
* `description` - The description of the token.
* `created_at` - The timestamp when the token was created.
* `updated_at` - The timestamp when the token was last updated.

~> **Note:** For security reasons, token secrets are not available through this data source. 
