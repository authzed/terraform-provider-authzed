package client

import (
	"context"
	"errors"
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

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  DefaultMaxRetries,
		BaseDelay:   DefaultBaseRetryDelay,
		MaxDelay:    DefaultMaxRetryDelay,
		MaxJitter:   DefaultMaxJitter,
		ShouldRetry: shouldRetryForConflict,
	}
}

// shouldRetryForConflict determines if a status code should trigger a retry
func shouldRetryForConflict(statusCode int) bool {
	return statusCode == http.StatusConflict ||
		statusCode == http.StatusPreconditionFailed ||
		statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusNotFound || // For service account visibility delays
		(statusCode >= 500 && statusCode <= 599)
}

// calculateDelay calculates the delay for a given retry attempt with exponential backoff and jitter
func (rc *RetryConfig) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	exponentialDelay := time.Duration(float64(rc.BaseDelay) * math.Pow(2, float64(attempt)))

	// Cap at max delay
	exponentialDelay = min(exponentialDelay, rc.MaxDelay)

	// Add random jitter
	jitter := time.Duration(rand.Intn(int(rc.MaxJitter)))

	return exponentialDelay + jitter
}

// RetryWithExponentialBackoff executes a function with exponential backoff retry logic
// Returns RetryResult with diagnostics
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

			diagnostics.AddWarning(
				"Retrying Operation",
				fmt.Sprintf("Configuration changed during %s. Retrying in %v (attempt %d of %d).",
					operationName, delay, attempt+1, rc.MaxRetries+1),
			)

			tflog.Warn(ctx, "conflict detected, retrying with exponential backoff", map[string]any{
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

		// If there's an error, check if it's retryable
		if lastErr != nil {
			// Check if this is a retryable error (APIError with retryable status code)
			apiErr := &APIError{}
			if errors.As(lastErr, &apiErr) && rc.ShouldRetry(apiErr.StatusCode) {
				// This is a retryable error, continue the retry loop
				tflog.Debug(ctx, "retryable error encountered, continuing retry loop", map[string]any{
					"operation":   operationName,
					"status_code": apiErr.StatusCode,
					"attempt":     attempt + 1,
					"error":       lastErr.Error(),
				})
				continue
			}
			// Non-retryable error, exit immediately
			return &RetryResult{
				Response:    nil,
				Diagnostics: diagnostics,
			}
		}

		// Check if we should retry based on status code (for successful responses)
		if !rc.ShouldRetry(resp.Response.StatusCode) {
			if attempt > 0 {
				diagnostics.AddWarning(
					"Retry Successful",
					fmt.Sprintf("%s succeeded after %d retries.",
						operationName, attempt),
				)

				tflog.Info(ctx, "retry succeeded", map[string]any{
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

	diagnostics.AddError(
		"Retries Exhausted",
		fmt.Sprintf("Maximum retries (%d) exceeded for %s. The configuration is changing too frequently.",
			rc.MaxRetries, operationName),
	)

	tflog.Error(ctx, "retries exhausted", map[string]any{
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

		// If there's an error, check if it's retryable
		if lastErr != nil {
			// Check if this is a retryable error (APIError with retryable status code)
			apiErr := &APIError{}
			if errors.As(lastErr, &apiErr) && rc.ShouldRetry(apiErr.StatusCode) {
				// This is a retryable error, continue the retry loop
				tflog.Debug(ctx, "retryable error encountered, continuing retry loop", map[string]any{
					"operation":   operationName,
					"status_code": apiErr.StatusCode,
					"attempt":     attempt + 1,
					"error":       lastErr.Error(),
				})
				continue
			}
			// Non-retryable error, exit immediately
			return nil, lastErr
		}

		// Check if we should retry based on status code (for successful responses)
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
