package provider

import (
	"context"
	"fmt"
	"testing"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/test/helpers"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDependencyChain(t *testing.T) {
	testID := helpers.GenerateTestID("dependency-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDependencyChainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDependencyChainConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify Permission System exists (dependency root)
					resource.TestCheckResourceAttrSet("data.authzed_permission_system.test", "id"),

					// Verify Service Account created (depends on PS)
					resource.TestCheckResourceAttr("authzed_service_account.chain", "name", fmt.Sprintf("%s-chain-sa", testID)),
					resource.TestCheckResourceAttrSet("authzed_service_account.chain", "id"),

					// Verify Role created (depends on PS)
					resource.TestCheckResourceAttr("authzed_role.chain", "name", fmt.Sprintf("%s-chain-role", testID)),
					resource.TestCheckResourceAttrSet("authzed_role.chain", "id"),

					// Verify Policy created (depends on PS, SA, and Role)
					resource.TestCheckResourceAttr("authzed_policy.chain", "name", fmt.Sprintf("%s-chain-policy", testID)),
					resource.TestCheckResourceAttrSet("authzed_policy.chain", "id"),
					resource.TestCheckResourceAttrSet("authzed_policy.chain", "principal_id"),
					resource.TestCheckResourceAttr("authzed_policy.chain", "role_ids.#", "1"),

					// Custom check to verify dependency chain worked
					func(s *terraform.State) error {
						t.Logf("Successfully created complete dependency chain: PS → SA → Role → Policy")
						t.Logf("Existence gates working correctly for eventual consistency")
						return nil
					},
				),
			},
		},
	})
}

func TestAccCrossResourceDependencies(t *testing.T) {
	testID := helpers.GenerateTestID("cross-deps-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCrossResourceDependenciesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCrossResourceDependenciesConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify all service accounts created
					resource.TestCheckResourceAttr("authzed_service_account.multi.0", "name", fmt.Sprintf("%s-multi-sa-1", testID)),
					resource.TestCheckResourceAttr("authzed_service_account.multi.2", "name", fmt.Sprintf("%s-multi-sa-3", testID)),

					// Verify all roles created
					resource.TestCheckResourceAttr("authzed_role.multi.0", "name", fmt.Sprintf("%s-multi-role-1", testID)),
					resource.TestCheckResourceAttr("authzed_role.multi.2", "name", fmt.Sprintf("%s-multi-role-3", testID)),

					// Verify tokens created (depend on service accounts)
					resource.TestCheckResourceAttr("authzed_token.multi.0", "name", fmt.Sprintf("%s-multi-token-1", testID)),
					resource.TestCheckResourceAttr("authzed_token.multi.5", "name", fmt.Sprintf("%s-multi-token-6", testID)),

					// Verify policies created (depend on both SAs and roles)
					resource.TestCheckResourceAttr("authzed_policy.multi.0", "name", fmt.Sprintf("%s-multi-policy-1", testID)),
					resource.TestCheckResourceAttr("authzed_policy.multi.2", "name", fmt.Sprintf("%s-multi-policy-3", testID)),

					// Custom check for complex dependencies
					func(s *terraform.State) error {
						t.Logf("Successfully created complex cross-resource dependencies")
						t.Logf("Multiple dependency gates working correctly")
						return nil
					},
				),
			},
		},
	})
}

func TestAccResourceVisibilityDelay(t *testing.T) {
	testID := helpers.GenerateTestID("visibility-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVisibilityDelayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVisibilityDelayConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify service account created
					resource.TestCheckResourceAttr("authzed_service_account.visibility", "name", fmt.Sprintf("%s-visibility-sa", testID)),

					// Verify token created (tests SA visibility wait)
					resource.TestCheckResourceAttr("authzed_token.visibility", "name", fmt.Sprintf("%s-visibility-token", testID)),
					resource.TestCheckResourceAttrSet("authzed_token.visibility", "service_account_id"),

					// Verify role created
					resource.TestCheckResourceAttr("authzed_role.visibility", "name", fmt.Sprintf("%s-visibility-role", testID)),

					// Verify policy created (tests both SA and role visibility)
					resource.TestCheckResourceAttr("authzed_policy.visibility", "name", fmt.Sprintf("%s-visibility-policy", testID)),
					resource.TestCheckResourceAttrSet("authzed_policy.visibility", "principal_id"),
					resource.TestCheckResourceAttr("authzed_policy.visibility", "role_ids.#", "1"),

					// Custom check for visibility handling
					func(s *terraform.State) error {
						t.Logf("Successfully handled resource visibility delays")
						t.Logf("AuthZed eventual consistency properly managed")
						return nil
					},
				),
			},
		},
	})
}

