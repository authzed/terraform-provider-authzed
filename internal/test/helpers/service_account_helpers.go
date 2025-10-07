package helpers

import (
	"context"
	"fmt"
	"os"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
)

// BuildServiceAccountConfigBasic creates a basic service account config for testing
func BuildServiceAccountConfigBasic(serviceAccountName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Test service account description"
  permission_system_id = %[2]q
}
`,
		serviceAccountName,
		GetTestPermissionSystemID(),
	)
}

// BuildServiceAccountConfigUpdate creates a service account config for update testing
func BuildServiceAccountConfigUpdate(serviceAccountName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Updated service account description"
  permission_system_id = %[2]q
}
`,
		serviceAccountName,
		GetTestPermissionSystemID(),
	)
}

// BuildServiceAccountConfigWithTokens creates a service account config with tokens
func BuildServiceAccountConfigWithTokens(serviceAccountName string, tokenNames []string) string {
	config := BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Service account with tokens"
  permission_system_id = %[2]q
}
`,
		serviceAccountName,
		GetTestPermissionSystemID(),
	)

	for i, tokenName := range tokenNames {
		config += fmt.Sprintf(`
resource "authzed_token" "token_%[2]d" {
  name                 = %[1]q
  description          = "Token %[2]d for service account"
  permission_system_id = %[3]q
  service_account_id   = authzed_service_account.test.id
}
`,
			tokenName,
			i+1,
			GetTestPermissionSystemID(),
		)
	}

	return config
}

// CreateTestServiceAccountClient creates a test client for service account operations
func CreateTestServiceAccountClient() *client.CloudClient {
	clientConfig := &client.CloudClientConfig{
		Host:       GetTestHost(),
		Token:      GetTestToken(),
		APIVersion: GetTestAPIVersion(),
	}
	return client.NewCloudClient(clientConfig)
}

// ValidateServiceAccountExists validates that a service account exists via API
func ValidateServiceAccountExists(permissionSystemID, serviceAccountID string) error {
	testClient := CreateTestServiceAccountClient()
	_, err := testClient.GetServiceAccount(context.Background(), permissionSystemID, serviceAccountID)
	if err != nil {
		return fmt.Errorf("service account does not exist: %w", err)
	}
	return nil
}

// ValidateServiceAccountDestroyed validates that a service account has been destroyed
func ValidateServiceAccountDestroyed(permissionSystemID, serviceAccountID string) error {
	testClient := CreateTestServiceAccountClient()
	_, err := testClient.GetServiceAccount(context.Background(), permissionSystemID, serviceAccountID)
	if err == nil {
		return fmt.Errorf("service account still exists")
	}
	if !IsNotFoundError(err) {
		return fmt.Errorf("unexpected error checking service account destruction: %w", err)
	}
	return nil
}

// GenerateServiceAccountTestData generates test data for service account testing
func GenerateServiceAccountTestData(baseName string) map[string]any {
	testID := GenerateTestID(baseName)
	return map[string]any{
		"name":                 testID,
		"description":          fmt.Sprintf("Test service account %s", testID),
		"permission_system_id": GetTestPermissionSystemID(),
		"created_at":           "",
		"creator":              "",
		"etag":                 "",
	}
}

// CreateTestServiceAccount creates a service account for testing and returns it
func CreateTestServiceAccount(name string) (*models.ServiceAccount, error) {
	testClient := CreateTestServiceAccountClient()

	serviceAccount := &models.ServiceAccount{
		Name:                name,
		Description:         fmt.Sprintf("Test service account %s", name),
		PermissionsSystemID: GetTestPermissionSystemID(),
	}

	created, err := testClient.CreateServiceAccount(context.Background(), serviceAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to create test service account: %w", err)
	}

	return created.ServiceAccount, nil
}

// CleanupTestServiceAccount removes a test service account
func CleanupTestServiceAccount(permissionSystemID, serviceAccountID string) error {
	testClient := CreateTestServiceAccountClient()
	return testClient.DeleteServiceAccount(permissionSystemID, serviceAccountID)
}

// GetServiceAccountTestEnvironment gets the test environment variables for service account testing
func GetServiceAccountTestEnvironment() map[string]string {
	return map[string]string{
		"AUTHZED_HOST":        GetTestHost(),
		"AUTHZED_TOKEN":       GetTestToken(),
		"AUTHZED_API_VERSION": GetTestAPIVersion(),
		"AUTHZED_PS_ID":       GetTestPermissionSystemID(),
	}
}

// SetServiceAccountTestEnvironment sets up the test environment for service account testing
func SetServiceAccountTestEnvironment() error {
	env := GetServiceAccountTestEnvironment()
	for key, value := range env {
		if value == "" {
			return fmt.Errorf("required environment variable %s is not set", key)
		}
		_ = os.Setenv(key, value)
	}
	return nil
}

// CleanupServiceAccountTestEnvironment cleans up the test environment
func CleanupServiceAccountTestEnvironment() {
	env := GetServiceAccountTestEnvironment()
	for key := range env {
		_ = os.Unsetenv(key)
	}
}
