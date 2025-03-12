package provider

import (
	"context"
	"fmt"
	"terraform-provider-platform-api/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure TokensDataSource satisfies the datasource interfaces
var _ datasource.DataSource = &TokensDataSource{}

// NewTokensDataSource returns a new tokens data source instance
func NewTokensDataSource() datasource.DataSource {
	return &TokensDataSource{}
}

// TokensDataSource defines the tokens data source implementation
type TokensDataSource struct {
	client *client.PlatformClient
}

// TokensDataSourceModel represents the tokens data source schema model
type TokensDataSourceModel struct {
	ID                 types.String    `tfsdk:"id"`
	PermissionSystemID types.String    `tfsdk:"permission_system_id"`
	ServiceAccountID   types.String    `tfsdk:"service_account_id"`
	Tokens             []TokenDataItem `tfsdk:"tokens"`
	TokensCount        types.Int64     `tfsdk:"tokens_count"`
}

// TokenDataItem represents a single token in the list of tokens
type TokenDataItem struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	Creator     types.String `tfsdk:"creator"`
}

// Metadata returns the data source type name
func (d *TokensDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tokens"
}

// Schema defines the schema for the data source
func (d *TokensDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all tokens for a service account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite ID for this tokens list",
				Computed:    true,
			},
			"permission_system_id": schema.StringAttribute{
				Description: "The globally unique ID for the permission system",
				Required:    true,
			},
			"service_account_id": schema.StringAttribute{
				Description: "The globally unique ID for the service account",
				Required:    true,
			},
			"tokens_count": schema.Int64Attribute{
				Description: "The number of tokens in the list",
				Computed:    true,
			},
			"tokens": schema.ListNestedAttribute{
				Description: "The list of tokens",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The globally unique ID for this token",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the token",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The human-supplied description of the token",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The timestamp when the token was created",
							Computed:    true,
						},
						"creator": schema.StringAttribute{
							Description: "The name of the user that created this token",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the data source with the provider client
func (d *TokensDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the tokens list from the API
func (d *TokensDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TokensDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissionSystemID := config.PermissionSystemID.ValueString()
	serviceAccountID := config.ServiceAccountID.ValueString()

	tokens, err := d.client.ListTokens(permissionSystemID, serviceAccountID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tokens",
			fmt.Sprintf("Unable to read tokens: %v", err),
		)
		return
	}

	// Generate the ID
	config.ID = types.StringValue(fmt.Sprintf("%s:%s", permissionSystemID, serviceAccountID))

	// Convert the tokens into the model
	tokensList := make([]TokenDataItem, 0, len(tokens))
	for _, token := range tokens {
		tokensList = append(tokensList, TokenDataItem{
			ID:          types.StringValue(token.ID),
			Name:        types.StringValue(token.Name),
			Description: types.StringValue(token.Description),
			CreatedAt:   types.StringValue(token.CreatedAt),
			Creator:     types.StringValue(token.Creator),
		})
	}

	config.Tokens = tokensList
	config.TokensCount = types.Int64Value(int64(len(tokens)))

	// Save the data into Terraform state
	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
