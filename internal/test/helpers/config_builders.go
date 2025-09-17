package helpers

import (
	"fmt"
)

// BuildProviderConfig creates a basic provider configuration for testing
func BuildProviderConfig() string {
	return fmt.Sprintf(`
provider "authzed" {
  endpoint    = %[1]q
  token       = %[2]q
  api_version = %[3]q
}
`,
		GetTestHost(),
		GetTestToken(),
		GetTestAPIVersion(),
	)
}

// BuildProviderConfigWithCustom creates a provider configuration with custom values
func BuildProviderConfigWithCustom(host, token, apiVersion string) string {
	return fmt.Sprintf(`
provider "authzed" {
  endpoint    = %[1]q
  token       = %[2]q
  api_version = %[3]q
}
`,
		host,
		token,
		apiVersion,
	)
}

func BuildPolicyConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "authzed_policy" "test" {
  name                   = %[2]q
  permission_system_id   = %[3]q
  principal_id          = "test-principal-%[4]s"
  role_ids              = []
}
`,
		BuildProviderConfig(),
		name,
		GetTestPermissionSystemID(),
		GenerateTestID("principal"),
	)
}

// BuildRoleConfig creates a basic role configuration for testing
func BuildRoleConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "authzed_role" "test" {
  name                 = %[2]q
  permission_system_id = %[3]q
  permissions = {
    "authzed.v1/ReadSchema" = "true"
  }
}
`,
		BuildProviderConfig(),
		name,
		GetTestPermissionSystemID(),
	)
}

// BuildServiceAccountConfig creates a basic service account configuration for testing
func BuildServiceAccountConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "authzed_service_account" "test" {
  name                 = %[2]q
  permission_system_id = %[3]q
  description         = "Test service account for acceptance testing"
}
`,
		BuildProviderConfig(),
		name,
		GetTestPermissionSystemID(),
	)
}

// BuildTokenConfig creates a basic token configuration for testing
func BuildTokenConfig(name string, serviceAccountName string) string {
	return fmt.Sprintf(`
%s

resource "authzed_service_account" "test" {
  name                 = %[2]q
  permission_system_id = %[3]q
  description         = "Test service account for token testing"
}

resource "authzed_token" "test" {
  name                 = %[4]q
  permission_system_id = %[3]q
  service_account_id   = authzed_service_account.test.id
  description         = "Test token for acceptance testing"
}
`,
		BuildProviderConfig(),
		serviceAccountName,
		GetTestPermissionSystemID(),
		name,
	)
}

// BuildDataSourceConfig creates a basic data source configuration for testing
func BuildDataSourceConfig(dataSourceType, name string) string {
	switch dataSourceType {
	case "permission_system":
		return fmt.Sprintf(`
%s

data "authzed_permission_system" "test" {
  id = %[2]q
}
`,
			BuildProviderConfig(),
			GetTestPermissionSystemID(),
		)
	case "permission_systems":
		return fmt.Sprintf(`
%s

data "authzed_permission_systems" "test" {}
`,
			BuildProviderConfig(),
		)
	default:
		return fmt.Sprintf(`
%s

data "authzed_%[2]s" "test" {
  permission_system_id = %[3]q
  name                = %[4]q
}
`,
			BuildProviderConfig(),
			dataSourceType,
			GetTestPermissionSystemID(),
			name,
		)
	}
}

// BuildComplexConfig creates a complex configuration with multiple resources for testing
func BuildComplexConfig(baseName string) string {
	return fmt.Sprintf(`
%s

resource "authzed_role" "test" {
  name                 = "%[2]s-role"
  permission_system_id = %[3]q
  permissions = {
    "authzed.v1/ReadSchema" = "true"
    "authzed.v1/WriteSchema" = "false"
  }
}

resource "authzed_policy" "test" {
  name                   = "%[2]s-policy"
  permission_system_id   = %[3]q
  principal_id          = "test-principal-%[2]s"
  role_ids              = [authzed_role.test.id]
}

resource "authzed_service_account" "test" {
  name                 = "%[2]s-sa"
  permission_system_id = %[3]q
  description         = "Test service account for complex testing"
}

resource "authzed_token" "test" {
  name                 = "%[2]s-token"
  permission_system_id = %[3]q
  service_account_id   = authzed_service_account.test.id
  description         = "Test token for complex testing"
}
`,
		BuildProviderConfig(),
		baseName,
		GetTestPermissionSystemID(),
	)
}
