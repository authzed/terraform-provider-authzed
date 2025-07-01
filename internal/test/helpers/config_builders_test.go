package helpers

import (
	"os"
	"strings"
	"testing"
)

func TestBuildProviderConfig(t *testing.T) {
	// Save original env vars
	originalHost := os.Getenv("AUTHZED_HOST")
	originalToken := os.Getenv("AUTHZED_TOKEN")
	originalAPIVersion := os.Getenv("AUTHZED_API_VERSION")

	// Clean up after test
	defer func() {
		_ = os.Setenv("AUTHZED_HOST", originalHost)
		_ = os.Setenv("AUTHZED_TOKEN", originalToken)
		_ = os.Setenv("AUTHZED_API_VERSION", originalAPIVersion)
	}()

	// Set test values
	_ = os.Setenv("AUTHZED_HOST", "test-host")
	_ = os.Setenv("AUTHZED_TOKEN", "test-token")
	_ = os.Setenv("AUTHZED_API_VERSION", "test-version")

	config := BuildProviderConfig()

	// Check that config contains expected values
	if !strings.Contains(config, "test-host") {
		t.Error("Config should contain test host")
	}
	if !strings.Contains(config, "test-token") {
		t.Error("Config should contain test token")
	}
	if !strings.Contains(config, "test-version") {
		t.Error("Config should contain test API version")
	}
	if !strings.Contains(config, `provider "authzed"`) {
		t.Error("Config should contain provider block")
	}
}

func TestBuildProviderConfigWithCustom(t *testing.T) {
	config := BuildProviderConfigWithCustom("custom-host", "custom-token", "custom-version")

	if !strings.Contains(config, "custom-host") {
		t.Error("Config should contain custom host")
	}
	if !strings.Contains(config, "custom-token") {
		t.Error("Config should contain custom token")
	}
	if !strings.Contains(config, "custom-version") {
		t.Error("Config should contain custom API version")
	}
}

func TestBuildPolicyConfig(t *testing.T) {
	// Save original env vars
	originalHost := os.Getenv("AUTHZED_HOST")
	originalToken := os.Getenv("AUTHZED_TOKEN")
	originalPSID := os.Getenv("AUTHZED_PS_ID")

	// Clean up after test
	defer func() {
		_ = os.Setenv("AUTHZED_HOST", originalHost)
		_ = os.Setenv("AUTHZED_TOKEN", originalToken)
		_ = os.Setenv("AUTHZED_PS_ID", originalPSID)
	}()

	// Set test values
	_ = os.Setenv("AUTHZED_HOST", "test-host")
	_ = os.Setenv("AUTHZED_TOKEN", "test-token")
	_ = os.Setenv("AUTHZED_PS_ID", "test-ps-id")

	config := BuildPolicyConfig("test-policy")

	if !strings.Contains(config, `resource "authzed_policy" "test"`) {
		t.Error("Config should contain policy resource")
	}
	if !strings.Contains(config, "test-policy") {
		t.Error("Config should contain policy name")
	}
	if !strings.Contains(config, "test-ps-id") {
		t.Error("Config should contain permission system ID")
	}
}

func TestBuildRoleConfig(t *testing.T) {
	// Save original env vars
	originalHost := os.Getenv("AUTHZED_HOST")
	originalToken := os.Getenv("AUTHZED_TOKEN")
	originalPSID := os.Getenv("AUTHZED_PS_ID")

	// Clean up after test
	defer func() {
		_ = os.Setenv("AUTHZED_HOST", originalHost)
		_ = os.Setenv("AUTHZED_TOKEN", originalToken)
		_ = os.Setenv("AUTHZED_PS_ID", originalPSID)
	}()

	// Set test values
	_ = os.Setenv("AUTHZED_HOST", "test-host")
	_ = os.Setenv("AUTHZED_TOKEN", "test-token")
	_ = os.Setenv("AUTHZED_PS_ID", "test-ps-id")

	config := BuildRoleConfig("test-role")

	if !strings.Contains(config, `resource "authzed_role" "test"`) {
		t.Error("Config should contain role resource")
	}
	if !strings.Contains(config, "test-role") {
		t.Error("Config should contain role name")
	}
	if !strings.Contains(config, "authzed.v1/ReadSchema") {
		t.Error("Config should contain permissions")
	}
}

func TestBuildDataSourceConfig(t *testing.T) {
	// Save original env vars
	originalHost := os.Getenv("AUTHZED_HOST")
	originalToken := os.Getenv("AUTHZED_TOKEN")
	originalPSID := os.Getenv("AUTHZED_PS_ID")

	// Clean up after test
	defer func() {
		_ = os.Setenv("AUTHZED_HOST", originalHost)
		_ = os.Setenv("AUTHZED_TOKEN", originalToken)
		_ = os.Setenv("AUTHZED_PS_ID", originalPSID)
	}()

	// Set test values
	_ = os.Setenv("AUTHZED_HOST", "test-host")
	_ = os.Setenv("AUTHZED_TOKEN", "test-token")
	_ = os.Setenv("AUTHZED_PS_ID", "test-ps-id")

	// Test permission_system data source
	config := BuildDataSourceConfig("permission_system", "test-name")
	if !strings.Contains(config, `data "authzed_permission_system" "test"`) {
		t.Error("Config should contain permission system data source")
	}

	// Test permission_systems data source
	config = BuildDataSourceConfig("permission_systems", "test-name")
	if !strings.Contains(config, `data "authzed_permission_systems" "test"`) {
		t.Error("Config should contain permission systems data source")
	}

	// Test other data sources
	config = BuildDataSourceConfig("policy", "test-policy")
	if !strings.Contains(config, `data "authzed_policy" "test"`) {
		t.Error("Config should contain policy data source")
	}
	if !strings.Contains(config, "test-policy") {
		t.Error("Config should contain policy name")
	}
}
