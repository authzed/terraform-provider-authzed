package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-authzed/internal/test/helpers"
)

// testAccMaterializePreCheck skips Materialize acceptance tests unless the
// Materialize-specific environment is configured, on top of the standard
// acceptance env validation.
func testAccMaterializePreCheck(t *testing.T) {
	testAccPreCheck(t)
	if helpers.GetTestDeploymentID() == "" {
		t.Skip("AUTHZED_DEPLOYMENT_ID not set; skipping materialize deployment acceptance tests")
	}
	if len(helpers.GetTestMaterializeWatchedPermissions()) == 0 {
		t.Skip("AUTHZED_MATERIALIZE_WATCHED_PERMISSIONS not set; skipping materialize deployment acceptance tests")
	}
}

func testAccCheckMaterializeDeploymentDestroy(s *terraform.State) error {
	testClient := helpers.CreateTestClient()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "authzed_materialize_deployment" {
			continue
		}
		_, err := testClient.GetMaterializeDeployment(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("materialize deployment %s still exists", rs.Primary.ID)
		}
		// A deleted deployment reads as 404 — or 403 on API versions where
		// deletion removes the authorization relationships.
		if !materializeDeploymentGone(err) {
			return fmt.Errorf("unexpected error checking materialize deployment %s: %w", rs.Primary.ID, err)
		}
	}
	return nil
}

func TestAccAuthzedMaterializeDeployment_basic(t *testing.T) {
	name := helpers.GenerateTestID("tf-test-mat")
	watched := helpers.GetTestMaterializeWatchedPermissions()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccMaterializePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMaterializeDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: helpers.BuildMaterializeDeploymentConfigWithOptions(name, watched, 1, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("authzed_materialize_deployment.test", "id"),
					resource.TestCheckResourceAttr("authzed_materialize_deployment.test", "name", name),
					resource.TestCheckResourceAttr("authzed_materialize_deployment.test", "permission_system_id", helpers.GetTestPermissionSystemID()),
					resource.TestCheckResourceAttr("authzed_materialize_deployment.test", "deployment_id", helpers.GetTestDeploymentID()),
					resource.TestCheckResourceAttr("authzed_materialize_deployment.test", "watched_permissions.#", fmt.Sprintf("%d", len(watched))),
					resource.TestCheckResourceAttr("authzed_materialize_deployment.test", "replicas", "1"),
					resource.TestCheckResourceAttr("authzed_materialize_deployment.test", "accelerated_queries", "false"),
				),
			},
			{
				ResourceName:            "authzed_materialize_deployment.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func TestAccAuthzedMaterializeDeployment_update(t *testing.T) {
	watched := helpers.GetTestMaterializeWatchedPermissions()
	if len(watched) < 2 {
		t.Skip("need at least 2 entries in AUTHZED_MATERIALIZE_WATCHED_PERMISSIONS for the update test")
	}
	name := helpers.GenerateTestID("tf-test-mat-upd")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccMaterializePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMaterializeDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: helpers.BuildMaterializeDeploymentConfig(name, watched[:1]),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("authzed_materialize_deployment.test", "watched_permissions.#", "1"),
				),
			},
			{
				Config: helpers.BuildMaterializeDeploymentConfig(name, watched),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("authzed_materialize_deployment.test", "watched_permissions.#", fmt.Sprintf("%d", len(watched))),
				),
			},
		},
	})
}
