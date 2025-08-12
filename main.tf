terraform {
  required_providers {
    authzed = {
      source  = "authzed/authzed"
      version = "0.5.0"
    }
  }
}

provider "authzed" {
  # Uses environment variables:
  # AUTHZED_API_TOKEN
  # AUTHZED_ENDPOINT (optional)
}

# Simple test to verify the provider works
resource "authzed_service_account" "test_v050" {
  name                 = "test-v050-plan-modifier-fix"
  description          = "Test v0.5.0 plan modifier fix"
  permission_system_id = var.permission_system_id
}

# Output computed fields to verify they work correctly
output "test_computed_fields" {
  value = {
    id         = authzed_service_account.test_v050.id
    created_at = authzed_service_account.test_v050.created_at
    creator    = authzed_service_account.test_v050.creator
    updated_at = authzed_service_account.test_v050.updated_at
    updater    = authzed_service_account.test_v050.updater
  }
}

variable "permission_system_id" {
  description = "The permission system ID to use for testing"
  type        = string
  default     = "ps-test123456789"
}