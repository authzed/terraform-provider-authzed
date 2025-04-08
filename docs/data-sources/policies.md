---
page_title: "Policies Data Source - cloudapi"
subcategory: ""
description: |-
  Lists all policies in a permission system.
---

# cloudapi_policies (Data Source)

This data source retrieves a list of all policies defined in an AuthZed Cloud permission system.

## Example Usage

```terraform
data "cloudapi_policies" "all" {
  permission_system_id = "sys_123456789"
}

output "policy_names" {
  value = [for policy in data.cloudapi_policies.all.policies : policy.name]
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system to list policies from.

## Attributes Reference

The following attributes are exported:

* `id` - A unique identifier for this data source in the Terraform state.
* `policies` - A list of policies. Each policy contains the following attributes:
  * `id` - The ID of the policy.
  * `name` - The name of the policy.
  * `description` - The description of the policy.
  * `principal_id` - The ID of the service account this policy applies to.
  * `role_ids` - A list of role IDs assigned by this policy.
  * `created_at` - The timestamp when the policy was created.
  * `updated_at` - The timestamp when the policy was last updated.
* `policies_count` - The total number of policies found. 
