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
  default     = "Test service account with tokens"
}

variable "token_names" {
  description = "List of token names to create"
  type        = list(string)
  default     = ["api-token", "backup-token"]
}

resource "authzed_service_account" "test" {
  name                 = var.service_account_name
  description          = var.service_account_description
  permission_system_id = var.permission_system_id
}

resource "authzed_token" "tokens" {
  count = length(var.token_names)
  
  name                 = var.token_names[count.index]
  description          = "Token ${count.index + 1} for ${var.service_account_name}"
  permission_system_id = var.permission_system_id
  service_account_id   = authzed_service_account.test.id
}

output "service_account_id" {
  description = "ID of the created service account"
  value       = authzed_service_account.test.id
}

output "service_account_name" {
  description = "Name of the created service account"
  value       = authzed_service_account.test.name
}

output "token_ids" {
  description = "IDs of the created tokens"
  value       = authzed_token.tokens[*].id
}

output "token_names" {
  description = "Names of the created tokens"
  value       = authzed_token.tokens[*].name
}

output "token_hashes" {
  description = "Hashes of the created tokens"
  value       = authzed_token.tokens[*].hash
}

output "token_plain_texts" {
  description = "Plain text values of the created tokens (sensitive)"
  value       = authzed_token.tokens[*].plain_text
  sensitive   = true
} 