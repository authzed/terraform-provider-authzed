---
page_title: "Resource: cloudapi_token"
description: |-
  Manages API tokens for service accounts to securely access AuthZed.
---

# cloudapi_token

This resource allows you to create and manage API tokens that service accounts use to authenticate with AuthZed. Tokens provide secure, programmatic access to your permission systems.

-> **Note:** The AuthZed API does not support updating tokens. Any change to token attributes will force the creation of a new token and the deletion of the old one. Remember that when a token is recreated, a new secret is generated and the old one becomes invalid.

## Example Usage

```terraform
resource "cloudapi_token" "api_token" {
  name                 = "api-service-token"
  description          = "Token for our API service"
  permission_system_id = "sys_123456789"
  service_account_id   = cloudapi_service_account.api_service.id
}

output "token_secret" {
  value     = cloudapi_token.api_token.secret
  sensitive = true
}
```

## Argument Reference

* `name` - (Required) A name for the token.
* `description` - (Optional) A description explaining the token's purpose.
* `permission_system_id` - (Required) The ID of the permission system this token belongs to.
* `service_account_id` - (Required) The ID of the service account this token is for.

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The unique identifier for the token.
* `secret` - The actual token value that should be used for authentication. This is only available when the token is first created and cannot be retrieved later.
* `created_at` - The timestamp when the token was created.
* `updated_at` - The timestamp when the token was last updated.

## Import

Tokens can be imported using a composite ID with the format `permission_system_id:service_account_id:token_id`, for example:

```bash
terraform import cloudapi_token.api_token sys_123456789:sva_abcdef123456:tok_987654321
``` 
