package provider

import (
	"fmt"
	"regexp"
	"testing"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/test/helpers"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAuthzedServiceAccount_basic(t *testing.T) {
	resourceName := "authzed_service_account.test"
	testID := helpers.GenerateTestID("test-sa")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceAccountConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "description", "Test service account description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "permission_system_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "creator"),
					resource.TestCheckResourceAttrSet(resourceName, "etag"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccServiceAccountImportStateIdFunc(resourceName),
			},
		},
	})
}

func TestAccAuthzedServiceAccount_update(t *testing.T) {
	resourceName := "authzed_service_account.test"
	testID := helpers.GenerateTestID("test-sa-update")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			// Create initial service account
			{
				Config: testAccServiceAccountConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test service account description"),
				),
			},
			// Update service account description
			{
				Config: testAccServiceAccountConfig_updated(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated service account description"),
				),
			},
		},
	})
}

func TestAccAuthzedServiceAccount_import(t *testing.T) {
	resourceName := "authzed_service_account.test"
	testID := helpers.GenerateTestID("test-sa-import")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			// Create service account
			{
				Config: testAccServiceAccountConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(resourceName),
				),
			},
			// Test import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccServiceAccountImportStateIdFunc(resourceName),
			},
		},
	})
}

func TestAccAuthzedServiceAccount_validation(t *testing.T) {
	testID := helpers.GenerateTestID("test-sa-validation")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test invalid permission system ID
			{
				Config:      testAccServiceAccountConfig_invalidPermissionSystemID(testID),
				ExpectError: regexp.MustCompile("Client Error"),
			},
			// Test empty name
			{
				Config:      testAccServiceAccountConfig_emptyName(),
				ExpectError: regexp.MustCompile("Inappropriate value for attribute \"name\""),
			},
		},
	})
}

// Helper functions

func testAccCheckServiceAccountExists(resourceName string) resource.TestCheckFunc {
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

		_, err := testClient.GetServiceAccount(permissionSystemID, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving service account: %s", err)
		}

		return nil
	}
}

func testAccCheckServiceAccountDestroy(s *terraform.State) error {
	// Create client directly for testing
	clientConfig := &client.CloudClientConfig{
		Host:       helpers.GetTestHost(),
		Token:      helpers.GetTestToken(),
		APIVersion: helpers.GetTestAPIVersion(),
	}
	testClient := client.NewCloudClient(clientConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "authzed_service_account" {
			continue
		}

		permissionSystemID := rs.Primary.Attributes["permission_system_id"]
		if permissionSystemID == "" {
			continue
		}

		_, err := testClient.GetServiceAccount(permissionSystemID, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("service account still exists: %s", rs.Primary.ID)
		}

		// Verify it's actually a 404 error, not another error
		if !helpers.IsNotFoundError(err) {
			return fmt.Errorf("unexpected error checking service account destruction: %v", err)
		}
	}

	return nil
}

func testAccServiceAccountImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
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

func testAccServiceAccountConfig_basic(serviceAccountName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Test service account description"
  permission_system_id = %[2]q
}
`,
		serviceAccountName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccServiceAccountConfig_updated(serviceAccountName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Updated service account description"
  permission_system_id = %[2]q
}
`,
		serviceAccountName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccServiceAccountConfig_invalidPermissionSystemID(serviceAccountName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Test service account with invalid permission system ID"
  permission_system_id = "invalid-ps-id"
}
`,
		serviceAccountName,
	)
}

func testAccServiceAccountConfig_emptyName() string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = ""
  description          = "Test service account with empty name"
  permission_system_id = %[1]q
}
`,
		helpers.GetTestPermissionSystemID(),
	)
}
