package provider

import (
	"context"
	"fmt"

	"terraform-provider-authzed/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &serviceAccountDataSource{}

func NewServiceAccountDataSource() datasource.DataSource {
	return &serviceAccountDataSource{}
}

type serviceAccountDataSource struct {
	client *client.CloudClient
}

type serviceAccountDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	PermissionsSystemID types.String `tfsdk:"permission_system_id"`
	ServiceAccountID    types.String `tfsdk:"service_account_id"`
	CreatedAt           types.String `tfsdk:"created_at"`
	Creator             types.String `tfsdk:"creator"`
	ETag                types.String `tfsdk:"etag"`
}

func (d *serviceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (d *serviceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a service account by ID",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier for the data source",
			},
			"service_account_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the service account to fetch",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system this service account belongs to",
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
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "Version identifier for the resource, used by update operations to prevent conflicts",
			},
		},
	}
}

func (d *serviceAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read refreshes the Terraform state with the latest data
func (d *serviceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serviceAccountDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccountWithETag, err := d.client.GetServiceAccount(
		data.PermissionsSystemID.ValueString(),
		data.ServiceAccountID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account, got error: %s", err))
		return
	}

	data.ID = types.StringValue(serviceAccountWithETag.ServiceAccount.ID)
	data.Name = types.StringValue(serviceAccountWithETag.ServiceAccount.Name)
	data.Description = types.StringValue(serviceAccountWithETag.ServiceAccount.Description)
	data.CreatedAt = types.StringValue(serviceAccountWithETag.ServiceAccount.CreatedAt)
	data.Creator = types.StringValue(serviceAccountWithETag.ServiceAccount.Creator)
	data.ETag = types.StringValue(serviceAccountWithETag.ETag)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
