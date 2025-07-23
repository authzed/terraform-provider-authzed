package client

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// RetryConfig holds configuration for retry behaviour
type RetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxJitter   time.Duration
	ShouldRetry func(statusCode int) bool
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
func (rc *RetryConfig) RetryWithExponentialBackoff(
	operation func() (*ResponseWithETag, error),
	getLatestETag func() (string, error),
	updateWithETag func(string) (*ResponseWithETag, error),
	operationName string,
) (*ResponseWithETag, error) {
	var lastErr error
	var resp *ResponseWithETag

	for attempt := 0; attempt <= rc.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying
			delay := rc.calculateDelay(attempt - 1)
			time.Sleep(delay)
		}

		// First attempt uses the operation as-is, subsequent attempts need fresh ETag
		if attempt == 0 {
			resp, lastErr = operation()
		} else {
			// Get fresh ETag for retry
			latestETag, etagErr := getLatestETag()
			if etagErr != nil {
				lastErr = fmt.Errorf("failed to get latest ETag for retry attempt %d (%s): %w", attempt, operationName, etagErr)
				continue
			}

			// Retry with fresh ETag
			resp, lastErr = updateWithETag(latestETag)
		}

		// If no error or non-retryable error, return
		if lastErr != nil {
			return nil, lastErr
		}

		// Check if we should retry based on status code
		if !rc.ShouldRetry(resp.Response.StatusCode) {
			return resp, nil
		}

		// Close the response body before retrying
		if resp.Response.Body != nil {
			_ = resp.Response.Body.Close()
		}

		// Log retry attempt for debugging
		if attempt < rc.MaxRetries {
			fmt.Printf("FGAM conflict detected (status %d) for %s, retrying in %v (attempt %d/%d)\n",
				resp.Response.StatusCode, operationName, rc.calculateDelay(attempt), attempt+1, rc.MaxRetries)
		}
	}

	// Max retries exceeded
	return nil, fmt.Errorf("max retries (%d) exceeded for %s after FGAM conflicts", rc.MaxRetries, operationName)
}
