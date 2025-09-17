// Package backoff provides retry logic with exponential backoff and jitter
package backoff

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

// Config holds configuration for retry behavior
type Config struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxJitter   time.Duration
	ShouldRetry func(statusCode int) bool
}

// Result contains the result of a retry operation and any diagnostics
type Result struct {
	Response    HTTPResponse
	Diagnostics diag.Diagnostics
}

// HTTPResponse interface for responses that support retry operations
type HTTPResponse interface {
	GetETag() string
	GetResponse() *http.Response
}

func DefaultConfig() *Config {
	return &Config{
		MaxRetries:  5,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		MaxJitter:   500 * time.Millisecond,
		ShouldRetry: ShouldRetryForConflict,
	}
}

// ShouldRetryForConflict determines if a status code should trigger a retry
func ShouldRetryForConflict(statusCode int) bool {
	return statusCode == http.StatusConflict ||
		statusCode == http.StatusPreconditionFailed ||
		statusCode == http.StatusTooManyRequests
}

// calculateDelay calculates the delay for a given retry attempt with exponential backoff and jitter
func (c *Config) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	exponentialDelay := time.Duration(float64(c.BaseDelay) * math.Pow(2, float64(attempt)))

	// Cap at max delay
	exponentialDelay = min(exponentialDelay, c.MaxDelay)

	// Add random jitter
	jitter := time.Duration(rand.Intn(int(c.MaxJitter)))

	return exponentialDelay + jitter
}

// min returns the minimum of two time.Duration values
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// WithExponentialBackoff executes a function with exponential backoff retry logic
// Returns Result with diagnostics
func (c *Config) WithExponentialBackoff(
	ctx context.Context,
	operation func() (HTTPResponse, error),
	getLatestETag func() (string, error),
	updateWithETag func(string) (HTTPResponse, error),
	operationName string,
) *Result {
	var lastErr error
	var resp HTTPResponse
	var diagnostics diag.Diagnostics

	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateDelay(attempt - 1)

			diagnostics.AddWarning(
				"Retrying Operation",
				fmt.Sprintf("Configuration changed during %s. Retrying in %v (attempt %d of %d).",
					operationName, delay, attempt+1, c.MaxRetries+1),
			)

			tflog.Warn(ctx, "conflict detected, retrying with exponential backoff", map[string]interface{}{
				"operation":    operationName,
				"status_code":  resp.GetResponse().StatusCode,
				"attempt":      attempt + 1,
				"max_attempts": c.MaxRetries + 1,
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
				lastErr = etagErr
				resp = nil
			} else {
				// Retry with fresh e-tag
				resp, lastErr = updateWithETag(latestETag)
			}
		}

		if lastErr != nil {
			return &Result{
				Response:    nil,
				Diagnostics: diagnostics,
			}
		}

		// Check if we should retry based on status code
		if !c.ShouldRetry(resp.GetResponse().StatusCode) {
			if attempt > 0 {
				diagnostics.AddWarning(
					"Retry Successful",
					fmt.Sprintf("%s succeeded after %d retries.",
						operationName, attempt),
				)

				tflog.Info(ctx, "retry succeeded", map[string]interface{}{
					"operation":     operationName,
					"final_attempt": attempt + 1,
					"total_retries": attempt,
				})
			}
			return &Result{
				Response:    resp,
				Diagnostics: diagnostics,
			}
		}

		// Close the response body before retrying
		if resp.GetResponse().Body != nil {
			_ = resp.GetResponse().Body.Close()
		}
	}

	// Max retries exceeded
	diagnostics.AddError(
		"Max Retries Exceeded",
		fmt.Sprintf("Maximum retries (%d) exceeded for %s. The configuration is changing too frequently.",
			c.MaxRetries, operationName),
	)

	tflog.Error(ctx, "max retries exceeded", map[string]interface{}{
		"operation":    operationName,
		"max_retries":  c.MaxRetries,
		"final_status": resp.GetResponse().StatusCode,
	})

	return &Result{
		Response:    nil,
		Diagnostics: diagnostics,
	}
}

