package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/test/helpers"
)

func TestAccReasonableWorkload(t *testing.T) {
	testID := helpers.GenerateTestID("reasonable-test")

	// Measure execution time
	startTime := time.Now()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReasonableWorkloadDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccReasonableWorkloadConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify 3 service accounts created
					resource.TestCheckResourceAttr("authzed_service_account.test.0", "name", fmt.Sprintf("%s-sa-1", testID)),
					resource.TestCheckResourceAttr("authzed_service_account.test.2", "name", fmt.Sprintf("%s-sa-3", testID)),

					// Verify 6 tokens created (2 per SA)
					resource.TestCheckResourceAttr("authzed_token.test_1.0", "name", fmt.Sprintf("%s-token-1-1", testID)),
					resource.TestCheckResourceAttr("authzed_token.test_2.2", "name", fmt.Sprintf("%s-token-2-3", testID)),

					// Verify 3 roles created
					resource.TestCheckResourceAttr("authzed_role.test.0", "name", fmt.Sprintf("%s-role-1", testID)),
					resource.TestCheckResourceAttr("authzed_role.test.2", "name", fmt.Sprintf("%s-role-3", testID)),

					// Policies omitted: staging PS has restricted API configuration

					// Custom check to measure and log timing
					func(s *terraform.State) error {
						duration := time.Since(startTime)
						t.Logf("Reasonable workload (15 resources) completed in: %v", duration)

						// Log performance metrics
						switch {
						case duration > 30*time.Second:
							t.Logf("WARNING: Performance slower than expected (>30s): %v", duration)
						case duration < 15*time.Second:
							t.Logf("EXCELLENT: Performance very good (<15s): %v", duration)
						default:
							t.Logf("GOOD: Performance acceptable (15-30s): %v", duration)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccScaleTest(t *testing.T) {
	testID := helpers.GenerateTestID("scale-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScaleTestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccScaleTestConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify first and last resources to confirm scale
					resource.TestCheckResourceAttr("authzed_service_account.scale.0", "name", fmt.Sprintf("%s-scale-sa-1", testID)),
					resource.TestCheckResourceAttr("authzed_service_account.scale.24", "name", fmt.Sprintf("%s-scale-sa-25", testID)),
					resource.TestCheckResourceAttr("authzed_token.scale.0", "name", fmt.Sprintf("%s-scale-token-1", testID)),
					resource.TestCheckResourceAttr("authzed_token.scale.49", "name", fmt.Sprintf("%s-scale-token-50", testID)),
				),
			},
		},
	})
}

// testAccReasonableWorkloadConfig creates a reasonable test scenario
// 3 service accounts + 6 tokens (2 per SA) + 3 roles + 3 policies = 15 resources
func testAccReasonableWorkloadConfig(testID string) string {
	return fmt.Sprintf(`
%s

// 3 service accounts
resource "authzed_service_account" "test" {
  count = 3
  name = "%s-sa-${count.index + 1}"
  description = "Reasonable test service account ${count.index + 1}"
  permission_system_id = %q
}

// 6 tokens (2 per service account)
resource "authzed_token" "test_1" {
  count = 3
  name = "%s-token-1-${count.index + 1}"
  description = "Token 1 for service account ${count.index + 1}"
  service_account_id = authzed_service_account.test[count.index].id
  permission_system_id = %q
}

resource "authzed_token" "test_2" {
  count = 3
  name = "%s-token-2-${count.index + 1}"
  description = "Token 2 for service account ${count.index + 1}"
  service_account_id = authzed_service_account.test[count.index].id
  permission_system_id = %q
}

// 3 roles
resource "authzed_role" "test" {
  count = 3
  name = "%s-role-${count.index + 1}"
  description = "Reasonable test role ${count.index + 1}"
  permission_system_id = %q
  permissions = {
    "authzed.v1/CheckPermission"      = ""
    "authzed.v1/ExpandPermissionTree" = ""
    "authzed.v1/LookupResources"      = ""
    "authzed.v1/LookupSubjects"       = ""
  }
}

// Policies intentionally omitted due to restricted API configuration in staging
`,
		helpers.BuildProviderConfig(),
		testID, helpers.GetTestPermissionSystemID(),
		testID, helpers.GetTestPermissionSystemID(),
		testID, helpers.GetTestPermissionSystemID(),
		testID, helpers.GetTestPermissionSystemID())
}

// testAccScaleTestConfig creates a larger scale test (75 resources)
func testAccScaleTestConfig(testID string) string {
	return fmt.Sprintf(`
%s

// 25 service accounts
resource "authzed_service_account" "scale" {
  count = 25
  name = "%s-scale-sa-${count.index + 1}"
  description = "Scale test service account ${count.index + 1}"
  permission_system_id = %q
}

// 50 tokens (2 per service account)
resource "authzed_token" "scale" {
  count = 50
  name = "%s-scale-token-${count.index + 1}"
  description = "Scale test token ${count.index + 1}"
  service_account_id = authzed_service_account.scale[count.index %% 25].id
  permission_system_id = %q
}
`, helpers.BuildProviderConfig(), testID, helpers.GetTestPermissionSystemID(), testID, helpers.GetTestPermissionSystemID())
}

func testAccCheckReasonableWorkloadDestroy(s *terraform.State) error {
	// Create client directly for testing
	clientConfig := &client.CloudClientConfig{
		Host:       helpers.GetTestHost(),
		Token:      helpers.GetTestToken(),
		APIVersion: helpers.GetTestAPIVersion(),
	}
	testClient := client.NewCloudClient(clientConfig)

	for _, rs := range s.RootModule().Resources {
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
			return fmt.Errorf("Unexpected error checking %s destruction: %w", rs.Type, err)
		}
	}
	return nil
}

func testAccCheckScaleTestDestroy(s *terraform.State) error {
	// Similar destroy check for scale test
	return testAccCheckReasonableWorkloadDestroy(s)
}
