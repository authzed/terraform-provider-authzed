package provider

import (
	"context"
	"fmt"

	"terraform-provider-authzed/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &permissionsSystemsDataSource{}

func NewPermissionsSystemsDataSource() datasource.DataSource {
	return &permissionsSystemsDataSource{}
}

type permissionsSystemsDataSource struct {
	client *client.CloudClient
}

// permissionsSystemsDataSourceModel maps the data source schema to values
type permissionsSystemsDataSourceModel struct {
	ID                 types.String                    `tfsdk:"id"`
	PermissionsSystems []permissionsSystemModelForList `tfsdk:"permission_systems"`
}

type permissionsSystemModelForList struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	GlobalDnsPath types.String `tfsdk:"global_dns_path"`
	SystemType    types.String `tfsdk:"system_type"`
}

func (d *permissionsSystemsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission_systems"
}

func (d *permissionsSystemsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of all permission systems",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier for the data source",
			},
			"permission_systems": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of permission systems",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of the permission system",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the permission system",
						},
						"global_dns_path": schema.StringAttribute{
							Computed:    true,
							Description: "Global DNS path for the permission system",
						},
						"system_type": schema.StringAttribute{
							Computed:    true,
							Description: "Type of the permission system (development or production)",
						},
					},
				},
			},
		},
	}
}

func (d *permissionsSystemsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.CloudClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.CloudClient, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data
func (d *permissionsSystemsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data permissionsSystemsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// List permission systems from API
	permissionsSystems, err := d.client.ListPermissionsSystems()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list permission systems, got error: %s", err))
		return
	}

	// Set ID for the data source
	data.ID = types.StringValue("all_permission_systems")

	permissionsSystemsList := []permissionsSystemModelForList{}
	for _, ps := range permissionsSystems {
		permissionsSystemsList = append(permissionsSystemsList, permissionsSystemModelForList{
			ID:            types.StringValue(ps.ID),
			Name:          types.StringValue(ps.Name),
			GlobalDnsPath: types.StringValue(ps.GlobalDnsPath),
			SystemType:    types.StringValue(ps.SystemType),
		})
	}

	data.PermissionsSystems = permissionsSystemsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
