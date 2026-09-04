package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
)

func TestWaitForMaterializeDeploymentReadySnapshotRunning(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		// First poll: not yet visible. Second: provisioning, snapshot pending.
		// Third+: still provisioning overall, but the snapshot job is running —
		// that is the readiness signal (full hydration can take hours).
		if n == 1 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		snapshotPhase := "Pending"
		if n >= 3 {
			snapshotPhase = "Running"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Provisioning", "snapshot": {"phase": %q}}}}`, snapshotPhase)
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deployment, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	if err != nil {
		t.Fatalf("waitForMaterializeDeploymentReady failed: %v", err)
	}
	if deployment == nil || deployment.ID != "mc-test" {
		t.Fatalf("expected the observed deployment to be returned, got %+v", deployment)
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 polls (404, snapshot Pending, snapshot Running), got %d", calls)
	}
}

func TestWaitForMaterializeDeploymentReadyRunningPhase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Running"}}}`)
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deployment, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	if err != nil {
		t.Fatalf("waitForMaterializeDeploymentReady failed on Running phase: %v", err)
	}
	if deployment == nil || deployment.ID != "mc-test" {
		t.Fatalf("expected the observed deployment to be returned, got %+v", deployment)
	}
}

func TestWaitForMaterializeDeploymentReadyTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Provisioning", "snapshot": {"phase": "Pending"}}}}`)
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// The timeout must say what the last poll observed — a bare "context
	// deadline exceeded" tells the operator nothing about why it is stuck.
	if !strings.Contains(err.Error(), `phase "Provisioning"`) || !strings.Contains(err.Error(), `snapshot phase "Pending"`) {
		t.Fatalf("expected timeout error to include the last observed state, got: %v", err)
	}
}

func TestWaitForMaterializeDeploymentReadyFailsFastOn403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"message": "forbidden"}`)
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	// Generous ctx timeout: the point of this test is that we return well
	// before it expires, proving the 403 short-circuits the wait instead of
	// being retried until the context deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error for a permanent 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected error to mention 403, got: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("expected fail-fast return well under the 10s ctx deadline, took %v", elapsed)
	}
}

func TestMaterializeDeploymentReady(t *testing.T) {
	cases := []struct {
		name   string
		status *models.MaterializeDeploymentStatus
		want   bool
	}{
		{"nil status", nil, false},
		{"provisioning, no snapshot info", &models.MaterializeDeploymentStatus{Phase: "Provisioning"}, false},
		{"snapshot pending", &models.MaterializeDeploymentStatus{Phase: "Provisioning", Snapshot: &models.MaterializeDeploymentSnapshotInfo{Phase: "Pending"}}, false},
		{"snapshot failed", &models.MaterializeDeploymentStatus{Phase: "Provisioning", Snapshot: &models.MaterializeDeploymentSnapshotInfo{Phase: "Failed"}}, false},
		{"snapshot running", &models.MaterializeDeploymentStatus{Phase: "Provisioning", Snapshot: &models.MaterializeDeploymentSnapshotInfo{Phase: "Running"}}, true},
		{"snapshot complete", &models.MaterializeDeploymentStatus{Phase: "Provisioning", Snapshot: &models.MaterializeDeploymentSnapshotInfo{Phase: "Complete"}}, true},
		{"snapshot disabled", &models.MaterializeDeploymentStatus{Phase: "Provisioning", Snapshot: &models.MaterializeDeploymentSnapshotInfo{Phase: "Disabled"}}, true},
		{"running phase without snapshot info", &models.MaterializeDeploymentStatus{Phase: "Running"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := materializeDeploymentReady(&models.MaterializeDeployment{Status: tc.status})
			if got != tc.want {
				t.Fatalf("materializeDeploymentReady = %v, want %v", got, tc.want)
			}
		})
	}
}

// preflightFailedBody is what the API returns when a preflight check rejects
// the config: the failure shows up only in the condition, while the phases sit
// at "Unknown"/"Pending" forever, same as a deployment that is still starting.
func preflightFailedBody(reason, message string) string {
	return preflightFailedBodyAtGeneration(reason, message, 1)
}

func preflightFailedBodyAtGeneration(reason, message string, generation int64) string {
	return fmt.Sprintf(`{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Unknown", "snapshot": {"phase": "Pending"}, "conditions": [{"type": "PreflightFailed", "status": "True", "reason": %q, "message": %q, "observedGeneration": %d}]}}}`, reason, message, generation)
}

