---
page_title: "Resource: cloudapi_policy"
description: |-
  Manages access control policies in an AuthZed permission system.
---

# cloudapi_policy

This resource allows you to create and manage access control policies that connect service accounts with roles. Policies are the mechanism for controlling what actions service accounts can perform on your AuthZed permission systems.

-> **Note:** The AuthZed API does not support updating policies. Any change to policy attributes will force the creation of a new policy and the deletion of the old one.

## Example Usage

```terraform
resource "cloudapi_policy" "reader_policy" {
  name                 = "reader-policy"
  description          = "Grant read-only access"
  permission_system_id = "sys_123456789"
  principal_id         = cloudapi_service_account.api_service.id
  role_ids             = [cloudapi_role.reader.id]
}
```

## Argument Reference

* `name` - (Required) A name for the policy.
* `description` - (Optional) A description explaining the policy's purpose.
* `permission_system_id` - (Required) The ID of the permission system this policy applies to.
* `principal_id` - (Required) The ID of the service account receiving these permissions.
* `role_ids` - (Required) A list of role IDs to assign to the service account.

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The unique identifier for the policy.
* `created_at` - The timestamp when the policy was created.
* `updated_at` - The timestamp when the policy was last updated.

## Import

Policies can be imported using a composite ID with the format `permission_system_id:policy_id`, for example:

```bash
terraform import cloudapi_policy.reader_policy sys_123456789:pol_abcdef123456
``` 
