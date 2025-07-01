package helpers

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

// generateTestID prevents conflicts between parallel test runs
func GenerateTestID(prefix string) string {
	// Timestamp and random number for uniqueness
	timestamp := time.Now().Unix()
	random := rand.Intn(10000)
	return fmt.Sprintf("%s-%d-%d", prefix, timestamp, random)
}

// GetTestEnvironmentVariables returns a map of required env variables
// for acceptance testing with validation
func GetTestEnvironmentVariables() (map[string]string, error) {
	envVars := map[string]string{
		"AUTHZED_HOST":        os.Getenv("AUTHZED_HOST"),
		"AUTHZED_TOKEN":       os.Getenv("AUTHZED_TOKEN"),
		"AUTHZED_PS_ID":       os.Getenv("AUTHZED_PS_ID"),
		"AUTHZED_API_VERSION": os.Getenv("AUTHZED_API_VERSION"),
	}

	// Set default API version if not provided
	if envVars["AUTHZED_API_VERSION"] == "" {
		envVars["AUTHZED_API_VERSION"] = "25r1"
	}

	// Validate required variables
	requiredVars := []string{"AUTHZED_HOST", "AUTHZED_TOKEN", "AUTHZED_PS_ID"}
	for _, varName := range requiredVars {
		if envVars[varName] == "" {
			return nil, fmt.Errorf("required environment variable %s is not set", varName)
		}
	}

	return envVars, nil
}

func IsAcceptanceTest() bool {
	return os.Getenv("TF_ACC") != ""
}

func ValidateTestEnvironment() error {
	if !IsAcceptanceTest() {
		return fmt.Errorf("TF_ACC must be set for acceptance tests")
	}

	_, err := GetTestEnvironmentVariables()
	return err
}

func GenerateUniqueResourceName(baseName string) string {
	return GenerateTestID(baseName)
}

func GetPermissionSystemID() string {
	return os.Getenv("AUTHZED_PS_ID")
}

// GetTestHost returns the AuthZed host for testing
func GetTestHost() string {
	return os.Getenv("AUTHZED_HOST")
}

func GetTestToken() string {
	return os.Getenv("AUTHZED_TOKEN")
}

func GetTestAPIVersion() string {
	version := os.Getenv("AUTHZED_API_VERSION")
	if version == "" {
		return "25r1"
	}
	return version
}

// GetTestPermissionSystemID returns the permission system ID for testing
func GetTestPermissionSystemID() string {
	return os.Getenv("AUTHZED_PS_ID")
}

func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "status 404") || strings.Contains(errStr, "not found")
}
