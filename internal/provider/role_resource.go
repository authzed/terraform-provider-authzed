package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
	"terraform-provider-authzed/internal/provider/pslanes"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	client  *client.CloudClient
	psLanes *pslanes.PSLanes
}

type roleResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	Description         types.String   `tfsdk:"description"`
	PermissionsSystemID types.String   `tfsdk:"permission_system_id"`
	Permissions         types.Map      `tfsdk:"permissions"`
	CreatedAt           types.String   `tfsdk:"created_at"`
	Creator             types.String   `tfsdk:"creator"`
	UpdatedAt           types.String   `tfsdk:"updated_at"`
	Updater             types.String   `tfsdk:"updater"`
	ETag                types.String   `tfsdk:"etag"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a role",
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
				Description: "Name of the role",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the role",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system this role belongs to",
			},
			"permissions": schema.MapAttribute{
				Required:    true,
				Description: "Map of permission name to expression",
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"creator": schema.StringAttribute{
				Computed:    true,
				Description: "User who created the role",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was last updated",
			},
			"updater": schema.StringAttribute{
				Computed:    true,
				Description: "User who last updated the role",
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

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.psLanes = providerData.PSLanes
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var data roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get context with create timeout (default 10 minutes for roles)
	createTimeout, diags := data.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	// Existence gate: ensure Permission System exists before role create (use createCtx)
	if psID := data.PermissionsSystemID.ValueString(); psID != "" {
		_ = waitForExists(createCtx, func(c context.Context) (bool, error) {
			_, err := r.client.GetPermissionsSystem(c, psID)
			if err != nil {
				return false, err
			}
			return true, nil
		})

	}

	// Extract permissions map from types.Map
	permissionsMap := make(models.PermissionExprMap)
	data.Permissions.ElementsAs(ctx, &permissionsMap, false)

	// Create role
	role := &models.Role{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		PermissionsSystemID: data.PermissionsSystemID.ValueString(),
		Permissions:         permissionsMap,
	}

	// Serialize role creation per Permission System to prevent FGAM conflicts
	var createdRoleWithETag *client.RoleWithETag
	err := r.psLanes.WithCreateLane(createCtx, role.PermissionsSystemID, func() error {
		var createErr error
		createdRoleWithETag, createErr = r.client.CreateRole(createCtx, role)
		return createErr
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdRoleWithETag.Role.ID)
	data.CreatedAt = types.StringValue(createdRoleWithETag.Role.CreatedAt)
	data.Creator = types.StringValue(createdRoleWithETag.Role.Creator)
	if createdRoleWithETag.Role.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(createdRoleWithETag.Role.UpdatedAt)
	}
	if createdRoleWithETag.Role.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(createdRoleWithETag.Role.Updater)
	}
	data.ETag = types.StringValue(createdRoleWithETag.ETag)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the role state
func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleWithETag, err := r.client.GetRole(ctx, data.PermissionsSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		// Check if the resource was not found (404 error)
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "not found") {
			// Resource no longer exists, remove it from state
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role, got error: %s", err))
		return
	}

	// Map response to model
	data.Name = types.StringValue(roleWithETag.Role.Name)
	data.Description = types.StringValue(roleWithETag.Role.Description)
	data.CreatedAt = types.StringValue(roleWithETag.Role.CreatedAt)
	data.Creator = types.StringValue(roleWithETag.Role.Creator)
	if roleWithETag.Role.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(roleWithETag.Role.UpdatedAt)
	}
	if roleWithETag.Role.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(roleWithETag.Role.Updater)
	}
	data.ETag = types.StringValue(roleWithETag.ETag)

	// Map permissions
	permissions := make(map[string]types.String)
	for k, v := range roleWithETag.Role.Permissions {
		permissions[k] = types.StringValue(v)
	}
	permissionsValue, diags := types.MapValueFrom(ctx, types.StringType, permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Permissions = permissionsValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current state to get the ETag
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract permissions from the model
	permissionsMap := make(models.PermissionExprMap)
	data.Permissions.ElementsAs(ctx, &permissionsMap, false)

	// Create role with updated data, use state values for immutable fields
	role := &models.Role{
		ID:                  state.ID.ValueString(),
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		PermissionsSystemID: data.PermissionsSystemID.ValueString(),
		Permissions:         permissionsMap,
		CreatedAt:           state.CreatedAt.ValueString(),
	}

	if !state.Creator.IsNull() {
		role.Creator = state.Creator.ValueString()
	}

	// Use the ETag from state for optimistic concurrency control
	updatedRoleWithETag, err := r.client.UpdateRole(ctx, role, state.ETag.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update role, got error: %s", err))
		return
	}

	// Update resource data with the response - preserve immutable fields from state
	data.ID = state.ID
	data.CreatedAt = state.CreatedAt
	data.Creator = state.Creator
	if updatedRoleWithETag.Role.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(updatedRoleWithETag.Role.UpdatedAt)
	}
	if updatedRoleWithETag.Role.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(updatedRoleWithETag.Role.Updater)
	}
	data.ETag = types.StringValue(updatedRoleWithETag.ETag)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var data roleResourceModel
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

	// Serialize role deletion per Permission System with 409 retry
	err := r.psLanes.WithDeleteLane(deleteCtx, permissionSystemID, func() error {
		return pslanes.Retry409Delete(deleteCtx, func() error {
			return r.client.DeleteRole(permissionSystemID, data.ID.ValueString())
		})
	})

	if err != nil {
		resp.Diagnostics.AddError("Error deleting role", fmt.Sprintf("Unable to delete role: %v", err))
		return
	}
}

// ImportState handles importing an existing role into Terraform state
func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import id in format 'permission_system_id:role_id', got: %s", req.ID),
		)
		return
	}

	permissionSystemID := idParts[0]
	roleID := idParts[1]

	// Set the main identifiers
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_system_id"), permissionSystemID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), roleID)...)

	// Terraform automatically calls Read to fetch the rest of the attributes
}
