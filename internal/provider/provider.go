package provider

import (
	"context"

	"terraform-provider-platform-api/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PlatformProvider struct {
	version string
}

type PlatformProviderModel struct {
	Endpoint   types.String `tfsdk:"endpoint"`
	Token      types.String `tfsdk:"token"`
	APIVersion types.String `tfsdk:"api_version"`
}

var _ provider.Provider = &PlatformProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PlatformProvider{
			version: version,
		}
	}
}

func (p *PlatformProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "platform-api"
	resp.Version = p.version
}

func (p *PlatformProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "The host address of Platform API",
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
		},
	}
}

func (p *PlatformProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config PlatformProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientConfig := &client.PlatformClientConfig{
		Host:       config.Endpoint.ValueString(),
		Token:      config.Token.ValueString(),
		APIVersion: config.APIVersion.ValueString(),
	}

	platformClient := client.NewPlatformClient(clientConfig)

	resp.DataSourceData = platformClient
	resp.ResourceData = platformClient
}

func (p *PlatformProvider) Resources(_ context.Context) []func() resource.Resource {
	resources := []func() resource.Resource{
		NewRoleResource,
		NewPolicyResource,
		NewServiceAccountResource,
	}
	return resources
}

func (p *PlatformProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	dataSources := []func() datasource.DataSource{
		NewPermissionSystemDataSource,
		NewPermissionSystemsDataSource,
		NewRoleDataSource,
		NewRolesDataSource,
		NewPolicyDataSource,
		NewPoliciesDataSource,
		NewServiceAccountDataSource,
		NewServiceAccountsDataSource,
	}
	return dataSources
}