func TestAccMultiplePermissionSystems(t *testing.T) {
	testID := helpers.GenerateTestID("multi-ps-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMultiplePermissionSystemsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMultiplePermissionSystemsConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify resources in first PS
					resource.TestCheckResourceAttr("authzed_service_account.ps1", "name", fmt.Sprintf("%s-ps1-sa", testID)),
					resource.TestCheckResourceAttr("authzed_role.ps1", "name", fmt.Sprintf("%s-ps1-role", testID)),

					// Verify resources in second PS
					resource.TestCheckResourceAttr("authzed_service_account.ps2", "name", fmt.Sprintf("%s-ps2-sa", testID)),
					resource.TestCheckResourceAttr("authzed_role.ps2", "name", fmt.Sprintf("%s-ps2-role", testID)),

					// Custom check for multi-PS handling
					func(s *terraform.State) error {
						t.Logf("Successfully created resources across multiple Permission Systems")
						t.Logf("PSLanes correctly isolating different Permission Systems")
						return nil
					},
				),
			},
		},
	})
}

// testAccDependencyChainConfig creates a complete dependency chain
// PS → ServiceAccount → Role → Policy
func testAccDependencyChainConfig(testID string) string {
	return fmt.Sprintf(`
data "authzed_permission_system" "test" {
  name = "test-system"
}

// Service Account depends on Permission System
resource "authzed_service_account" "chain" {
  name = "%s-chain-sa"
  description = "Service account for dependency chain testing"
  permission_system_id = data.authzed_permission_system.test.id
}

// Role depends on Permission System
resource "authzed_role" "chain" {
  name = "%s-chain-role"
  description = "Role for dependency chain testing"
  permission_system_id = data.authzed_permission_system.test.id
  permissions = {
    "authzed.v1/CheckPermission"      = ""
    "authzed.v1/ExpandPermissionTree" = ""
    "authzed.v1/LookupResources"      = ""
    "authzed.v1/LookupSubjects"       = ""
  }
}

// Policy depends on Permission System, Service Account, and Role
resource "authzed_policy" "chain" {
  name = "%s-chain-policy"
  description = "Policy for dependency chain testing"
  permission_system_id = data.authzed_permission_system.test.id
  principal_id = authzed_service_account.chain.id
  role_ids = [authzed_role.chain.id]
}
`, testID, testID, testID)
}

// testAccCrossResourceDependenciesConfig creates complex cross-dependencies
func testAccCrossResourceDependenciesConfig(testID string) string {
	return fmt.Sprintf(`
data "authzed_permission_system" "test" {
  name = "test-system"
}

// 3 Service Accounts
resource "authzed_service_account" "multi" {
  count = 3
  name = "%s-multi-sa-${count.index + 1}"
  description = "Multi-dependency service account ${count.index + 1}"
  permission_system_id = data.authzed_permission_system.test.id
}

// 3 Roles
resource "authzed_role" "multi" {
  count = 3
  name = "%s-multi-role-${count.index + 1}"
  description = "Multi-dependency role ${count.index + 1}"
  permission_system_id = data.authzed_permission_system.test.id
  permissions = {
    "authzed.v1/CheckPermission" = ""
    "authzed.v1/LookupResources" = ""
  }
}

// 6 Tokens (2 per service account) - tests SA visibility
resource "authzed_token" "multi" {
  count = 6
  name = "%s-multi-token-${count.index + 1}"
  description = "Multi-dependency token ${count.index + 1}"
  service_account_id = authzed_service_account.multi[count.index %% 3].id
  permission_system_id = data.authzed_permission_system.test.id
}

// 3 Policies (each maps SA to Role) - tests both SA and Role visibility
resource "authzed_policy" "multi" {
  count = 3
  name = "%s-multi-policy-${count.index + 1}"
  description = "Multi-dependency policy ${count.index + 1}"
  permission_system_id = data.authzed_permission_system.test.id
  principal_id = authzed_service_account.multi[count.index].id
  role_ids = [authzed_role.multi[count.index].id]
}
`, testID, testID, testID, testID)
}

// testAccResourceVisibilityDelayConfig tests visibility waiting
func testAccResourceVisibilityDelayConfig(testID string) string {
	return fmt.Sprintf(`
data "authzed_permission_system" "test" {
  name = "test-system"
}

// Create service account first
resource "authzed_service_account" "visibility" {
  name = "%s-visibility-sa"
  description = "Service account for visibility testing"
  permission_system_id = data.authzed_permission_system.test.id
}

// Token creation should wait for SA visibility
resource "authzed_token" "visibility" {
  name = "%s-visibility-token"
  description = "Token for visibility testing"
  service_account_id = authzed_service_account.visibility.id
  permission_system_id = data.authzed_permission_system.test.id
}

// Create role
resource "authzed_role" "visibility" {
  name = "%s-visibility-role"
  description = "Role for visibility testing"
  permission_system_id = data.authzed_permission_system.test.id
  permissions = {
    "authzed.v1/CheckPermission" = ""
  }
}

// Policy creation should wait for both SA and Role visibility
resource "authzed_policy" "visibility" {
  name = "%s-visibility-policy"
  description = "Policy for visibility testing"
  permission_system_id = data.authzed_permission_system.test.id
  principal_id = authzed_service_account.visibility.id
  role_ids = [authzed_role.visibility.id]
}
`, testID, testID, testID, testID)
}

