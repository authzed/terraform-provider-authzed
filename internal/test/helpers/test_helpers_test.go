package helpers

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateTestID(t *testing.T) {
	prefix := "test"
	id1 := GenerateTestID(prefix)
	id2 := GenerateTestID(prefix)

	// IDs should be different
	if id1 == id2 {
		t.Error("GenerateTestID should generate unique IDs")
	}

	// IDs should start with prefix
	if !strings.HasPrefix(id1, prefix) {
		t.Errorf("ID %s should start with prefix %s", id1, prefix)
	}
}

func TestGetTestEnvironmentVariables(t *testing.T) {
	// Save original env vars
	originalHost := os.Getenv("AUTHZED_HOST")
	originalToken := os.Getenv("AUTHZED_TOKEN")
	originalPSID := os.Getenv("AUTHZED_PS_ID")
	originalAPIVersion := os.Getenv("AUTHZED_API_VERSION")

	// Clean up after test
	defer func() {
		os.Setenv("AUTHZED_HOST", originalHost)
		os.Setenv("AUTHZED_TOKEN", originalToken)
		os.Setenv("AUTHZED_PS_ID", originalPSID)
		os.Setenv("AUTHZED_API_VERSION", originalAPIVersion)
	}()

	// Test missing required variables
	os.Unsetenv("AUTHZED_HOST")
	os.Unsetenv("AUTHZED_TOKEN")
	os.Unsetenv("AUTHZED_PS_ID")
	os.Unsetenv("AUTHZED_API_VERSION")

	_, err := GetTestEnvironmentVariables()
	if err == nil {
		t.Error("Expected error when required environment variables are missing")
	}

	// Test with all required variables set
	os.Setenv("AUTHZED_HOST", "test-host")
	os.Setenv("AUTHZED_TOKEN", "test-token")
	os.Setenv("AUTHZED_PS_ID", "test-ps-id")

	envVars, err := GetTestEnvironmentVariables()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if envVars["AUTHZED_HOST"] != "test-host" {
		t.Error("AUTHZED_HOST not set correctly")
	}

	// Test default API version
	if envVars["AUTHZED_API_VERSION"] != "25r1" {
		t.Error("Default API version should be 25r1")
	}
}

func TestIsAcceptanceTest(t *testing.T) {
	// Save original TF_ACC
	originalTFAcc := os.Getenv("TF_ACC")
	defer os.Setenv("TF_ACC", originalTFAcc)

	// Test without TF_ACC
	os.Unsetenv("TF_ACC")
	if IsAcceptanceTest() {
		t.Error("IsAcceptanceTest should return false when TF_ACC is not set")
	}

	// Test with TF_ACC set
	os.Setenv("TF_ACC", "1")
	if !IsAcceptanceTest() {
		t.Error("IsAcceptanceTest should return true when TF_ACC is set")
	}
}

func TestValidateTestEnvironment(t *testing.T) {
	// Save original env vars
	originalTFAcc := os.Getenv("TF_ACC")
	originalHost := os.Getenv("AUTHZED_HOST")
	originalToken := os.Getenv("AUTHZED_TOKEN")
	originalPSID := os.Getenv("AUTHZED_PS_ID")

	// Clean up after test
	defer func() {
		os.Setenv("TF_ACC", originalTFAcc)
		os.Setenv("AUTHZED_HOST", originalHost)
		os.Setenv("AUTHZED_TOKEN", originalToken)
		os.Setenv("AUTHZED_PS_ID", originalPSID)
	}()

	// Test without TF_ACC
	os.Unsetenv("TF_ACC")
	err := ValidateTestEnvironment()
	if err == nil {
		t.Error("Expected error when TF_ACC is not set")
	}

	// Test with TF_ACC but missing other vars
	os.Setenv("TF_ACC", "1")
	os.Unsetenv("AUTHZED_HOST")
	err = ValidateTestEnvironment()
	if err == nil {
		t.Error("Expected error when required environment variables are missing")
	}

	// Test with all vars set
	os.Setenv("AUTHZED_HOST", "test-host")
	os.Setenv("AUTHZED_TOKEN", "test-token")
	os.Setenv("AUTHZED_PS_ID", "test-ps-id")
	err = ValidateTestEnvironment()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
