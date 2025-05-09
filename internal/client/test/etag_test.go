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
	var firstReceivedETag string // Track the first ETag received for verification
	var ifMatchHeaderReceived bool
	var firstPUTRequest bool = true // Track first PUT request for retry test

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set content type for all responses
		w.Header().Set("Content-Type", "application/json")

		// For the retry test, track sequence of requests
		if r.URL.Path == "/ps/ps-test123/access/service-accounts/asa-test123" {
			// Check for If-Match header in PUT requests
			if r.Method == http.MethodPut {
				receivedETag = r.Header.Get("If-Match")
				ifMatchHeaderReceived = receivedETag != ""

				// Save the first ETag for verification in the retry test
				if firstPUTRequest {
					firstReceivedETag = receivedETag
				}

				// In retry test (DetectsConcurrentModification):
				// - First PUT with wrong ETag fails with 412
				// - Second PUT (after GET) with correct ETag succeeds
				if receivedETag == "W/\"wrong-etag\"" && firstPUTRequest {
					// Fail first PUT with wrong ETag
					firstPUTRequest = false
					w.WriteHeader(http.StatusPreconditionFailed)
					_, err := w.Write([]byte(`{"error":"Precondition failed: resource has been modified"}`))
					if err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
					return
				}
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
	t.Run("GetCaptures_ETag", func(t *testing.T) {
		result, err := c.GetServiceAccount("ps-test123", "asa-test123")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testETag, result.ETag, "ETag should be captured from response")
		assert.Equal(t, "asa-test123", result.ServiceAccount.ID)
		assert.Equal(t, "Test Service Account", result.ServiceAccount.Name)
	})
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
	t.Run("DetectsConcurrentModification", func(t *testing.T) {
		// Reset tracking variables
		ifMatchHeaderReceived = false
		receivedETag = ""
		firstReceivedETag = ""
		firstPUTRequest = true

		// Perform update with wrong ETag
		sa := &models.ServiceAccount{
			ID:                  "asa-test123",
			PermissionsSystemID: "ps-test123",
			Name:                "Updated Service Account",
			Description:         "Updated Description",
		}

		result, err := c.UpdateServiceAccount(sa, "W/\"wrong-etag\"")
		// With auto-retry, this should succeed
		assert.NoError(t, err, "Update with wrong ETag should retry and succeed")
		assert.NotNil(t, result, "Result should not be nil after successful retry")
		assert.True(t, ifMatchHeaderReceived, "If-Match header should be sent")
		assert.Equal(t, "W/\"wrong-etag\"", firstReceivedETag, "Wrong ETag should be sent initially")

		// After retry, the result should have the updated ETag
		assert.Equal(t, updatedETag, result.ETag, "Result should have updated ETag after retry")
	})
}