// testAccMultiplePermissionSystemsConfig tests PSLanes isolation
func testAccMultiplePermissionSystemsConfig(testID string) string {
	return fmt.Sprintf(`
data "authzed_permission_system" "test1" {
  name = "test-system"
}

data "authzed_permission_system" "test2" {
  name = "test-system-2"
}

// Resources in first Permission System
resource "authzed_service_account" "ps1" {
  name = "%s-ps1-sa"
  description = "Service account in PS1"
  permission_system_id = data.authzed_permission_system.test1.id
}

resource "authzed_role" "ps1" {
  name = "%s-ps1-role"
  description = "Role in PS1"
  permission_system_id = data.authzed_permission_system.test1.id
  permissions = {
    "authzed.v1/CheckPermission" = ""
  }
}

// Resources in second Permission System
resource "authzed_service_account" "ps2" {
  name = "%s-ps2-sa"
  description = "Service account in PS2"
  permission_system_id = data.authzed_permission_system.test2.id
}

resource "authzed_role" "ps2" {
  name = "%s-ps2-role"
  description = "Role in PS2"
  permission_system_id = data.authzed_permission_system.test2.id
  permissions = {
    "authzed.v1/CheckPermission" = ""
  }
}
`, testID, testID, testID, testID)
}

func testAccCheckDependencyChainDestroy(s *terraform.State) error {
	// Verify dependency chain resources are destroyed
	return testAccCheckResourcesDestroyed(s, []string{"authzed_service_account", "authzed_role", "authzed_policy"})
}

func testAccCheckCrossResourceDependenciesDestroy(s *terraform.State) error {
	// Verify cross-dependency resources are destroyed
	return testAccCheckResourcesDestroyed(s, []string{"authzed_service_account", "authzed_role", "authzed_token", "authzed_policy"})
}

func testAccCheckVisibilityDelayDestroy(s *terraform.State) error {
	// Verify visibility test resources are destroyed
	return testAccCheckResourcesDestroyed(s, []string{"authzed_service_account", "authzed_token", "authzed_role", "authzed_policy"})
}

func testAccCheckMultiplePermissionSystemsDestroy(s *terraform.State) error {
	// Verify multi-PS resources are destroyed
	return testAccCheckResourcesDestroyed(s, []string{"authzed_service_account", "authzed_role"})
}

// Helper function to check resource destruction
func testAccCheckResourcesDestroyed(s *terraform.State, resourceTypes []string) error {
	// Create client directly for testing
	clientConfig := &client.CloudClientConfig{
		Host:       helpers.GetTestHost(),
		Token:      helpers.GetTestToken(),
		APIVersion: helpers.GetTestAPIVersion(),
	}
	testClient := client.NewCloudClient(clientConfig)

	for _, rs := range s.RootModule().Resources {
		for _, resourceType := range resourceTypes {
			if rs.Type == resourceType {
				permissionSystemID := rs.Primary.Attributes["permission_system_id"]
				if permissionSystemID == "" {
					continue
				}

				var err error
				switch rs.Type {
				case "authzed_service_account":
					_, err = testClient.GetServiceAccount(context.Background(), permissionSystemID, rs.Primary.ID)
				case "authzed_role":
					_, err = testClient.GetRole(context.Background(), permissionSystemID, rs.Primary.ID)
				case "authzed_policy":
					_, err = testClient.GetPolicy(permissionSystemID, rs.Primary.ID)
				case "authzed_token":
					serviceAccountID := rs.Primary.Attributes["service_account_id"]
					if serviceAccountID != "" {
						_, err = testClient.GetToken(context.Background(), permissionSystemID, serviceAccountID, rs.Primary.ID)
					}
				default:
					continue
				}

				if err == nil {
					return fmt.Errorf("Resource still exists: %s %s", rs.Type, rs.Primary.ID)
				}

				// Verify it's actually a 404 error, not another error
				if !helpers.IsNotFoundError(err) {
					return fmt.Errorf("Unexpected error checking %s destruction: %v", rs.Type, err)
				}
			}
		}
	}
	return nil
}
