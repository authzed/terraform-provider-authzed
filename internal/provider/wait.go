package provider

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
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

// classifyMaterializeDeploymentError inspects err for an *client.APIError.
func classifyMaterializeDeploymentError(err error) (isNotFound, isPermanent bool) {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return false, false
	}
	if apiErr.StatusCode == http.StatusNotFound {
		return true, false
	}
	return false, apiErr.StatusCode >= 400 && apiErr.StatusCode < 500
}

// materializeDeploymentGone reports whether err means the deployment no
// longer exists: 404, or 403 on API versions where deletion removes the
// deployment's authorization relationships and reads of a deleted deployment
// are rejected as forbidden — matching the delete polling's gone semantics.
func materializeDeploymentGone(err error) bool {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusForbidden
}

// materializeDeploymentReady reports whether a deployment has been
// provisioned and started doing useful work. Snapshotting and hydration are
// data-dependent and can take hours, so waiting for the overall Running
// phase (fully hydrated and serving) is unrealistic; instead the deployment
// counts as ready once its snapshot job is underway (Running), already done
// (Complete), or intentionally off (Disabled) — or once the deployment
// somehow reached Running outright.
func materializeDeploymentReady(deployment *models.MaterializeDeployment) bool {
	if deployment.Status == nil {
		return false
	}
	if deployment.Status.Phase == models.MaterializePhaseRunning {
		return true
	}
	if deployment.Status.Snapshot == nil {
		return false
	}
	switch deployment.Status.Snapshot.Phase {
	case models.MaterializeJobPhaseRunning, models.MaterializeJobPhaseComplete, models.MaterializeJobPhaseDisabled:
		return true
	}
	return false
}

// waitForMaterializeDeploymentReady waits for a materialize deployment to be
// provisioned and snapshotting (see materializeDeploymentReady) — NOT fully
// hydrated, which can take hours — and returns the deployment observed by
// its final poll, saving callers a follow-up read. Callers control the
// overall wait via ctx. A 404 means the deployment is not yet visible;
// Degraded/Unknown phases can be transient during provisioning, so both keep
// polling until ctx expires. Unlike waitForExists, this fails fast on any
// other 4xx response (e.g. an invalid or expired token) instead of burning
// the full wait budget on an error that will never resolve on its own.
func waitForMaterializeDeploymentReady(ctx context.Context, cloudClient *client.CloudClient, id string) (*models.MaterializeDeployment, error) {
	var ready *models.MaterializeDeployment
	// lastState remembers what the most recent poll observed: on ctx expiry
	// backoff.Retry returns a bare context error, and without this a stuck
	// deployment times out with no hint of why (e.g. a Degraded phase).
	var lastState error
	operation := func() error {
		deployment, err := cloudClient.GetMaterializeDeployment(ctx, id)
		if err != nil {
			isNotFound, isPermanent := classifyMaterializeDeploymentError(err)
			switch {
			case isNotFound:
				// Not visible yet, keep waiting.
				lastState = fmt.Errorf("not visible yet: %w", err)
			case isPermanent:
				// Permanent client error (e.g. a bad token); the client
				// already retried 429s, so this won't resolve itself and
				// there is no point burning the rest of the wait budget.
				return backoff.Permanent(err)
			default:
				// Network error or a 5xx that survived the client's own retries.
				lastState = err
			}
			return lastState
		}
		if materializeDeploymentReady(deployment) {
			ready = deployment
			return nil
		}
		phase := ""
		snapshotPhase := ""
		if deployment.Status != nil {
			phase = deployment.Status.Phase
			if deployment.Status.Snapshot != nil {
				snapshotPhase = deployment.Status.Snapshot.Phase
			}
		}
		lastState = fmt.Errorf("not ready yet (phase %q, snapshot phase %q)", phase, snapshotPhase)
		return lastState
	}

	if err := backoff.Retry(operation, backoff.WithContext(client.NewPollBackOff(), ctx)); err != nil {
		if lastState != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
			err = fmt.Errorf("%w (last state: %w)", err, lastState)
		}
		return nil, fmt.Errorf("waiting for materialize deployment %s to start snapshotting: %w", id, err)
	}
	return ready, nil
}
