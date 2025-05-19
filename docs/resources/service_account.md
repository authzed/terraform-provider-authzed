---
page_title: "Resource: cloudapi_service_account"
description: |-
  Manages service accounts for secure access to AuthZed permission systems.
---

# cloudapi_service_account

This resource allows you to create, update, and delete service accounts for your AuthZed permission systems. Service accounts provide a secure way for applications and services to authenticate with and access your permission systems.

## Example Usage

```terraform
resource "cloudapi_service_account" "api_service" {
  name                 = "api-service"
  description          = "Service account for our API backend"
  permission_system_id = "ps-123456789"
}
```

## Argument Reference

* `name` - (Required) A name for the service account. Must be between 1 and 50 characters.
* `description` - (Optional) A description explaining the service account's purpose. Maximum length is 200 characters.
* `permission_system_id` - (Required) The ID of the permission system this service account belongs to. Must start with `ps-` followed by alphanumeric characters or hyphens.

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The unique identifier for the service account. Will start with `sva-` followed by alphanumeric characters or hyphens.
* `created_at` - The timestamp when the service account was created (RFC 3339 format).
* `creator` - The name of the user that created this service account.
* `updated_at` - The timestamp when the service account was last updated (RFC 3339 format).
* `updater` - The name of the user that last updated this service account.

## Import

Service accounts can be imported using a composite ID with the format `permission_system_id:service_account_id`, for example:

```bash
terraform import cloudapi_service_account.api_service ps-123456789:sva-abcdef123456
``` 
