resource "authzed_role" "test" {
  name                 = var.role_name
  description          = "Test role for policy acceptance tests"
  permission_system_id = var.permission_system_id
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}

resource "authzed_policy" "test" {
  name                 = var.policy_name
  description          = var.policy_description
  permission_system_id = var.permission_system_id
  principal_id         = var.principal_id
  role_ids             = [authzed_role.test.id]
}

variable "policy_name" {
  description = "Name of the policy"
  type        = string
}

variable "role_name" {
  description = "Name of the role"
  type        = string
}

variable "policy_description" {
  description = "Updated description of the policy"
  type        = string
  default     = "Updated test policy description"
}

variable "permission_system_id" {
  description = "Permission system ID"
  type        = string
}

variable "principal_id" {
  description = "Principal ID for the policy"
  type        = string
  default     = "test-principal"
} 