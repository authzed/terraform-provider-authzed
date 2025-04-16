package provider

import (
	"context"

	"terraform-provider-authzed/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CloudProvider struct {
	version string
}

type CloudProviderModel struct {
	Endpoint   types.String `tfsdk:"endpoint"`
	Token      types.String `tfsdk:"token"`
	APIVersion types.String `tfsdk:"api_version"`
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

	resp.DataSourceData = cloudClient
	resp.ResourceData = cloudClient
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
