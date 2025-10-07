package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-authzed/internal/client"
)

var _ datasource.DataSourceWithConfigure = &roleDataSource{}

func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

type roleDataSource struct {
	client *client.CloudClient
}

type roleDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	RoleID              types.String `tfsdk:"role_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	PermissionsSystemID types.String `tfsdk:"permission_system_id"`
	Permissions         types.Map    `tfsdk:"permissions"`
	CreatedAt           types.String `tfsdk:"created_at"`
	Creator             types.String `tfsdk:"creator"`
	ETag                types.String `tfsdk:"etag"`
}

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a role by ID",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this resource",
			},
			"role_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the role",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the role",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the role",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system this role belongs to",
			},
			"permissions": schema.MapAttribute{
				Computed:    true,
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
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "Version identifier for the resource, used by update operations to prevent conflicts",
			},
		},
	}
}

func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*CloudProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *CloudProviderData, got: %T", req.ProviderData),
		)
		return
	}

	d.client = providerData.Client
}

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleWithETag, err := d.client.GetRole(ctx, data.PermissionsSystemID.ValueString(), data.RoleID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role, got error: %s", err))
		return
	}

	data.ID = types.StringValue(roleWithETag.Role.ID)
	data.Name = types.StringValue(roleWithETag.Role.Name)
	data.Description = types.StringValue(roleWithETag.Role.Description)
	data.CreatedAt = types.StringValue(roleWithETag.Role.CreatedAt)
	data.Creator = types.StringValue(roleWithETag.Role.Creator)
	data.ETag = types.StringValue(roleWithETag.ETag)

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
