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

~> **Performance Note:** When creating mixed resource types (policies, service accounts, tokens, roles) with more than 8 total resources, use `terraform apply -parallelism=1` to avoid FGAM conflicts due to temporary API limitations. See the [troubleshooting guide](../guides/troubleshooting.md#performance-and-parallelism) for details.

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
* `etag` - Version identifier used for optimistic concurrency control; updates when the policy changes.

## Timeouts

This resource supports a `timeouts` block for create and delete operations.

Example:

```hcl
resource "authzed_policy" "example" {
  # ... arguments ...

  timeouts {
    create = "5m"
    delete = "10m"
  }
}
```

## Import

Policies can be imported using a composite ID with the format `permission_system_id:policy_id`. The permission system ID must start with `ps-` and the policy ID must start with `apc-`. For example:

```bash
# Import a policy with:
# - Permission System ID: ps-example123
# - Policy ID: apc-mypolicy
terraform import authzed_policy.example "ps-example123:apc-mypolicy"
```

After import, you can manage the policy using Terraform. The imported policy will include all computed attributes like `created_at`, `creator`, etc. and the associated role IDs. 
