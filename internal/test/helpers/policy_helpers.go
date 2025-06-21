package helpers

import (
	"fmt"

	"terraform-provider-authzed/internal/client"
)

// BuildPolicyConfigBasic creates a basic policy config for testing
func BuildPolicyConfigBasic(policyName, roleName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[2]q
  description          = "Test role for policy acceptance tests"
  permission_system_id = %[3]q
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}

resource "authzed_policy" "test" {
  name                 = %[1]q
  description          = "Test policy description"
  permission_system_id = %[3]q
  principal_id         = "test-principal"
  role_ids             = [authzed_role.test.id]
}
`,
		policyName,
		roleName,
		GetTestPermissionSystemID(),
	)
}

// BuildPolicyConfigUpdate creates a policy config for update testing
func BuildPolicyConfigUpdate(policyName, roleName, description string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[2]q
  description          = "Test role for policy acceptance tests"
  permission_system_id = %[4]q
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}

resource "authzed_policy" "test" {
  name                 = %[1]q
  description          = %[3]q
  permission_system_id = %[4]q
  principal_id         = "test-principal"
  role_ids             = [authzed_role.test.id]
}
`,
		policyName,
		roleName,
		description,
		GetTestPermissionSystemID(),
	)
}

// BuildPolicyConfigWithMultipleRoles creates a policy config with multiple roles
func BuildPolicyConfigWithMultipleRoles(policyName string, roleNames []string) string {
	config := BuildProviderConfig()

	// Add role resources
	for i, roleName := range roleNames {
		config += fmt.Sprintf(`
resource "authzed_role" "test_%d" {
  name                 = %[2]q
  description          = "Test role %[1]d for policy acceptance tests"
  permission_system_id = %[3]q
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}
`, i, roleName, GetTestPermissionSystemID())
	}

	// Add policy resource with all roles
	roleRefs := ""
	for i := range roleNames {
		if i > 0 {
			roleRefs += ", "
		}
		roleRefs += fmt.Sprintf("authzed_role.test_%d.id", i)
	}

	config += fmt.Sprintf(`
resource "authzed_policy" "test" {
  name                 = %[1]q
  description          = "Test policy with multiple roles"
  permission_system_id = %[2]q
  principal_id         = "test-principal"
  role_ids             = [%[3]s]
}
`,
		policyName,
		GetTestPermissionSystemID(),
		roleRefs,
	)

	return config
}

// CreateTestClient creates a client for testing purposes
func CreateTestClient() *client.CloudClient {
	clientConfig := &client.CloudClientConfig{
		Host:       GetTestHost(),
		Token:      GetTestToken(),
		APIVersion: GetTestAPIVersion(),
	}
	return client.NewCloudClient(clientConfig)
}

// ValidatePolicyExists checks if a policy exists in the API
func ValidatePolicyExists(permissionSystemID, policyID string) error {
	testClient := CreateTestClient()
	_, err := testClient.GetPolicy(permissionSystemID, policyID)
	return err
}

// ValidatePolicyDestroyed checks if a policy has been properly destroyed
func ValidatePolicyDestroyed(permissionSystemID, policyID string) error {
	testClient := CreateTestClient()
	_, err := testClient.GetPolicy(permissionSystemID, policyID)
	if err == nil {
		return fmt.Errorf("policy still exists: %s", policyID)
	}

	if !IsNotFoundError(err) {
		return fmt.Errorf("unexpected error checking policy destruction: %v", err)
	}

	return nil
}

// GeneratePolicyTestData creates test data for policy testing
func GeneratePolicyTestData(prefix string) map[string]string {
	testID := GenerateTestID(prefix)
	return map[string]string{
		"policy_name":          testID,
		"role_name":            fmt.Sprintf("%s-role", testID),
		"permission_system_id": GetTestPermissionSystemID(),
		"principal_id":         "test-principal",
		"description":          "Test policy description",
		"updated_description":  "Updated test policy description",
	}
}

// BuildPolicyConfigWithCustomPrincipal creates a policy configuration with custom principal
func BuildPolicyConfigWithCustomPrincipal(policyName, roleName, principalID string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[2]q
  description          = "Test role for policy acceptance tests"
  permission_system_id = %[4]q
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}

resource "authzed_policy" "test" {
  name                 = %[1]q
  description          = "Test policy with custom principal"
  permission_system_id = %[4]q
  principal_id         = %[3]q
  role_ids             = [authzed_role.test.id]
}
`,
		policyName,
		roleName,
		principalID,
		GetTestPermissionSystemID(),
	)
}
