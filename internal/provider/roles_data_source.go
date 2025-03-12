package provider

import (
	"context"
	"fmt"

	"terraform-provider-platform-api/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &rolesDataSource{}

// NewRolesDataSource creates a new data source for listing roles
func NewRolesDataSource() datasource.DataSource {
	return &rolesDataSource{}
}

type rolesDataSource struct {
	client *client.PlatformClient
}

type rolesDataSourceModel struct {
	ID                 types.String    `tfsdk:"id"`
	PermissionSystemID types.String    `tfsdk:"permission_system_id"`
	Roles              []roleDataModel `tfsdk:"roles"`
}

// roleDataModel describes a single role in the roles list
type roleDataModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Creator     types.String `tfsdk:"creator"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func (d *rolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *rolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all roles for a given permission system",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Placeholder identifier for this data source",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system to list roles for",
			},
			"roles": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of roles in the permission system",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Unique identifier for the role",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the role",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "Description of the role",
						},
						"creator": schema.StringAttribute{
							Computed:    true,
							Description: "User who created the role",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "Timestamp when the role was created",
						},
					},
				},
			},
		},
	}
}

func (d *rolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.PlatformClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.PlatformClient, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data
func (d *rolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data rolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get roles from API
	roles, err := d.client.ListRoles(data.PermissionSystemID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles, got error: %s", err))
		return
	}

	// Map response to model
	data.ID = types.StringValue(data.PermissionSystemID.ValueString())

	// Map roles
	rolesList := make([]roleDataModel, 0, len(roles))
	for _, role := range roles {
		roleData := roleDataModel{
			ID:          types.StringValue(role.ID),
			Name:        types.StringValue(role.Name),
			Description: types.StringValue(role.Description),
			Creator:     types.StringValue(role.Creator),
			CreatedAt:   types.StringValue(role.CreatedAt),
		}
		rolesList = append(rolesList, roleData)
	}
	data.Roles = rolesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
