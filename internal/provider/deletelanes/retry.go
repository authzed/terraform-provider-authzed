package deletelanes

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"time"

	"terraform-provider-authzed/internal/client"
)

// isRetryable checks if an error should be retried based on HTTP status code
func isRetryable(err error) bool {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.StatusCode
		return code == http.StatusConflict ||
			code == http.StatusTooManyRequests ||
			(code >= 500 && code <= 599)
	}
	return false
}

// Retry409Delete retries delete operations on retryable errors (409, 429, 5xx) with jittered backoff
func Retry409Delete(ctx context.Context, fn func(context.Context) error) error {
	const (
		initialDelay = 500 * time.Millisecond
		maxDelay     = 5 * time.Second
		jitterFactor = 0.2
	)

	delay := initialDelay

	for {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		// Treat 404 as success (resource already deleted)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil
		}

		// Only retry on retryable errors (409, 429, 5xx)
		if isRetryable(err) {
			// Check if we have time for another attempt
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Add jitter to delay
			jitteredDelay := addJitter(delay, jitterFactor)

			// Check if we can fit this delay within the deadline
			deadline, hasDeadline := ctx.Deadline()
			if hasDeadline && time.Now().Add(jitteredDelay).After(deadline) {
				return err // No time for retry
			}

			// Sleep with jitter
			timer := time.NewTimer(jitteredDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				// Continue to next attempt
			}

			// Exponential backoff with cap
			delay = time.Duration(float64(delay) * 2)
			if delay > maxDelay {
				delay = maxDelay
			}

			continue
		}

		// Non-retryable error
		return err
	}
}

// addJitter adds random jitter to a duration
func addJitter(duration time.Duration, factor float64) time.Duration {
	if factor <= 0 {
		return duration
	}

	jitter := time.Duration(float64(duration) * factor * (rand.Float64()*2 - 1))
	result := duration + jitter

	// Ensure we don't go negative
	if result < 0 {
		return duration / 2
	}

	return result
}
