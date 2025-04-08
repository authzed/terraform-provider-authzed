---
page_title: "Data Source: cloudapi_permission_systems"
description: |-
  Lists all permission systems available in your AuthZed account.
---

# cloudapi_permission_systems

This data source retrieves information about all permission systems in your AuthZed account. Use this to discover your permission systems and their properties.

## Example Usage

```terraform
data "cloudapi_permission_systems" "all" {}

output "all_system_names" {
  value = [for ps in data.cloudapi_permission_systems.all.permission_systems : ps.name]
}

output "system_count" {
  value = data.cloudapi_permission_systems.all.permission_systems_count
}
```

## Attribute Reference

The following attributes are exported:

* `permission_systems` - A list of all permission systems. Each permission system contains:
  * `id` - The unique identifier of the permission system.
  * `name` - The name of the permission system.
  * `system_type` - The type of the permission system (e.g., "dedicated", "developer").
  * `created_at` - The timestamp when the permission system was created.
  * `updated_at` - The timestamp when the permission system was last updated.
* `permission_systems_count` - The total number of permission systems found. 
