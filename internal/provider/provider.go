package provider

import (
	"context"
	"sync"

	"terraform-provider-authzed/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FGAMCoordinator handles serialization of FGAM operations per permission system
type FGAMCoordinator struct {
	enabled  bool
	mutexes  map[string]*sync.Mutex
	mapMutex sync.RWMutex
}

// NewFGAMCoordinator creates a new FGAM coordinator
func NewFGAMCoordinator(enabled bool) *FGAMCoordinator {
	return &FGAMCoordinator{
		enabled: enabled,
		mutexes: make(map[string]*sync.Mutex),
	}
}

// Lock acquires a lock for the given permission system ID
func (fc *FGAMCoordinator) Lock(permissionSystemID string) {
	if !fc.enabled {
		return
	}

	fc.mapMutex.RLock()
	mutex, exists := fc.mutexes[permissionSystemID]
	fc.mapMutex.RUnlock()

	if !exists {
		fc.mapMutex.Lock()
		// Double-check after acquiring write lock
		if mutex, exists = fc.mutexes[permissionSystemID]; !exists {
			mutex = &sync.Mutex{}
			fc.mutexes[permissionSystemID] = mutex
		}
		fc.mapMutex.Unlock()
	}

	mutex.Lock()
}

// Unlock releases the lock for the given permission system ID
func (fc *FGAMCoordinator) Unlock(permissionSystemID string) {
	if !fc.enabled {
		return
	}

	fc.mapMutex.RLock()
	if mutex, exists := fc.mutexes[permissionSystemID]; exists {
		mutex.Unlock()
	}
	fc.mapMutex.RUnlock()
}

type CloudProvider struct {
	version string
}

type CloudProviderModel struct {
	Endpoint          types.String `tfsdk:"endpoint"`
	Token             types.String `tfsdk:"token"`
	APIVersion        types.String `tfsdk:"api_version"`
	FGAMSerialization types.Bool   `tfsdk:"fgam_serialization"`
}

// CloudProviderData contains the configured client and coordinator
type CloudProviderData struct {
	Client          *client.CloudClient
	FGAMCoordinator *FGAMCoordinator
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
			"fgam_serialization": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable serialization of Fine-Grained Access Management operations to prevent conflicts. When enabled, FGAM resources within the same permission system will be created/updated sequentially instead of in parallel.",
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

	// Create FGAM coordinator based on configuration
	fgamSerialization := config.FGAMSerialization.ValueBool()
	fgamCoordinator := NewFGAMCoordinator(fgamSerialization)

	providerData := &CloudProviderData{
		Client:          cloudClient,
		FGAMCoordinator: fgamCoordinator,
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
