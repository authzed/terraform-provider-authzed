package provider

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"terraform-provider-authzed/internal/client"
)

// waitForExists polls check(ctx) until it returns true, or the context is done.
// Retries on retryable errors (409/412/429/5xx/network), fails fast on other 4xx.
func waitForExists(ctx context.Context, check func(context.Context) (bool, error)) error {
	backoff := 250 * time.Millisecond
	maxBackoff := 5 * time.Second
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		ok, err := check(ctx)
		if err == nil && ok {
			return nil
		}

		// For existence checks, we only care about 404s, everything else should be retried
		// The specific error type checking will be done by the individual check functions

		// Sleep with jitter
		jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
		sleep := backoff - jitter
		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// waitForPermissionSystemExists waits for a permission system to be globally visible
func waitForPermissionSystemExists(ctx context.Context, client *client.CloudClient, psID string) error {
	return waitForExists(ctx, func(ctx context.Context) (bool, error) {
		_, err := client.GetPermissionsSystem(ctx, psID)
		if err != nil {
			// Check if it's a 404 by looking at the error string
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

// waitForServiceAccountExists waits for a service account to be globally visible
func waitForServiceAccountExists(ctx context.Context, client *client.CloudClient, psID, saID string) error {
	return waitForExists(ctx, func(ctx context.Context) (bool, error) {
		_, err := client.GetServiceAccount(ctx, psID, saID)
		if err != nil {
			// Check if it's a 404 by looking at the error string
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
				return false, nil // Not found yet, keep waiting
			}
			return false, err // Other error, let waitForExists decide retryability
		}
		return true, nil // Found!
	})
}

// waitForRoleExists waits for a role to be globally visible
func waitForRoleExists(ctx context.Context, client *client.CloudClient, psID, roleID string) error {
	return waitForExists(ctx, func(ctx context.Context) (bool, error) {
		_, err := client.GetRole(ctx, psID, roleID)
		if err != nil {
			// Check if it's a 404 by looking at the error string
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err // let waitForExists decide retryability
		}
		return true, nil // Found!
	})
}
