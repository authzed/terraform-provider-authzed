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

variable "role_name" {
  description = "Name of the role"
  type        = string
}

variable "role_description" {
  description = "Description of the role"
  type        = string
  default     = "Test role description"
}

resource "authzed_role" "test" {
  name                 = var.role_name
  description          = var.role_description
  permission_system_id = var.permission_system_id
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}

output "role_id" {
  description = "ID of the created role"
  value       = authzed_role.test.id
}

output "role_etag" {
  description = "ETag of the created role"
  value       = authzed_role.test.etag
} 