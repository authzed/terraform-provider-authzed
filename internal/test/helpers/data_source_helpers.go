package helpers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Data source configuration builders

// BuildPermissionSystemDataSourceConfig creates a permission system data source configuration
func BuildPermissionSystemDataSourceConfig(permissionSystemID string) string {
	return fmt.Sprintf(`
%s

data "authzed_permission_system" "test" {
  id = %q
}
`, BuildProviderConfig(), permissionSystemID)
}

// BuildPermissionSystemsDataSourceConfig creates a permission systems data source configuration
func BuildPermissionSystemsDataSourceConfig() string {
	return fmt.Sprintf(`
%s

data "authzed_permission_systems" "test" {}
`, BuildProviderConfig())
}

// BuildPolicyDataSourceConfig creates a policy data source configuration with dependencies
func BuildPolicyDataSourceConfig(policyName, roleName, serviceAccountName string) string {
	return fmt.Sprintf(`
%s

resource "authzed_role" "test" {
  name                 = %q
  permission_system_id = %q
  description         = "Test role for policy data source"
  permissions = {
    "document:read" = "true"
    "document:write" = "request.user.id == resource.owner_id"
  }
}

resource "authzed_service_account" "test" {
  name                 = %q
  permission_system_id = %q
  description         = "Test service account for policy data source"
}

resource "authzed_policy" "test" {
  name                 = %q
  permission_system_id = %q
  description         = "Test policy for data source"
  principal_id        = authzed_service_account.test.id
  role_ids           = [authzed_role.test.id]
}

data "authzed_policy" "test" {
  permission_system_id = %q
  policy_id           = authzed_policy.test.id
}
`, BuildProviderConfig(), roleName, GetPermissionSystemID(), serviceAccountName,
		GetPermissionSystemID(), policyName, GetPermissionSystemID(), GetPermissionSystemID())
}

// BuildPoliciesDataSourceConfig creates a policies data source configuration
func BuildPoliciesDataSourceConfig() string {
	return fmt.Sprintf(`
%s

data "authzed_policies" "test" {
  permission_system_id = %q
}
`, BuildProviderConfig(), GetPermissionSystemID())
}

// BuildRoleDataSourceConfig creates a role data source configuration with dependencies
func BuildRoleDataSourceConfig(roleName string) string {
	return fmt.Sprintf(`
%s

resource "authzed_role" "test" {
  name                 = %q
  permission_system_id = %q
  description         = "Test role for data source"
  permissions = {
    "document:read" = "true"
    "document:write" = "request.user.id == resource.owner_id"
  }
}

data "authzed_role" "test" {
  permission_system_id = %q
  role_id             = authzed_role.test.id
}
`, BuildProviderConfig(), roleName, GetPermissionSystemID(), GetPermissionSystemID())
}

// BuildRolesDataSourceConfig creates a roles data source configuration
func BuildRolesDataSourceConfig() string {
	return fmt.Sprintf(`
%s

data "authzed_roles" "test" {
  permission_system_id = %q
}
`, BuildProviderConfig(), GetPermissionSystemID())
}

// BuildServiceAccountDataSourceConfig creates a service account data source configuration
func BuildServiceAccountDataSourceConfig(serviceAccountName string) string {
	return fmt.Sprintf(`
%s

resource "authzed_service_account" "test" {
  name                 = %q
  permission_system_id = %q
  description         = "Test service account for data source"
}

data "authzed_service_account" "test" {
  permission_system_id = %q
  service_account_id   = authzed_service_account.test.id
}
`, BuildProviderConfig(), serviceAccountName, GetPermissionSystemID(), GetPermissionSystemID())
}

// BuildServiceAccountsDataSourceConfig creates a service accounts data source configuration
func BuildServiceAccountsDataSourceConfig() string {
	return fmt.Sprintf(`
%s

data "authzed_service_accounts" "test" {
  permission_system_id = %q
}
`, BuildProviderConfig(), GetPermissionSystemID())
}

// BuildTokenDataSourceConfig creates a token data source configuration with dependencies
func BuildTokenDataSourceConfig(tokenName, serviceAccountName string) string {
	return fmt.Sprintf(`
%s

resource "authzed_service_account" "test" {
  name                 = %q
  permission_system_id = %q
  description         = "Test service account for token data source"
}

resource "authzed_token" "test" {
  name                 = %q
  permission_system_id = %q
  service_account_id   = authzed_service_account.test.id
  description         = "Test token for data source"
}

data "authzed_token" "test" {
  permission_system_id = %q
  service_account_id   = authzed_service_account.test.id
  token_id            = authzed_token.test.id
}
`, BuildProviderConfig(), serviceAccountName, GetPermissionSystemID(), tokenName,
		GetPermissionSystemID(), GetPermissionSystemID())
}

