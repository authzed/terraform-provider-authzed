package provider

import (
	"context"
	"fmt"

	"terraform-provider-platform-api/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &serviceAccountsDataSource{}

func NewServiceAccountsDataSource() datasource.DataSource {
	return &serviceAccountsDataSource{}
}

type serviceAccountsDataSource struct {
	client *client.CloudClient
}

// serviceAccountsDataSourceModel maps the data source schema to values
type serviceAccountsDataSourceModel struct {
	ID                 types.String                 `tfsdk:"id"`
	PermissionSystemID types.String                 `tfsdk:"permission_system_id"`
	ServiceAccounts    []serviceAccountModelForList `tfsdk:"service_accounts"`
}

type serviceAccountModelForList struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	Creator     types.String `tfsdk:"creator"`
}

func (d *serviceAccountsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_accounts"
}

func (d *serviceAccountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of service accounts for a permission system",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier for the data source",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system to list service accounts for",
			},
			"service_accounts": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of service accounts",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of the service account",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the service account",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "Description of the service account",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "Timestamp when the service account was created",
						},
						"creator": schema.StringAttribute{
							Computed:    true,
							Description: "User who created the service account",
						},
					},
				},
			},
		},
	}
}

func (d *serviceAccountsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *serviceAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serviceAccountsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccounts, err := d.client.ListServiceAccounts(data.PermissionSystemID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list service accounts, got error: %s", err))
		return
	}

	// Set ID for the data source
	data.ID = types.StringValue("service_accounts_for_" + data.PermissionSystemID.ValueString())

	serviceAccountsList := []serviceAccountModelForList{}
	for _, sa := range serviceAccounts {
		serviceAccountsList = append(serviceAccountsList, serviceAccountModelForList{
			ID:          types.StringValue(sa.ID),
			Name:        types.StringValue(sa.Name),
			Description: types.StringValue(sa.Description),
			CreatedAt:   types.StringValue(sa.CreatedAt),
			Creator:     types.StringValue(sa.Creator),
		})
	}

	data.ServiceAccounts = serviceAccountsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
