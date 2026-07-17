package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"terraform-provider-authzed/internal/models"
)

// newMaterializeTestClient returns a CloudClient pointed at an httptest server.
func newMaterializeTestClient(t *testing.T, handler http.Handler) *CloudClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewCloudClient(&CloudClientConfig{
		Host:          server.URL,
		Token:         "test-token",
		DeleteTimeout: 10 * time.Second,
	})
}

func TestWithAPIVersionOverridesHeader(t *testing.T) {
	var gotVersion atomic.Value
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotVersion.Store(r.Header.Get("X-API-Version"))
		w.WriteHeader(http.StatusOK)
	}))

	req, err := c.NewRequest(http.MethodGet, "/materialize/mc-123", nil, WithAPIVersion(MaterializeAPIVersion))
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	_ = resp.Response.Body.Close()

	if got := gotVersion.Load(); got != "internal" {
		t.Fatalf("expected X-API-Version %q, got %q", "internal", got)
	}
}

func TestDeleteResourcePassesOptionsToAllRequests(t *testing.T) {
	var deleteCalls, getCalls int32
	var badVersions int32
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Version") != "internal" {
			atomic.AddInt32(&badVersions, 1)
		}
		switch r.Method {
		case http.MethodDelete:
			atomic.AddInt32(&deleteCalls, 1)
			w.WriteHeader(http.StatusAccepted) // async delete
		case http.MethodGet:
			// First poll: still present; afterwards: gone.
			if atomic.AddInt32(&getCalls, 1) == 1 {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	if err := c.DeleteResource("/materialize/mc-123", WithAPIVersion(MaterializeAPIVersion)); err != nil {
		t.Fatalf("DeleteResource failed: %v", err)
	}
	if deleteCalls != 1 {
		t.Fatalf("expected 1 DELETE, got %d", deleteCalls)
	}
	if getCalls < 2 {
		t.Fatalf("expected at least 2 polling GETs, got %d", getCalls)
	}
	if badVersions != 0 {
		t.Fatalf("%d requests were sent without X-API-Version: internal", badVersions)
	}
}

func TestCreateMaterializeDeployment(t *testing.T) {
	var gotBody models.CreateMaterializeDeploymentRequest
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/materialize" {
			t.Errorf("expected POST /materialize, got %s %s", r.Method, r.URL.Path)
		}
		if v := r.Header.Get("X-API-Version"); v != "internal" {
			t.Errorf("expected X-API-Version internal, got %q", v)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"id": "mc-abc123", "deploymentID": "dp-xyz", "name": "my-materialize"}`)
	}))

	replicas := int64(2)
	accelerated := true
	created, err := c.CreateMaterializeDeployment(context.Background(), &models.CreateMaterializeDeploymentRequest{
		PermissionsSystemID: "ps-xyz",
		DeploymentID:        "dp-xyz",
		ServerTemplateID:    "mtsc-default",
		SnapshotTemplateID:  "mtssc-default",
		HydrationTemplateID: "mthc-default",
		Name:                "my-materialize",
		WatchedPermissions:  []string{"document#view@user"},
		Replicas:            &replicas,
		AcceleratedQueries:  &accelerated,
	})
	if err != nil {
		t.Fatalf("CreateMaterializeDeployment failed: %v", err)
	}
	if created.ID != "mc-abc123" {
		t.Fatalf("expected id mc-abc123, got %q", created.ID)
	}
	if gotBody.Name != "my-materialize" || gotBody.PermissionsSystemID != "ps-xyz" ||
		gotBody.DeploymentID != "dp-xyz" || gotBody.ServerTemplateID != "mtsc-default" ||
		gotBody.SnapshotTemplateID != "mtssc-default" || gotBody.HydrationTemplateID != "mthc-default" {
		t.Fatalf("request body missing required fields: %+v", gotBody)
	}
	if gotBody.Replicas == nil || *gotBody.Replicas != 2 {
		t.Fatalf("expected replicas pointer 2, got %v", gotBody.Replicas)
	}
	if gotBody.AcceleratedQueries == nil || !*gotBody.AcceleratedQueries {
		t.Fatalf("expected acceleratedQueries pointer true, got %v", gotBody.AcceleratedQueries)
	}
}

func TestCreateMaterializeDeploymentOmitsNilOptionals(t *testing.T) {
	var rawBody map[string]any
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&rawBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"id": "mc-abc123", "deploymentID": "dp-xyz", "name": "n"}`)
	}))

	_, err := c.CreateMaterializeDeployment(context.Background(), &models.CreateMaterializeDeploymentRequest{
		PermissionsSystemID: "ps-xyz",
		DeploymentID:        "dp-xyz",
		ServerTemplateID:    "mtsc-default",
		SnapshotTemplateID:  "mtssc-default",
		HydrationTemplateID: "mthc-default",
		Name:                "n",
		WatchedPermissions:  []string{"document#view@user"},
	})
	if err != nil {
		t.Fatalf("CreateMaterializeDeployment failed: %v", err)
	}
	for _, key := range []string{"replicas", "acceleratedQueries", "enableEventStreams", "downloadPermissionSetsFormats"} {
		if _, present := rawBody[key]; present {
			t.Errorf("expected %q to be omitted from request body, got %v", key, rawBody[key])
		}
	}
}

