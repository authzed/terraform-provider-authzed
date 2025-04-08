---
page_title: "Resource: cloudapi_service_account"
description: |-
  Manages service accounts for secure access to AuthZed permission systems.
---

# cloudapi_service_account

This resource allows you to create, update, and delete service accounts for your AuthZed permission systems. Service accounts provide a secure way for applications and services to authenticate with and access your permission systems.

-> **Note:** The AuthZed API does not support updating service accounts. Any change to service account attributes will force the creation of a new service account and the deletion of the old one.

## Example Usage

```terraform
resource "cloudapi_service_account" "api_service" {
  name                 = "api-service"
  description          = "Service account for our API backend"
  permission_system_id = "sys_123456789"
}
```

## Argument Reference

* `name` - (Required) A name for the service account.
* `description` - (Optional) A description explaining the service account's purpose.
* `permission_system_id` - (Required) The ID of the permission system this service account belongs to.

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The unique identifier for the service account.
* `created_at` - The timestamp when the service account was created.
* `updated_at` - The timestamp when the service account was last updated.

## Import

Service accounts can be imported using a composite ID with the format `permission_system_id:service_account_id`, for example:

```bash
terraform import cloudapi_service_account.api_service sys_123456789:sva_abcdef123456
``` 
