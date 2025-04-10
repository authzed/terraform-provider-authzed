terraform {
  required_providers {
    authzed = {
      source  = "authzed/authzed"
      version = "0.1.0"
    }
  }
}

provider "authzed" {
  endpoint    = "https://api.admin.stage.aws.authzed.net"
  token       = "your-api-token-here"
  api_version = "25r1"
}

# Example resources
resource "authzed_service_account" "example" {
  name                 = "example-service-account"
  description          = "Example service account created by Terraform"
  permission_system_id = "ps-example123456789"
}

output "service_account_id" {
  value = authzed_service_account.example.id
} 