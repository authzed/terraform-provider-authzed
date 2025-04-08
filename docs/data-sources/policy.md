---
page_title: "Data Source: cloudapi_policy"
description: |-
  Gets information about a specific policy in a permission system.
---

# cloudapi_policy

This data source retrieves information about a specific policy in an AuthZed permission system.

## Example Usage

```terraform
data "cloudapi_policy" "example" {
  permission_system_id = "sys_123456789"
  policy_id            = "pol_abcdef123456"
}

output "policy_roles" {
  value = data.cloudapi_policy.example.role_ids
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system containing the policy.
* `policy_id` - (Required) The ID of the policy to look up.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A composite ID uniquely identifying this policy.
* `name` - The name of the policy.
* `description` - The description of the policy.
* `principal_id` - The ID of the service account this policy applies to.
* `role_ids` - A list of role IDs assigned by this policy.
* `created_at` - The timestamp when the policy was created.
* `updated_at` - The timestamp when the policy was last updated. 
