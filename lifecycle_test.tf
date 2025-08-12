terraform {
  required_providers {
    authzed = {
      source  = "registry.terraform.io/authzed/authzed"
      version = "0.0.0-dev"
    }
  }
}

provider "authzed" {
  endpoint    = "https://api.admin.stage.aws.authzed.net"
  token       = "7TPV0Qpn89rKxZf6sNmhUHav3L2ujCG5"
  api_version = "25r1"
}

# Test Service Account - FGAM resource
resource "authzed_service_account" "test_sa" {
  name                 = "lifecycle-test-sa-${formatdate("YYYYMMDD-hhmmss", timestamp())}"
  description          = "Testing service account lifecycle and FGAM fields"
  permission_system_id = "ps-IRC1S6Epg4cW-0L6xXaUq"
}

# Test Role - FGAM resource  
resource "authzed_role" "test_role" {
  name                 = "lifecycle-test-role-${formatdate("YYYYMMDD-hhmmss", timestamp())}"
  description          = "Testing role lifecycle and FGAM fields"
  permission_system_id = "ps-IRC1S6Epg4cW-0L6xXaUq"
  permissions = {
    "read"   = "true"
    "write"  = "false"
    "delete" = "false"
  }
}

# Test Policy - FGAM resource (this was the original failing resource)
resource "authzed_policy" "test_policy" {
  name                 = "lifecycle-test-policy-${formatdate("YYYYMMDD-hhmmss", timestamp())}"
  description          = "Testing policy lifecycle and etag consistency"
  permission_system_id = "ps-IRC1S6Epg4cW-0L6xXaUq"
  principal_id         = authzed_service_account.test_sa.id
  role_ids             = [authzed_role.test_role.id]
}

# Test Token - FGAM resource
resource "authzed_token" "test_token" {
  name                 = "lifecycle-test-token-${formatdate("YYYYMMDD-hhmmss", timestamp())}"
  description          = "Testing token lifecycle and FGAM fields"
  permission_system_id = "ps-IRC1S6Epg4cW-0L6xXaUq"
  service_account_id   = authzed_service_account.test_sa.id
}

# Output ALL FGAM fields for each resource to verify they're populated
output "service_account_fgam" {
  value = {
    id         = authzed_service_account.test_sa.id
    name       = authzed_service_account.test_sa.name
    created_at = authzed_service_account.test_sa.created_at
    creator    = authzed_service_account.test_sa.creator
    updated_at = authzed_service_account.test_sa.updated_at
    updater    = authzed_service_account.test_sa.updater
    etag       = authzed_service_account.test_sa.etag
  }
}

output "role_fgam" {
  value = {
    id         = authzed_role.test_role.id
    name       = authzed_role.test_role.name
    created_at = authzed_role.test_role.created_at
    creator    = authzed_role.test_role.creator
    updated_at = authzed_role.test_role.updated_at
    updater    = authzed_role.test_role.updater
    etag       = authzed_role.test_role.etag
  }
}

output "policy_fgam" {
  value = {
    id         = authzed_policy.test_policy.id
    name       = authzed_policy.test_policy.name
    created_at = authzed_policy.test_policy.created_at
    creator    = authzed_policy.test_policy.creator
    updated_at = authzed_policy.test_policy.updated_at
    updater    = authzed_policy.test_policy.updater
    etag       = authzed_policy.test_policy.etag
  }
}

output "token_fgam" {
  value = {
    id         = authzed_token.test_token.id
    name       = authzed_token.test_token.name
    created_at = authzed_token.test_token.created_at
    creator    = authzed_token.test_token.creator
    updated_at = authzed_token.test_token.updated_at
    updater    = authzed_token.test_token.updater
    etag       = authzed_token.test_token.etag
    hash       = authzed_token.test_token.hash
  }
  sensitive = true
}