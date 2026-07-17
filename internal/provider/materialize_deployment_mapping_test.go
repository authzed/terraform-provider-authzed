package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-authzed/internal/models"
)

func fullTestDeployment() *models.MaterializeDeployment {
	replicas := int64(2)
	return &models.MaterializeDeployment{
		ID:                  "mc-abc123",
		Name:                "my-materialize",
		PermissionsSystemID: "ps-xyz",
		DeploymentID:        "dp-xyz",
		ServerTemplateID:    "mtsc-default",
		SnapshotTemplateID:  "mtssc-default",
		HydrationTemplateID: "mthc-default",
		WatchedPermissions:  []string{"document#view@user"},
		URL:                 "https://mc-abc123.example.com",
		CreatedAt:           "2026-07-13T00:00:00Z",
		Status: &models.MaterializeDeploymentStatus{
			Phase:    models.MaterializePhaseRunning,
			Features: &models.MaterializeDeploymentFeatures{AcceleratedQueries: true},
			Compute: &models.MaterializeDeploymentCompute{
				CacheServer: &models.MaterializeComponentCompute{Replicas: &replicas},
			},
		},
	}
}

func TestApplyMaterializeDeploymentToModelFull(t *testing.T) {
	var data materializeDeploymentResourceModel
	diags := applyMaterializeDeploymentToModel(context.Background(), fullTestDeployment(), &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.ID.ValueString() != "mc-abc123" || data.Name.ValueString() != "my-materialize" {
		t.Fatalf("identity not mapped: %+v", data)
	}
	if data.PermissionSystemID.ValueString() != "ps-xyz" || data.DeploymentID.ValueString() != "dp-xyz" {
		t.Fatalf("references not mapped: %+v", data)
	}
	if data.ServerTemplateID.ValueString() != "mtsc-default" ||
		data.SnapshotTemplateID.ValueString() != "mtssc-default" ||
		data.HydrationTemplateID.ValueString() != "mthc-default" {
		t.Fatalf("templates not mapped: %+v", data)
	}
	if data.URL.ValueString() != "https://mc-abc123.example.com" || data.CreatedAt.ValueString() != "2026-07-13T00:00:00Z" {
		t.Fatalf("computed fields not mapped: %+v", data)
	}
	if !data.AcceleratedQueries.ValueBool() {
		t.Fatalf("expected accelerated_queries true, got %v", data.AcceleratedQueries)
	}
	if data.Replicas.ValueInt64() != 2 {
		t.Fatalf("expected replicas 2, got %v", data.Replicas)
	}
	var watched []string
	if d := data.WatchedPermissions.ElementsAs(context.Background(), &watched, false); d.HasError() {
		t.Fatalf("watched_permissions not a string list: %v", d)
	}
	if len(watched) != 1 || watched[0] != "document#view@user" {
		t.Fatalf("unexpected watched_permissions: %v", watched)
	}
}

// When status sub-objects are missing (transiently unresolved), known prior
// values must be preserved rather than overwritten with null.
func TestApplyMaterializeDeploymentToModelPreservesKnownValues(t *testing.T) {
	deployment := fullTestDeployment()
	deployment.Status = nil
	deployment.ServerTemplateID = ""
	deployment.SnapshotTemplateID = ""
	deployment.HydrationTemplateID = ""
	deployment.URL = ""
	deployment.CreatedAt = ""

	data := materializeDeploymentResourceModel{
		ServerTemplateID:    types.StringValue("mtsc-prior"),
		SnapshotTemplateID:  types.StringValue("mtssc-prior"),
		HydrationTemplateID: types.StringValue("mthc-prior"),
		Replicas:            types.Int64Value(4),
		AcceleratedQueries:  types.BoolValue(true),
		URL:                 types.StringValue("https://prior.example.com"),
		CreatedAt:           types.StringValue("2026-01-01T00:00:00Z"),
	}
	diags := applyMaterializeDeploymentToModel(context.Background(), deployment, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.ServerTemplateID.ValueString() != "mtsc-prior" {
		t.Fatalf("expected prior server template preserved, got %v", data.ServerTemplateID)
	}
	if data.SnapshotTemplateID.ValueString() != "mtssc-prior" {
		t.Fatalf("expected prior snapshot template preserved, got %v", data.SnapshotTemplateID)
	}
	if data.HydrationTemplateID.ValueString() != "mthc-prior" {
		t.Fatalf("expected prior hydration template preserved, got %v", data.HydrationTemplateID)
	}
	if data.Replicas.ValueInt64() != 4 {
		t.Fatalf("expected prior replicas preserved, got %v", data.Replicas)
	}
	if !data.AcceleratedQueries.ValueBool() {
		t.Fatalf("expected prior accelerated_queries preserved, got %v", data.AcceleratedQueries)
	}
	if data.URL.ValueString() != "https://prior.example.com" {
		t.Fatalf("expected prior url preserved, got %v", data.URL)
	}
	if data.CreatedAt.ValueString() != "2026-01-01T00:00:00Z" {
		t.Fatalf("expected prior created_at preserved, got %v", data.CreatedAt)
	}
}

// Unknown values cannot be persisted to state, so when the API provides
// nothing they must resolve to null, not stay unknown.
func TestApplyMaterializeDeploymentToModelResolvesUnknowns(t *testing.T) {
	deployment := fullTestDeployment()
	deployment.Status = nil
	deployment.ServerTemplateID = ""
	deployment.SnapshotTemplateID = ""
	deployment.HydrationTemplateID = ""
	deployment.URL = ""
	deployment.CreatedAt = ""

	data := materializeDeploymentResourceModel{
		ServerTemplateID:    types.StringUnknown(),
		SnapshotTemplateID:  types.StringUnknown(),
		HydrationTemplateID: types.StringUnknown(),
		Replicas:            types.Int64Unknown(),
		AcceleratedQueries:  types.BoolUnknown(),
		URL:                 types.StringUnknown(),
		CreatedAt:           types.StringUnknown(),
	}
	diags := applyMaterializeDeploymentToModel(context.Background(), deployment, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !data.ServerTemplateID.IsNull() {
		t.Fatalf("expected server template resolved to null, got %v", data.ServerTemplateID)
	}
	if !data.SnapshotTemplateID.IsNull() {
		t.Fatalf("expected snapshot template resolved to null, got %v", data.SnapshotTemplateID)
	}
	if !data.HydrationTemplateID.IsNull() {
		t.Fatalf("expected hydration template resolved to null, got %v", data.HydrationTemplateID)
	}
	if !data.Replicas.IsNull() {
		t.Fatalf("expected replicas resolved to null, got %v", data.Replicas)
	}
	if !data.AcceleratedQueries.IsNull() {
		t.Fatalf("expected accelerated_queries resolved to null, got %v", data.AcceleratedQueries)
	}
	if !data.URL.IsNull() {
		t.Fatalf("expected url resolved to null, got %v", data.URL)
	}
	if !data.CreatedAt.IsNull() {
		t.Fatalf("expected created_at resolved to null, got %v", data.CreatedAt)
	}
}
