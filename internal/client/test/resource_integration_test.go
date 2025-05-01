package test

import (
	"os"
	"testing"
	"time"

	"terraform-provider-authzed/internal/client"
)

// TestResourceIntegration is an integration test that can be run with real API credentials
// Skip this test unless explicitly enabled with TEST_INTEGRATION=1
func TestResourceIntegration(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("TEST_INTEGRATION") != "1" {
		t.Skip("Skipping integration test; set TEST_INTEGRATION=1 to run")
	}

	// Get API credentials from environment
	host := os.Getenv("AUTHZED_HOST")
	if host == "" {
		host = "https://grpcapi.authzed.com"
	}

	token := os.Getenv("AUTHZED_TOKEN")
	if token == "" {
		t.Fatal("AUTHZED_TOKEN environment variable must be set")
	}

	permissionsSystemID := os.Getenv("AUTHZED_PS_ID")
	if permissionsSystemID == "" {
		t.Fatal("AUTHZED_PS_ID environment variable must be set")
	}

	// Create client
	cfg := &client.CloudClientConfig{
		Host:       host,
		Token:      token,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}
	c := client.NewCloudClient(cfg)

	// Test GetPermissionsSystem
	t.Run("GetPermissionsSystem", func(t *testing.T) {
		ps, err := c.GetPermissionsSystem(permissionsSystemID)
		if err != nil {
			t.Fatalf("Failed to get permissions system: %v", err)
		}

		// Verify Resource interface methods
		if ps.GetID() != permissionsSystemID {
			t.Errorf("Expected ID %s, got %s", permissionsSystemID, ps.GetID())
		}

		if ps.GetETag() == "" {
			t.Error("ETag is empty")
		}

		t.Logf("Successfully got permissions system with ID: %s and ETag: %s",
			ps.GetID(), ps.GetETag())
	})

	// Test ListServiceAccounts
	t.Run("ListServiceAccounts", func(t *testing.T) {
		accounts, err := c.ListServiceAccounts(permissionsSystemID)
		if err != nil {
			t.Fatalf("Failed to list service accounts: %v", err)
		}

		t.Logf("Successfully listed %d service accounts", len(accounts))
	})
}
