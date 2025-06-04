---
page_title: "Data Source: authzed_roles"
description: |-
  Lists all roles in a permission system.
---

# authzed_roles (Data Source)

This data source retrieves a list of all roles defined in an AuthZed permission system.

## Example Usage

```terraform
data "authzed_roles" "all" {
  permission_system_id = "sys_123456789"
}

output "role_names" {
  value = [for role in data.authzed_roles.all.roles : role.name]
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system to list roles from.

## Attributes Reference

The following attributes are exported:

* `id` - A unique identifier for this data source in the Terraform state.
* `roles` - A list of roles. Each role contains the following attributes:
  * `id` - The ID of the role.
  * `name` - The name of the role.
  * `description` - The description of the role.
  * `permissions` - A map of permissions granted by this role.
  * `created_at` - The timestamp when the role was created.
  * `updated_at` - The timestamp when the role was last updated.
* `roles_count` - The total number of roles found. 