func TestWaitForMaterializeDeploymentReadyFailsFastOnTerminalPreflightFailure(t *testing.T) {
	const message = "invalid watched permission: unknown definition thumper/userrr"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, preflightFailedBody("InvalidWatchedPermission", message))
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	// Generous deadline: the point is that we return long before it expires.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error for a terminal preflight failure, got nil")
	}
	// The user needs to be told which permission is wrong.
	if !strings.Contains(err.Error(), "InvalidWatchedPermission") {
		t.Errorf("expected error to include the condition reason, got: %v", err)
	}
	if !strings.Contains(err.Error(), message) {
		t.Errorf("expected error to include the condition message, got: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("expected fail-fast return well under the 10s ctx deadline, took %v", elapsed)
	}
}

func TestWaitForMaterializeDeploymentReadyKeepsPollingOnTransientPreflightFailure(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// A preflight failure the service retries must not abort the wait.
		// Here it clears and the snapshot job starts.
		if atomic.AddInt32(&calls, 1) < 3 {
			_, _ = fmt.Fprint(w, preflightFailedBody("DatastoreUnreachable", "could not reach the datastore"))
			return
		}
		_, _ = fmt.Fprint(w, `{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Provisioning", "snapshot": {"phase": "Running"}, "conditions": [{"type": "PreflightComplete", "status": "True", "reason": "JobComplete", "message": "preflight job completed", "observedGeneration": 1, "lastTransitionTime": "2026-09-03T11:05:00Z"}]}}}`)
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deployment, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	if err != nil {
		t.Fatalf("a retryable preflight failure must not abort the wait: %v", err)
	}
	if deployment == nil || deployment.ID != "mc-test" {
		t.Fatalf("expected the observed deployment to be returned, got %+v", deployment)
	}
}

