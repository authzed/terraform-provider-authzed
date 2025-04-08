---
page_title: "Getting Started with AuthZed Cloud API Provider"
description: |-
  Learn how to set up and use the AuthZed Cloud API provider.
---

# Getting Started with AuthZed Cloud API Provider

This guide will walk you through the process of setting up and using the AuthZed Cloud API provider with Terraform. You'll learn how to authenticate with [AuthZed Cloud API](https://www.postman.com/authzed/spicedb/collection/5fm402n/authzed-cloud-api), manage resources, and implement common access control patterns.

## What this API Does

The AuthZed Cloud API allows you to manage access to the AuthZed platform itself, not the permission relationships within your systems. Using this provider, you can:

* Control who can access your AuthZed environment
* Create and manage service accounts that can access your permission systems
* Generate and manage API tokens for secure programmatic access
* View and monitor your hosted permission systems

> **Important:** This API manages platform-level access. For defining permission relationships within your systems (e.g., defining who can view what document), you would use the AuthZed permissions API directly.

## Tech Preview Limitations and Considerations

The AuthZed API has certain limitations you should be aware of when working with Terraform:

1. **No Support for Updates**: The API does not support direct updates to resources. When you change attributes of a resource in your Terraform configuration, Terraform will need to delete the existing resource and create a new one with the updated attributes.

2. **Token Regeneration**: When a token resource is recreated, a new secret is generated. Applications using the previous token will need to be updated with the new secret.

3. **Resource Dependencies**: When a resource like a service account is recreated, it will receive a new ID. Any resources that depend on this ID (like tokens or policies) will also need to be recreated.

4. **Access Continuity**: Plan your changes carefully to avoid disrupting access for production systems. Consider using separate resources for critical vs. non-critical access.

## Before You Begin

To follow this guide, you'll need:

