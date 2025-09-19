---
page_title: "Resource: authzed_role"
description: |-
  Manages a role with permissions for access to an AuthZed permission system.
---

# authzed_role

This resource allows you to create, update and delete roles that define sets of permissions for accessing and managing an AuthZed permission system. Roles are used with policies to control what actions service accounts can perform.

## Example Usage

```terraform
resource "authzed_role" "reader" {
  name                 = "reader"
  description          = "Role for read-only operations"
  permission_system_id = "ps-123456789"
  permissions = {
    "authzed.v1/ReadSchema"        = ""
    "authzed.v1/ReadRelationships" = ""
    "authzed.v1/CheckPermission"   = "CheckPermissionRequest.permission == \"admin\""
  }
}
```

~> **Performance Note:** When creating mixed resource types (roles, service accounts, tokens, policies) with more than 8 total resources, use `terraform apply -parallelism=1` to avoid FGAM conflicts due to temporary API limitations. See the [troubleshooting guide](../guides/troubleshooting.md#performance-and-parallelism) for details.

## Argument Reference

* `name` - (Required) The name of the role. Must be between 1 and 50 characters.
* `description` - (Optional) A description of the role's purpose. Maximum length is 200 characters.
* `permission_system_id` - (Required) The ID of the permission system this role belongs to. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `permissions` - (Required) A map of permission names to CEL filter expressions for more fine grained control access to API methods. Examples can be found [here](https://authzed.com/docs/authzed/concepts/restricted-api-access#example-rule-expressions).  If no filter is required, provide an empty string.

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
* `etag` - Version identifier used for optimistic concurrency control; updates when the role changes.

## Timeouts

This resource supports a `timeouts` block for create and delete operations.

Example:

```hcl
resource "authzed_role" "example" {
  # ... arguments ...

  timeouts {
    create = "10m"
    delete = "10m"
  }
}
```

## Import

Roles can be imported using a composite ID with the format `permission_system_id:role_id`. The permission system ID must start with `ps-` and the role ID must start with `arl-`. For example:

```bash
# Import a role with:
# - Permission System ID: ps-example123
# - Role ID: arl-myrole
terraform import authzed_role.example "ps-example123:arl-myrole"
```

After import, you can manage the role using Terraform. The imported role will include all computed attributes like `created_at`, `creator`, etc. and the full permissions map. 
