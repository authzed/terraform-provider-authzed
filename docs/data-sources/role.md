---
page_title: "Data Source: cloudapi_role"
description: |-
  Gets information about a specific role in a permission system.
---

# cloudapi_role

This data source retrieves information about a specific role in an AuthZed permission system.

## Example Usage

```terraform
data "cloudapi_role" "example" {
  permission_system_id = "sys_123456789"
  role_id              = "rol_abcdef123456"
}

output "role_permissions" {
  value = data.cloudapi_role.example.permissions
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system containing the role.
* `role_id` - (Required) The ID of the role to look up.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A unique identifier for this role.
* `name` - The name of the role.
* `description` - The description of the role.
* `permissions` - A map of permissions granted by this role.
* `created_at` - The timestamp when the role was created.
* `updated_at` - The timestamp when the role was last updated. 
