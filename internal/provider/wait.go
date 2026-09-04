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

// classifyAPIError inspects err for an *client.APIError.
func classifyAPIError(err error) (isNotFound, isPermanent bool) {
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

// terminalPreflightReasons are the failures worth giving up on. The service
// retries a failed check on its own, so everything else keeps polling.
var terminalPreflightReasons = map[string]struct{}{
	models.MaterializePreflightReasonInvalidWatchedPermission: {},
	models.MaterializePreflightReasonSchemaEmpty:              {},
	models.MaterializePreflightReasonNoWatchedPermissions:     {},
}

// materializeConditions returns the deployment's conditions, or none.
func materializeConditions(deployment *models.MaterializeDeployment) []models.MaterializeDeploymentStatusCondition {
	if deployment.Status == nil {
		return nil
	}
	return deployment.Status.Conditions
}

// materializeConditionActive reports whether the named condition holds.
func materializeConditionActive(deployment *models.MaterializeDeployment, conditionType string) bool {
	for _, condition := range materializeConditions(deployment) {
		if condition.Type == conditionType {
			return condition.Status == models.MaterializeConditionTrue
		}
	}
	return false
}

// describeMaterializeCondition renders a condition as "Type: Reason: message".
func describeMaterializeCondition(condition models.MaterializeDeploymentStatusCondition) string {
	parts := make([]string, 0, 3)
	for _, part := range []string{condition.Type, condition.Reason, condition.Message} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ": ")
}

// materializeFailureDetail describes every failure the deployment reports, or
// "" if none. Retryable ones are included: if the deployment never recovers,
// this is the only clue the user gets about why. The name match is loose on
// purpose, so failures added later are still described.
func materializeFailureDetail(deployment *models.MaterializeDeployment) string {
	var failures []string
	for _, condition := range materializeConditions(deployment) {
		if condition.Status != models.MaterializeConditionTrue {
			continue
		}
		if strings.HasSuffix(condition.Type, "Failed") || strings.HasSuffix(condition.Type, "Error") {
			failures = append(failures, describeMaterializeCondition(condition))
		}
	}
	return strings.Join(failures, "; ")
}

// materializeObservedGeneration returns the newest configuration version the
// deployment's status describes, or 0 if it reports none.
func materializeObservedGeneration(deployment *models.MaterializeDeployment) int64 {
	var newest int64
	for _, condition := range materializeConditions(deployment) {
		if condition.ObservedGeneration > newest {
			newest = condition.ObservedGeneration
		}
	}
	return newest
}

// materializeConfigCheckRunning reports whether the deployment is checking
// its configuration, so there is no verdict on it yet.
func materializeConfigCheckRunning(deployment *models.MaterializeDeployment) bool {
	return materializeConditionActive(deployment, models.MaterializeConditionPreflightInProgress)
}

// materializeSettleWindow caps how long a tracker waits for a configuration
// check that may never run.
var materializeSettleWindow = 20 * time.Second

// materializeChangeTracker reports when a deployment's status describes the
// change just made, rather than the configuration it replaced. For a moment
// after a change the deployment still reports the old configuration, and
// reports it as ready, so two things have to happen before its status means
// anything: it has to name a newer configuration version, and the
// configuration check has to run and finish. Some changes need no check, so
// the wait also ends materializeSettleWindow after the newer version first
// shows up.
type materializeChangeTracker struct {
	previousGeneration int64
	settleBy           time.Time
	newGenerationSeen  bool
	checkSeen          bool
	tookEffect         bool
}

// newMaterializeChangeTracker tracks a change to a deployment that was on
// previousGeneration. Zero means nothing was there before, as on create,
// where the status can be trusted straight away.
func newMaterializeChangeTracker(previousGeneration int64) *materializeChangeTracker {
	return &materializeChangeTracker{
		previousGeneration: previousGeneration,
		tookEffect:         previousGeneration == 0,
	}
}