// WithExponentialBackoffLegacy provides legacy compatibility for older retry patterns
func (c *Config) WithExponentialBackoffLegacy(
	ctx context.Context,
	operation func() (HTTPResponse, error),
	getLatestETag func() (string, error),
	updateWithETag func(string) (HTTPResponse, error),
	operationName string,
) (HTTPResponse, error) {
	var lastErr error
	var resp HTTPResponse

	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateDelay(attempt - 1)

			// Use tflog for legacy compatibility
			tflog.Info(ctx, fmt.Sprintf("FGAM Conflict: Fine-Grained Access Management configuration changed during %s. Retrying in %v (attempt %d of %d)...",
				operationName, delay, attempt+1, c.MaxRetries+1))

			time.Sleep(delay)
		}

		// First attempt uses the operation as-is, subsequent attempts need fresh e-tag
		if attempt == 0 {
			resp, lastErr = operation()
		} else {
			// Get fresh e-tag for retry
			latestETag, etagErr := getLatestETag()
			if etagErr != nil {
				lastErr = etagErr
				resp = nil
			} else {
				// Retry with fresh e-tag
				resp, lastErr = updateWithETag(latestETag)
			}
		}

		if lastErr != nil {
			return nil, lastErr
		}

		// Check if we should retry based on status code
		if !c.ShouldRetry(resp.GetResponse().StatusCode) {
			if attempt > 0 {
				// Log successful retry
				tflog.Info(ctx, fmt.Sprintf("FGAM Retry Successful: %s completed after %d retries.", operationName, attempt))
			}
			return resp, nil
		}

		// Close the response body before retrying
		if resp.GetResponse().Body != nil {
			_ = resp.GetResponse().Body.Close()
		}
	}

	// Max retries exceeded
	tflog.Error(ctx, fmt.Sprintf("FGAM Retries Exhausted: Maximum retries (%d) exceeded for %s. The Fine-Grained Access Management configuration is changing too frequently.",
		c.MaxRetries, operationName))

	return nil, fmt.Errorf("max retries (%d) exceeded for %s after FGAM conflicts", c.MaxRetries, operationName)
}

// RetryTerraformOperation provides Terraform-specific retry logic with timeout handling
// This matches the pattern from the previous implementation
func RetryTerraformOperation(ctx context.Context, timeout time.Duration, operation func() error) error {
	config := DefaultConfig()

	// Create timeout context if not already bounded
	opCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		opCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := config.calculateDelay(attempt - 1)

			tflog.Warn(opCtx, "terraform operation failed, retrying with exponential backoff", map[string]interface{}{
				"attempt":      attempt + 1,
				"max_attempts": config.MaxRetries + 1,
				"retry_delay":  delay.String(),
			})

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-opCtx.Done():
				return fmt.Errorf("operation cancelled during retry delay: %w", opCtx.Err())
			}
		}

		// Execute the operation
		lastErr = operation()
		if lastErr == nil {
			if attempt > 0 {
				tflog.Info(opCtx, "terraform operation succeeded after retries", map[string]interface{}{
					"total_attempts": attempt + 1,
					"total_retries":  attempt,
				})
			}
			return nil
		}

		// Check if we should retry based on the error
		// For now, retry on all errors - can be made more sophisticated later
		tflog.Debug(opCtx, "terraform operation failed", map[string]interface{}{
			"attempt": attempt + 1,
			"error":   lastErr.Error(),
		})
	}

	// Max retries exceeded
	tflog.Error(opCtx, "terraform operation failed after max retries", map[string]interface{}{
		"max_retries": config.MaxRetries,
		"final_error": lastErr.Error(),
	})

	return fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries, lastErr)
}

// RetryOnEventualConsistency provides retry logic for read-after-write consistency verification
// This matches the pattern from the previous implementation for eventual consistency handling
func RetryOnEventualConsistency(ctx context.Context, maxAttempts int, operation func() error) error {
	if maxAttempts <= 0 {
		maxAttempts = 3 // Default from previous implementation
	}

	var lastErr error
	baseDelay := 100 * time.Millisecond // Start with short delay for consistency checks

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter for consistency checks
			delay := time.Duration(float64(baseDelay) * math.Pow(1.5, float64(attempt-1)))
			jitter := time.Duration(rand.Intn(int(delay.Nanoseconds() / 2))) // Up to 50% jitter
			totalDelay := delay + jitter

			tflog.Debug(ctx, "retrying eventual consistency check", map[string]interface{}{
				"attempt":      attempt + 1,
				"max_attempts": maxAttempts,
				"delay":        totalDelay.String(),
			})

			select {
			case <-time.After(totalDelay):
				// Continue with retry
			case <-ctx.Done():
				return fmt.Errorf("consistency check cancelled: %w", ctx.Err())
			}
		}

		lastErr = operation()
		if lastErr == nil {
			if attempt > 0 {
				tflog.Debug(ctx, "eventual consistency achieved", map[string]interface{}{
					"attempts_needed": attempt + 1,
				})
			}
			return nil
		}

		tflog.Debug(ctx, "consistency check failed, will retry", map[string]interface{}{
			"attempt": attempt + 1,
			"error":   lastErr.Error(),
		})
	}

	tflog.Warn(ctx, "eventual consistency not achieved within max attempts", map[string]interface{}{
		"max_attempts": maxAttempts,
		"final_error":  lastErr.Error(),
	})

	return fmt.Errorf("consistency check failed after %d attempts: %w", maxAttempts, lastErr)
}
