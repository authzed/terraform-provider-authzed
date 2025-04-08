---
page_title: "Data Source: cloudapi_permission_system"
description: |-
  Retrieves information about a specific permission system in AuthZed.
---

# cloudapi_permission_system

This data source retrieves information about a specific permission system in your AuthZed account. Permission systems are where you define and store your permission relationships.

## Example Usage

```terraform
data "cloudapi_permission_system" "development" {
  id = "sys_123456789"
}

output "system_name" {
  value = data.cloudapi_permission_system.development.name
}

output "system_type" {
  value = data.cloudapi_permission_system.development.system_type
}
```

## Argument Reference

* `id` - (Required) The unique identifier of the permission system to retrieve.

## Attribute Reference

The following attributes are exported:

* `name` - The name of the permission system.
* `system_type` - The type of the permission system (e.g., "dedicated", "developer").
* `created_at` - The timestamp when the permission system was created.
* `updated_at` - The timestamp when the permission system was last updated.

