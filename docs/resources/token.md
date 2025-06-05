---
page_title: "Resource: authzed_token"
description: |-
  Manages permission system access-management tokens for service accounts.
---

# authzed_token

This resource allows you to create and manage permission system access-management tokens under service accounts. These tokens are used for access management of a permissions system.

~> **Security Warning** Token values are sensitive and grant access-management capabilities to your permissions system. They are stored in the Terraform state file. Please ensure your state file is stored securely and encrypted. Consider using remote state with encryption enabled.

!> **One-Time Token Access** The token's plaintext value is only available during initial creation via the `plain_text` attribute. After that, only its hash remains available. Make sure to capture and securely store the token when you first see it!

## Example Usage

```hcl
# Create a service account
resource "authzed_service_account" "example" {
  permission_system_id = "ps-example"
  name                = "example-sa"
  description         = "Example service account"
}

# Create a token for the service account
resource "authzed_token" "example" {
  permission_system_id = authzed_service_account.example.permission_system_id
  service_account_id   = authzed_service_account.example.id
  name                = "example-token"
  description         = "Example token"
}

# Output the token value (only available during creation)
output "token_plain_text" {
  value     = authzed_token.example.plain_text
  sensitive = true    # Keeps the token hidden by default
}

# Output the token hash for verification
output "token_hash" {
  value = authzed_token.example.hash
}
```

To securely retrieve the token value during creation:

```bash
# Option 1: Use the terraform output command (recommended):
terraform output -raw token_plain_text

# Option 2: Set sensitive = false in the output block to see it in apply output
```

~> **Security Note:** When using Option 2, the token will be displayed in the terraform apply output.

## Recommended Workflow

Here's a recommended pattern for handling service account tokens:

1. **Create & capture**  
   Run `terraform apply` and copy the token from the CLI output. This is the only time `plain_text` is available.

2. **Securely store**  
   Paste the token into your organization's secret management system (Vault, AWS Secrets Manager, Kubernetes Secrets, etc.) immediately.

3. **Keep it in state**  
   Allow Terraform to preserve the token (marked sensitive) in its state file—and optionally as an `output`—so other resources or modules can reference it without re-creating the token.

4. **Rotate when needed**  
   When it's time to rotate, create a new `authzed_token` resource, update your consumers to use the new token, then destroy the old token:

   ```hcl
   resource "authzed_token" "ci_v2" { ... }
   # then remove or destroy the old one
   ```

5. **Clean up**  
   Once everything is migrated off the old token, remove its Terraform resource (and any associated output) and run `terraform apply` to purge it from state.

This gives you a reliable "one-time" capture of the token in your CLI, plus a safe, state-backed credential for all downstream Terraform-driven workflows.

## Argument Reference

* `name` - (Required) A name for the token. Must be between 1 and 50 characters.
* `description` - (Optional) A description explaining the token's purpose. Maximum length is 200 characters.
* `permission_system_id` - (Required) The ID of the permission system this token belongs to. Must start with `ps-` followed by alphanumeric characters or hyphens.
* `service_account_id` - (Required) The ID of the service account this token is for. Must start with `asa-` followed by alphanumeric characters or hyphens.

## Attribute Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The unique identifier for the token. Will start with `atk-` followed by alphanumeric characters or hyphens.
* `plain_text` - The actual token value that should be used for authentication. **This is only available when the token is first created and cannot be retrieved later.**
* `hash` - The SHA256 hash of the token value, without the prefix.
* `created_at` - The timestamp when the token was created (RFC 3339 format). May not be specified.
* `creator` - The name of the user that created this token. May be empty.
* `updated_at` - The timestamp when the token was last updated (RFC 3339 format). May not be specified.
* `updater` - The name of the user that last updated this token. May be empty.

## State File Security

The token value is stored in the state file as a sensitive value. To protect sensitive values:

1. Use remote state with encryption (e.g., S3 with encryption, HashiCorp Cloud, etc.)
2. Restrict access to the state file
3. Never store state files in version control
4. Use separate state files for resources containing sensitive values
5. Consider using [dynamic credentials](https://developer.hashicorp.com/terraform/tutorials/cloud/dynamic-credentials) where possible

## Import

Tokens can be imported using a composite ID with the format `permission_system_id:service_account_id:token_id`. The permission system ID must start with `ps-`, the service account ID must start with `asa-`, and the token ID must start with `atk-`. For example:

```bash
# Import a token with:
# - Permission System ID: ps-example123
# - Service Account ID: asa-myserviceaccount
# - Token ID: atk-mytoken
terraform import authzed_token.example "ps-example123:asa-myserviceaccount:atk-mytoken"
```

After import, you can manage the token using Terraform. The imported token will include all computed attributes like `created_at`, `creator`, etc. and the token hash.

~> **Note:** When importing a token, the `plain_text` value is **not available**—only the hash can be imported. This is because tokens are only returned in plaintext during their initial creation.
