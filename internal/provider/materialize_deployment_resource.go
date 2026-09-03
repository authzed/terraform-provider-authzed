package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
)

var (
	_ resource.Resource                = &materializeDeploymentResource{}
	_ resource.ResourceWithImportState = &materializeDeploymentResource{}
)

func NewMaterializeDeploymentResource() resource.Resource {
	return &materializeDeploymentResource{}
}

type materializeDeploymentResource struct {
	client *client.CloudClient
}

type materializeDeploymentResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	PermissionSystemID  types.String   `tfsdk:"permission_system_id"`
	DeploymentID        types.String   `tfsdk:"deployment_id"`
	ServerTemplateID    types.String   `tfsdk:"server_template_id"`
	SnapshotTemplateID  types.String   `tfsdk:"snapshot_template_id"`
	HydrationTemplateID types.String   `tfsdk:"hydration_template_id"`
	WatchedPermissions  types.List     `tfsdk:"watched_permissions"`
	Replicas            types.Int64    `tfsdk:"replicas"`
	AcceleratedQueries  types.Bool     `tfsdk:"accelerated_queries"`
	URL                 types.String   `tfsdk:"url"`
	CreatedAt           types.String   `tfsdk:"created_at"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

// The *OrPreserve helpers implement the read-back rule shared by every
// asymmetric attribute: an API-provided value wins; an absent one keeps the
// model's current value — except values that are still unknown, which must
// resolve to null because unknowns cannot be persisted to state.

func stringOrPreserve(apiValue string, current types.String) types.String {
	if apiValue != "" {
		return types.StringValue(apiValue)
	}
	if current.IsUnknown() {
		return types.StringNull()
	}
	return current
}

func boolOrPreserve(apiValue *bool, current types.Bool) types.Bool {
	if apiValue != nil {
		return types.BoolValue(*apiValue)
	}
	if current.IsUnknown() {
		return types.BoolNull()
	}
	return current
}

func int64OrPreserve(apiValue *int64, current types.Int64) types.Int64 {
	if apiValue != nil {
		return types.Int64Value(*apiValue)
	}
	if current.IsUnknown() {
		return types.Int64Null()
	}
	return current
}

// applyMaterializeDeploymentToModel maps API fields onto the resource model.
// GET composes template IDs, features, and compute best-effort (they can be
// transiently absent while the operator resolves the spec), so absent values
// follow the *OrPreserve rule above.
func applyMaterializeDeploymentToModel(ctx context.Context, deployment *models.MaterializeDeployment, data *materializeDeploymentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(deployment.ID)
	data.Name = types.StringValue(deployment.Name)
	data.PermissionSystemID = types.StringValue(deployment.PermissionsSystemID)
	data.DeploymentID = types.StringValue(deployment.DeploymentID)

	data.ServerTemplateID = stringOrPreserve(deployment.ServerTemplateID, data.ServerTemplateID)
	data.SnapshotTemplateID = stringOrPreserve(deployment.SnapshotTemplateID, data.SnapshotTemplateID)
	data.HydrationTemplateID = stringOrPreserve(deployment.HydrationTemplateID, data.HydrationTemplateID)
	data.URL = stringOrPreserve(deployment.URL, data.URL)
	data.CreatedAt = stringOrPreserve(deployment.CreatedAt, data.CreatedAt)

	watched, d := types.ListValueFrom(ctx, types.StringType, deployment.WatchedPermissions)
	diags.Append(d...)
	data.WatchedPermissions = watched

	var acceleratedQueries *bool
	if deployment.Status != nil && deployment.Status.Features != nil {
		acceleratedQueries = &deployment.Status.Features.AcceleratedQueries
	}
	data.AcceleratedQueries = boolOrPreserve(acceleratedQueries, data.AcceleratedQueries)

	var replicas *int64
	if deployment.Status != nil && deployment.Status.Compute != nil && deployment.Status.Compute.CacheServer != nil {
		replicas = deployment.Status.Compute.CacheServer.Replicas
	}
	data.Replicas = int64OrPreserve(replicas, data.Replicas)

	return diags
}

// optionalInt64/optionalBool convert an optional attribute to the request's
// pointer form: nil when the plan leaves the value to the server.

func optionalInt64(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueInt64()
	return &value
}

func optionalBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueBool()
	return &value
}

// applyPlannedOverrides restores plan-specified values for the
// optional+computed attributes after a post-apply read: the API accepted the
// requested configuration, and the just-read status is composed
// asynchronously and may still be stale. Read detects real drift later.
func applyPlannedOverrides(data *materializeDeploymentResourceModel, plannedReplicas types.Int64, plannedAcceleratedQueries types.Bool) {
	if !plannedReplicas.IsNull() && !plannedReplicas.IsUnknown() {
		data.Replicas = plannedReplicas
	}
	if !plannedAcceleratedQueries.IsNull() && !plannedAcceleratedQueries.IsUnknown() {
		data.AcceleratedQueries = plannedAcceleratedQueries
	}
}

func (r *materializeDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_materialize_deployment"
}

func (r *materializeDeploymentResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an AuthZed Materialize deployment for a permission system deployment. Requires the internal API version, which the provider sends automatically for this resource. The internal API comes with no compatibility guarantees and can change or break at any time without notice; promotion to the public API is planned to align with the Materialize GA release.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this Materialize deployment (mc-...)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the Materialize deployment (RFC-1123, max 50 chars). Changing it forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system this Materialize deployment watches (ps-...). Changing it forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system deployment providing updates (dp-...). Changing it forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"server_template_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the Materialize online hydration server template (e.g. mtsc-4-vcpu)",
			},
			"snapshot_template_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the Materialize snapshot template (e.g. mtssc-4-vcpu)",
			},
			"hydration_template_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the Materialize offline hydration template (e.g. mthc-4-vcpu)",
			},
			"watched_permissions": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Permissions to materialize, in the format resource_type#relation@subject_type[#subject_relation] (1-100 entries). Updates replace the entire list.",
			},
			"replicas": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Number of online hydration server replicas (0-16). Zero temporarily disables the deployment without deleting it. Omitted uses the server template default.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"accelerated_queries": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this deployment accelerates the permission system's CheckPermission, LookupResources, and LookupSubjects queries by serving as a hedged, lower-latency cache.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "Endpoint URL of the Materialize deployment",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the Materialize deployment was created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
			}),
		},
	}
}

func (r *materializeDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*CloudProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *CloudProviderData, got: %T", req.ProviderData),
		)
		return
	}

	r.client = providerData.Client
}

func (r *materializeDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data materializeDeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create waits until the deployment is provisioned and snapshotting —
	// not fully hydrated, which is data-dependent and can take hours. The
	// 30m here is only the default: users override it with the resource's
	// `timeouts { create = "..." }` block.
	createTimeout, diags := data.Timeouts.Create(ctx, 30*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	// Capture plan-specified values before any demotion; after the
	// post-apply read they take precedence over possibly-stale status.
	plannedReplicas := data.Replicas
	plannedAcceleratedQueries := data.AcceleratedQueries

	var watched []string
	resp.Diagnostics.Append(data.WatchedPermissions.ElementsAs(ctx, &watched, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &models.CreateMaterializeDeploymentRequest{
		PermissionsSystemID: data.PermissionSystemID.ValueString(),
		DeploymentID:        data.DeploymentID.ValueString(),
		ServerTemplateID:    data.ServerTemplateID.ValueString(),
		SnapshotTemplateID:  data.SnapshotTemplateID.ValueString(),
		HydrationTemplateID: data.HydrationTemplateID.ValueString(),
		Name:                data.Name.ValueString(),
		WatchedPermissions:  watched,
		Replicas:            optionalInt64(data.Replicas),
		AcceleratedQueries:  optionalBool(data.AcceleratedQueries),
	}

	created, err := r.client.CreateMaterializeDeployment(createCtx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create materialize deployment, got error: %s", err))
		return
	}

	// Persist the ID before waiting so a failed wait leaves a tainted
	// resource in state instead of an untracked deployment.
	data.ID = types.StringValue(created.ID)
	data.URL = types.StringNull()
	data.CreatedAt = types.StringNull()
	data.Replicas = int64OrPreserve(nil, data.Replicas)
	data.AcceleratedQueries = boolOrPreserve(nil, data.AcceleratedQueries)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The wait's final poll returns the deployment it observed, so no extra
	// read is needed afterwards.
	deployment, err := waitForMaterializeDeploymentReady(createCtx, r.client, created.ID, 0)
	if err != nil {
		var rejected *materializeConfigRejectedError
		if errors.As(err, &rejected) {
			resp.Diagnostics.AddError(
				"Invalid Materialize Deployment Configuration",
				fmt.Sprintf("Deployment %s was created but rejected its configuration: %s. Fix the configuration and apply again.", created.ID, err),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Materialize Deployment Not Ready",
			fmt.Sprintf("Deployment %s was created but did not start snapshotting: %s. Increase the create timeout if provisioning needs more time.", created.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(applyMaterializeDeploymentToModel(ctx, deployment, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	applyPlannedOverrides(&data, plannedReplicas, plannedAcceleratedQueries)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *materializeDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data materializeDeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deployment, err := r.client.GetMaterializeDeployment(ctx, data.ID.ValueString())
	if err != nil {
		if materializeDeploymentGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read materialize deployment, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(applyMaterializeDeploymentToModel(ctx, deployment, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *materializeDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data materializeDeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	var state materializeDeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The 30m here is only the default: users override it with the
	// resource's `timeouts { update = "..." }` block.
	updateTimeout, diags := data.Timeouts.Update(ctx, 30*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	// Capture plan-specified values before any demotion; after the
	// post-apply read they take precedence over possibly-stale status.
	plannedReplicas := data.Replicas
	plannedAcceleratedQueries := data.AcceleratedQueries

	var watched []string
	resp.Diagnostics.Append(data.WatchedPermissions.ElementsAs(ctx, &watched, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Send only the fields whose planned value differs from state; the API
	// treats omitted fields as "leave unchanged", and PATCHing an unchanged
	// template would needlessly re-resolve it.
	updateReq := &models.UpdateMaterializeDeploymentRequest{}
	hasAPIChanges := false
	if !data.ServerTemplateID.Equal(state.ServerTemplateID) {
		serverTemplateID := data.ServerTemplateID.ValueString()
		updateReq.ServerTemplateID = &serverTemplateID
		hasAPIChanges = true
	}
	if !data.SnapshotTemplateID.Equal(state.SnapshotTemplateID) {
		snapshotTemplateID := data.SnapshotTemplateID.ValueString()
		updateReq.SnapshotTemplateID = &snapshotTemplateID
		hasAPIChanges = true
	}
	if !data.HydrationTemplateID.Equal(state.HydrationTemplateID) {
		hydrationTemplateID := data.HydrationTemplateID.ValueString()
		updateReq.HydrationTemplateID = &hydrationTemplateID
		hasAPIChanges = true
	}
	if !data.WatchedPermissions.Equal(state.WatchedPermissions) {
		updateReq.WatchedPermissions = &watched
		hasAPIChanges = true
	}
	if !data.Replicas.Equal(state.Replicas) {
		updateReq.Replicas = optionalInt64(data.Replicas)
		hasAPIChanges = true
	}
	if !data.AcceleratedQueries.Equal(state.AcceleratedQueries) {
		updateReq.AcceleratedQueries = optionalBool(data.AcceleratedQueries)
		hasAPIChanges = true
	}

	if !hasAPIChanges {
		// Only non-API attributes changed (e.g. the timeouts block); the API
		// rejects an empty update, and there is nothing to wait for.
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	id := state.ID.ValueString()

	// Note the version being replaced so the wait below can tell the new
	// configuration's status from the old one's. Failing to read it only
	// costs that, so it must not fail the update.
	var currentGeneration int64
	if current, err := r.client.GetMaterializeDeployment(updateCtx, id); err == nil {
		currentGeneration = materializeObservedGeneration(current)
	} else {
		tflog.Warn(ctx, "could not read the deployment's version before updating it", map[string]any{
			"id":    id,
			"error": err.Error(),
		})
	}

	if err := r.client.UpdateMaterializeDeployment(updateCtx, id, updateReq); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update materialize deployment, got error: %s", err))
		return
	}

	// The wait's final poll returns the deployment it observed, so no extra
	// read is needed afterwards.
	deployment, err := waitForMaterializeDeploymentReady(updateCtx, r.client, id, currentGeneration)
	if err != nil {
		var rejected *materializeConfigRejectedError
		if errors.As(err, &rejected) {
			resp.Diagnostics.AddError(
				"Invalid Materialize Deployment Configuration",
				fmt.Sprintf("Deployment %s rejected its updated configuration: %s. Fix the configuration and apply again.", id, err),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Materialize Deployment Not Ready",
			fmt.Sprintf("Deployment %s was updated but did not return to a ready state: %s. Increase the update timeout if the rollout needs more time.", id, err),
		)
		return
	}

	resp.Diagnostics.Append(applyMaterializeDeploymentToModel(ctx, deployment, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	applyPlannedOverrides(&data, plannedReplicas, plannedAcceleratedQueries)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *materializeDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data materializeDeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The API answers 202 Accepted; the client polls until the deployment is
	// gone, bounded by the client-level DeleteTimeout.
	if err := r.client.DeleteMaterializeDeployment(data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting materialize deployment", fmt.Sprintf("Unable to delete materialize deployment: %v", err))
		return
	}
}

// ImportState imports by the mc-... deployment ID alone; unlike other
// resources, GET needs no permission system ID.
func (r *materializeDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