func TestTerminalMaterializeFailure(t *testing.T) {
	cond := func(condType, status, reason string) models.MaterializeDeploymentStatusCondition {
		return models.MaterializeDeploymentStatusCondition{
			Type: condType, Status: status, Reason: reason, Message: "m",
		}
	}
	cases := []struct {
		name       string
		conditions []models.MaterializeDeploymentStatusCondition
		wantErr    bool
	}{
		{"no conditions", nil, false},
		{"healthy conditions", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightComplete", "True", "JobComplete"),
			cond("CacheReady", "True", "CacheCreated"),
		}, false},
		{"terminal: invalid watched permission", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightFailed", "True", "InvalidWatchedPermission"),
		}, true},
		{"terminal: schema empty", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightFailed", "True", "SchemaEmpty"),
		}, true},
		{"terminal: no watched permissions", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightFailed", "True", "NoWatchedPermissions"),
		}, true},
		// The operator retries these on its own; aborting the apply would
		// give up on a failure that was going to clear.
		{"retryable: datastore unreachable", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightFailed", "True", "DatastoreUnreachable"),
		}, false},
		{"retryable: internal", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightFailed", "True", "Internal"),
		}, false},
		{"retryable: generic catch-all", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightFailed", "True", "PreflightFailed"),
		}, false},
		// A cleared condition may be set to False or dropped; both count.
		{"terminal reason but condition False", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightFailed", "False", "InvalidWatchedPermission"),
		}, false},
		// Snapshot failures are retried and already show up in snapshot.phase.
		{"snapshot failed is not terminal", []models.MaterializeDeploymentStatusCondition{
			cond("SnapshotFailed", "True", "JobFailed"),
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := terminalMaterializeFailure(&models.MaterializeDeployment{
				Status: &models.MaterializeDeploymentStatus{Conditions: tc.conditions},
			})
			if tc.wantErr != (err != nil) {
				t.Fatalf("terminalMaterializeFailure() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestWaitForMaterializeDeploymentReadyTimeoutSurfacesFailureDetail(t *testing.T) {
	const message = "preflight could not reach the datastore"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, preflightFailedBody("DatastoreUnreachable", message))
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// A failure that never clears still has to say what went wrong; the
	// phases alone cannot, since a starting deployment looks identical.
	if !strings.Contains(err.Error(), "PreflightFailed") {
		t.Errorf("expected timeout error to name the failing condition, got: %v", err)
	}
	if !strings.Contains(err.Error(), "DatastoreUnreachable") {
		t.Errorf("expected timeout error to include the condition reason, got: %v", err)
	}
	if !strings.Contains(err.Error(), message) {
		t.Errorf("expected timeout error to include the condition message, got: %v", err)
	}
}

func TestMaterializeFailureDetail(t *testing.T) {
	cond := func(condType, status, reason, message string) models.MaterializeDeploymentStatusCondition {
		return models.MaterializeDeploymentStatusCondition{
			Type: condType, Status: status, Reason: reason, Message: message,
		}
	}
	cases := []struct {
		name       string
		conditions []models.MaterializeDeploymentStatusCondition
		want       string
	}{
		{"no conditions", nil, ""},
		{"only healthy conditions", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightComplete", "True", "JobComplete", "done"),
			cond("CacheReady", "True", "CacheCreated", "2 replicas ready"),
		}, ""},
		{"preflight failure", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightComplete", "False", "", ""),
			cond("PreflightFailed", "True", "SchemaEmpty", "the schema is empty"),
		}, "PreflightFailed: SchemaEmpty: the schema is empty"},
		// Reported even though they are not terminal: the service retries
		// them, but if they never clear the timeout has to explain why.
		{"snapshot failure", []models.MaterializeDeploymentStatusCondition{
			cond("SnapshotFailed", "True", "JobFailed", "snapshot job failed"),
		}, "SnapshotFailed: JobFailed: snapshot job failed"},
		{"error-suffixed condition", []models.MaterializeDeploymentStatusCondition{
			cond("HydrationError", "True", "Stalled", "hydration stalled"),
		}, "HydrationError: Stalled: hydration stalled"},
		{"cleared condition ignored", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightFailed", "False", "SchemaEmpty", "the schema is empty"),
		}, ""},
		{"multiple failures joined", []models.MaterializeDeploymentStatusCondition{
			cond("PreflightFailed", "True", "Internal", "boom"),
			cond("CacheError", "True", "Crash", "cache crashed"),
		}, "PreflightFailed: Internal: boom; CacheError: Crash: cache crashed"},
		{"failure without a message", []models.MaterializeDeploymentStatusCondition{
			cond("SnapshotFailed", "True", "JobFailed", ""),
		}, "SnapshotFailed: JobFailed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := materializeFailureDetail(&models.MaterializeDeployment{
				Status: &models.MaterializeDeploymentStatus{Conditions: tc.conditions},
			})
			if got != tc.want {
				t.Fatalf("materializeFailureDetail() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWaitForMaterializeDeploymentReadyRejectedConfigIsIdentifiable(t *testing.T) {
	const message = "invalid watched permission: unknown definition thumper/userrr"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, preflightFailedBody("InvalidWatchedPermission", message))
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	// Callers need to tell a rejected configuration apart from a slow one so
	// they do not tell the user to raise a timeout that is not the problem.
	var rejected *materializeConfigRejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a materializeConfigRejectedError, got %T: %v", err, err)
	}
	// The generic "waiting for ... to start snapshotting" framing is wrong
	// here: it was never going to start, whatever the wait budget.
	if strings.Contains(err.Error(), "to start snapshotting") {
		t.Errorf("expected no waiting-framed prefix, got: %v", err)
	}
	if !strings.Contains(err.Error(), message) {
		t.Errorf("expected the condition message, got: %v", err)
	}
}

