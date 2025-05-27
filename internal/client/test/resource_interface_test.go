package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
)

func TestResourceInterface(t *testing.T) {
	// Test ServiceAccountWithETag
	saID := "asa-123abc456def"
	saETag := "W/\"etag-service-account\""
	sa := &models.ServiceAccount{ID: saID}
	saResource := &client.ServiceAccountWithETag{
		ServiceAccount: sa,
		ETag:           saETag,
	}

	testResource(t, saResource, saID, saETag)

	// Test RoleWithETag
	roleID := "arl-789def012ghi"
	roleETag := "W/\"etag-role\""
	role := &models.Role{ID: roleID}
	roleResource := &client.RoleWithETag{
		Role: role,
		ETag: roleETag,
	}

	testResource(t, roleResource, roleID, roleETag)

	// Test PolicyWithETag
	policyID := "apc-345ghi678jkl"
	policyETag := "W/\"etag-policy\""
	policy := &models.Policy{ID: policyID}
	policyResource := &client.PolicyWithETag{
		Policy: policy,
		ETag:   policyETag,
	}

	testResource(t, policyResource, policyID, policyETag)

	// Test TokenWithETag
	tokenID := "atk-901jkl234mno"
	tokenETag := "W/\"etag-token\""
	token := &models.TokenRequest{ID: tokenID}
	tokenResource := &client.TokenWithETag{
		Token: token,
		ETag:  tokenETag,
	}

	testResource(t, tokenResource, tokenID, tokenETag)

	// Test PermissionsSystemWithETag
	psID := "ps-567mno890pqr"
	psETag := "W/\"etag-ps\""
	ps := &models.PermissionsSystem{ID: psID}
	psResource := &client.PermissionsSystemWithETag{
		PermissionsSystem: ps,
		ETag:              psETag,
	}

	testResource(t, psResource, psID, psETag)
}

func testResource(t *testing.T, resource client.Resource, expectedID, expectedETag string) {
	t.Helper()

	// Test GetID
	if id := resource.GetID(); id != expectedID {
		t.Errorf("Expected ID %s, got %s", expectedID, id)
	}

	// Test GetETag
	if etag := resource.GetETag(); etag != expectedETag {
		t.Errorf("Expected ETag %s, got %s", expectedETag, etag)
	}

	// Test SetETag
	newETag := "new-etag-456"
	resource.SetETag(newETag)
	if etag := resource.GetETag(); etag != newETag {
		t.Errorf("Expected ETag to be updated to %s, got %s", newETag, etag)
	}

	// Test GetResource returns non-nil
	if resource.GetResource() == nil {
		t.Error("GetResource returned nil")
	}
}

// TestFactories tests the resource factory functions
func TestFactories(t *testing.T) {
	// Test ServiceAccount factory
	saID := "asa-123abc456def"
	saETag := "W/\"etag-service-account\""
	sa := &models.ServiceAccount{ID: saID}

	saResource := client.NewServiceAccountResource(sa, saETag)
	if saResource.GetID() != saID {
		t.Errorf("ServiceAccount factory: expected ID %s, got %s", saID, saResource.GetID())
	}
	if saResource.GetETag() != saETag {
		t.Errorf("ServiceAccount factory: expected ETag %s, got %s", saETag, saResource.GetETag())
	}

	// Test Role factory
	roleID := "arl-789def012ghi"
	roleETag := "W/\"etag-role\""
	role := &models.Role{ID: roleID}

	roleResource := client.NewRoleResource(role, roleETag)
	if roleResource.GetID() != roleID {
		t.Errorf("Role factory: expected ID %s, got %s", roleID, roleResource.GetID())
	}
	if roleResource.GetETag() != roleETag {
		t.Errorf("Role factory: expected ETag %s, got %s", roleETag, roleResource.GetETag())
	}
}

// TestUpdateResource tests the generic UpdateResource method
func TestUpdateResource(t *testing.T) {
	// Create a mock server to test UpdateResource
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify HTTP method is PUT
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		// Verify If-Match header contains the expected ETag
		etag := r.Header.Get("If-Match")
		if etag != "W/\"etag-service-account\"" {
			t.Errorf("Expected If-Match header with %s, got %s", "W/\"etag-service-account\"", etag)
		}

		// Return successful response with new ETag
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", "W/\"new-etag\"")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"id":"asa-123abc456def","name":"Updated Service Account"}`))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create client with mock server
	c := client.NewCloudClient(&client.CloudClientConfig{
		Host:  server.URL,
		Token: "test-token",
	})

	// Create a Resource for testing
	saID := "asa-123abc456def"
	saETag := "W/\"etag-service-account\""
	sa := &models.ServiceAccount{
		ID:   saID,
		Name: "Service Account",
	}
	resource := &client.ServiceAccountWithETag{
		ServiceAccount: sa,
		ETag:           saETag,
	}

	// Call UpdateResource
	endpoint := "/ps/test-ps/access/service-accounts/asa-123abc456def"
	body := map[string]any{"name": "Updated Service Account"}

	updated, err := c.UpdateResource(resource, endpoint, body)

	// Verify results
	if err != nil {
		t.Fatalf("UpdateResource failed: %v", err)
	}

	if updated.GetETag() != "W/\"new-etag\"" {
		t.Errorf("Expected new ETag %s, got %s", "W/\"new-etag\"", updated.GetETag())
	}

	// Verify it's the same resource instance (not a copy)
	if updated != resource {
		t.Error("UpdateResource should update and return the same resource instance")
	}
}
