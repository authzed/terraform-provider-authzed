package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestETagSupport(t *testing.T) {
	const testETag = "W/\"etag-service-account\""
	const updatedETag = "W/\"updated-etag-service-account\""
	updateRequestCount := 0

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// For GET requests, return service account with ETag
			w.Header().Set("ETag", testETag)
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
			updateRequestCount++

			// Check if If-Match header is present and correct
			ifMatch := r.Header.Get("If-Match")

			if updateRequestCount == 1 {
				// First update: simulate concurrent modification (412)
				if ifMatch != testETag {
					w.WriteHeader(http.StatusBadRequest)
					_, err := w.Write([]byte(`{"error": "Invalid ETag"}`))
					if err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
					return
				}

				// Return 412 to simulate concurrent modification
				w.WriteHeader(http.StatusPreconditionFailed)
				_, err := w.Write([]byte(`{"error": "Precondition failed"}`))
				if err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
				return
			}

			// Second PUT succeeds
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
	}))
	defer server.Close()

	// Create client
	cfg := &client.CloudClientConfig{
		Host:       server.URL,
		Token:      "test-token",
		APIVersion: "v1",
		Timeout:    client.DefaultTimeout,
	}
	c := client.NewCloudClient(cfg)

	t.Run("GetCaptures_ETag", func(t *testing.T) {
		sa, err := c.GetServiceAccount("ps-test123", "asa-test123")
		assert.NoError(t, err)
		assert.Equal(t, testETag, sa.GetETag())
	})

	t.Run("UpdateSends_IfMatch", func(t *testing.T) {
		serviceAccount := &models.ServiceAccount{
			ID:                  "asa-test123",
			PermissionsSystemID: "ps-test123",
			Name:                "Updated Service Account",
			Description:         "Updated Description",
		}

		// This should work after retry
		result := c.UpdateServiceAccount(context.Background(), serviceAccount, testETag)
		assert.False(t, result.Diagnostics.HasError())
		assert.NotNil(t, result.ServiceAccount)
		assert.Equal(t, updatedETag, result.ServiceAccount.ETag)
	})

	t.Run("DetectsConcurrentModification", func(t *testing.T) {
		// Reset counter for this test
		updateRequestCount = 0

		serviceAccount := &models.ServiceAccount{
			ID:                  "asa-test123",
			PermissionsSystemID: "ps-test123",
			Name:                "Updated Service Account",
			Description:         "Updated Description",
		}

		result := c.UpdateServiceAccount(context.Background(), serviceAccount, testETag)
		assert.False(t, result.Diagnostics.HasError())
		assert.NotNil(t, result.ServiceAccount)
		assert.Equal(t, updatedETag, result.ServiceAccount.ETag)
		assert.Equal(t, 2, updateRequestCount)
	})

	t.Run("Handles409ConflictRetry", func(t *testing.T) {
		// Create a separate server for 409 testing
		conflictRequestCount := 0
		server409 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				// For GET requests, return service account with ETag
				w.Header().Set("ETag", testETag)
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
				conflictRequestCount++

				if conflictRequestCount == 1 {
					// First update: simulate FGAM conflict (409)
					w.WriteHeader(http.StatusConflict)
					_, err := w.Write([]byte(`{"error": "restricted API access configuration for permission system has changed"}`))
					if err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
					return
				}

				// Second PUT succeeds
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
		}))
		defer server409.Close()

		// Create client for 409 testing
		cfg409 := &client.CloudClientConfig{
			Host:       server409.URL,
			Token:      "test-token",
			APIVersion: "v1",
			Timeout:    client.DefaultTimeout,
		}
		c409 := client.NewCloudClient(cfg409)

		serviceAccount := &models.ServiceAccount{
			ID:                  "asa-test123",
			PermissionsSystemID: "ps-test123",
			Name:                "Updated Service Account",
			Description:         "Updated Description",
		}

		// This should succeed after retry
		result := c409.UpdateServiceAccount(context.Background(), serviceAccount, testETag)
		assert.False(t, result.Diagnostics.HasError())
		assert.NotNil(t, result.ServiceAccount)
		assert.Equal(t, updatedETag, result.ServiceAccount.ETag)
		assert.Equal(t, 2, conflictRequestCount)
	})
}
