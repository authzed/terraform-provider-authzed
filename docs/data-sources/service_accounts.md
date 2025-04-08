---
page_title: "Service Accounts Data Source - cloudapi"
subcategory: ""
description: |-
  Lists all service accounts in a permission system.
---

# cloudapi_service_accounts (Data Source)

This data source retrieves a list of all service accounts defined in an AuthZed Cloud permission system.

## Example Usage

```terraform
data "cloudapi_service_accounts" "all" {
  permission_system_id = "sys_123456789"
}

output "service_account_names" {
  value = [for sa in data.cloudapi_service_accounts.all.service_accounts : sa.name]
}
```

## Argument Reference

The following arguments are supported:

* `permission_system_id` - (Required) The ID of the permission system to list service accounts from.

## Attributes Reference

The following attributes are exported:

* `id` - A unique identifier for this data source in the Terraform state.
* `service_accounts` - A list of service accounts. Each service account contains the following attributes:
  * `id` - The ID of the service account.
  * `name` - The name of the service account.
  * `description` - The description of the service account.
  * `created_at` - The timestamp when the service account was created.
  * `updated_at` - The timestamp when the service account was last updated.
* `service_accounts_count` - The total number of service accounts found. 
