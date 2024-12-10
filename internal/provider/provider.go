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
	Host  types.String `tfsdk:"host"`
	Token types.String `tfsdk:"token"`
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
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The host address of the Platform API",
			},
			"token": schema.StringAttribute{
				Required:    true,
				Description: "The bearer token for authentication",
				Sensitive:   true,
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

	conf := client.NewConfiguration()
	conf.Servers = []client.ServerConfiguration{
		{
			URL:         config.Host.ValueString(),
			Description: "v1",
		},
	}
	conf.DefaultHeader["X-API-Version"] = "20241017"
	conf.AddDefaultHeader("Authorization", "Bearer "+config.Token.ValueString())

	apiClient := client.NewAPIClient(conf)

	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
}

func (p *PlatformProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHelloResource,
	}
}

func (p *PlatformProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