// TookEffect reports whether this observation of the deployment describes the
// change. Once it does, it always does.
func (t *materializeChangeTracker) TookEffect(deployment *models.MaterializeDeployment) bool {
	if t.tookEffect {
		return true
	}
	if materializeObservedGeneration(deployment) <= t.previousGeneration {
		return false
	}
	if !t.newGenerationSeen {
		// The window gives the configuration check time to start, so it can
		// only start counting once there is a newer version for it to check.
		// Started any earlier, a service that takes longer than the window
		// to bump the version would have the restamped previous verdict —
		// ready, and passing a check that has not run — accepted on sight.
		t.newGenerationSeen = true
		t.settleBy = time.Now().Add(materializeSettleWindow)
	}
	switch {
	case materializeConfigCheckRunning(deployment):
		t.checkSeen = true
	case t.checkSeen, time.Now().After(t.settleBy):
		t.tookEffect = true
	}
	return t.tookEffect
}

// materializeConfigRejectedError means the user has to change their
// configuration. Callers check for it so they do not blame a timeout.
type materializeConfigRejectedError struct{ err error }

func (e *materializeConfigRejectedError) Error() string { return e.err.Error() }
func (e *materializeConfigRejectedError) Unwrap() error { return e.err }

// terminalMaterializeFailure returns an error if the deployment can never
// start with the configuration it was given, so waiting longer is pointless.
// Only call it once the change has taken effect, or it can report a failure
// left over from the configuration being replaced.
func terminalMaterializeFailure(deployment *models.MaterializeDeployment) error {
	for _, condition := range materializeConditions(deployment) {
		if condition.Type != models.MaterializeConditionPreflightFailed ||
			condition.Status != models.MaterializeConditionTrue {
			continue
		}
		if _, terminal := terminalPreflightReasons[condition.Reason]; !terminal {
			continue
		}
		detail := condition.Reason
		if condition.Message != "" {
			detail += ": " + condition.Message
		}
		return &materializeConfigRejectedError{
			err: fmt.Errorf("preflight check failed: %s", detail),
		}
	}
	return nil
}

// waitForMaterializeDeploymentReady waits for a materialize deployment to be
// provisioned and snapshotting (see materializeDeploymentReady) — NOT fully
// hydrated, which can take long — and returns the deployment observed by its
// final poll, saving callers a follow-up read.
//
// previousGeneration is the configuration version the deployment was on
// before the change being waited on; pass 0 on create (see
// materializeChangeTracker).
//
// A 404 means the deployment is not yet visible, and Degraded/Unknown phases
// can be transient while provisioning, so both keep polling until ctx
// expires. Unlike waitForExists, this fails fast on any other 4xx response
// (e.g. an invalid or expired token) and on a rejected configuration, rather
// than waiting for something will never resolve itself.
func waitForMaterializeDeploymentReady(ctx context.Context, cloudClient *client.CloudClient, id string, previousGeneration int64) (*models.MaterializeDeployment, error) {
	change := newMaterializeChangeTracker(previousGeneration)

	var ready *models.MaterializeDeployment
	// lastState remembers what the most recent poll observed: on ctx expiry
	// backoff.Retry returns a bare context error, and without this a stuck
	// deployment times out with no hint of why (e.g. a Degraded phase).
	var lastState error
	operation := func() error {
		deployment, err := cloudClient.GetMaterializeDeployment(ctx, id)
		if err != nil {
			isNotFound, isPermanent := classifyAPIError(err)
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
		if !change.TookEffect(deployment) {
			lastState = errors.New("the new configuration has not taken effect yet")
			// On timeout this message is all the user sees, so say what the
			// deployment is complaining about while it fails to catch up.
			if detail := materializeFailureDetail(deployment); detail != "" {
				lastState = fmt.Errorf("%w: %s", lastState, detail)
			}
			return lastState
		}
		// Before the ready test: the previous configuration's jobs can still
		// be running, which looks ready even though the new one was rejected.
		if failure := terminalMaterializeFailure(deployment); failure != nil {
			return backoff.Permanent(failure)
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
		// On timeout this message is all the user sees, so say what failed.
		if detail := materializeFailureDetail(deployment); detail != "" {
			lastState = fmt.Errorf("%w: %s", lastState, detail)
		}
		return lastState
	}

	if err := backoff.Retry(operation, backoff.WithContext(client.NewPollBackOff(), ctx)); err != nil {
		// A rejected configuration is terminal; the "waiting for ..." framing would be misleading.
		var rejected *materializeConfigRejectedError
		if errors.As(err, &rejected) {
			return nil, err
		}
		if lastState != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
			err = fmt.Errorf("%w (last state: %w)", err, lastState)
		}
		return nil, fmt.Errorf("waiting for materialize deployment %s to start snapshotting: %w", id, err)
	}
	return ready, nil
}
