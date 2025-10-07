package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-authzed/internal/test/helpers"
)

// Proves PSLanes serialization for roles in same Permission System (small N)
func TestAccConcurrentRoles_Serialized(t *testing.T) {
	name := helpers.GenerateTestID("acc-concurrent-role")
	config := fmt.Sprintf(`
%s

resource "authzed_role" "concurrent" {
  count = 3
  name  = "%s-${count.index + 1}"
  permission_system_id = %q
  permissions = {
    "authzed.v1/CheckPermission" = ""
  }
}
`, helpers.BuildProviderConfig(), name, helpers.GetTestPermissionSystemID())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             config,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("authzed_role.concurrent.0", "name", fmt.Sprintf("%s-1", name)),
					resource.TestCheckResourceAttr("authzed_role.concurrent.2", "name", fmt.Sprintf("%s-3", name)),
				),
			},
		},
	})
}

// Proves PSLanes serialization for tokens under the same Permission System (small N)
func TestAccConcurrentTokens_Serialized(t *testing.T) {
	serviceAccountName := helpers.GenerateTestID("acc-concurrent-token-sa")
	name := helpers.GenerateTestID("acc-concurrent-token")

	// Step 0: only service account (allow server-managed fields to settle)
	configSA := fmt.Sprintf(`
%s

resource "authzed_service_account" "sa" {
  name                 = %q
  permission_system_id = %q
}
`, helpers.BuildProviderConfig(), serviceAccountName, helpers.GetTestPermissionSystemID())

	// Step 1: add t1
	config1 := fmt.Sprintf(`
%s

resource "authzed_service_account" "sa" {
  name                 = %q
  permission_system_id = %q
}

resource "authzed_token" "t1" {
  name                 = "%s-1"
  permission_system_id = %q
  service_account_id   = authzed_service_account.sa.id
}
`, helpers.BuildProviderConfig(), serviceAccountName, helpers.GetTestPermissionSystemID(), name, helpers.GetTestPermissionSystemID())

	config2 := fmt.Sprintf(`
%s

resource "authzed_service_account" "sa" {
  name                 = %q
  permission_system_id = %q
}

resource "authzed_token" "t1" {
  name                 = "%s-1"
  permission_system_id = %q
  service_account_id   = authzed_service_account.sa.id
}

resource "authzed_token" "t2" {
  name                 = "%s-2"
  permission_system_id = %q
  service_account_id   = authzed_service_account.sa.id
}
`, helpers.BuildProviderConfig(), serviceAccountName, helpers.GetTestPermissionSystemID(), name, helpers.GetTestPermissionSystemID(), name, helpers.GetTestPermissionSystemID())

	config3 := fmt.Sprintf(`
%s

resource "authzed_service_account" "sa" {
  name                 = %q
  permission_system_id = %q
}

resource "authzed_token" "t1" {
  name                 = "%s-1"
  permission_system_id = %q
  service_account_id   = authzed_service_account.sa.id
}

resource "authzed_token" "t2" {
  name                 = "%s-2"
  permission_system_id = %q
  service_account_id   = authzed_service_account.sa.id
}

resource "authzed_token" "t3" {
  name                 = "%s-3"
  permission_system_id = %q
  service_account_id   = authzed_service_account.sa.id
}
`, helpers.BuildProviderConfig(), serviceAccountName, helpers.GetTestPermissionSystemID(), name, helpers.GetTestPermissionSystemID(), name, helpers.GetTestPermissionSystemID(), name, helpers.GetTestPermissionSystemID())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configSA,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("authzed_service_account.sa", "name", serviceAccountName),
				),
			},
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("authzed_token.t1", "name", fmt.Sprintf("%s-1", name)),
				),
			},
			{
				Config: config2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("authzed_token.t1", "name", fmt.Sprintf("%s-1", name)),
					resource.TestCheckResourceAttr("authzed_token.t2", "name", fmt.Sprintf("%s-2", name)),
				),
			},
			{
				Config: config3,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("authzed_token.t1", "name", fmt.Sprintf("%s-1", name)),
					resource.TestCheckResourceAttr("authzed_token.t2", "name", fmt.Sprintf("%s-2", name)),
					resource.TestCheckResourceAttr("authzed_token.t3", "name", fmt.Sprintf("%s-3", name)),
				),
			},
		},
	})
}
