package pslanes

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"terraform-provider-authzed/internal/client"
)

// Retry409Delete retries delete operations on 409 conflicts with exponential backoff
func Retry409Delete(ctx context.Context, operation func() error) error {
	backoff := 500 * time.Millisecond
	maxBackoff := 5 * time.Second
	maxRetries := 10

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		// Treat 404 as success, idempotent deletes
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == http.StatusNotFound {
			return nil
		}

		// Only retry on 409 conflicts
		if !is409Conflict(err) {
			return err
		}

		// Don't sleep on the last attempt
		if attempt == maxRetries-1 {
			return err
		}

		// Exponential backoff with jitter to prevent thundering herd
		jitter := time.Duration(rand.Int63n(int64(backoff / 4)))
		sleep := backoff + jitter

		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			return ctx.Err()
		}

		// Increase backoff for next attempt
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return fmt.Errorf("delete operation failed after %d attempts", maxRetries)
}

// is409Conflict checks if the error is a 409 conflict
func is409Conflict(err error) bool {
	if apiErr, ok := err.(*client.APIError); ok {
		return apiErr.StatusCode == http.StatusConflict
	}

	// Also check for string-based conflict indicators
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "conflict") || strings.Contains(errStr, "409")
}
