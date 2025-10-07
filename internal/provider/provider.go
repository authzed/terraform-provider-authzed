package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/provider/pslanes"
)

type CloudProvider struct {
	version string
}

type CloudProviderModel struct {
	Endpoint                     types.String `tfsdk:"endpoint"`
	Token                        types.String `tfsdk:"token"`
	APIVersion                   types.String `tfsdk:"api_version"`
	DeleteTimeout                types.String `tfsdk:"delete_timeout"`
	AutoParallelism              types.Bool   `tfsdk:"auto_parallelism"`
	MaxConcurrentServiceAccounts types.Int64  `tfsdk:"max_concurrent_service_accounts"`
	MaxConcurrentTokens          types.Int64  `tfsdk:"max_concurrent_tokens"`
	MaxConcurrentPolicies        types.Int64  `tfsdk:"max_concurrent_policies"`
	MaxConcurrentRoles           types.Int64  `tfsdk:"max_concurrent_roles"`
}

// CloudProviderData contains the configured client and essential components
type CloudProviderData struct {
	Client  *client.CloudClient
	PSLanes *pslanes.PSLanes
}

var _ provider.Provider = &CloudProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CloudProvider{
			version: version,
		}
	}
}

func (p *CloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "authzed"
	resp.Version = p.version
}

func (p *CloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "The host address of AuthZed Cloud API",
			},
			"token": schema.StringAttribute{
				Required:    true,
				Description: "The bearer token for authentication",
				Sensitive:   true,
			},
			"api_version": schema.StringAttribute{
				Optional:    true,
				Description: "The version of the API to use (defaults to 25r1)",
			},
			"delete_timeout": schema.StringAttribute{
				Optional:    true,
				Description: "Maximum time to wait for asynchronous deletes to complete (e.g., 5m, 15m).",
			},
			"auto_parallelism": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable automatic parallelism recommendations based on resource count. Can also be set via AUTHZED_AUTO_PARALLELISM.",
			},
			"max_concurrent_service_accounts": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of concurrent service account operations (default: 6). Can also be set via AUTHZED_MAX_CONCURRENT_SERVICE_ACCOUNTS.",
			},
			"max_concurrent_tokens": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of concurrent token operations (default: 8). Can also be set via AUTHZED_MAX_CONCURRENT_TOKENS.",
			},
			"max_concurrent_policies": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of concurrent policy operations (default: 3). Can also be set via AUTHZED_MAX_CONCURRENT_POLICIES.",
			},
			"max_concurrent_roles": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of concurrent role operations (default: 3). Can also be set via AUTHZED_MAX_CONCURRENT_ROLES.",
			},
		},
	}
}

func (p *CloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config CloudProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientConfig := &client.CloudClientConfig{
		Host:       config.Endpoint.ValueString(),
		Token:      config.Token.ValueString(),
		APIVersion: config.APIVersion.ValueString(),
	}

	cloudClient := client.NewCloudClient(clientConfig)

	// Initialize PSLanes for per-Permission System serialization
	psLanes := pslanes.NewPSLanes()

	providerData := &CloudProviderData{
		Client:  cloudClient,
		PSLanes: psLanes,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *CloudProvider) Resources(_ context.Context) []func() resource.Resource {
	resources := []func() resource.Resource{
		NewRoleResource,
		NewPolicyResource,
		NewServiceAccountResource,
		NewTokenResource,
	}
	return resources
}

func (p *CloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	dataSources := []func() datasource.DataSource{
		NewPermissionsSystemDataSource,
		NewPermissionsSystemsDataSource,
		NewRoleDataSource,
		NewRolesDataSource,
		NewPolicyDataSource,
		NewPoliciesDataSource,
		NewServiceAccountDataSource,
		NewServiceAccountsDataSource,
		NewTokenDataSource,
		NewTokensDataSource,
	}
	return dataSources
}