// BuildTokensDataSourceConfig creates a tokens data source configuration with dependencies
func BuildTokensDataSourceConfig(serviceAccountName string) string {
	return fmt.Sprintf(`
%s

resource "authzed_service_account" "test" {
  name                 = %q
  permission_system_id = %q
  description         = "Test service account for tokens data source"
}

data "authzed_tokens" "test" {
  permission_system_id = %q
  service_account_id   = authzed_service_account.test.id
}
`, BuildProviderConfig(), serviceAccountName, GetPermissionSystemID(), GetPermissionSystemID())
}

// Data source validation helpers

// ValidateDataSourceExists checks if a data source exists in the terraform state
func ValidateDataSourceExists(resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("data source %s not found in state", resourceName)
		}
		return nil
	}
}

// ValidateDataSourceAttribute checks if a data source attribute has the expected value
func ValidateDataSourceAttribute(resourceName, attributeName, expectedValue string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("data source %s not found in state", resourceName)
		}

		actualValue := rs.Primary.Attributes[attributeName]
		if actualValue != expectedValue {
			return fmt.Errorf("attribute %s: expected %q, got %q", attributeName, expectedValue, actualValue)
		}
		return nil
	}
}

// ValidateDataSourceAttributeSet checks if a data source attribute is set (not empty)
func ValidateDataSourceAttributeSet(resourceName, attributeName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("data source %s not found in state", resourceName)
		}

		actualValue := rs.Primary.Attributes[attributeName]
		if actualValue == "" {
			return fmt.Errorf("attribute %s is empty", attributeName)
		}
		return nil
	}
}

// ValidateListContainsValue checks if a list attribute contains a specific value
func ValidateListContainsValue(resourceName, listAttributePrefix, valueAttribute, expectedValue string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("data source %s not found in state", resourceName)
		}

		countStr := rs.Primary.Attributes[listAttributePrefix+".#"]
		count, err := strconv.Atoi(countStr)
		if err != nil {
			return fmt.Errorf("list count is not a number: %s", countStr)
		}

		for i := 0; i < count; i++ {
			key := fmt.Sprintf("%s.%d.%s", listAttributePrefix, i, valueAttribute)
			if rs.Primary.Attributes[key] == expectedValue {
				return nil
			}
		}

		return fmt.Errorf("value %q not found in list %s", expectedValue, listAttributePrefix)
	}
}

// ValidateIDFormat checks if an ID follows the expected format pattern
func ValidateIDFormat(resourceName, attributeName, pattern string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("data source %s not found in state", resourceName)
		}

		actualValue := rs.Primary.Attributes[attributeName]
		if actualValue == "" {
			return fmt.Errorf("attribute %s is empty", attributeName)
		}

		if !strings.Contains(actualValue, strings.Split(pattern, "-")[0]) {
			return fmt.Errorf("attribute %s has invalid format: %s (expected pattern: %s)",
				attributeName, actualValue, pattern)
		}
		return nil
	}
}

// ValidatePermissionSystemIDFormat validates permission system ID format (ps-*)
func ValidatePermissionSystemIDFormat(resourceName, attributeName string) func(*terraform.State) error {
	return ValidateIDFormat(resourceName, attributeName, "ps-*")
}

// ValidateRoleIDFormat validates role ID format (arb-*)
func ValidateRoleIDFormat(resourceName, attributeName string) func(*terraform.State) error {
	return ValidateIDFormat(resourceName, attributeName, "arb-*")
}

// ValidatePolicyIDFormat validates policy ID format (apc-*)
func ValidatePolicyIDFormat(resourceName, attributeName string) func(*terraform.State) error {
	return ValidateIDFormat(resourceName, attributeName, "apc-*")
}

// ValidateServiceAccountIDFormat validates service account ID format (asa-*)
func ValidateServiceAccountIDFormat(resourceName, attributeName string) func(*terraform.State) error {
	return ValidateIDFormat(resourceName, attributeName, "asa-*")
}

// ValidateTokenIDFormat validates token ID format (atk-*)
func ValidateTokenIDFormat(resourceName, attributeName string) func(*terraform.State) error {
	return ValidateIDFormat(resourceName, attributeName, "atk-*")
}

// Data source test utilities

// GenerateDataSourceTestName creates a unique test name for data source tests
func GenerateDataSourceTestName(dataSourceType, testType string) string {
	return GenerateTestID(fmt.Sprintf("%s-ds-%s", dataSourceType, testType))
}

// GetDataSourceTestPermissionSystemID returns the permission system ID for data source tests
func GetDataSourceTestPermissionSystemID() string {
	return GetPermissionSystemID()
}

// IsDataSourceTestEnvironmentReady checks if the environment is ready for data source tests
func IsDataSourceTestEnvironmentReady() error {
	return ValidateTestEnvironment()
}
