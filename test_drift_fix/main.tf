terraform {
  required_providers {
    authzed = {
      source  = "authzed/authzed"
      version = ">= 0.1.8"
    }
  }
}

provider "authzed" {
  // Configure the provider as needed, e.g., credentials and endpoint.
  // Assuming environment variables are used for credentials.
}

resource "authzed_service_account" "test_sa" {
  name        = "test-sa-for-drift-fix"
  description = "Service account for testing state drift fix."
}

resource "authzed_token" "test_token" {
  service_account_id = authzed_service_account.test_sa.id
  name               = "test-token-for-drift-fix"
}

resource "authzed_role" "test_role" {
  name        = "test-role-for-drift-fix"
  description = "Role for testing state drift fix."
  permissions = {
    "test/resource:read"  = "true",
    "test/resource:write" = "true"
  }
}

resource "authzed_policy" "test_policy" {
  name        = "test-policy-for-drift-fix"
  description = "Policy for testing state drift fix."
  rules = {
    "test_rule" = jsonencode({
      "expression" : "has(user.roles)",
      "parameters" : {},
      "variables" : {}
    })
  }
} 