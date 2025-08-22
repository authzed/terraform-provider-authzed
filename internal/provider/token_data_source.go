package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-authzed/internal/client"
)

var _ datasource.DataSourceWithConfigure = &TokenDataSource{}

func NewTokenDataSource() datasource.DataSource {
	return &TokenDataSource{}
}

type TokenDataSource struct {
	client *client.CloudClient
}

type TokenDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	TokenID             types.String `tfsdk:"token_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	PermissionsSystemID types.String `tfsdk:"permission_system_id"`
	ServiceAccountID    types.String `tfsdk:"service_account_id"`
	CreatedAt           types.String `tfsdk:"created_at"`
	Creator             types.String `tfsdk:"creator"`
	ETag                types.String `tfsdk:"etag"`
}

func (d *TokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (d *TokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Gets a token by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite ID for this token",
				Computed:    true,
			},
			"token_id": schema.StringAttribute{
				Description: "The globally unique ID for this token",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the token",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The human-supplied description of the token",
				Computed:    true,
			},
			"permission_system_id": schema.StringAttribute{
				Description: "The globally unique ID for the permission system",
				Required:    true,
			},
			"service_account_id": schema.StringAttribute{
				Description: "The globally unique ID for the containing service account",
				Required:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the token was created",
				Computed:    true,
			},
			"creator": schema.StringAttribute{
				Description: "The name of the user that created this token",
				Computed:    true,
			},
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "Version identifier for the resource, used by update operations to prevent conflicts",
			},
		},
	}
}

func (d *TokenDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TokenDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tokenWithETag, err := d.client.GetToken(
		config.PermissionsSystemID.ValueString(),
		config.ServiceAccountID.ValueString(),
		config.TokenID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading token",
			fmt.Sprintf("Unable to read token: %v", err),
		)
		return
	}

	// Populate the model
	config.ID = types.StringValue(fmt.Sprintf("%s:%s:%s",
		tokenWithETag.Token.PermissionsSystemID, tokenWithETag.Token.ServiceAccountID, tokenWithETag.Token.ID))
	config.Name = types.StringValue(tokenWithETag.Token.Name)
	config.Description = types.StringValue(tokenWithETag.Token.Description)
	config.CreatedAt = types.StringValue(tokenWithETag.Token.CreatedAt)
	config.Creator = types.StringValue(tokenWithETag.Token.Creator)
	config.ETag = types.StringValue(tokenWithETag.ETag)

	// Save the data into Terraform state
	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
