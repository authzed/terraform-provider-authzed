package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/test/helpers"
)

func TestAccAuthzedRole_basic(t *testing.T) {
	resourceName := "authzed_role.test"
	testID := helpers.GenerateTestID("test-role")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRoleConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "description", "Test role description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "permission_system_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "creator"),
					resource.TestCheckResourceAttrSet(resourceName, "etag"),
					resource.TestCheckResourceAttr(resourceName, "permissions.%", "4"),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/CheckPermission", ""),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccRoleImportStateIdFunc(resourceName),
				ImportStateVerifyIgnore: []string{
					"etag",       // ETag changes between operations
					"updated_at", // Server-managed timestamp
					"updater",    // Server-managed field
				},
			},
		},
	})
}

func TestAccAuthzedRole_update(t *testing.T) {
	resourceName := "authzed_role.test"
	testID := helpers.GenerateTestID("test-role-update")
	updatedDescription := "Updated test role description"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test role description"),
				),
			},
			{
				Config: testAccRoleConfig_updated(testID, updatedDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", updatedDescription),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
				),
			},
		},
	})
}

func TestAccAuthzedRole_permissions(t *testing.T) {
	resourceName := "authzed_role.test"
	testID := helpers.GenerateTestID("test-role-perms")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_minimalPermissions(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "permissions.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/CheckPermission", ""),
				),
			},
			{
				Config: testAccRoleConfig_fullPermissions(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "permissions.%", "4"),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/CheckPermission", ""),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/ExpandPermissionTree", ""),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/LookupResources", ""),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/LookupSubjects", ""),
				),
			},
		},
	})
}

func testAccRoleConfig_basic(testID string) string {
	return fmt.Sprintf(`
%s

resource "authzed_role" "test" {
  name                 = "%s"
  description          = "Test role description"
  permission_system_id = %q
  permissions = {
    "authzed.v1/CheckPermission"      = ""
    "authzed.v1/ExpandPermissionTree" = ""
    "authzed.v1/LookupResources"      = ""
    "authzed.v1/LookupSubjects"       = ""
  }
}
`, helpers.BuildProviderConfig(), testID, helpers.GetTestPermissionSystemID())
}

func testAccRoleConfig_updated(testID, description string) string {
	return fmt.Sprintf(`
%s

resource "authzed_role" "test" {
  name                 = "%s"
  description          = "%s"
  permission_system_id = %q
  permissions = {
    "authzed.v1/CheckPermission"      = ""
    "authzed.v1/ExpandPermissionTree" = ""
    "authzed.v1/LookupResources"      = ""
    "authzed.v1/LookupSubjects"       = ""
  }
}
`, helpers.BuildProviderConfig(), testID, description, helpers.GetTestPermissionSystemID())
}

func testAccRoleConfig_minimalPermissions(testID string) string {
	return fmt.Sprintf(`
%s

resource "authzed_role" "test" {
  name                 = "%s"
  description          = "Test role with minimal permissions"
  permission_system_id = %q
  permissions = {
    "authzed.v1/CheckPermission" = ""
  }
}
`, helpers.BuildProviderConfig(), testID, helpers.GetTestPermissionSystemID())
}

func testAccRoleConfig_fullPermissions(testID string) string {
	return fmt.Sprintf(`
%s

resource "authzed_role" "test" {
  name                 = "%s"
  description          = "Test role with full permissions"
  permission_system_id = %q
  permissions = {
    "authzed.v1/CheckPermission"      = ""
    "authzed.v1/ExpandPermissionTree" = ""
    "authzed.v1/LookupResources"      = ""
    "authzed.v1/LookupSubjects"       = ""
  }
}
`, helpers.BuildProviderConfig(), testID, helpers.GetTestPermissionSystemID())
}

func testAccCheckRoleExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Role ID is set")
		}

		// Create client directly for testing
		clientConfig := &client.CloudClientConfig{
			Host:       helpers.GetTestHost(),
			Token:      helpers.GetTestToken(),
			APIVersion: helpers.GetTestAPIVersion(),
		}
		testClient := client.NewCloudClient(clientConfig)

		permissionSystemID := rs.Primary.Attributes["permission_system_id"]
		if permissionSystemID == "" {
			return fmt.Errorf("No Permission System ID is set")
		}

		_, err := testClient.GetRole(context.Background(), permissionSystemID, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Role does not exist: %w", err)
		}

		return nil
	}
}

func testAccCheckRoleDestroy(s *terraform.State) error {
	// Create client directly for testing
	clientConfig := &client.CloudClientConfig{
		Host:       helpers.GetTestHost(),
		Token:      helpers.GetTestToken(),
		APIVersion: helpers.GetTestAPIVersion(),
	}
	testClient := client.NewCloudClient(clientConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "authzed_role" {
			continue
		}

		permissionSystemID := rs.Primary.Attributes["permission_system_id"]
		if permissionSystemID == "" {
			continue
		}

		_, err := testClient.GetRole(context.Background(), permissionSystemID, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Role still exists: %s", rs.Primary.ID)
		}

		// Verify it's actually a 404 error, not another error
		if !helpers.IsNotFoundError(err) {
			return fmt.Errorf("Unexpected error checking role destruction: %w", err)
		}
	}

	return nil
}

func testAccRoleImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		permissionSystemID := rs.Primary.Attributes["permission_system_id"]
		roleID := rs.Primary.ID

		if permissionSystemID == "" || roleID == "" {
			return "", fmt.Errorf("Permission system ID or role ID not set")
		}

		return fmt.Sprintf("%s:%s", permissionSystemID, roleID), nil
	}
}
