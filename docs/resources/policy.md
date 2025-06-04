---
page_title: "Resource: authzed_policy"
description: |-
  Manages access control policies in an AuthZed permission system.
---

# authzed_policy

This resource allows you to create and manage access control policies that connect service accounts with roles. Policies are the mechanism for controlling what actions service accounts can perform on your AuthZed permission systems.

## Example Usage

```terraform
resource "authzed_policy" "reader_policy" {
  name                 = "reader-policy"
  description          = "Grant read-only access"
  permission_system_id = "ps-123456789"
  principal_id         = authzed_service_account.api_service.id
  role_ids             = [authzed_role.reader.id]
}
```

## Argument Reference

* `name` - (Required) A name for the policy. Must be between 1 and 50 characters.
* `description` - (Optional) A description explaining the policy's purpose. Maximum length is 200 characters.
* `permission_system_id` - (Required) The ID of the permission system this policy applies to. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `principal_id` - (Required) The ID of the service account receiving these permissions.
* `role_ids` - (Required) A list of role IDs to assign to the service account. Currently limited to exactly one role ID.

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The unique identifier for the policy. Will start with `apc-` followed by alphanumeric characters or hyphens.
* `created_at` - The timestamp when the policy was created (RFC 3339 format).
* `creator` - The name of the user that created this policy.
* `updated_at` - The timestamp when the policy was last updated (RFC 3339 format).
* `updater` - The name of the user that last updated this policy.

## Import

Policies can be imported using a composite ID with the format `permission_system_id:policy_id`, for example:

```bash
terraform import authzed_policy.reader_policy ps-123456789:apc-abcdef123456
``` 
