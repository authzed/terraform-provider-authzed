package test

import (
	"testing"

	"terraform-provider-authzed/internal/client"

	"github.com/stretchr/testify/assert"
)

func TestFGAMErrorMessages(t *testing.T) {
	t.Run("DetectsFGAMConfigConflict", func(t *testing.T) {
		// Test error message that indicates FGAM configuration conflict
		apiErr := &client.APIError{
			StatusCode: 409,
			Message:    "restricted API access configuration for permission system \"ps-test123\" has changed",
		}

		errorMsg := apiErr.Error()

		// Should contain the helpful context message
		assert.Contains(t, errorMsg, "Fine-Grained Access Management (FGAM) configuration")
		assert.Contains(t, errorMsg, "automatically retry")
		assert.Contains(t, errorMsg, "status 409")
	})

	t.Run("DoesNotDetectNonFGAMConflict", func(t *testing.T) {
		// Test error message that is 409 but not FGAM related
		apiErr := &client.APIError{
			StatusCode: 409,
			Message:    "resource conflict: another process is modifying this resource",
		}

		errorMsg := apiErr.Error()

		// Should NOT contain the FGAM-specific context message
		assert.NotContains(t, errorMsg, "Fine-Grained Access Management")
		assert.Contains(t, errorMsg, "status 409")
		assert.Contains(t, errorMsg, "resource conflict")
	})

	t.Run("HandlesCaseInsensitiveFGAMDetection", func(t *testing.T) {
		// Test with different case variations
		testCases := []string{
			"Restricted API Access Configuration for permission system \"ps-test\" has changed",
			"RESTRICTED API ACCESS CONFIGURATION for permission system \"ps-test\" HAS CHANGED",
			"restricted api access configuration for permission system \"ps-test\" has changed",
		}

		for _, message := range testCases {
			apiErr := &client.APIError{
				StatusCode: 409,
				Message:    message,
			}

			errorMsg := apiErr.Error()
			assert.Contains(t, errorMsg, "Fine-Grained Access Management (FGAM) configuration",
				"Should detect FGAM conflict in message: %s", message)
		}
	})

	t.Run("HandlesNon409Errors", func(t *testing.T) {
		// Test that non-409 errors don't get FGAM treatment
		apiErr := &client.APIError{
			StatusCode: 400,
			Message:    "restricted API access configuration for permission system \"ps-test\" has changed",
		}

		errorMsg := apiErr.Error()

		// Should NOT contain the FGAM-specific context message for non-409 errors
		assert.NotContains(t, errorMsg, "Fine-Grained Access Management")
		assert.Contains(t, errorMsg, "status 400")
	})
}