func TestGetMaterializeDeployment(t *testing.T) {
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/materialize/mc-abc123" {
			t.Errorf("expected GET /materialize/mc-abc123, got %s %s", r.Method, r.URL.Path)
		}
		if v := r.Header.Get("X-API-Version"); v != "internal" {
			t.Errorf("expected X-API-Version internal, got %q", v)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"deployment": {
			"id": "mc-abc123",
			"name": "my-materialize",
			"permissionsSystemID": "ps-xyz",
			"deploymentID": "dp-xyz",
			"serverTemplateID": "mtsc-default",
			"snapshotTemplateID": "mtssc-default",
			"hydrationTemplateID": "mthc-default",
			"watchedPermissions": ["document#view@user"],
			"url": "https://mc-abc123.example.com",
			"createdAt": "2026-07-13T00:00:00Z",
			"status": {
				"phase": "Running",
				"features": {"acceleratedQueries": true, "eventStreams": false},
				"compute": {"cacheServer": {"cpu": "4", "memory": "8Gi", "replicas": 2}}
			}
		}}`)
	}))

	deployment, err := c.GetMaterializeDeployment(context.Background(), "mc-abc123")
	if err != nil {
		t.Fatalf("GetMaterializeDeployment failed: %v", err)
	}
	if deployment.ID != "mc-abc123" || deployment.Name != "my-materialize" {
		t.Fatalf("unexpected deployment identity: %+v", deployment)
	}
	if deployment.ServerTemplateID != "mtsc-default" {
		t.Fatalf("expected serverTemplateID mtsc-default, got %q", deployment.ServerTemplateID)
	}
	if len(deployment.WatchedPermissions) != 1 || deployment.WatchedPermissions[0] != "document#view@user" {
		t.Fatalf("unexpected watchedPermissions: %v", deployment.WatchedPermissions)
	}
	if deployment.Status == nil || deployment.Status.Phase != models.MaterializePhaseRunning {
		t.Fatalf("expected Running status, got %+v", deployment.Status)
	}
	if deployment.Status.Features == nil || !deployment.Status.Features.AcceleratedQueries {
		t.Fatalf("expected acceleratedQueries true, got %+v", deployment.Status.Features)
	}
	if deployment.Status.Compute == nil || deployment.Status.Compute.CacheServer == nil ||
		deployment.Status.Compute.CacheServer.Replicas == nil || *deployment.Status.Compute.CacheServer.Replicas != 2 {
		t.Fatalf("expected cacheServer replicas 2, got %+v", deployment.Status.Compute)
	}
}

// GET is idempotent, so transient 5xx responses retry.
func TestGetMaterializeDeploymentRetriesOn5xx(t *testing.T) {
	var calls int32
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"deployment": {"id": "mc-abc123", "name": "n", "permissionsSystemID": "ps-x", "deploymentID": "dp-x", "watchedPermissions": []}}`)
	}))

	deployment, err := c.GetMaterializeDeployment(context.Background(), "mc-abc123")
	if err != nil {
		t.Fatalf("expected retry to recover from 503, got: %v", err)
	}
	if deployment.ID != "mc-abc123" {
		t.Fatalf("unexpected deployment: %+v", deployment)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls (503 then 200), got %d", calls)
	}
}

// POST must not retry an ambiguous 5xx: the create may have been processed
// server-side, and a retry could create a duplicate deployment.
func TestCreateMaterializeDeploymentDoesNotRetryOn5xx(t *testing.T) {
	var calls int32
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))

	_, err := c.CreateMaterializeDeployment(context.Background(), &models.CreateMaterializeDeploymentRequest{
		PermissionsSystemID: "ps-x",
		DeploymentID:        "dp-x",
		ServerTemplateID:    "mtsc-default",
		SnapshotTemplateID:  "mtssc-default",
		HydrationTemplateID: "mthc-default",
		Name:                "n",
		WatchedPermissions:  []string{"document#view@user"},
	})
	if err == nil {
		t.Fatal("expected error for 502 on POST, got nil")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 POST attempt (no retry on ambiguous 5xx), got %d", calls)
	}
}

