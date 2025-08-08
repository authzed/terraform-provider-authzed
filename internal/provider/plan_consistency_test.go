package provider

import (
	"fmt"
	"testing"

	"terraform-provider-authzed/internal/test/helpers"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestPlanConsistency_PolicyImmutableFields validates plan modifier behavior for policy updates
func TestPlanConsistency_PolicyImmutableFields(t *testing.T) {
	testID := helpers.GenerateTestID("test-policy-plan-consistency")
	roleName := fmt.Sprintf("%s-role", testID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			// Create initial policy
			{
				Config: testAccPolicyConfig_basic(testID, roleName),
			},
			// Plan an update to test plan modifier behavior
			{
				Config:             testAccPolicyConfig_update(testID, roleName, "Updated description for plan consistency test"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestPlanConsistency_ServiceAccountImmutableFields tests the same pattern for service accounts
func TestPlanConsistency_ServiceAccountImmutableFields(t *testing.T) {
	testID := helpers.GenerateTestID("test-sa-plan-consistency")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			// Create initial service account
			{
				Config: testAccServiceAccountConfig_basic(testID),
			},
			// Plan an update to test plan modifier behavior
			{
				Config:             testAccServiceAccountConfig_updated(testID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestPlanConsistency_MultipleUpdates tests sequential updates to catch edge cases
func TestPlanConsistency_MultipleUpdates(t *testing.T) {
	testID := helpers.GenerateTestID("test-policy-multiple-updates")
	roleName := fmt.Sprintf("%s-role", testID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			// Create initial policy
			{
				Config: testAccPolicyConfig_basic(testID, roleName),
			},
			// First update
			{
				Config: testAccPolicyConfig_update(testID, roleName, "First update"),
			},
			// Second update
			{
				Config: testAccPolicyConfig_update(testID, roleName, "Second update"),
			},
			// Third update with different field
			{
				Config: testAccPolicyConfig_updateName(testID, roleName, "updated-name", "Third update"),
			},
		},
	})
}

// Helper config function for name updates
func testAccPolicyConfig_updateName(policyName, roleName, updatedName, description string) string {
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
  name                 = %[3]q
  description          = %[5]q
  permission_system_id = %[4]q
  principal_id         = "test-principal"
  role_ids             = [authzed_role.test.id]
}
`,
		policyName,
		roleName,
		updatedName,
		helpers.GetTestPermissionSystemID(),
		description,
	)
}
