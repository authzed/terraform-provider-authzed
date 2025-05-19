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
  id = "ps-123456789"
}

output "system_name" {
  value = data.cloudapi_permission_system.development.name
}

output "system_type" {
  value = data.cloudapi_permission_system.development.system_type
}
```

## Argument Reference

* `id` - (Required) The unique identifier of the permission system to retrieve. Must start with `ps-` followed by alphanumeric characters or hyphens.

## Attribute Reference

The following attributes are exported:

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

