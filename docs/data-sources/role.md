---
page_title: "Data Source: authzed_role"
description: |-
  Gets information about a specific role in a permission system.
---

# authzed_role (Data Source)

This data source retrieves information about a specific role in an AuthZed permission system.

## Example Usage

```terraform
data "authzed_role" "example" {
  permission_system_id = "ps-123456789"
  role_id             = "arl-abcdef123456"
}

output "role_permissions" {
  value = data.authzed_role.example.permissions
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system containing the role. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `role_id` - (Required) The ID of the role to look up. Must start with `arl-` followed by alphanumeric characters or hyphens.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A unique identifier for this role.
* `name` - The name of the role. Will be between 1 and 50 characters.
* `description` - The description of the role. Maximum length is 200 characters.
* `permissions` - A map of permissions granted by this role. Most permissions are boolean and use "true" as the value.
* `created_at` - The timestamp when the role was created (RFC 3339 format).
* `creator` - The name of the user that created this role.
* `updated_at` - The timestamp when the role was last updated (RFC 3339 format).
* `updater` - The name of the user that last updated this role. 
