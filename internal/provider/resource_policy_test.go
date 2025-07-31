package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/test/helpers"
)

func TestAccAuthzedPolicy_basic(t *testing.T) {
	resourceName := "authzed_policy.test"
	testID := helpers.GenerateTestID("test-policy")
	roleName := fmt.Sprintf("%s-role", testID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicyConfig_basic(testID, roleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "description", "Test policy description"),
					resource.TestCheckResourceAttr(resourceName, "principal_id", "test-principal"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "permission_system_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "creator"),
					resource.TestCheckResourceAttrSet(resourceName, "etag"),
					resource.TestCheckResourceAttr(resourceName, "role_ids.#", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccPolicyImportStateIdFunc(resourceName),
			},
		},
	})
}

func TestAccAuthzedPolicy_import(t *testing.T) {
	resourceName := "authzed_policy.test"
	testID := helpers.GenerateTestID("test-policy-import")
	roleName := fmt.Sprintf("%s-role", testID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			// Create policy
			{
				Config: testAccPolicyConfig_basic(testID, roleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPolicyExists(resourceName),
				),
			},
			// Test import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccPolicyImportStateIdFunc(resourceName),
			},
		},
	})
}

func TestAccAuthzedPolicy_update(t *testing.T) {
	resourceName := "authzed_policy.test"
	testID := helpers.GenerateTestID("test-policy-update")
	roleName := fmt.Sprintf("%s-role", testID)
	updatedDescription := "Updated test policy description"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			// Create initial policy
			{
				Config: testAccPolicyConfig_basic(testID, roleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "Test policy description"),
				),
			},
			// Update policy
			{
				Config: testAccPolicyConfig_update(testID, roleName, updatedDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", updatedDescription),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
				),
			},
		},
	})
}

func TestAccAuthzedPolicy_validation(t *testing.T) {
	testID := helpers.GenerateTestID("test-policy-validation")
	roleName := fmt.Sprintf("%s-role", testID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test invalid permission system ID
			{
				Config:      testAccPolicyConfig_invalidPermissionSystemID(testID, roleName),
				ExpectError: regexp.MustCompile("Client Error"),
			},
			// Test empty name
			{
				Config:      testAccPolicyConfig_emptyName(roleName),
				ExpectError: regexp.MustCompile("Inappropriate value for attribute \"name\""),
			},
			// Test empty role IDs
			{
				Config:      testAccPolicyConfig_emptyRoleIDs(testID),
				ExpectError: regexp.MustCompile("Inappropriate value for attribute \"role_ids\""),
			},
		},
	})
}

func TestAccAuthzedPolicy_noDrift(t *testing.T) {
	resourceName := "authzed_policy.test"
	testID := helpers.GenerateTestID("test-policy-drift")
	roleName := fmt.Sprintf("%s-role", testID)
	updatedDescription := "Updated test policy description"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			// Create initial policy
			{
				Config: testAccPolicyConfig_basic(testID, roleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "description", "Test policy description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "creator"),
					resource.TestCheckResourceAttrSet(resourceName, "etag"),
				),
			},
			// Verify no drift on second plan - this is the key test
			{
				Config:   testAccPolicyConfig_basic(testID, roleName),
				PlanOnly: true,
				// If there's drift in computed fields (id, created_at, creator, etag),
				// this step will fail because Terraform will detect changes
			},
			// Update mutable fields and verify computed fields don't drift
			{
				Config: testAccPolicyConfig_update(testID, roleName, updatedDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", testID),
					resource.TestCheckResourceAttr(resourceName, "description", updatedDescription),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "creator"),
					resource.TestCheckResourceAttrSet(resourceName, "etag"),
				),
			},
			{
				Config:   testAccPolicyConfig_update(testID, roleName, updatedDescription),
				PlanOnly: true,
				// This verifies that after an update, computed fields don't show as changing
			},
		},
	})
}

// Helper functions

func testAccCheckPolicyExists(resourceName string) resource.TestCheckFunc {
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

		_, err := testClient.GetPolicy(permissionSystemID, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error retrieving policy: %w", err)
		}

		return nil
	}
}

func testAccCheckPolicyDestroy(s *terraform.State) error {
	// Create client directly for testing
	clientConfig := &client.CloudClientConfig{
		Host:       helpers.GetTestHost(),
		Token:      helpers.GetTestToken(),
		APIVersion: helpers.GetTestAPIVersion(),
	}
	testClient := client.NewCloudClient(clientConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "authzed_policy" {
			continue
		}

		permissionSystemID := rs.Primary.Attributes["permission_system_id"]
		if permissionSystemID == "" {
			continue
		}

		_, err := testClient.GetPolicy(permissionSystemID, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Policy still exists: %s", rs.Primary.ID)
		}

		// Verify it's actually a 404 error, not another error
		if !helpers.IsNotFoundError(err) {
			return fmt.Errorf("Unexpected error checking policy destruction: %w", err)
		}
	}
	return nil
}

func testAccPolicyImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Resource not found: %s", resourceName)
		}

		permissionSystemID := rs.Primary.Attributes["permission_system_id"]
		policyID := rs.Primary.ID

		if permissionSystemID == "" || policyID == "" {
			return "", fmt.Errorf("Permission system ID or policy ID not set")
		}

		return fmt.Sprintf("%s:%s", permissionSystemID, policyID), nil
	}
}

// Configuration templates

func testAccPolicyConfig_basic(policyName, roleName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
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
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccPolicyConfig_update(policyName, roleName, description string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
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
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccPolicyConfig_invalidPermissionSystemID(policyName, roleName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
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
  description          = "Test policy with invalid permission system ID"
  permission_system_id = "invalid-ps-id"
  principal_id         = "test-principal"
  role_ids             = [authzed_role.test.id]
}
`,
		policyName,
		roleName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccPolicyConfig_emptyName(roleName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_role" "test" {
  name                 = %[1]q
  description          = "Test role for policy acceptance tests"
  permission_system_id = %[2]q
  permissions = {
    "authzed.v1/ReadSchema" = ""
  }
}

resource "authzed_policy" "test" {
  name                 = ""
  description          = "Test policy with empty name"
  permission_system_id = %[2]q
  principal_id         = "test-principal"
  role_ids             = [authzed_role.test.id]
}
`,
		roleName,
		helpers.GetTestPermissionSystemID(),
	)
}

func testAccPolicyConfig_emptyRoleIDs(policyName string) string {
	return helpers.BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_policy" "test" {
  name                 = %[1]q
  description          = "Test policy with empty role IDs"
  permission_system_id = %[2]q
  principal_id         = "test-principal"
  role_ids             = []
}
`,
		policyName,
		helpers.GetTestPermissionSystemID(),
	)
}
