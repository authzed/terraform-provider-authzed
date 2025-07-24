package client

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// RetryConfig holds configuration for retry behaviour
type RetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxJitter   time.Duration
	ShouldRetry func(statusCode int) bool
}

// RetryResult contains the result of a retry operation and any diagnostics
type RetryResult struct {
	Response    *ResponseWithETag
	Diagnostics diag.Diagnostics
}

// DefaultRetryConfig returns the default retry configuration for FGAM conflicts
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  DefaultMaxRetries,
		BaseDelay:   DefaultBaseRetryDelay,
		MaxDelay:    DefaultMaxRetryDelay,
		MaxJitter:   DefaultMaxJitter,
		ShouldRetry: shouldRetryForFGAMConflict,
	}
}

// shouldRetryForFGAMConflict determines if a status code should trigger a retry
func shouldRetryForFGAMConflict(statusCode int) bool {
	return statusCode == http.StatusConflict || statusCode == http.StatusPreconditionFailed
}

// calculateDelay calculates the delay for a given retry attempt with exponential backoff and jitter
func (rc *RetryConfig) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	exponentialDelay := time.Duration(float64(rc.BaseDelay) * math.Pow(2, float64(attempt)))

	// Cap at max delay
	exponentialDelay = min(exponentialDelay, rc.MaxDelay)

	// Add random jitter to prevent thundering herd
	jitter := time.Duration(rand.Intn(int(rc.MaxJitter)))

	return exponentialDelay + jitter
}

// RetryWithExponentialBackoff executes a function with exponential backoff retry logic
// Returns RetryResult with diagnostics that appear in normal Terraform output
func (rc *RetryConfig) RetryWithExponentialBackoff(
	ctx context.Context,
	operation func() (*ResponseWithETag, error),
	getLatestETag func() (string, error),
	updateWithETag func(string) (*ResponseWithETag, error),
	operationName string,
) *RetryResult {
	var lastErr error
	var resp *ResponseWithETag
	var diagnostics diag.Diagnostics

	for attempt := 0; attempt <= rc.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := rc.calculateDelay(attempt - 1)

			// Add user-visible warning about retry - this shows in normal terraform output
			diagnostics.AddWarning(
				"FGAM Conflict - Retrying Operation",
				fmt.Sprintf("Fine-Grained Access Management configuration changed during %s. Retrying in %v (attempt %d of %d).",
					operationName, delay, attempt+1, rc.MaxRetries+1),
			)

			// Also log for debugging
			tflog.Warn(ctx, "FGAM conflict detected, retrying with exponential backoff", map[string]interface{}{
				"operation":    operationName,
				"status_code":  resp.Response.StatusCode,
				"attempt":      attempt + 1,
				"max_attempts": rc.MaxRetries + 1,
				"retry_delay":  delay.String(),
			})

			time.Sleep(delay)
		}

		// First attempt uses the operation as-is, subsequent attempts need fresh e-tag
		if attempt == 0 {
			resp, lastErr = operation()
		} else {
			// Get fresh e-tag for retry
			latestETag, etagErr := getLatestETag()
			if etagErr != nil {
				continue
			}

			// Retry with fresh e-tag
			resp, lastErr = updateWithETag(latestETag)
		}

		if lastErr != nil {
			return &RetryResult{
				Response:    nil,
				Diagnostics: diagnostics,
			}
		}

		// Check if we should retry based on status code
		if !rc.ShouldRetry(resp.Response.StatusCode) {
			if attempt > 0 {
				// Add user-visible info about successful retry
				diagnostics.AddWarning(
					"FGAM Retry Successful",
					fmt.Sprintf("%s succeeded after %d retries due to FGAM configuration changes.",
						operationName, attempt),
				)

				// Also log for debugging
				tflog.Info(ctx, "FGAM retry succeeded", map[string]interface{}{
					"operation":     operationName,
					"final_attempt": attempt + 1,
					"total_retries": attempt,
				})
			}
			return &RetryResult{
				Response:    resp,
				Diagnostics: diagnostics,
			}
		}

		// Close the response body before retrying
		if resp.Response.Body != nil {
			_ = resp.Response.Body.Close()
		}
	}

	// Max retries exceeded - add user-visible error
	diagnostics.AddError(
		"FGAM Retries Exhausted",
		fmt.Sprintf("Maximum retries (%d) exceeded for %s due to persistent FGAM conflicts. The Fine-Grained Access Management configuration is changing too frequently.",
			rc.MaxRetries, operationName),
	)

	// Also log for debugging
	tflog.Error(ctx, "FGAM retries exhausted", map[string]interface{}{
		"operation":    operationName,
		"max_attempts": rc.MaxRetries + 1,
		"final_status": resp.Response.StatusCode,
	})

	return &RetryResult{
		Response:    nil,
		Diagnostics: diagnostics,
	}
}

// RetryWithExponentialBackoffLegacy executes a function with exponential backoff retry logic (OG version)
func (rc *RetryConfig) RetryWithExponentialBackoffLegacy(
	ctx context.Context,
	operation func() (*ResponseWithETag, error),
	getLatestETag func() (string, error),
	updateWithETag func(string) (*ResponseWithETag, error),
	operationName string,
) (*ResponseWithETag, error) {
	var lastErr error
	var resp *ResponseWithETag

	for attempt := 0; attempt <= rc.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := rc.calculateDelay(attempt - 1)

			// Use tflog for legacy compatibility
			tflog.Info(ctx, fmt.Sprintf("FGAM Conflict: Fine-Grained Access Management configuration changed during %s. Retrying in %v (attempt %d of %d)...",
				operationName, delay, attempt+1, rc.MaxRetries+1))

			time.Sleep(delay)
		}

		// First attempt uses the operation as-is, subsequent attempts need fresh e-tag
		if attempt == 0 {
			resp, lastErr = operation()
		} else {
			// Get fresh e-tag for retry
			latestETag, etagErr := getLatestETag()
			if etagErr != nil {
				continue
			}

			// Retry with fresh e-tag
			resp, lastErr = updateWithETag(latestETag)
		}

		if lastErr != nil {
			return nil, lastErr
		}

		// Check if we should retry based on status code
		if !rc.ShouldRetry(resp.Response.StatusCode) {
			if attempt > 0 {
				// Log successful retry
				tflog.Info(ctx, fmt.Sprintf("FGAM Retry Successful: %s completed after %d retries.", operationName, attempt))
			}
			return resp, nil
		}

		// Close the response body before retrying
		if resp.Response.Body != nil {
			_ = resp.Response.Body.Close()
		}
	}

	// Max retries exceeded
	tflog.Error(ctx, fmt.Sprintf("FGAM Retries Exhausted: Maximum retries (%d) exceeded for %s. The Fine-Grained Access Management configuration is changing too frequently.",
		rc.MaxRetries, operationName))

	return nil, fmt.Errorf("max retries (%d) exceeded for %s after FGAM conflicts", rc.MaxRetries, operationName)
}
