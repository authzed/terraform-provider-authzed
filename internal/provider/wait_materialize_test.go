package provider

import (
	"context"
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

	deployment, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test")
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

	deployment, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test")
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

	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test")
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
	_, err := waitForMaterializeDeploymentReady(ctx, c, "mc-test")
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
