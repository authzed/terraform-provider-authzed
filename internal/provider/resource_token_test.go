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

func TestAccAuthzedToken_basic(t *testing.T) {
	resourceName := "authzed_token.test"
	serviceAccountResourceName := "authzed_service_account.test"
	testID := helpers.GenerateTestID("test-token")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTokenDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTokenConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenExists(resourceName),
					testAccCheckServiceAccountExists(serviceAccountResourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "description", "Test token description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "permission_system_id"),
					resource.TestCheckResourceAttrSet(resourceName, "service_account_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "hash"),
					resource.TestCheckResourceAttrSet(resourceName, "plain_text"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccTokenImportStateIdFunc(resourceName),
				ImportStateVerifyIgnore: []string{
					"plain_text", // plain_text is only available during creation
				},
			},
		},
	})
}

func TestAccAuthzedToken_withServiceAccount(t *testing.T) {
	resourceName := "authzed_token.test"
	serviceAccountResourceName := "authzed_service_account.test"
	testID := helpers.GenerateTestID("test-token-sa")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTokenDestroy,
		Steps: []resource.TestStep{
			// Create token with service account relationship
			{
				Config: testAccTokenConfig_withServiceAccount(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenExists(resourceName),
					testAccCheckServiceAccountExists(serviceAccountResourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID+"-token"),
					resource.TestCheckResourceAttr(resourceName, "description", "Token for service account"),
					resource.TestCheckResourceAttrPair(resourceName, "service_account_id", serviceAccountResourceName, "id"),
					resource.TestCheckResourceAttrPair(resourceName, "permission_system_id", serviceAccountResourceName, "permission_system_id"),
				),
			},
		},
	})
}

func TestAccAuthzedToken_update(t *testing.T) {
	resourceName := "authzed_token.test"
	testID := helpers.GenerateTestID("test-token-update")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTokenDestroy,
		Steps: []resource.TestStep{
			// Create initial token
			{
				Config: testAccTokenConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test token description"),
				),
			},
			// Update token description
			{
				Config: testAccTokenConfig_updated(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated token description"),
				),
			},
		},
	})
}

func TestAccAuthzedToken_import(t *testing.T) {
	resourceName := "authzed_token.test"
	testID := helpers.GenerateTestID("test-token-import")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTokenDestroy,
		Steps: []resource.TestStep{
			// Create token
			{
				Config: testAccTokenConfig_basic(testID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTokenExists(resourceName),
				),
			},
			// Test import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccTokenImportStateIdFunc(resourceName),
				ImportStateVerifyIgnore: []string{
					"plain_text", // plain_text is only available during creation
				},
			},
		},
	})
}

func TestAccAuthzedToken_validation(t *testing.T) {
	testID := helpers.GenerateTestID("test-token-validation")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test invalid service account ID
			{
				Config:      testAccTokenConfig_invalidServiceAccountID(testID),
				ExpectError: regexp.MustCompile("Client Error"),
			},
			// Test empty name
			{
				Config:      testAccTokenConfig_emptyName(),
				ExpectError: regexp.MustCompile("Inappropriate value for attribute \"name\""),
			},
		},
	})
}

// Helper functions

func testAccCheckTokenExists(resourceName string) resource.TestCheckFunc {
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
		serviceAccountID := rs.Primary.Attributes["service_account_id"]
		if permissionSystemID == "" || serviceAccountID == "" {
			return fmt.Errorf("Permission system ID or service account ID not set")
		}

		_, err := testClient.GetToken(permissionSystemID, serviceAccountID, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving token: %s", err)
		}

		return nil
	}
}

func testAccCheckTokenDestroy(s *terraform.State) error {
	// Create client directly for testing
	clientConfig := &client.CloudClientConfig{
		Host:       helpers.GetTestHost(),
		Token:      helpers.GetTestToken(),
		APIVersion: helpers.GetTestAPIVersion(),
	}
	testClient := client.NewCloudClient(clientConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "authzed_token" {
			continue
		}

		permissionSystemID := rs.Primary.Attributes["permission_system_id"]
		serviceAccountID := rs.Primary.Attributes["service_account_id"]
		if permissionSystemID == "" || serviceAccountID == "" {
			continue
		}

		_, err := testClient.GetToken(permissionSystemID, serviceAccountID, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("token still exists: %s", rs.Primary.ID)
		}

		// Verify it's actually a 404 error, not another error
		if !helpers.IsNotFoundError(err) {
			return fmt.Errorf("unexpected error checking token destruction: %v", err)
		}
	}

	return nil
}

func testAccTokenImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		permissionSystemID := rs.Primary.Attributes["permission_system_id"]
		serviceAccountID := rs.Primary.Attributes["service_account_id"]
		if permissionSystemID == "" || serviceAccountID == "" {
			return "", fmt.Errorf("Permission system ID or service account ID not set")
		}

		return fmt.Sprintf("%s:%s:%s", permissionSystemID, serviceAccountID, rs.Primary.ID), nil
	}
}

// Configuration templates

func testAccTokenConfig_basic(tokenName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Test service account for token"
  permission_system_id = %[2]q
}

resource "authzed_token" "test" {
  name                 = %[1]q
  description          = "Test token description"
  permission_system_id = %[2]q
  service_account_id   = authzed_service_account.test.id
}
`,
		tokenName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccTokenConfig_withServiceAccount(baseName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Service account for token testing"
  permission_system_id = %[2]q
}

resource "authzed_token" "test" {
  name                 = "%[1]s-token"
  description          = "Token for service account"
  permission_system_id = %[2]q
  service_account_id   = authzed_service_account.test.id
}
`,
		baseName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccTokenConfig_updated(tokenName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Test service account for token"
  permission_system_id = %[2]q
}

resource "authzed_token" "test" {
  name                 = %[1]q
  description          = "Updated token description"
  permission_system_id = %[2]q
  service_account_id   = authzed_service_account.test.id
}
`,
		tokenName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccTokenConfig_invalidServiceAccountID(tokenName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_token" "test" {
  name                 = %[1]q
  description          = "Test token with invalid service account ID"
  permission_system_id = %[2]q
  service_account_id   = "invalid-sa-id"
}
`,
		tokenName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccTokenConfig_emptyName() string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = "test-sa-for-empty-token"
  description          = "Test service account for empty token name test"
  permission_system_id = %[1]q
}

resource "authzed_token" "test" {
  name                 = ""
  description          = "Test token with empty name"
  permission_system_id = %[1]q
  service_account_id   = authzed_service_account.test.id
}
`,
		helpers.GetTestPermissionSystemID(),
	)
}
