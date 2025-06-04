---
page_title: "Data Source: authzed_permission_systems"
description: |-
  Lists all permission systems available in your AuthZed account.
---

# authzed_permission_systems

This data source retrieves information about all permission systems in your AuthZed account. Use this to discover your permission systems and their properties.

## Example Usage

```terraform
data "authzed_permission_systems" "all" {}

output "all_system_names" {
  value = [for ps in data.authzed_permission_systems.all.permission_systems : ps.name]
}

output "system_count" {
  value = data.authzed_permission_systems.all.permission_systems_count
}
```

## Attribute Reference

The following attributes are exported:

* `permission_systems` - A list of all permission systems. Each permission system contains:
  * `id` - The unique identifier of the permission system. Must start with `ps-` followed by alphanumeric characters or hyphens.
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
* `permission_systems_count` - The total number of permission systems.