func TestWaitForMaterializeDeploymentReadyTimeoutIsNotRejectedConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, preflightFailedBody("DatastoreUnreachable", "cannot reach datastore"))
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	// A retryable failure that ran out of time really may just need longer,
	// so it must not be reported as a configuration the user has to fix.
	var rejected *materializeConfigRejectedError
	if errors.As(err, &rejected) {
		t.Fatalf("timeout must not be reported as a rejected configuration: %v", err)
	}
}

func TestWaitForMaterializeDeploymentReadyRejectsConfigDespiteStaleReadyState(t *testing.T) {
	// What an update to a bad config actually looks like: the previous
	// config's snapshot job is still running (so the deployment still looks
	// ready) while the new config's preflight has already been rejected.
	// Checking readiness first would report success and write the broken
	// configuration to state.
	body := `{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Provisioning", "snapshot": {"phase": "Running"}, "conditions": [
		{"type": "SnapshotInProgress", "status": "True", "reason": "JobStarted", "message": "Snapshot job started"},
		{"type": "CacheUpdating", "status": "True", "reason": "FreshGroupStore", "message": "rolling update is in-progress"},
		{"type": "PreflightFailed", "status": "True", "reason": "InvalidWatchedPermission", "message": "invalid watched permission: unknown definition thumper/userrr", "lastTransitionTime": "` + time.Now().UTC().Format(time.RFC3339) + `"}
	]}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, body)
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 0)
	if err == nil {
		t.Fatal("expected a rejected configuration, got success")
	}
	var rejected *materializeConfigRejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a materializeConfigRejectedError, got %T: %v", err, err)
	}
}

// shortSettleWindow keeps tests that rely on the settle fallback quick.
func shortSettleWindow(t *testing.T) {
	t.Helper()
	previous := materializeSettleWindow
	materializeSettleWindow = 250 * time.Millisecond
	t.Cleanup(func() { materializeSettleWindow = previous })
}

func TestWaitForMaterializeDeploymentReadyIgnoresLeftoverFailureAfterFix(t *testing.T) {
	shortSettleWindow(t)
	// The recovery an operator makes after a rejected config: the deployment
	// still reports the old configuration's failure until it catches up.
	// Treating that as terminal would leave them unable to apply the very fix
	// that resolves it.
	stale := preflightFailedBodyAtGeneration("InvalidWatchedPermission",
		"invalid watched permission: unknown definition thumper/userrr", 4)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if atomic.AddInt32(&calls, 1) < 3 {
			_, _ = fmt.Fprint(w, stale)
			return
		}
		_, _ = fmt.Fprint(w, `{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Provisioning", "snapshot": {"phase": "Running"}, "conditions": [{"type": "PreflightComplete", "status": "True", "reason": "JobComplete", "message": "ok", "observedGeneration": 5}]}}}`)
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deployment, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 4)
	if err != nil {
		t.Fatalf("a leftover failure must not fail the fix that clears it: %v", err)
	}
	if deployment == nil || deployment.ID != "mc-test" {
		t.Fatalf("expected the observed deployment, got %+v", deployment)
	}
}

func TestWaitForMaterializeDeploymentReadyIgnoresStaleReadyStateAfterUpdate(t *testing.T) {
	shortSettleWindow(t)
	// The bad update: right after the change the deployment still reports the
	// previous configuration, fully ready. Trusting that reports success and
	// writes the rejected configuration to state.
	const staleReady = `{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Running", "snapshot": {"phase": "Complete"}, "conditions": [{"type": "PreflightComplete", "status": "True", "reason": "JobComplete", "message": "ok", "observedGeneration": 7}]}}}`
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if atomic.AddInt32(&calls, 1) < 3 {
			_, _ = fmt.Fprint(w, staleReady)
			return
		}
		_, _ = fmt.Fprint(w, preflightFailedBodyAtGeneration("InvalidWatchedPermission",
			"invalid watched permission: unknown definition thumper/userrr", 8))
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 7)
	if err == nil {
		t.Fatal("expected the rejected configuration to be reported, got success")
	}
	var rejected *materializeConfigRejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a materializeConfigRejectedError, got %T: %v", err, err)
	}
}

