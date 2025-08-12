terraform {
  required_providers {
    authzed = {
      source = "authzed/authzed"
    }
  }
}

provider "authzed" {
  endpoint    = "https://api.admin.stage.aws.authzed.net"
  token       = "7TPV0Qpn89rKxZf6sNmhUHav3L2ujCG5"
  api_version = "25r1"
}

# Create a single service account first
resource "authzed_service_account" "test_sa" {
  name                   = "test-retry-conflict"
  description           = "Test service account for retry testing"
  permission_system_id  = "ps-IRC1S6Epg4cW-0L6xXaUq"
} 