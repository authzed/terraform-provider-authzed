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
  permission_system_id = "sys_123456789"
  service_account_id   = "sa_abcdef123456"
}

output "service_account_name" {
  value = data.cloudapi_service_account.example.name
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system containing the service account.
* `service_account_id` - (Required) The ID of the service account to look up.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A composite ID uniquely identifying this service account.
* `name` - The name of the service account.
* `description` - The description of the service account.
* `created_at` - The timestamp when the service account was created.
* `updated_at` - The timestamp when the service account was last updated. 