func TestMaterializeChangeTracker(t *testing.T) {
	at := func(generation int64, conditionType string) *models.MaterializeDeployment {
		return &models.MaterializeDeployment{Status: &models.MaterializeDeploymentStatus{
			Conditions: []models.MaterializeDeploymentStatusCondition{
				{Type: conditionType, Status: "True", ObservedGeneration: generation},
			},
		}}
	}

	t.Run("create trusts the status straight away", func(t *testing.T) {
		if !newMaterializeChangeTracker(0).TookEffect(at(1, "PreflightComplete")) {
			t.Fatal("expected a create to need no settling")
		}
	})

	t.Run("waits for a newer configuration version", func(t *testing.T) {
		if newMaterializeChangeTracker(3).TookEffect(at(3, "PreflightComplete")) {
			t.Fatal("expected the old configuration version to be ignored")
		}
	})

	t.Run("waits for the check to finish", func(t *testing.T) {
		tracker := newMaterializeChangeTracker(3)
		// The old verdict, already restamped onto the new version.
		if tracker.TookEffect(at(4, "PreflightComplete")) {
			t.Fatal("expected a restamped verdict not to count")
		}
		if tracker.TookEffect(at(4, "PreflightInProgress")) {
			t.Fatal("expected a running check not to count")
		}
		if !tracker.TookEffect(at(4, "PreflightFailed")) {
			t.Fatal("expected the finished check to count")
		}
	})

	t.Run("stays true once the change has taken effect", func(t *testing.T) {
		tracker := newMaterializeChangeTracker(3)
		tracker.TookEffect(at(4, "PreflightInProgress"))
		if !tracker.TookEffect(at(4, "PreflightComplete")) {
			t.Fatal("expected the check to have finished")
		}
		if !tracker.TookEffect(at(3, "PreflightComplete")) {
			t.Fatal("expected the result to stick")
		}
	})

	t.Run("settles when no check ever runs", func(t *testing.T) {
		shortSettleWindow(t)
		tracker := newMaterializeChangeTracker(3)
		if tracker.TookEffect(at(4, "PreflightComplete")) {
			t.Fatal("expected the settle window not to have passed yet")
		}
		time.Sleep(2 * materializeSettleWindow)
		if !tracker.TookEffect(at(4, "PreflightComplete")) {
			t.Fatal("expected the settle window to release the wait")
		}
	})

	// The window exists to give the configuration check time to start, so it
	// can only start counting once there is a newer version to check. A
	// service slower than the window would otherwise have its restamped
	// previous verdict — ready, and passing a check that never ran —
	// accepted the moment the version finally bumps.
	t.Run("a slow version bump does not consume the settle window", func(t *testing.T) {
		shortSettleWindow(t)
		tracker := newMaterializeChangeTracker(3)
		if tracker.TookEffect(at(3, "PreflightComplete")) {
			t.Fatal("expected the old configuration version to be ignored")
		}
		time.Sleep(2 * materializeSettleWindow)
		// The bump finally lands, still carrying the previous verdict.
		if tracker.TookEffect(at(4, "PreflightComplete")) {
			t.Fatal("expected the window to start from the newer version, not from the change")
		}
		if tracker.TookEffect(at(4, "PreflightInProgress")) {
			t.Fatal("expected a running check not to count")
		}
		if !tracker.TookEffect(at(4, "PreflightFailed")) {
			t.Fatal("expected the finished check to count")
		}
	})
}

