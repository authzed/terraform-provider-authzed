package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-authzed/internal/test/helpers"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. It will be invoked for every Terraform
// command executed to create a provider server to which the CLI can reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"authzed": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates that all required environment variables are set
// for acceptance testing. It is called before each acceptance test to ensure
// the test environment is properly configured.
func testAccPreCheck(t *testing.T) {
	// Use helper function for validation
	if err := helpers.ValidateTestEnvironment(); err != nil {
		t.Fatal(err)
	}
}

// testAccProviderConfig returns a basic provider configuration for testing
func testAccProviderConfig() string {
	return helpers.BuildProviderConfig()
}

// TestProvider verifies that the provider can be instantiated
func TestProvider(t *testing.T) {
	p := New("dev")()
	if p == nil {
		t.Fatal("Provider should not be nil")
	}
}

// TestAccProvider verifies that the provider can be configured for acceptance testing
func TestAccProvider(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(),
			},
		},
	})
}
