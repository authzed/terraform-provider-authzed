---
page_title: "Resource: cloudapi_role"
description: |-
  Manages a role with permissions for access to an AuthZed permission system.
---

# cloudapi_role

This resource allows you to create, update and delete roles that define sets of permissions for accessing and managing an AuthZed permission system. Roles are used with policies to control what actions service accounts can perform.

## Example Usage

```terraform
resource "cloudapi_role" "reader" {
  name                 = "reader"
  description          = "Role for read-only operations"
  permission_system_id = "ps-123456789"
  permissions = {
    "authzed.v1/ReadSchema"        = "true"
    "authzed.v1/ReadRelationships" = "true"
    "authzed.v1/CheckPermission"   = "true"
  }
}
```

## Argument Reference

* `name` - (Required) The name of the role. Must be between 1 and 50 characters.
* `description` - (Optional) A description of the role's purpose. Maximum length is 200 characters.
* `permission_system_id` - (Required) The ID of the permission system this role belongs to. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `permissions` - (Required) A map of permission names to their values. Most permissions are boolean and use "true" as the value. These control what actions can be performed on the permission system.

## Permission Reference

Here are common permissions that can be granted:

* `authzed.v1/ReadSchema` - Allows reading the schema of the permission system
* `authzed.v1/WriteSchema` - Allows modifying the schema of the permission system
* `authzed.v1/ReadRelationships` - Allows reading relationships in the permission system
* `authzed.v1/WriteRelationships` - Allows creating and updating relationships
* `authzed.v1/DeleteRelationships` - Allows deleting relationships
* `authzed.v1/CheckPermission` - Allows checking permissions (performing lookups)
* `authzed.v1/LookupResources` - Allows looking up resources
* `authzed.v1/LookupSubjects` - Allows looking up subjects
* `authzed.v1/ExpandPermissionTree` - Allows expanding permission trees
* `authzed.v1/Watch` - Allows watching for changes

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The unique identifier for the role. Will start with `arl-` followed by alphanumeric characters or hyphens.
* `created_at` - The timestamp when the role was created (RFC 3339 format).
* `creator` - The name of the user that created this role.
* `updated_at` - The timestamp when the role was last updated (RFC 3339 format).
* `updater` - The name of the user that last updated this role.

## Import

Roles can be imported using a composite ID with the format `permission_system_id:role_id`, for example:

```bash
terraform import cloudapi_role.example ps-123456789:arl-987654321
``` 
