package helpers

import (
	"context"
	"fmt"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
)

// BuildRoleConfigBasic creates a basic role configuration for testing
func BuildRoleConfigBasic(roleName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[1]q
  description          = "Test role description"
  permission_system_id = %[2]q
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}
`,
		roleName,
		GetTestPermissionSystemID(),
	)
}

// BuildRoleConfigWithPermissions creates a role configuration with specified permissions
func BuildRoleConfigWithPermissions(roleName string, permissions map[string]string) string {
	config := BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[1]q
  description          = "Test role with custom permissions"
  permission_system_id = %[2]q
  permissions = {
`,
		roleName,
		GetTestPermissionSystemID(),
	)

	for permission, expression := range permissions {
		config += fmt.Sprintf("    %q = %q\n", permission, expression)
	}

	config += `  }
}
`
	return config
}

// BuildRoleConfigUpdate creates an updated role configuration for testing updates
func BuildRoleConfigUpdate(roleName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[1]q
  description          = "Updated test role description"
  permission_system_id = %[2]q
  permissions = {
    "authzed.v1/ReadSchema"        = ""
    "authzed.v1/ReadRelationships" = ""
    "authzed.v1/CheckPermission"   = "CheckPermissionRequest.permission == \"admin\""
  }
}
`,
		roleName,
		GetTestPermissionSystemID(),
	)
}

// BuildRoleConfigComplexPermissions creates a role with complex permissions for testing
func BuildRoleConfigComplexPermissions(roleName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[1]q
  description          = "Test role with complex permissions"
  permission_system_id = %[2]q
  permissions = {
    "authzed.v1/ReadSchema"           = ""
    "authzed.v1/WriteSchema"          = ""
    "authzed.v1/ReadRelationships"    = ""
    "authzed.v1/WriteRelationships"   = ""
    "authzed.v1/CheckPermission"      = "CheckPermissionRequest.permission == \"admin\" || CheckPermissionRequest.permission == \"read\""
    "authzed.v1/LookupResources"      = "LookupResourcesRequest.resource_object_type == \"document\""
    "authzed.v1/LookupSubjects"       = ""
    "authzed.v1/ExpandPermissionTree" = ""
  }
}
`,
		roleName,
		GetTestPermissionSystemID(),
	)
}

