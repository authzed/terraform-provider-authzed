---
page_title: "Data Source: authzed_policy"
description: |-
  Gets information about a specific policy in a permission system.
---

# authzed_policy (Data Source)

This data source retrieves information about a specific policy in an AuthZed permission system.

## Example Usage

```terraform
data "authzed_policy" "example" {
  permission_system_id = "ps-123456789"
  policy_id            = "apc-abcdef123456"
}

output "policy_roles" {
  value = data.authzed_policy.example.role_ids
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system containing the policy. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `policy_id` - (Required) The ID of the policy to look up. Must start with `apc-` followed by alphanumeric characters or hyphens.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A composite ID uniquely identifying this policy.
* `name` - The name of the policy. Will be between 1 and 50 characters.
* `description` - The description of the policy. Maximum length is 200 characters.
* `principal_id` - The ID of the service account this policy applies to.
* `role_ids` - A list of role IDs assigned by this policy. Currently limited to exactly one role ID.
* `created_at` - The timestamp when the policy was created (RFC 3339 format).
* `creator` - The name of the user that created this policy.
* `updated_at` - The timestamp when the policy was last updated (RFC 3339 format).
* `updater` - The name of the user that last updated this policy. 
