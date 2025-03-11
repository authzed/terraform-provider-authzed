package provider

import (
	"context"
	"fmt"

	"terraform-provider-platform-api/internal/client"
	"terraform-provider-platform-api/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &roleResource{}

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	client *client.PlatformClient
}

type roleResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	PermissionSystemID types.String `tfsdk:"permission_system_id"`
	Permissions        types.Map    `tfsdk:"permissions"`
	CreatedAt          types.String `tfsdk:"created_at"`
	Creator            types.String `tfsdk:"creator"`
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a role",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this resource",
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
			},
			"creator": schema.StringAttribute{
				Computed:    true,
				Description: "User who created the role",
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.PlatformClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.PlatformClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract permissions map from types.Map
	permissionsMap := make(models.PermissionExprMap)
	data.Permissions.ElementsAs(ctx, &permissionsMap, false)

	// Create role
	role := &models.Role{
		Name:               data.Name.ValueString(),
		Description:        data.Description.ValueString(),
		PermissionSystemID: data.PermissionSystemID.ValueString(),
		Permissions:        permissionsMap,
	}

	createdRole, err := r.client.CreateRole(role)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdRole.ID)
	data.CreatedAt = types.StringValue(createdRole.CreatedAt)
	data.Creator = types.StringValue(createdRole.Creator)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the role state
func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRole(data.PermissionSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role, got error: %s", err))
		return
	}

	// Map response to model
	data.Name = types.StringValue(role.Name)
	data.Description = types.StringValue(role.Description)
	data.CreatedAt = types.StringValue(role.CreatedAt)
	data.Creator = types.StringValue(role.Creator)

	// Map permissions
	permissions := make(map[string]types.String)
	for k, v := range role.Permissions {
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

	err := r.client.DeleteRole(data.PermissionSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role during update, got error: %s", err))
		return
	}

	permissionsMap := make(models.PermissionExprMap)
	data.Permissions.ElementsAs(ctx, &permissionsMap, false)

	// Create role with updated data, since we
	role := &models.Role{
		Name:               data.Name.ValueString(),
		Description:        data.Description.ValueString(),
		PermissionSystemID: data.PermissionSystemID.ValueString(),
		Permissions:        permissionsMap,
	}

	createdRole, err := r.client.CreateRole(role)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to recreate role during update, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdRole.ID)
	data.CreatedAt = types.StringValue(createdRole.CreatedAt)
	data.Creator = types.StringValue(createdRole.Creator)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRole(data.PermissionSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role, got error: %s", err))
		return
	}
}
