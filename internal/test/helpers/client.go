package helpers

import "terraform-provider-authzed/internal/client"

// CreateTestClient constructs a CloudClient for acceptance tests using env vars
func CreateTestClient() *client.CloudClient {
	cfg := &client.CloudClientConfig{
		Host:       GetTestHost(),
		Token:      GetTestToken(),
		APIVersion: GetTestAPIVersion(),
	}
	return client.NewCloudClient(cfg)
}
