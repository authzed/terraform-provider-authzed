package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestETagSupport(t *testing.T) {
	const testETag = "W/\"etag-service-account\""
	const updatedETag = "W/\"etag-updated\""
	var receivedETag string
	var ifMatchHeaderReceived bool

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set content type for all responses
		w.Header().Set("Content-Type", "application/json")

		// Check for If-Match header in PUT requests
		if r.Method == http.MethodPut {
			receivedETag = r.Header.Get("If-Match")
			ifMatchHeaderReceived = receivedETag != ""

			// Simulate concurrency detection
			if receivedETag != testETag && receivedETag != "" {
				w.WriteHeader(http.StatusPreconditionFailed)
				_, err := w.Write([]byte(`{"error":"Precondition failed: resource has been modified"}`))
				if err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
				return
			}
		}

		// Test paths
		switch r.URL.Path {
		case "/ps/ps-test123/access/service-accounts/asa-test123":
			// Set ETag header in response
			w.Header().Set("ETag", testETag)

			switch r.Method {
			case http.MethodGet:
				// Return service account for GET
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{
					"id": "asa-test123",
					"permissionsSystemId": "ps-test123",
					"name": "Test Service Account",
					"description": "Test Description",
					"createdAt": "2023-05-01T12:00:00Z",
					"creator": "test-user"
				}`))
				if err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			case http.MethodPut:
				// Return updated service account for PUT with new ETag
				w.Header().Set("ETag", updatedETag)
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{
					"id": "asa-test123",
					"permissionsSystemId": "ps-test123",
					"name": "Updated Service Account",
					"description": "Updated Description",
					"createdAt": "2023-05-01T12:00:00Z",
					"creator": "test-user"
				}`))
				if err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}
		}
	}))
	defer server.Close()

	// Create client
	c := client.NewCloudClient(&client.CloudClientConfig{
		Host:  server.URL,
		Token: "test-token",
	})

	// Test 1: GET captures ETag
	t.Run("GetCaptures_ETag", func(t *testing.T) {
		result, err := c.GetServiceAccount("ps-test123", "asa-test123")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testETag, result.ETag, "ETag should be captured from response")
		assert.Equal(t, "asa-test123", result.ServiceAccount.ID)
		assert.Equal(t, "Test Service Account", result.ServiceAccount.Name)
	})

	// Test 2: Update sends ETag in If-Match header
	t.Run("UpdateSends_IfMatch", func(t *testing.T) {
		// Reset tracking variables
		ifMatchHeaderReceived = false
		receivedETag = ""

		// Perform update
		sa := &models.ServiceAccount{
			ID:                  "asa-test123",
			PermissionsSystemID: "ps-test123",
			Name:                "Updated Service Account",
			Description:         "Updated Description",
		}

		result, err := c.UpdateServiceAccount(sa, testETag)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, ifMatchHeaderReceived, "If-Match header should be sent")
		assert.Equal(t, testETag, receivedETag, "ETag should be sent correctly")
		assert.Equal(t, updatedETag, result.ETag, "New ETag should be returned")
	})

	// Test 3: Concurrent modification detection
	t.Run("DetectsConcurrentModification", func(t *testing.T) {
		// Reset tracking variables
		ifMatchHeaderReceived = false
		receivedETag = ""

		// Perform update with wrong ETag
		sa := &models.ServiceAccount{
			ID:                  "asa-test123",
			PermissionsSystemID: "ps-test123",
			Name:                "Updated Service Account",
			Description:         "Updated Description",
		}

		result, err := c.UpdateServiceAccount(sa, "W/\"wrong-etag\"")
		assert.Error(t, err, "Update with wrong ETag should fail")
		assert.Nil(t, result)
		assert.True(t, ifMatchHeaderReceived, "If-Match header should be sent")
		assert.Equal(t, "W/\"wrong-etag\"", receivedETag, "Wrong ETag should be sent")

		// Check error type
		apiErr, ok := err.(*client.APIError)
		assert.True(t, ok, "Error should be an APIError")
		assert.Equal(t, http.StatusPreconditionFailed, apiErr.StatusCode, "Should receive 412 Precondition Failed")
	})
}
