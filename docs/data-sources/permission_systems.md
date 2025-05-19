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
  * `id` - The unique identifier of the permission system. Will start with `ps-` followed by alphanumeric characters or hyphens.
  * `name` - The name of the permission system. Will be between 1 and 50 characters.
  * `description` - The description of the permission system. Maximum length is 200 characters.
  * `system_type` - The type of the permission system (e.g., "dedicated", "developer").
  * `created_at` - The timestamp when the permission system was created (RFC 3339 format).
  * `creator` - The name of the user that created this permission system.
  * `updated_at` - The timestamp when the permission system was last updated (RFC 3339 format).
  * `updater` - The name of the user that last updated this permission system.
  * `version` - A complex object containing version information with the following attributes:
    * `current_version` - An object containing:
      * `display_name` - The display name of the current version.
      * `supported_feature_names` - List of supported feature names.
      * `version` - The version string.
    * `has_update_available` - Whether an update is available.
    * `is_locked_to_version` - Whether the version is locked.
    * `override_image` - The image to use for the SpiceDB instance.
    * `selected_channel` - The selected channel for updates.
* `permission_systems_count` - The total number of permission systems found. 
