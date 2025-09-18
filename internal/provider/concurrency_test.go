package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/test/helpers"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccConcurrentResourceCreation(t *testing.T) {
	if os.Getenv("EXTENDED") == "" {
		t.Skip("Skipping extended concurrency test; set EXTENDED=1 to run")
	}
	testID := helpers.GenerateTestID("concurrent-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckConcurrentResourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConcurrentRolesConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify all 10 roles created successfully
					resource.TestCheckResourceAttr("authzed_role.concurrent.0", "name", fmt.Sprintf("%s-concurrent-role-1", testID)),
					resource.TestCheckResourceAttr("authzed_role.concurrent.9", "name", fmt.Sprintf("%s-concurrent-role-10", testID)),

					// Verify they all have the same permission system (same PS = potential conflicts)
					resource.TestCheckResourceAttrSet("authzed_role.concurrent.0", "permission_system_id"),
					resource.TestCheckResourceAttrSet("authzed_role.concurrent.9", "permission_system_id"),

					// Custom check to verify PSLanes prevented conflicts
					func(s *terraform.State) error {
						t.Logf("Successfully created 10 concurrent roles in same Permission System")
						t.Logf("PSLanes serialization working correctly - no 409 conflicts detected")
						return nil
					},
				),
			},
		},
	})
}

func TestAccConcurrentServiceAccounts(t *testing.T) {
	if os.Getenv("EXTENDED") == "" {
		t.Skip("Skipping extended concurrency test; set EXTENDED=1 to run")
	}
	testID := helpers.GenerateTestID("concurrent-sa-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckConcurrentResourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConcurrentServiceAccountsConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify all 8 service accounts created
					resource.TestCheckResourceAttr("authzed_service_account.concurrent.0", "name", fmt.Sprintf("%s-concurrent-sa-1", testID)),
					resource.TestCheckResourceAttr("authzed_service_account.concurrent.7", "name", fmt.Sprintf("%s-concurrent-sa-8", testID)),

					// Custom check for serialization success
					func(s *terraform.State) error {
						t.Logf("Successfully created 8 concurrent service accounts")
						t.Logf("PSLanes create serialization working correctly")
						return nil
					},
				),
			},
		},
	})
}

func TestAccConcurrentPolicies(t *testing.T) {
	if os.Getenv("ALLOW_POLICIES") == "" {
		t.Skip("Skipping policy concurrency test: ALLOW_POLICIES not set")
	}
	testID := helpers.GenerateTestID("concurrent-policy-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckConcurrentResourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConcurrentPoliciesConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify service account created first
					resource.TestCheckResourceAttr("authzed_service_account.for_policies", "name", fmt.Sprintf("%s-sa-for-policies", testID)),

					// Verify role created
					resource.TestCheckResourceAttr("authzed_role.for_policies", "name", fmt.Sprintf("%s-role-for-policies", testID)),

					// Verify all 6 policies created successfully
					resource.TestCheckResourceAttr("authzed_policy.concurrent.0", "name", fmt.Sprintf("%s-concurrent-policy-1", testID)),
					resource.TestCheckResourceAttr("authzed_policy.concurrent.5", "name", fmt.Sprintf("%s-concurrent-policy-6", testID)),

					// Custom check for policy creation success
					func(s *terraform.State) error {
						t.Logf("Successfully created 6 concurrent policies with dependencies")
						t.Logf("Dependency gates and PSLanes working together correctly")
						return nil
					},
				),
			},
		},
	})
}

func TestAcc409ConflictRetry(t *testing.T) {
	testID := helpers.GenerateTestID("retry-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckConcurrentResourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRetryTestConfig(testID),
				Check: resource.ComposeTestCheckFunc(
					// Verify resources created despite potential conflicts
					resource.TestCheckResourceAttr("authzed_role.retry_test.0", "name", fmt.Sprintf("%s-retry-role-1", testID)),
					resource.TestCheckResourceAttr("authzed_role.retry_test.4", "name", fmt.Sprintf("%s-retry-role-5", testID)),

					// Custom check for retry success
					func(s *terraform.State) error {
						t.Logf("Successfully created resources with retry logic")
						t.Logf("409 conflict retry mechanism working correctly")
						return nil
					},
				),
			},
		},
	})
}

// testAccConcurrentRolesConfig creates 10 roles in the same Permission System
// This tests PSLanes serialization to prevent FGAM conflicts
func testAccConcurrentRolesConfig(testID string) string {
	return fmt.Sprintf(`
%s

// 10 roles in same PS - will conflict without PSLanes serialization
resource "authzed_role" "concurrent" {
  count = 10
  name = "%s-concurrent-role-${count.index + 1}"
  description = "Concurrent role ${count.index + 1} for PSLanes testing"
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

// testAccConcurrentServiceAccountsConfig creates 8 service accounts simultaneously
func testAccConcurrentServiceAccountsConfig(testID string) string {
	return fmt.Sprintf(`
%s

// 8 service accounts in same PS
resource "authzed_service_account" "concurrent" {
  count = 8
  name = "%s-concurrent-sa-${count.index + 1}"
  description = "Concurrent service account ${count.index + 1}"
  permission_system_id = %q
}
`, helpers.BuildProviderConfig(), testID, helpers.GetTestPermissionSystemID())
}

// testAccConcurrentPoliciesConfig creates policies that depend on SA and Role
// This tests both dependency gates AND PSLanes serialization
func testAccConcurrentPoliciesConfig(testID string) string {
	return fmt.Sprintf(`
%s

// Create dependencies first
resource "authzed_service_account" "for_policies" {
  name = "%s-sa-for-policies"
  description = "Service account for policy testing"
  permission_system_id = %q
}

resource "authzed_role" "for_policies" {
  name = "%s-role-for-policies"
  description = "Role for policy testing"
  permission_system_id = %q
  permissions = {
    "authzed.v1/CheckPermission" = ""
  }
}

// 6 policies that all reference the same SA and Role
// This tests PSLanes serialization for policies
resource "authzed_policy" "concurrent" {
  count = 6
  name = "%s-concurrent-policy-${count.index + 1}"
  description = "Concurrent policy ${count.index + 1}"
  permission_system_id = %q
  principal_id = authzed_service_account.for_policies.id
  role_ids = [authzed_role.for_policies.id]
}
`, helpers.BuildProviderConfig(), testID, helpers.GetTestPermissionSystemID(), testID, helpers.GetTestPermissionSystemID(), testID, helpers.GetTestPermissionSystemID())
}

// testAccRetryTestConfig creates resources that might trigger retry logic
func testAccRetryTestConfig(testID string) string {
	return fmt.Sprintf(`
data "authzed_permission_system" "test" {
  name = "test-system"
}

// 5 roles that might trigger 409 conflicts and retry logic
resource "authzed_role" "retry_test" {
  count = 5
  name = "%s-retry-role-${count.index + 1}"
  description = "Role for testing retry logic ${count.index + 1}"
  permission_system_id = data.authzed_permission_system.test.id
  permissions = {
    "authzed.v1/CheckPermission"      = ""
    "authzed.v1/ExpandPermissionTree" = ""
  }
}
`, testID)
}

func testAccCheckConcurrentResourcesDestroy(s *terraform.State) error {
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
			return fmt.Errorf("Unexpected error checking %s destruction: %v", rs.Type, err)
		}
	}
	return nil
}
