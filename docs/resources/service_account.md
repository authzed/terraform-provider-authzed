---
page_title: "Resource: authzed_service_account"
description: |-
  Manages service accounts for secure access to AuthZed permission systems.
---

# authzed_service_account

This resource allows you to create, update, and delete service accounts for your AuthZed permission systems. Service accounts provide a secure way for applications and services to authenticate with and access your permission systems.

## Example Usage

```terraform
resource "authzed_service_account" "api_service" {
  name                 = "api-service"
  description          = "Service account for our API backend"
  permission_system_id = "ps-123456789"
}
```

~> **Performance Note:** When creating 5 or more service accounts, use `terraform apply -parallelism=1` to avoid state management issues due to temporary API eventual consistency limitations. See the [troubleshooting guide](../guides/troubleshooting.md#performance-and-parallelism) for details.

## Argument Reference

* `name` - (Required) A name for the service account. Must be between 1 and 50 characters.
* `description` - (Optional) A description explaining the service account's purpose. Maximum length is 200 characters.
* `permission_system_id` - (Required) The ID of the permission system this service account belongs to. Must start with `ps-` followed by alphanumeric characters or hyphens.

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The unique identifier for the service account. Will start with `asa-` followed by alphanumeric characters or hyphens.
* `created_at` - The timestamp when the service account was created (RFC 3339 format).
* `creator` - The name of the user that created this service account.
* `updated_at` - The timestamp when the service account was last updated (RFC 3339 format).
* `updater` - The name of the user that last updated this service account.
* `etag` - Version identifier used for optimistic concurrency control; updates when the service account changes.

## Timeouts

This resource supports a `timeouts` block for create and delete operations.

Example:

```hcl
resource "authzed_service_account" "example" {
  # ... arguments ...

  timeouts {
    create = "10m"
    delete = "10m"
  }
}
```

## Import

Service accounts can be imported using a composite ID with the format `permission_system_id:service_account_id`. The permission system ID must start with `ps-` and the service account ID must start with `asa-`. For example:

```bash
# Import a service account with:
# - Permission System ID: ps-example123
# - Service Account ID: asa-myserviceaccount
terraform import authzed_service_account.example "ps-example123:asa-myserviceaccount"
```

After import, you can manage the service account using Terraform. The imported service account will include all computed attributes like `created_at`, `creator`, etc. 
