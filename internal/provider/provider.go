package provider

import (
	"context"
	"fmt"

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
	Host       types.String `tfsdk:"host"`
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
			"host": schema.StringAttribute{
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
		Host:       config.Host.ValueString(),
		Token:      config.Token.ValueString(),
		APIVersion: config.APIVersion.ValueString(),
	}

	platformClient := client.NewPlatformClient(clientConfig)

	resp.DataSourceData = platformClient
	resp.ResourceData = platformClient
}

func (p *PlatformProvider) Resources(_ context.Context) []func() resource.Resource {
	fmt.Println("PROVIDER DEBUG: Resources method called")

	roleResource := NewRoleResource()
	if roleResource == nil {
		panic("NewRoleResource() returned nil")
	}
	fmt.Printf("PROVIDER DEBUG: Role resource type: %T\n", roleResource)

	resources := []func() resource.Resource{
		NewRoleResource,
	}
	fmt.Printf("PROVIDER DEBUG: Returning %d resources\n", len(resources))
	return resources
}

func (p *PlatformProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	fmt.Println("PROVIDER DEBUG: DataSources method called")

	// Check each data source constructor
	permSysDS := NewPermissionSystemDataSource()
	if permSysDS == nil {
		panic("NewPermissionSystemDataSource() returned nil")
	}
	fmt.Printf("PROVIDER DEBUG: Permission System DS type: %T\n", permSysDS)

	permSysListDS := NewPermissionSystemsDataSource()
	if permSysListDS == nil {
		panic("NewPermissionSystemsDataSource() returned nil")
	}
	fmt.Printf("PROVIDER DEBUG: Permission Systems List DS type: %T\n", permSysListDS)

	roleDS := NewRoleDataSource()
	if roleDS == nil {
		panic("NewRoleDataSource() returned nil")
	}
	fmt.Printf("PROVIDER DEBUG: Role DS type: %T\n", roleDS)

	rolesDS := NewRolesDataSource()
	if rolesDS == nil {
		panic("NewRolesDataSource() returned nil")
	}
	fmt.Printf("PROVIDER DEBUG: Roles List DS type: %T\n", rolesDS)

	dataSources := []func() datasource.DataSource{
		NewPermissionSystemDataSource,
		NewPermissionSystemsDataSource,
		NewRoleDataSource,
		NewRolesDataSource,
	}
	fmt.Printf("PROVIDER DEBUG: Returning %d data sources\n", len(dataSources))
	return dataSources
}
