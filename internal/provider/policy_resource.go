package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
	"terraform-provider-authzed/internal/provider/deletelanes"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &policyResource{}
	_ resource.ResourceWithImportState = &policyResource{}
)

func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

type policyResource struct {
	client      *client.CloudClient
	deleteLanes *deletelanes.DeleteLanes
}

type policyResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	Description         types.String   `tfsdk:"description"`
	PermissionsSystemID types.String   `tfsdk:"permission_system_id"`
	PrincipalID         types.String   `tfsdk:"principal_id"`
	RoleIDs             types.List     `tfsdk:"role_ids"`
	CreatedAt           types.String   `tfsdk:"created_at"`
	Creator             types.String   `tfsdk:"creator"`
	UpdatedAt           types.String   `tfsdk:"updated_at"`
	Updater             types.String   `tfsdk:"updater"`
	ETag                types.String   `tfsdk:"etag"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

func (r *policyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *policyResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a policy",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the policy",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the policy",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system this policy belongs to",
			},
			"principal_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the principal this policy is associated with",
			},
			"role_ids": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "IDs of the roles this policy is associated with",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the policy was created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"creator": schema.StringAttribute{
				Computed:    true,
				Description: "User who created the policy",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the policy was last updated (managed by API)",
			},
			"updater": schema.StringAttribute{
				Computed:    true,
				Description: "User who last updated the policy (managed by API)",
			},
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "Version identifier used to prevent conflicts from concurrent updates, ensuring safe resource modifications",
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *policyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.deleteLanes = providerData.DeleteLanes
}

func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var data policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get context with create timeout (default 5 minutes for policies)
	createTimeout, diags := data.Timeouts.Create(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	// Extract role IDs from types.List
	var roleIDs []string
	resp.Diagnostics.Append(data.RoleIDs.ElementsAs(ctx, &roleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Existence gates: ensure PS and referenced role(s) exist before creating policy
	psID := data.PermissionsSystemID.ValueString()

	// Wait for Permission System (uses createCtx timeout)
	if err := waitForPermissionSystemExists(createCtx, r.client, psID); err != nil {
		resp.Diagnostics.AddError("Permission System Not Ready",
			fmt.Sprintf("Permission System %s is not yet globally visible: %v", psID, err))
		return
	}

	// Wait for Service Account (principal) (uses createCtx timeout)
	if principalID := data.PrincipalID.ValueString(); principalID != "" {
		if err := waitForServiceAccountExists(createCtx, r.client, psID, principalID); err != nil {
			resp.Diagnostics.AddError("Service Account Not Ready",
				fmt.Sprintf("Service Account %s is not yet globally visible: %v", principalID, err))
			return
		}
	}

	// Wait for all referenced roles (uses createCtx timeout)
	for _, roleID := range roleIDs {
		if err := waitForRoleExists(createCtx, r.client, psID, roleID); err != nil {
			resp.Diagnostics.AddError("Role Not Ready",
				fmt.Sprintf("Role %s is not yet globally visible: %v", roleID, err))
			return
		}
	}

	// Create policy
	policy := &models.Policy{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		PermissionsSystemID: data.PermissionsSystemID.ValueString(),
		PrincipalID:         data.PrincipalID.ValueString(),
		RoleIDs:             roleIDs,
	}

	createdPolicyWithETag, err := r.client.CreatePolicy(createCtx, policy)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Policy", err.Error())
		return
	}

	data.ID = types.StringValue(createdPolicyWithETag.Policy.ID)
	data.CreatedAt = types.StringValue(createdPolicyWithETag.Policy.CreatedAt)
	if createdPolicyWithETag.Policy.Creator == "" {
		data.Creator = types.StringNull()
	} else {
		data.Creator = types.StringValue(createdPolicyWithETag.Policy.Creator)
	}

	if createdPolicyWithETag.Policy.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(createdPolicyWithETag.Policy.UpdatedAt)
	}
	if createdPolicyWithETag.Policy.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(createdPolicyWithETag.Policy.Updater)
	}

	data.ETag = types.StringValue(createdPolicyWithETag.ETag)

	// Update role IDs in case the order or values changed
	roleIDList, diags := types.ListValueFrom(ctx, types.StringType, createdPolicyWithETag.Policy.RoleIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RoleIDs = roleIDList

	// Post-create GET to stabilize metadata and ETag
	fresh, gerr := r.client.GetPolicy(data.PermissionsSystemID.ValueString(), data.ID.ValueString())
	if gerr == nil {
		data.CreatedAt = types.StringValue(fresh.Policy.CreatedAt)
		if fresh.Policy.Creator == "" {
			data.Creator = types.StringNull()
		} else {
			data.Creator = types.StringValue(fresh.Policy.Creator)
		}
		if fresh.Policy.UpdatedAt == "" {
			data.UpdatedAt = types.StringNull()
		} else {
			data.UpdatedAt = types.StringValue(fresh.Policy.UpdatedAt)
		}
		if fresh.Policy.Updater == "" {
			data.Updater = types.StringNull()
		} else {
			data.Updater = types.StringValue(fresh.Policy.Updater)
		}
		data.ETag = types.StringValue(fresh.ETag)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyWithETag, err := r.client.GetPolicy(data.PermissionsSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		// Check if the resource was not found (404 error)
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "not found") {
			// Resource no longer exists, remove it from state
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy, got error: %s", err))
		return
	}

	policy := policyWithETag.Policy

	// Map response to model
	data.ID = types.StringValue(policy.ID)
	data.Name = types.StringValue(policy.Name)
	data.Description = types.StringValue(policy.Description)
	data.PrincipalID = types.StringValue(policy.PrincipalID)
	data.CreatedAt = types.StringValue(policy.CreatedAt)
	if policy.Creator == "" {
		data.Creator = types.StringNull()
	} else {
		data.Creator = types.StringValue(policy.Creator)
	}

	if policy.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(policy.UpdatedAt)
	}
	if policy.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(policy.Updater)
	}

	data.ETag = types.StringValue(policyWithETag.ETag)

	// Map role IDs
	roleIDList, diags := types.ListValueFrom(ctx, types.StringType, policy.RoleIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RoleIDs = roleIDList
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current state to get the ETag
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract role IDs from types.List
	var roleIDs []string
	resp.Diagnostics.Append(data.RoleIDs.ElementsAs(ctx, &roleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create policy with updated data - use state values for immutable fields
	policy := &models.Policy{
		ID:                  state.ID.ValueString(), // Use state for immutable ID
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		PermissionsSystemID: data.PermissionsSystemID.ValueString(),
		PrincipalID:         data.PrincipalID.ValueString(),
		RoleIDs:             roleIDs,
		CreatedAt:           state.CreatedAt.ValueString(), // Preserve immutable CreatedAt
	}

	// Handle Creator field - it might be null in state
	if !state.Creator.IsNull() {
		policy.Creator = state.Creator.ValueString()
	}

	updatedPolicyWithETag, err := r.client.UpdatePolicy(ctx, policy, state.ETag.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update policy, got error: %s", err))
		return
	}

	// Update resource data with the response. Preserve immutable fields from state
	data.ID = state.ID
	data.CreatedAt = state.CreatedAt
	data.Creator = state.Creator

	if updatedPolicyWithETag.Policy.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(updatedPolicyWithETag.Policy.UpdatedAt)
	}
	if updatedPolicyWithETag.Policy.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(updatedPolicyWithETag.Policy.Updater)
	}

	// ETag should always update to the new value from API response
	data.ETag = types.StringValue(updatedPolicyWithETag.ETag)

	// Update role IDs in case the order or values changed
	roleIDList, diags := types.ListValueFrom(ctx, types.StringType, updatedPolicyWithETag.Policy.RoleIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RoleIDs = roleIDList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var data policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get context with delete timeout (default 10 minutes for deletes)
	deleteTimeout, diags := data.Timeouts.Delete(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	permissionSystemID := data.PermissionsSystemID.ValueString()

	// Serialize delete per Permission System with 409 retry
	err := r.deleteLanes.WithDelete(deleteCtx, permissionSystemID, func(ctx context.Context) error {
		return deletelanes.Retry409Delete(ctx, func(ctx context.Context) error {
			return r.client.DeletePolicy(permissionSystemID, data.ID.ValueString())
		})
	})

	if err != nil {
		resp.Diagnostics.AddError("Error deleting policy", fmt.Sprintf("Unable to delete policy: %v", err))
		return
	}
}

func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: permission_system_id:policy_id
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import id in format 'permission_system_id:policy_id', got: %s", req.ID),
		)
		return
	}

	permissionSystemID := idParts[0]
	policyID := idParts[1]

	// Set the main identifiers
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_system_id"), permissionSystemID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), policyID)...)

	// Terraform automatically calls Read to fetch the rest of the attributes
}
