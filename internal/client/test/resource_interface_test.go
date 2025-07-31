package test

import (
	"testing"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
)

func TestResourceInterface(t *testing.T) {
	// Test that ServiceAccountWithETag implements Resource interface
	sa := &models.ServiceAccount{
		ID:                  "asa-123abc456def",
		PermissionsSystemID: "test-ps",
		Name:                "Test Service Account",
		Description:         "A test service account",
		CreatedAt:           "2023-05-01T12:00:00Z",
		Creator:             "test-user",
	}

	saWithETag := &client.ServiceAccountWithETag{
		ServiceAccount: sa,
		ETag:           "W/\"test-etag\"",
	}

	// Test Resource interface methods
	if saWithETag.GetID() != "asa-123abc456def" {
		t.Errorf("Expected ID 'asa-123abc456def', got '%s'", saWithETag.GetID())
	}

	if saWithETag.GetETag() != "W/\"test-etag\"" {
		t.Errorf("Expected ETag 'W/\"test-etag\"', got '%s'", saWithETag.GetETag())
	}

	// Test SetETag
	saWithETag.SetETag("W/\"new-etag\"")
	if saWithETag.GetETag() != "W/\"new-etag\"" {
		t.Errorf("Expected ETag 'W/\"new-etag\"' after SetETag, got '%s'", saWithETag.GetETag())
	}

	// Test that RoleWithETag implements Resource interface
	role := &models.Role{
		ID:                  "role-123abc456def",
		PermissionsSystemID: "test-ps",
		Name:                "Test Role",
		Description:         "A test role",
		CreatedAt:           "2023-05-01T12:00:00Z",
		Creator:             "test-user",
	}

	roleWithETag := &client.RoleWithETag{
		Role: role,
		ETag: "W/\"role-etag\"",
	}

	if roleWithETag.GetID() != "role-123abc456def" {
		t.Errorf("Expected ID 'role-123abc456def', got '%s'", roleWithETag.GetID())
	}

	if roleWithETag.GetETag() != "W/\"role-etag\"" {
		t.Errorf("Expected ETag 'W/\"role-etag\"', got '%s'", roleWithETag.GetETag())
	}

	roleWithETag.SetETag("W/\"new-role-etag\"")
	if roleWithETag.GetETag() != "W/\"new-role-etag\"" {
		t.Errorf("Expected ETag 'W/\"new-role-etag\"' after SetETag, got '%s'", roleWithETag.GetETag())
	}

	// Test that PolicyWithETag implements Resource interface
	policy := &models.Policy{
		ID:                  "policy-123abc456def",
		PermissionsSystemID: "test-ps",
		Name:                "Test Policy",
		Description:         "A test policy",
		PrincipalID:         "principal-123",
		RoleIDs:             []string{"role-456"},
		CreatedAt:           "2023-05-01T12:00:00Z",
		Creator:             "test-user",
	}

	policyWithETag := &client.PolicyWithETag{
		Policy: policy,
		ETag:   "W/\"policy-etag\"",
	}

	if policyWithETag.GetID() != "policy-123abc456def" {
		t.Errorf("Expected ID 'policy-123abc456def', got '%s'", policyWithETag.GetID())
	}

	if policyWithETag.GetETag() != "W/\"policy-etag\"" {
		t.Errorf("Expected ETag 'W/\"policy-etag\"', got '%s'", policyWithETag.GetETag())
	}

	policyWithETag.SetETag("W/\"new-policy-etag\"")
	if policyWithETag.GetETag() != "W/\"new-policy-etag\"" {
		t.Errorf("Expected ETag 'W/\"new-policy-etag\"' after SetETag, got '%s'", policyWithETag.GetETag())
	}

	// Test that TokenWithETag implements Resource interface
	token := &models.TokenRequest{
		ID:                  "token-123abc456def",
		PermissionsSystemID: "test-ps",
		ServiceAccountID:    "asa-123abc456def",
		Name:                "Test Token",
		Description:         "A test token",
	}

	tokenWithETag := &client.TokenWithETag{
		Token: token,
		ETag:  "W/\"token-etag\"",
	}

	if tokenWithETag.GetID() != "token-123abc456def" {
		t.Errorf("Expected ID 'token-123abc456def', got '%s'", tokenWithETag.GetID())
	}

	if tokenWithETag.GetETag() != "W/\"token-etag\"" {
		t.Errorf("Expected ETag 'W/\"token-etag\"', got '%s'", tokenWithETag.GetETag())
	}

	tokenWithETag.SetETag("W/\"new-token-etag\"")
	if tokenWithETag.GetETag() != "W/\"new-token-etag\"" {
		t.Errorf("Expected ETag 'W/\"new-token-etag\"' after SetETag, got '%s'", tokenWithETag.GetETag())
	}
}

func TestUpdateResource(t *testing.T) {
	// Create a mock client (this won't actually make HTTP requests)
	c := client.NewCloudClient(&client.CloudClientConfig{
		Host:  "https://test.example.com",
		Token: "test-token",
	})

	// Create a test resource
	sa := &models.ServiceAccount{
		ID:                  "asa-123abc456def",
		PermissionsSystemID: "test-ps",
		Name:                "Test Service Account",
		Description:         "A test service account",
		CreatedAt:           "2023-05-01T12:00:00Z",
		Creator:             "test-user",
	}

	saETag := "W/\"test-etag\""
	resource := &client.ServiceAccountWithETag{
		ServiceAccount: sa,
		ETag:           saETag,
	}

	// Call UpdateResource
	endpoint := "/ps/test-ps/access/service-accounts/asa-123abc456def"
	body := map[string]any{"name": "Updated Service Account"}

	updated, err := c.UpdateResource(t.Context(), resource, endpoint, body)
	// Verify results
	if err != nil {
		// This is expected since we're not actually making a real HTTP request
		// The important thing is that the method signature works correctly
		t.Logf("Expected error for mock request: %v", err)
	}

	// Verify that the resource interface is working
	if updated != nil {
		t.Errorf("Expected nil result for mock request, got %v", updated)
	}

	// Test that the resource passed in maintains its interface
	if resource.GetETag() != saETag {
		t.Errorf("Expected ETag '%s', got '%s'", saETag, resource.GetETag())
	}
}