* [Terraform](https://www.terraform.io/downloads.html) 1.0 or later
* An AuthZed Dedicated account
* API token for AuthZed (contact your account team to obtain one)

## Step 1: Configure the Provider

Create a new directory for your Terraform configuration:

```bash
mkdir authzed-terraform
cd authzed-terraform
```

Create a file named `main.tf` with the following content:

```terraform
terraform {
  required_providers {
    cloudapi = {
      source  = "authzed/cloudapi"
      version = "~> 0.1.0"
    }
  }
}

provider "cloudapi" {
  endpoint = "https://api.admin.stage.aws.authzed.net"
  token    = var.authzed_token
}

variable "authzed_token" {
  description = "AuthZed API token"
  type        = string
  sensitive   = true
}

variable "permission_system_id" {
  description = "ID of your permission system"
  type        = string
}
```

Create a file named `terraform.tfvars` to store your token (this file should be added to .gitignore):

```hcl
authzed_token = "your-api-token-here"
permission_system_id = "sys_123456789"  # Replace with your permission system ID
```

## Step 2: List Your Permission Systems

Add the following to your `main.tf` to discover your permission systems:

```terraform
data "cloudapi_permission_systems" "all" {}

output "permission_systems" {
  value = [for ps in data.cloudapi_permission_systems.all.permission_systems : {
    id   = ps.id
    name = ps.name
    type = ps.system_type
  }]
}
```

Initialize Terraform and view your permission systems:

```bash
terraform init
terraform apply -target=data.cloudapi_permission_systems.all
```

Look at the output to find your permission system ID to use in the next steps.

## Step 3: Create a Service Account with Read-Only Access

Let's create a service account that can only read from your permission system. This is a common pattern for applications that need to check permissions but shouldn't modify them.

Add this to your `main.tf`:

```terraform
# Get the specific permission system
data "cloudapi_permission_system" "example" {
  id = var.permission_system_id
}

# Create service account
resource "cloudapi_service_account" "readonly_service" {
  name                 = "readonly-service"
  description          = "Service account with read-only access"
  permission_system_id = data.cloudapi_permission_system.example.id
}

# Create read-only role
resource "cloudapi_role" "readonly" {
  name                 = "readonly"
  description          = "Read-only access"
  permission_system_id = data.cloudapi_permission_system.example.id
  permissions = {
    "authzed.v1/ReadSchema"        = "true"
    "authzed.v1/ReadRelationships" = "true"
    "authzed.v1/CheckPermission"   = "true"
  }
}

# Assign role via policy
resource "cloudapi_policy" "readonly_policy" {
  name                 = "readonly-policy"
  description          = "Grant read-only access"
  permission_system_id = data.cloudapi_permission_system.example.id
  principal_id         = cloudapi_service_account.readonly_service.id
  role_ids             = [cloudapi_role.readonly.id]
}
```

Apply the changes:

```bash
terraform apply
```

## Step 4: Generate an API Token

Now that you have a service account, let's create a token that applications can use to authenticate:

```terraform
resource "cloudapi_token" "readonly_token" {
  name                 = "readonly-service-token"
  description          = "API token for read-only service account"
  permission_system_id = data.cloudapi_permission_system.example.id
  service_account_id   = cloudapi_service_account.readonly_service.id
}

output "token_secret" {
  value     = cloudapi_token.readonly_token.secret
  sensitive = true
}
```

Apply and get the token:

```bash
terraform apply
terraform output -raw token_secret
```

Save this token securely - it will only be shown once.

## Step 5: Create a Service Account with Write Access

For administrative tasks, you might need a service account with write access:

```terraform
# Create service account for management
resource "cloudapi_service_account" "admin_service" {
  name                 = "admin-service"
  description          = "Service account with write access"
  permission_system_id = data.cloudapi_permission_system.example.id
}

# Create writer role
resource "cloudapi_role" "writer" {
  name                 = "writer"
  description          = "Write access"
  permission_system_id = data.cloudapi_permission_system.example.id
  permissions = {
    "authzed.v1/ReadSchema"          = "true"
    "authzed.v1/WriteSchema"         = "true"
    "authzed.v1/ReadRelationships"   = "true"
    "authzed.v1/WriteRelationships"  = "true"
    "authzed.v1/DeleteRelationships" = "true"
    "authzed.v1/CheckPermission"     = "true"
  }
}

# Assign role via policy
resource "cloudapi_policy" "admin_policy" {
  name                 = "admin-policy"
  description          = "Grant write access"
  permission_system_id = data.cloudapi_permission_system.example.id
  principal_id         = cloudapi_service_account.admin_service.id
  role_ids             = [cloudapi_role.writer.id]
}

# Generate token for admin service account
resource "cloudapi_token" "admin_token" {
  name                 = "admin-service-token"
  description          = "API token for admin service account"
  permission_system_id = data.cloudapi_permission_system.example.id
  service_account_id   = cloudapi_service_account.admin_service.id
}

output "admin_token_secret" {
  value     = cloudapi_token.admin_token.secret
  sensitive = true
}
```

Apply these changes:

```bash
terraform apply
```

## Understanding Resource Limitations

When working with this provider, be aware of these limitations:

1. **No Support for Updates**: The AuthZed API doesn't support direct updates. Changing any resource attributes will cause Terraform to delete and recreate the resource.

2. **State Management**: When a token is recreated, a new secret is generated and the old token becomes invalid. Applications using the previous token will need to be updated with the new secret.

3. **Resource Dependencies**: When a service account is recreated, it receives a new ID, which means any tokens or policies attached to it will also need to be recreated.

## Best Practices

1. **Minimize Changes**: Group related resources together and try to avoid frequent changes to resources with dependencies.

2. **Secret Rotation**: If you need to rotate secrets, plan for a transition period where both old and new tokens might be valid.

3. **Separate Environments**: Use different Terraform workspaces or states for development and production environments.

4. **Use Variables**: Keep tokens and IDs in variables to make your configurations more flexible.

## Additional Resources 

- Learn about the available [resources](../index.md#resources) and [data sources](../index.md#data-sources)
- Explore the [AuthZed documentation](https://docs.authzed.com) to understand permissions concepts
