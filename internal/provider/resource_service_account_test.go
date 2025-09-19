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

func TestAccAuthzedServiceAccount_basic(t *testing.T) {
	resourceName := "authzed_service_account.test"
	name := helpers.GenerateTestID("acc-sa")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: helpers.BuildServiceAccountConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "permission_system_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccServiceAccountImportStateIdFunc(resourceName),
				ImportStateVerifyIgnore: []string{
					"etag",
					"updated_at",
					"updater",
				},
			},
		},
	})
}
func TestAccAuthzedServiceAccount_update(t *testing.T) {
	resourceName := "authzed_service_account.test"
	name := helpers.GenerateTestID("acc-sa-update")
	updatedDesc := "Updated description"

	initial := fmt.Sprintf(`
%s

resource "authzed_service_account" "test" {
  name                 = %q
  permission_system_id = %q
  description          = "Initial description"
}
`, helpers.BuildProviderConfig(), name, helpers.GetTestPermissionSystemID())

	updated := fmt.Sprintf(`
%s

resource "authzed_service_account" "test" {
  name                 = %q
  permission_system_id = %q
  description          = %q
}
`, helpers.BuildProviderConfig(), name, helpers.GetTestPermissionSystemID(), updatedDesc)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Initial description"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", updatedDesc),
				),
			},
		},
	})
}

func testAccCheckServiceAccountExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service Account ID is set")
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

		_, err := testClient.GetServiceAccount(context.Background(), permissionSystemID, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Service Account does not exist: %v", err)
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

		_, err := testClient.GetServiceAccount(context.Background(), permissionSystemID, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Service Account still exists: %s", rs.Primary.ID)
		}

		// Verify it's actually a 404 error, not another error
		if !helpers.IsNotFoundError(err) {
			return fmt.Errorf("Unexpected error checking service account destruction: %v", err)
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
		psID := rs.Primary.Attributes["permission_system_id"]
		id := rs.Primary.ID
		if psID == "" || id == "" {
			return "", fmt.Errorf("missing permission_system_id or id")
		}
		return fmt.Sprintf("%s:%s", psID, id), nil
	}
}
