---
page_title: "Resource: cloudapi_token"
description: |-
  Manages API tokens for service accounts to securely access AuthZed.
---

# cloudapi_token

This resource allows you to create and manage API tokens that service accounts use to authenticate with AuthZed. Tokens provide secure, programmatic access to your permission systems.

-> **Note:** The token's secret value is only available when the token is first created and cannot be retrieved later. The token remains valid until it is deleted.

## Example Usage

```terraform
resource "cloudapi_token" "api_token" {
  name                 = "api-service-token"
  description          = "Token for our API service"
  permission_system_id = "ps-123456789"
  service_account_id   = cloudapi_service_account.api_service.id
}

output "token_secret" {
  value     = cloudapi_token.api_token.secret
  sensitive = true
}
```

## Argument Reference

* `name` - (Required) A name for the token. Must be between 1 and 50 characters.
* `description` - (Optional) A description explaining the token's purpose. Maximum length is 200 characters.
* `permission_system_id` - (Required) The ID of the permission system this token belongs to. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `service_account_id` - (Required) The ID of the service account this token is for.

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The unique identifier for the token.
* `secret` - The actual token value that should be used for authentication. This is only available when the token is first created and cannot be retrieved later.
* `created_at` - The timestamp when the token was created (RFC 3339 format).
* `creator` - The name of the user that created this token.
* `updated_at` - The timestamp when the token was last updated (RFC 3339 format).
* `updater` - The name of the user that last updated this token.

## Import

Tokens can be imported using a composite ID with the format `permission_system_id:service_account_id:token_id`, for example:

```bash
terraform import cloudapi_token.api_token ps-123456789:sva-abcdef123456:tok-987654321
``` 
