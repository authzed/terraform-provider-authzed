---
page_title: "Service Account Data Source - cloudapi"
subcategory: ""
description: |-
  Gets information about a specific service account in a permission system.
---

# cloudapi_service_account (Data Source)

This data source retrieves information about a specific service account in an AuthZed Cloud permission system.

## Example Usage

```terraform
data "cloudapi_service_account" "example" {
  permission_system_id = "ps-123456789"
  service_account_id   = "sva-abcdef123456"
}

output "service_account_name" {
  value = data.cloudapi_service_account.example.name
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system containing the service account. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `service_account_id` - (Required) The ID of the service account to look up. Must start with `sva-` followed by alphanumeric characters or hyphens.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A composite ID uniquely identifying this service account.
* `name` - The name of the service account. Will be between 1 and 50 characters.
* `description` - The description of the service account. Maximum length is 200 characters.
* `created_at` - The timestamp when the service account was created (RFC 3339 format).
* `creator` - The name of the user that created this service account.
* `updated_at` - The timestamp when the service account was last updated (RFC 3339 format).
* `updater` - The name of the user that last updated this service account. 
