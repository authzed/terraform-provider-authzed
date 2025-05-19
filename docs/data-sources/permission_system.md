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

* `name` - The name of the permission system.
* `system_type` - The type of the permission system ("development" or "production").
* `system_state` - Information about the current state of the permission system:
  * `status` - The operational status of the system. Possible values: "CLUSTER_ISSUE", "DEGRADED", "MODIFYING", "PAUSED", "PROVISIONING", "PROVISION_ERROR", "RUNNING", "UNKNOWN", "UPGRADE_ERROR", "UPGRADING".
  * `message` - A human-readable message explaining the current state.
* `version` - Version information for the permission system:
  * `current_version` - Information about the current SpiceDB version:
    * `display_name` - The display name of the version.
    * `supported_feature_names` - List of features supported by this version.
    * `version` - The version of SpiceDB.
  * `has_update_available` - Whether an update is available for the SpiceDB version.
  * `is_locked_to_version` - Whether the version is locked to a specific version.
  * `override_image` - The image to use for the SpiceDB instance (if specified).
  * `selected_channel` - The channel selected for the SpiceDB version.
  * `selected_channel_display_name` - The display name of the selected channel.
* `features` - List of features enabled in this permission system. Each feature contains:
  * `id` - The feature identifier.
  * `display_name` - The display name for the feature.
  * `enabled` - Whether the feature is enabled or disabled.

