---
page_title: "Getting Started with AuthZed Provider"
description: |-
  Learn how to set up and use the AuthZed provider.
---

# Getting Started with AuthZed Provider

This guide will walk you through the process of setting up and using the AuthZed provider with Terraform. You'll learn how to authenticate with [AuthZed Cloud API](https://www.postman.com/authzed/spicedb/collection/5fm402n/authzed-cloud-api), manage resources, and implement common access control patterns.

## What this API Does

The AuthZed Cloud API allows you to manage access to the AuthZed platform itself, not the permission relationships within your systems. Using this provider, you can:

* Control who can access your AuthZed environment
* Create and manage service accounts that can access your permission systems
* Generate and manage API tokens for secure programmatic access
* View and monitor your hosted permission systems

## Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) 1.12.x or later
- An AuthZed Dedicated account
- An AuthZed Cloud API token

## Provider Configuration

> **Important:** This API manages platform-level access. For defining permission relationships within your systems (e.g., defining who can view what document), you would use the AuthZed permissions API directly.

## Tech Preview Limitations and Considerations

The AuthZed API has certain limitations you should be aware of when working with Terraform:

1. **Token Regeneration**: When a token resource is recreated, a new secret is generated. Applications using the previous token will need to be updated with the new secret.

2. **Access Continuity**: Plan your changes carefully to avoid disrupting access for production systems. Consider using separate resources for critical vs. non-critical access.

## Before You Begin

To follow this guide, you'll need:

* [Terraform](https://www.terraform.io/downloads.html) 1.12.x or later
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
    authzed = {
      source  = "authzed/authzed"
      version = "~> 0.1.0"
    }
  }
}

provider "authzed" {
  endpoint = "https://api.admin.stage.aws.authzed.net"
  token    = var.authzed_cloud_api_token
}

variable "authzed_cloud_api_token" {
  description = "AuthZed Cloud API token"
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
authzed_cloud_api_token = "your-api-token-here"
permission_system_id = "sys_123456789"  # Replace with your permission system ID
```

## Step 2: List Your Permission Systems

Add the following to your `main.tf` to discover your permission systems:

```terraform
data "authzed_permission_systems" "all" {}

output "permission_systems" {
  value = [for ps in data.authzed_permission_systems.all.permission_systems : {
    id   = ps.id
    name = ps.name
    type = ps.system_type
  }]
}
```

Initialize Terraform and view your permission systems:

```bash
terraform init
terraform apply -target=data.authzed_permission_systems.all
```

Look at the output to find your permission system ID to use in the next steps.

## Step 3: Create a Service Account with Read-Only Access

Let's create a service account that can only read from your permission system. This is a common pattern for applications that need to check permissions but shouldn't modify them.

Add this to your `main.tf`:

```terraform
# Get the specific permission system
data "authzed_permission_system" "example" {
  id = var.permission_system_id
}

# Create service account
resource "authzed_service_account" "readonly_service" {
  name                 = "readonly-service"
  description          = "Service account with read-only access"
  permission_system_id = data.authzed_permission_system.example.id
}

# Create read-only role
resource "authzed_role" "readonly" {
  name                 = "readonly"
  description          = "Read-only access"
  permission_system_id = data.authzed_permission_system.example.id
  permissions = {
    "authzed.v1/ReadSchema"        = ""
    "authzed.v1/ReadRelationships" = ""
    "authzed.v1/CheckPermission"   = ""
  }
}

# Assign role via policy
resource "authzed_policy" "readonly_policy" {
  name                 = "readonly-policy"
  description          = "Grant read-only access"
  permission_system_id = data.authzed_permission_system.example.id
  principal_id         = authzed_service_account.readonly_service.id
  role_ids             = [authzed_role.readonly.id]
}
```

Apply the changes:

```bash
terraform apply
```

## Step 4: Generate an API Token

Now that you have a service account, let's create a token that applications can use to authenticate:

```terraform
resource "authzed_token" "readonly_token" {
  name                 = "readonly-service-token"
  description          = "API token for read-only service account"
  permission_system_id = data.authzed_permission_system.example.id
  service_account_id   = authzed_service_account.readonly_service.id
}

output "token_plain_text" {
  value     = authzed_token.readonly_token.plain_text
  sensitive = true
}
```

Apply and get the token:

```bash
terraform apply
terraform output -raw token_plain_text
```

Save this token securely - it will only be shown once.

## Step 5: Create a Service Account with Write Access

For administrative tasks, you might need a service account with write access:

```terraform
# Create service account for management
resource "authzed_service_account" "admin_service" {
  name                 = "admin-service"
  description          = "Service account with write access"
  permission_system_id = data.authzed_permission_system.example.id
}

# Create writer role
resource "authzed_role" "writer" {
  name                 = "writer"
  description          = "Write access"
  permission_system_id = data.authzed_permission_system.example.id
  permissions = {
    "authzed.v1/ReadSchema"          = ""
    "authzed.v1/WriteSchema"         = ""
    "authzed.v1/ReadRelationships"   = ""
    "authzed.v1/WriteRelationships"  = ""
    "authzed.v1/DeleteRelationships" = ""
    "authzed.v1/CheckPermission"     = ""
  }
}

# Assign role via policy
resource "authzed_policy" "admin_policy" {
  name                 = "admin-policy"
  description          = "Grant write access"
  permission_system_id = data.authzed_permission_system.example.id
  principal_id         = authzed_service_account.admin_service.id
  role_ids             = [authzed_role.writer.id]
}

# Generate token for admin service account
resource "authzed_token" "admin_token" {
  name                 = "admin-service-token"
  description          = "API token for admin service account"
  permission_system_id = data.authzed_permission_system.example.id
  service_account_id   = authzed_service_account.admin_service.id
}

output "admin_token_plain_text" {
  value     = authzed_token.admin_token.plain_text
  sensitive = true
}
```

Apply these changes:

```bash
terraform apply
terraform output -raw admin_token_plain_text
```

## Best Practices

1. **Minimize Changes**: Group related resources together and try to avoid frequent changes to resources with dependencies.
2. **Secret Rotation**: If you need to rotate secrets, plan for a transition period where both old and new tokens might be valid.
3. **Separate Environments**: Use different Terraform workspaces or states for development and production environments.
4. **Use Variables**: Keep tokens and IDs in variables to make your configurations more flexible.

## Additional Resources 

- Learn about the available [resources](../index.md#resources) and [data sources](../index.md#data-sources)
- Explore the [AuthZed documentation](https://docs.authzed.com) to understand permissions concepts
