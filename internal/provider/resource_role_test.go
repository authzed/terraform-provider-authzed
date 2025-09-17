package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/test/helpers"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
					resource.TestCheckResourceAttr(resourceName, "permissions.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/ReadSchema", ""),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccRoleImportStateIdFunc(resourceName),
				ImportStateVerifyIgnore: []string{
					"etag",
					"updated_at",
					"updater",
				},
				Check: resource.ComposeTestCheckFunc(
					// Verify ETag presence (not equality)
					resource.TestCheckResourceAttrSet(resourceName, "etag"),
					// Verify stable config fields match
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "permissions.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/ReadSchema", ""),
				),
			},
		},
	})
}

func TestAccAuthzedRole_updatePermissions(t *testing.T) {
	resourceName := "authzed_role.test"
	testID := helpers.GenerateTestID("test-role-update")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			// Create initial role with basic permissions
			{
				Config: testAccRoleConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "permissions.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/ReadSchema", ""),
				),
			},
			// Update role with additional permissions
			{
				Config: testAccRoleConfig_updatePermissions(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "permissions.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/ReadSchema", ""),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/ReadRelationships", ""),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/CheckPermission", "CheckPermissionRequest.permission == \"admin\""),
				),
			},
		},
	})
}

func TestAccAuthzedRole_complexPermissions(t *testing.T) {
	resourceName := "authzed_role.test"
	testID := helpers.GenerateTestID("test-role-complex")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			// Create role with complex permissions including CEL expressions
			{
				Config: testAccRoleConfig_complexPermissions(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "permissions.%", "5"),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/ReadSchema", ""),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/WriteSchema", ""),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/ReadRelationships", ""),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/WriteRelationships", ""),
					resource.TestCheckResourceAttr(resourceName, "permissions.authzed.v1/CheckPermission", "CheckPermissionRequest.permission == \"admin\" || CheckPermissionRequest.permission == \"read\""),
					testAccCheckRolePermissions(resourceName, map[string]string{
						"authzed.v1/ReadSchema":         "",
						"authzed.v1/WriteSchema":        "",
						"authzed.v1/ReadRelationships":  "",
						"authzed.v1/WriteRelationships": "",
						"authzed.v1/CheckPermission":    "CheckPermissionRequest.permission == \"admin\" || CheckPermissionRequest.permission == \"read\"",
					}),
				),
			},
			// ImportState testing for complex permissions
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccRoleImportStateIdFunc(resourceName),
				ImportStateVerifyIgnore: []string{
					"etag", // ETag changes between operations
				},
			},
		},
	})
}

func TestAccAuthzedRole_import(t *testing.T) {
	resourceName := "authzed_role.test"
	testID := helpers.GenerateTestID("test-role-import")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			// Create role
			{
				Config: testAccRoleConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
				),
			},
			// Test import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccRoleImportStateIdFunc(resourceName),
				ImportStateVerifyIgnore: []string{
					"etag", // ETag changes between operations
				},
			},
		},
	})
}

func TestAccAuthzedRole_validation(t *testing.T) {
	testID := helpers.GenerateTestID("test-role-validation")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test invalid permission system ID
			{
				Config:      testAccRoleConfig_invalidPermissionSystemID(testID),
				ExpectError: regexp.MustCompile("Client Error"),
			},
			// Test empty name
			{
				Config:      testAccRoleConfig_emptyName(),
				ExpectError: regexp.MustCompile("Inappropriate value for attribute \"name\""),
			},
			// Test empty permissions
			{
				Config:      testAccRoleConfig_emptyPermissions(testID),
				ExpectError: regexp.MustCompile("Inappropriate value for attribute \"permissions\""),
			},
		},
	})
}

func TestAccAuthzedRole_noDrift(t *testing.T) {
	resourceName := "authzed_role.test"
	testID := helpers.GenerateTestID("test-role-drift")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			// Create initial role
			{
				Config: testAccRoleConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "description", "Test role description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "creator"),
					resource.TestCheckResourceAttrSet(resourceName, "etag"),
				),
			},
			// Verify no drift on second plan. Critical path.
			{
				Config:   testAccRoleConfig_basic(testID),
				PlanOnly: true,
			},
			// Update permissions and verify computed fields don't drift
			{
				Config: testAccRoleConfig_updatePermissions(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "creator"),
					resource.TestCheckResourceAttrSet(resourceName, "etag"),
				),
			},
			// Verify no drift after update - computed fields should remain stable
			{
				Config:   testAccRoleConfig_updatePermissions(testID),
				PlanOnly: true,
				// This verifies that after an update, computed fields don't show as changing
			},
		},
	})
}

// Helper functions

func testAccCheckRoleExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource ID not set")
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
			return fmt.Errorf("Permission system ID not set")
		}

		_, err := testClient.GetRole(context.Background(), permissionSystemID, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving role: %s", err)
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
			return fmt.Errorf("role still exists: %s", rs.Primary.ID)
		}

		// Verify it's actually a 404 error, not another error
		if !helpers.IsNotFoundError(err) {
			return fmt.Errorf("unexpected error checking role destruction: %v", err)
		}
	}

	return nil
}

func testAccCheckRolePermissions(resourceName string, expectedPermissions map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource not found: %s", resourceName)
		}

		// Check each expected permission
		for permission, expectedValue := range expectedPermissions {
			attrKey := fmt.Sprintf("permissions.%s", permission)
			actualValue, exists := rs.Primary.Attributes[attrKey]
			if !exists {
				return fmt.Errorf("Permission %s not found in role", permission)
			}
			if actualValue != expectedValue {
				return fmt.Errorf("Permission %s: expected %q, got %q", permission, expectedValue, actualValue)
			}
		}

		return nil
	}
}

func testAccRoleImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		permissionSystemID := rs.Primary.Attributes["permission_system_id"]
		if permissionSystemID == "" {
			return "", fmt.Errorf("Permission system ID not set")
		}

		return fmt.Sprintf("%s:%s", permissionSystemID, rs.Primary.ID), nil
	}
}

// Configuration templates

func testAccRoleConfig_basic(roleName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
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
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccRoleConfig_updatePermissions(roleName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[1]q
  description          = "Test role with updated permissions"
  permission_system_id = %[2]q
  permissions = {
    "authzed.v1/ReadSchema"        = ""
    "authzed.v1/ReadRelationships" = ""
    "authzed.v1/CheckPermission"   = "CheckPermissionRequest.permission == \"admin\""
  }
}
`,
		roleName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccRoleConfig_complexPermissions(roleName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[1]q
  description          = "Test role with complex permissions"
  permission_system_id = %[2]q
  permissions = {
    "authzed.v1/ReadSchema"         = ""
    "authzed.v1/WriteSchema"        = ""
    "authzed.v1/ReadRelationships"  = ""
    "authzed.v1/WriteRelationships" = ""
    "authzed.v1/CheckPermission"    = "CheckPermissionRequest.permission == \"admin\" || CheckPermissionRequest.permission == \"read\""
  }
}
`,
		roleName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccRoleConfig_invalidPermissionSystemID(roleName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[1]q
  description          = "Test role with invalid permission system ID"
  permission_system_id = "invalid-ps-id"
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}
`,
		roleName,
	)
}

func testAccRoleConfig_emptyName() string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = ""
  description          = "Test role with empty name"
  permission_system_id = %[1]q
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}
`,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccRoleConfig_emptyPermissions(roleName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[1]q
  description          = "Test role with empty permissions"
  permission_system_id = %[2]q
  permissions = {}
}
`,
		roleName,
		helpers.GetTestPermissionSystemID(),
	)
}
