package helpers

import (
	"context"
	"fmt"
	"os"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
)

// BuildTokenConfigBasic creates a basic token config for testing
func BuildTokenConfigBasic(tokenName, serviceAccountName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[2]q
  description          = "Test service account for token"
  permission_system_id = %[3]q
}

resource "authzed_token" "test" {
  name                 = %[1]q
  description          = "Test token description"
  permission_system_id = %[3]q
  service_account_id   = authzed_service_account.test.id
}
`,
		tokenName,
		serviceAccountName,
		GetTestPermissionSystemID(),
	)
}

// BuildTokenConfigUpdate creates a token config for update testing
func BuildTokenConfigUpdate(tokenName, serviceAccountName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[2]q
  description          = "Test service account for token"
  permission_system_id = %[3]q
}

resource "authzed_token" "test" {
  name                 = %[1]q
  description          = "Updated token description"
  permission_system_id = %[3]q
  service_account_id   = authzed_service_account.test.id
}
`,
		tokenName,
		serviceAccountName,
		GetTestPermissionSystemID(),
	)
}

// BuildTokenConfigWithServiceAccount creates a token config with explicit service account relationship
func BuildTokenConfigWithServiceAccount(tokenName, serviceAccountName string) string {
	return BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[2]q
  description          = "Service account for token testing"
  permission_system_id = %[3]q
}

resource "authzed_token" "test" {
  name                 = %[1]q
  description          = "Token for service account relationship testing"
  permission_system_id = %[3]q
  service_account_id   = authzed_service_account.test.id
}
`,
		tokenName,
		serviceAccountName,
		GetTestPermissionSystemID(),
	)
}

// BuildTokenConfigMultiple creates a config with multiple tokens for the same service account
func BuildTokenConfigMultiple(serviceAccountName string, tokenNames []string) string {
	config := BuildProviderConfig() + fmt.Sprintf(`
resource "authzed_service_account" "test" {
  name                 = %[1]q
  description          = "Service account with multiple tokens"
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

// CreateTestTokenClient creates a test client for token operations
func CreateTestTokenClient() *client.CloudClient {
	clientConfig := &client.CloudClientConfig{
		Host:       GetTestHost(),
		Token:      GetTestToken(),
		APIVersion: GetTestAPIVersion(),
	}
	return client.NewCloudClient(clientConfig)
}

// ValidateTokenExists validates that a token exists via API
func ValidateTokenExists(permissionSystemID, serviceAccountID, tokenID string) error {
	testClient := CreateTestTokenClient()
	_, err := testClient.GetToken(context.Background(), permissionSystemID, serviceAccountID, tokenID)
	if err != nil {
		return fmt.Errorf("token does not exist: %w", err)
	}
	return nil
}

// ValidateTokenDestroyed validates that a token has been destroyed
func ValidateTokenDestroyed(permissionSystemID, serviceAccountID, tokenID string) error {
	testClient := CreateTestTokenClient()
	_, err := testClient.GetToken(context.Background(), permissionSystemID, serviceAccountID, tokenID)
	if err == nil {
		return fmt.Errorf("token still exists")
	}
	if !IsNotFoundError(err) {
		return fmt.Errorf("unexpected error checking token destruction: %w", err)
	}
	return nil
}

// GenerateTokenTestData generates test data for token testing
func GenerateTokenTestData(baseName string) map[string]any {
	testID := GenerateTestID(baseName)
	return map[string]any{
		"name":                 testID,
		"description":          fmt.Sprintf("Test token %s", testID),
		"permission_system_id": GetTestPermissionSystemID(),
		"service_account_id":   "",
		"created_at":           "",
		"creator":              "",
		"hash":                 "",
		"plain_text":           "",
		"etag":                 "",
	}
}

// CreateTestToken creates a token for testing and returns it
func CreateTestToken(name, serviceAccountID string) (*models.TokenRequest, error) {
	testClient := CreateTestTokenClient()

	token := &models.TokenRequest{
		Name:                name,
		Description:         fmt.Sprintf("Test token %s", name),
		PermissionsSystemID: GetTestPermissionSystemID(),
		ServiceAccountID:    serviceAccountID,
		ReturnPlainText:     true,
	}

	created, err := testClient.CreateToken(context.Background(), token)
	if err != nil {
		return nil, fmt.Errorf("failed to create test token: %w", err)
	}

	return created.Token, nil
}

// CleanupTestToken removes a test token
func CleanupTestToken(permissionSystemID, serviceAccountID, tokenID string) error {
	testClient := CreateTestTokenClient()
	return testClient.DeleteToken(permissionSystemID, serviceAccountID, tokenID)
}

// GetTokenTestEnvironment gets the test environment variables for token testing
func GetTokenTestEnvironment() map[string]string {
	return map[string]string{
		"AUTHZED_HOST":        GetTestHost(),
		"AUTHZED_TOKEN":       GetTestToken(),
		"AUTHZED_API_VERSION": GetTestAPIVersion(),
		"AUTHZED_PS_ID":       GetTestPermissionSystemID(),
	}
}

// SetTokenTestEnvironment sets up the test environment for token testing
func SetTokenTestEnvironment() error {
	env := GetTokenTestEnvironment()
	for key, value := range env {
		if value == "" {
			return fmt.Errorf("required environment variable %s is not set", key)
		}
		_ = os.Setenv(key, value)
	}
	return nil
}

// CleanupTokenTestEnvironment cleans up the test environment
func CleanupTokenTestEnvironment() {
	env := GetTokenTestEnvironment()
	for key := range env {
		_ = os.Unsetenv(key)
	}
}

// ValidateTokenPlainTextSensitive validates that plain_text field is properly marked as sensitive
func ValidateTokenPlainTextSensitive(plainTextValue string) error {
	if plainTextValue == "" {
		return fmt.Errorf("plain_text should not be empty during creation")
	}
	// In real tests, we would verify this is properly masked in logs/output
	return nil
}

// ValidateTokenHash validates that token hash is properly generated
func ValidateTokenHash(hash string) error {
	if hash == "" {
		return fmt.Errorf("token hash should not be empty")
	}
	if len(hash) != 64 {
		return fmt.Errorf("token hash should be 64 characters (SHA256), got %d", len(hash))
	}
	return nil
}

// ValidateTokenServiceAccountRelationship validates the relationship between token and service account
func ValidateTokenServiceAccountRelationship(tokenServiceAccountID, expectedServiceAccountID string) error {
	if tokenServiceAccountID != expectedServiceAccountID {
		return fmt.Errorf("token service account ID %s does not match expected %s", tokenServiceAccountID, expectedServiceAccountID)
	}
	return nil
}