func TestMaterializeConditionActive(t *testing.T) {
	deployment := &models.MaterializeDeployment{Status: &models.MaterializeDeploymentStatus{
		Conditions: []models.MaterializeDeploymentStatusCondition{
			{Type: "PreflightInProgress", Status: "True"},
			{Type: "CacheReady", Status: "False"},
		},
	}}
	if !materializeConditionActive(deployment, "PreflightInProgress") {
		t.Error("expected a condition that holds to be reported as active")
	}
	if materializeConditionActive(deployment, "CacheReady") {
		t.Error("expected a condition that does not hold to be reported as inactive")
	}
	if materializeConditionActive(deployment, "Missing") {
		t.Error("expected an absent condition to be reported as inactive")
	}
	if materializeConditionActive(&models.MaterializeDeployment{}, "PreflightInProgress") {
		t.Error("expected no status to be reported as inactive")
	}
}

func TestWaitForMaterializeDeploymentReadyWaitsForConfigCheckVerdict(t *testing.T) {
	shortSettleWindow(t)
	// The real sequence after an update: the previous configuration still
	// reports fully ready while the new one is being checked, and only then
	// is it rejected. Accepting the ready state reports success and writes
	// the rejected configuration to state.
	const checking = `{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Running", "snapshot": {"phase": "Complete"}, "conditions": [
		{"type": "SnapshotComplete", "status": "True", "reason": "JobComplete", "message": "done", "observedGeneration": 8},
		{"type": "PreflightInProgress", "status": "True", "reason": "JobStarted", "message": "checking", "observedGeneration": 8}
	]}}}`
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if atomic.AddInt32(&calls, 1) < 3 {
			_, _ = fmt.Fprint(w, checking)
			return
		}
		_, _ = fmt.Fprint(w, preflightFailedBodyAtGeneration("InvalidWatchedPermission",
			"invalid watched permission: unknown definition thumper/userrr", 8))
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 7)
	if err == nil {
		t.Fatal("expected the rejected configuration to be reported, got success")
	}
	var rejected *materializeConfigRejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a materializeConfigRejectedError, got %T: %v", err, err)
	}
}

func TestWaitForMaterializeDeploymentReadyRejectsRestampedStaleStatus(t *testing.T) {
	// The sequence the service actually produces after an update: it first
	// restamps the previous configuration's status onto the new version, so
	// for a moment a rejected configuration reports a passed check and a
	// ready deployment. Only then does the new check start and fail.
	ready := func(preflight string, generation int64) string {
		return fmt.Sprintf(`{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Running", "snapshot": {"phase": "Complete"}, "conditions": [
			{"type": "SnapshotComplete", "status": "True", "reason": "JobComplete", "message": "done", "observedGeneration": %[2]d},
			{"type": %[1]q, "status": "True", "reason": "r", "message": "m", "observedGeneration": %[2]d}
		]}}}`, preflight, generation)
	}
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch n := atomic.AddInt32(&calls, 1); {
		case n <= 2:
			// Stale verdict already carrying the new generation.
			_, _ = fmt.Fprint(w, ready("PreflightComplete", 11))
		case n <= 4:
			_, _ = fmt.Fprint(w, ready("PreflightInProgress", 11))
		default:
			_, _ = fmt.Fprint(w, preflightFailedBodyAtGeneration("InvalidWatchedPermission",
				"invalid watched permission: unknown definition thumper/userrr", 11))
		}
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 10)
	if err == nil {
		t.Fatal("expected the rejected configuration to be reported, got success")
	}
	var rejected *materializeConfigRejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a materializeConfigRejectedError, got %T: %v", err, err)
	}
}

func TestWaitForMaterializeDeploymentReadySettlesWithoutAConfigCheck(t *testing.T) {
	// A change that needs no configuration check (replica count, say) never
	// reports one running, so the settle window is what releases the wait.
	shortSettleWindow(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"deployment": {"id": "mc-test", "name": "t", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": [], "status": {"phase": "Running", "snapshot": {"phase": "Complete"}, "conditions": [{"type": "PreflightComplete", "status": "True", "reason": "JobComplete", "message": "ok", "observedGeneration": 12}]}}}`)
	}))
	defer server.Close()

	c := client.NewCloudClient(&client.CloudClientConfig{Host: server.URL, Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test", 11); err != nil {
		t.Fatalf("expected the settle window to release the wait: %v", err)
	}
}
