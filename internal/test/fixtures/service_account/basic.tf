provider "authzed" {
  endpoint    = var.authzed_host
  token       = var.authzed_token
  api_version = var.authzed_api_version
}

variable "authzed_host" {
  description = "AuthZed API host"
  type        = string
}

variable "authzed_token" {
  description = "AuthZed API token"
  type        = string
  sensitive   = true
}

variable "authzed_api_version" {
  description = "AuthZed API version"
  type        = string
  default     = "25r1"
}

variable "permission_system_id" {
  description = "Permission system ID for testing"
  type        = string
}

variable "service_account_name" {
  description = "Name of the service account"
  type        = string
}

variable "service_account_description" {
  description = "Description of the service account"
  type        = string
  default     = "Test service account"
}

resource "authzed_service_account" "test" {
  name                 = var.service_account_name
  description          = var.service_account_description
  permission_system_id = var.permission_system_id
}

output "service_account_id" {
  description = "ID of the created service account"
  value       = authzed_service_account.test.id
}

output "service_account_name" {
  description = "Name of the created service account"
  value       = authzed_service_account.test.name
}

output "service_account_etag" {
  description = "ETag of the created service account"
  value       = authzed_service_account.test.etag
} 