// BuildRoleConfigWithMultipleRoles creates multiple roles for testing relationships
func BuildRoleConfigWithMultipleRoles(baseName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "admin" {
  name                 = "%[1]s-admin"
  description          = "Admin role for testing"
  permission_system_id = %[2]q
  permissions = {
    "authzed.v1/ReadSchema"         = ""
    "authzed.v1/WriteSchema"        = ""
    "authzed.v1/ReadRelationships"  = ""
    "authzed.v1/WriteRelationships" = ""
    "authzed.v1/CheckPermission"    = ""
  }
}

resource "authzed_role" "reader" {
  name                 = "%[1]s-reader"
  description          = "Reader role for testing"
  permission_system_id = %[2]q
  permissions = {
    "authzed.v1/ReadSchema"        = ""
    "authzed.v1/ReadRelationships" = ""
    "authzed.v1/CheckPermission"   = "CheckPermissionRequest.permission == \"read\""
  }
}
`,
		baseName,
		GetTestPermissionSystemID(),
	)
}

// CreateTestRoleClient creates a client for role testing
func CreateTestRoleClient() *client.CloudClient {
	clientConfig := &client.CloudClientConfig{
		Host:       GetTestHost(),
		Token:      GetTestToken(),
		APIVersion: GetTestAPIVersion(),
	}
	return client.NewCloudClient(clientConfig)
}

// ValidateRoleExists checks if a role exists via API
func ValidateRoleExists(permissionSystemID, roleID string) error {
	testClient := CreateTestRoleClient()
	_, err := testClient.GetRole(permissionSystemID, roleID)
	if err != nil {
		return fmt.Errorf("role does not exist: %s", err)
	}
	return nil
}

// ValidateRoleDestroyed checks if a role has been properly destroyed
func ValidateRoleDestroyed(permissionSystemID, roleID string) error {
	testClient := CreateTestRoleClient()
	_, err := testClient.GetRole(permissionSystemID, roleID)
	if err == nil {
		return fmt.Errorf("role still exists: %s", roleID)
	}

	if !IsNotFoundError(err) {
		return fmt.Errorf("unexpected error checking role destruction: %v", err)
	}

	return nil
}

// ValidateRolePermissions checks if a role has the expected permissions
func ValidateRolePermissions(permissionSystemID, roleID string, expectedPermissions map[string]string) error {
	testClient := CreateTestRoleClient()
	roleWithETag, err := testClient.GetRole(permissionSystemID, roleID)
	if err != nil {
		return fmt.Errorf("failed to retrieve role: %s", err)
	}

	role := roleWithETag.Role
	if len(role.Permissions) != len(expectedPermissions) {
		return fmt.Errorf("permission count mismatch: expected %d, got %d", len(expectedPermissions), len(role.Permissions))
	}

	for permission, expectedExpression := range expectedPermissions {
		actualExpression, exists := role.Permissions[permission]
		if !exists {
			return fmt.Errorf("permission %s not found in role", permission)
		}
		if actualExpression != expectedExpression {
			return fmt.Errorf("permission %s: expected expression %q, got %q", permission, expectedExpression, actualExpression)
		}
	}

	return nil
}

// GenerateRoleTestData creates test data for role testing
func GenerateRoleTestData(prefix string) *RoleTestData {
	testID := GenerateTestID(prefix)
	return &RoleTestData{
		Name:               testID,
		Description:        fmt.Sprintf("Test role: %s", testID),
		PermissionSystemID: GetTestPermissionSystemID(),
		BasicPermissions: map[string]string{
			"authzed.v1/ReadSchema": "",
		},
		ComplexPermissions: map[string]string{
			"authzed.v1/ReadSchema":         "",
			"authzed.v1/WriteSchema":        "",
			"authzed.v1/ReadRelationships":  "",
			"authzed.v1/WriteRelationships": "",
			"authzed.v1/CheckPermission":    "CheckPermissionRequest.permission == \"admin\"",
		},
	}
}

// RoleTestData holds test data for role testing
type RoleTestData struct {
	Name               string
	Description        string
	PermissionSystemID string
	BasicPermissions   map[string]string
	ComplexPermissions map[string]string
}

// CreateTestRole creates a role for testing purposes
func CreateTestRole(roleName string, permissions map[string]string) (*models.Role, error) {
	testClient := CreateTestRoleClient()

	role := &models.Role{
		Name:                roleName,
		Description:         fmt.Sprintf("Test role: %s", roleName),
		PermissionsSystemID: GetTestPermissionSystemID(),
		Permissions:         permissions,
	}

	roleWithETag, err := testClient.CreateRole(context.Background(), role)
	if err != nil {
		return nil, fmt.Errorf("failed to create test role: %s", err)
	}

	return roleWithETag.Role, nil
}

// DeleteTestRole deletes a role for cleanup purposes
func DeleteTestRole(permissionSystemID, roleID string) error {
	testClient := CreateTestRoleClient()
	err := testClient.DeleteRole(permissionSystemID, roleID)
	if err != nil && !IsNotFoundError(err) {
		return fmt.Errorf("failed to delete test role: %s", err)
	}
	return nil
}

// GetCommonRolePermissions returns a set of commonly used permissions for testing
func GetCommonRolePermissions() map[string]string {
	return map[string]string{
		"authzed.v1/ReadSchema":        "",
		"authzed.v1/ReadRelationships": "",
		"authzed.v1/CheckPermission":   "",
	}
}

// GetAdminRolePermissions returns a set of admin-level permissions for testing
func GetAdminRolePermissions() map[string]string {
	return map[string]string{
		"authzed.v1/ReadSchema":         "",
		"authzed.v1/WriteSchema":        "",
		"authzed.v1/ReadRelationships":  "",
		"authzed.v1/WriteRelationships": "",
		"authzed.v1/CheckPermission":    "",
		"authzed.v1/LookupResources":    "",
		"authzed.v1/LookupSubjects":     "",
	}
}

// GetReadOnlyRolePermissions returns a set of read-only permissions for testing
func GetReadOnlyRolePermissions() map[string]string {
	return map[string]string{
		"authzed.v1/ReadSchema":        "",
		"authzed.v1/ReadRelationships": "",
		"authzed.v1/CheckPermission":   "CheckPermissionRequest.permission == \"read\"",
	}
}
