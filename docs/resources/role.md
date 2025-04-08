---
page_title: "Resource: cloudapi_role"
description: |-
  Manages a role with permissions for access to an AuthZed permission system.
---

# cloudapi_role

This resource allows you to create, update and delete roles that define sets of permissions for accessing and managing an AuthZed permission system. Roles are used with policies to control what actions service accounts can perform.

-> **Note:** The AuthZed API does not support updating roles. Any change to role attributes will force the creation of a new role and the deletion of the old one.

## Example Usage

```terraform
resource "cloudapi_role" "reader" {
  name                 = "reader"
  description          = "Role for read-only operations"
  permission_system_id = "sys_123456789"
  permissions = {
    "authzed.v1/ReadSchema"        = "true"
    "authzed.v1/ReadRelationships" = "true"
    "authzed.v1/CheckPermission"   = "true"
  }
}
```

## Argument Reference

* `name` - (Required) The name of the role.
* `description` - (Required) A description of the role's purpose.
* `permission_system_id` - (Required) The ID of the permission system this role belongs to.
* `permissions` - (Required) A map of permission names to their values. Most permissions are boolean and use "true" as the value. These control what actions can be performed on the permission system.

## Permission Reference

Here are common permissions that can be granted:

* `authzed.v1/ReadSchema` - Allows reading the schema of the permission system
* `authzed.v1/WriteSchema` - Allows modifying the schema of the permission system
* `authzed.v1/ReadRelationships` - Allows reading relationships in the permission system
* `authzed.v1/WriteRelationships` - Allows creating and updating relationships
* `authzed.v1/DeleteRelationships` - Allows deleting relationships
* `authzed.v1/CheckPermission` - Allows checking permissions (performing lookups)

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the role.
* `created_at` - The timestamp when the role was created.
* `updated_at` - The timestamp when the role was last updated.

## Import

Roles can be imported using a composite ID with the format `permission_system_id:role_id`, for example:

```bash
terraform import cloudapi_role.example sys_123456789:rol_987654321
``` 