// POST retries on 429: the request provably was not processed.
func TestCreateMaterializeDeploymentRetriesOn429(t *testing.T) {
	var calls int32
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"id": "mc-abc123", "deploymentID": "dp-x", "name": "n"}`)
	}))

	created, err := c.CreateMaterializeDeployment(context.Background(), &models.CreateMaterializeDeploymentRequest{
		PermissionsSystemID: "ps-x",
		DeploymentID:        "dp-x",
		ServerTemplateID:    "mtsc-default",
		SnapshotTemplateID:  "mtssc-default",
		HydrationTemplateID: "mthc-default",
		Name:                "n",
		WatchedPermissions:  []string{"document#view@user"},
	})
	if err != nil {
		t.Fatalf("expected retry to recover from 429, got: %v", err)
	}
	if created.ID != "mc-abc123" {
		t.Fatalf("unexpected create response: %+v", created)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls (429 then 201), got %d", calls)
	}
}

func TestGetMaterializeDeploymentNotFound(t *testing.T) {
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	_, err := c.GetMaterializeDeployment(context.Background(), "mc-missing")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("expected error to contain 'status 404', got: %v", err)
	}
}

func TestUpdateMaterializeDeployment(t *testing.T) {
	var gotBody map[string]any
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/materialize/mc-abc123" {
			t.Errorf("expected PATCH /materialize/mc-abc123, got %s %s", r.Method, r.URL.Path)
		}
		if v := r.Header.Get("X-API-Version"); v != "internal" {
			t.Errorf("expected X-API-Version internal, got %q", v)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK) // empty body per spec
	}))

	replicas := int64(3)
	serverTemplateID := "mtsc-large"
	err := c.UpdateMaterializeDeployment(context.Background(), "mc-abc123", &models.UpdateMaterializeDeploymentRequest{
		ServerTemplateID: &serverTemplateID,
		Replicas:         &replicas,
	})
	if err != nil {
		t.Fatalf("UpdateMaterializeDeployment failed: %v", err)
	}
	if gotBody["serverTemplateID"] != "mtsc-large" {
		t.Fatalf("expected serverTemplateID mtsc-large, got %v", gotBody["serverTemplateID"])
	}
	if gotBody["replicas"] != float64(3) {
		t.Fatalf("expected replicas 3, got %v", gotBody["replicas"])
	}
	for _, key := range []string{"snapshotTemplateID", "hydrationTemplateID", "watchedPermissions", "acceleratedQueries"} {
		if _, present := gotBody[key]; present {
			t.Errorf("expected omitted field %q to be absent, got %v", key, gotBody[key])
		}
	}
}

func TestUpdateMaterializeDeploymentError(t *testing.T) {
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	serverTemplateID := "mtsc-bogus"
	err := c.UpdateMaterializeDeployment(context.Background(), "mc-abc123", &models.UpdateMaterializeDeploymentRequest{
		ServerTemplateID: &serverTemplateID,
	})
	if err == nil {
		t.Fatal("expected error for 400, got nil")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected error to contain 'status 400', got: %v", err)
	}
}

func TestDeleteMaterializeDeploymentAsync(t *testing.T) {
	var getCalls int32
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("X-API-Version"); v != "internal" {
			t.Errorf("expected X-API-Version internal on %s, got %q", r.Method, v)
		}
		switch r.Method {
		case http.MethodDelete:
			if r.URL.Path != "/materialize/mc-abc123" {
				t.Errorf("expected DELETE /materialize/mc-abc123, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			if atomic.AddInt32(&getCalls, 1) == 1 {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	if err := c.DeleteMaterializeDeployment("mc-abc123"); err != nil {
		t.Fatalf("DeleteMaterializeDeployment failed: %v", err)
	}
	if getCalls < 2 {
		t.Fatalf("expected polling until 404, got %d GETs", getCalls)
	}
}

// Deleting a materialize deployment removes its authorization relationships,
// so polls for the deleted deployment fail the API's permission check with
// 403 rather than 404. After an accepted DELETE that must count as gone.
func TestDeleteMaterializeDeploymentAsync403MeansGone(t *testing.T) {
	var getCalls int32
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			// Still deprovisioning on the first poll, then the authz
			// relationships are gone and the API answers 403 forever.
			if atomic.AddInt32(&getCalls, 1) == 1 {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusForbidden)
		}
	}))

	if err := c.DeleteMaterializeDeployment("mc-abc123"); err != nil {
		t.Fatalf("DeleteMaterializeDeployment failed on 403 poll: %v", err)
	}
	if getCalls < 2 {
		t.Fatalf("expected polling until 403, got %d GETs", getCalls)
	}
}

// A 403 on the DELETE request itself is a genuine permission error, not a
// deletion signal.
func TestDeleteMaterializeDeploymentForbidden(t *testing.T) {
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))

	err := c.DeleteMaterializeDeployment("mc-abc123")
	if err == nil {
		t.Fatal("expected error for 403 on DELETE, got nil")
	}
	if !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("expected error to contain 'status 403', got: %v", err)
	}
}

// A DELETE answered with 404 is an idempotent success (already gone).
func TestDeleteMaterializeDeploymentIdempotent(t *testing.T) {
	c := newMaterializeTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	if err := c.DeleteMaterializeDeployment("mc-abc123"); err != nil {
		t.Fatalf("expected idempotent success on 404 DELETE, got: %v", err)
	}
}
