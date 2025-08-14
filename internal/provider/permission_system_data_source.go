package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-authzed/internal/client"
)

var _ datasource.DataSourceWithConfigure = &permissionsSystemDataSource{}

func NewPermissionsSystemDataSource() datasource.DataSource {
	return &permissionsSystemDataSource{}
}

type permissionsSystemDataSource struct {
	client *client.CloudClient
}

type permissionsSystemDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	GlobalDnsPath types.String `tfsdk:"global_dns_path"`
	SystemType    types.String `tfsdk:"system_type"`
	SystemState   types.Object `tfsdk:"system_state"`
	Version       types.Object `tfsdk:"version"`
}

func (d *permissionsSystemDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission_system"
}

func (d *permissionsSystemDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a permission system by ID",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the permission system",
			},
			"global_dns_path": schema.StringAttribute{
				Computed:    true,
				Description: "Global DNS path for the permission system",
			},
			"system_type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of the permission system (development or production)",
			},
			"system_state": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "State of the permission system",
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						Computed:    true,
						Description: "Operational status of the permission system (e.g., RUNNING, PROVISIONING, UPGRADING, DEGRADED)",
					},
					"message": schema.StringAttribute{
						Computed:    true,
						Description: "Human-readable explanation of the current system state, including error details or operational status information",
					},
				},
			},
			"version": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Version information for the permission system",
				Attributes: map[string]schema.Attribute{
					"current_version": schema.SingleNestedAttribute{
						Computed:    true,
						Description: "Current SpiceDB version",
						Attributes: map[string]schema.Attribute{
							"display_name": schema.StringAttribute{
								Computed:    true,
								Description: "Display name of the version",
							},
							"supported_feature_names": schema.ListAttribute{
								Computed:    true,
								Description: "Features supported by this version",
								ElementType: types.StringType,
							},
							"version": schema.StringAttribute{
								Computed:    true,
								Description: "Version of SpiceDB",
							},
						},
					},
					"has_update_available": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether an update is available",
					},
					"is_locked_to_version": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether the version is locked",
					},
					"override_image": schema.StringAttribute{
						Computed:    true,
						Description: "Override image for SpiceDB",
					},
					"selected_channel": schema.StringAttribute{
						Computed:    true,
						Description: "Selected channel for updates",
					},
				},
			},
		},
	}
}

func (d *permissionsSystemDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *permissionsSystemDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data permissionsSystemDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissionsSystemWithETag, err := d.client.GetPermissionsSystem(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read permission system, got error: %s", err))
		return
	}

	permissionsSystem := permissionsSystemWithETag.PermissionsSystem
	data.Name = types.StringValue(permissionsSystem.Name)
	data.GlobalDnsPath = types.StringValue(permissionsSystem.GlobalDnsPath)
	data.SystemType = types.StringValue(permissionsSystem.SystemType)

	systemStateMap := map[string]attr.Value{
		"status":  types.StringValue(permissionsSystem.SystemState.Status),
		"message": types.StringValue(permissionsSystem.SystemState.Message),
	}
	systemStateObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"status":  types.StringType,
			"message": types.StringType,
		},
		systemStateMap,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.SystemState = systemStateObj

	supportedFeatureNames := []attr.Value{}
	for _, feature := range permissionsSystem.Version.CurrentVersion.SupportedFeatureNames {
		supportedFeatureNames = append(supportedFeatureNames, types.StringValue(feature))
	}

	supportedFeaturesList, diags := types.ListValue(
		types.StringType,
		supportedFeatureNames,
	)
	resp.Diagnostics.Append(diags...)

	currentVersionMap := map[string]attr.Value{
		"display_name":            types.StringValue(permissionsSystem.Version.CurrentVersion.DisplayName),
		"supported_feature_names": supportedFeaturesList,
		"version":                 types.StringValue(permissionsSystem.Version.CurrentVersion.Version),
	}

	currentVersionObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"display_name":            types.StringType,
			"supported_feature_names": types.ListType{ElemType: types.StringType},
			"version":                 types.StringType,
		},
		currentVersionMap,
	)
	resp.Diagnostics.Append(diags...)

	// Map full Version object
	versionMap := map[string]attr.Value{
		"current_version":      currentVersionObj,
		"has_update_available": types.BoolValue(permissionsSystem.Version.HasUpdateAvailable),
		"is_locked_to_version": types.BoolValue(permissionsSystem.Version.IsLockedToVersion),
		"override_image":       types.StringValue(permissionsSystem.Version.OverrideImage),
		"selected_channel":     types.StringValue(permissionsSystem.Version.SelectedChannel),
	}

	versionObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"current_version": types.ObjectType{AttrTypes: map[string]attr.Type{
				"display_name":            types.StringType,
				"supported_feature_names": types.ListType{ElemType: types.StringType},
				"version":                 types.StringType,
			}},
			"has_update_available": types.BoolType,
			"is_locked_to_version": types.BoolType,
			"override_image":       types.StringType,
			"selected_channel":     types.StringType,
		},
		versionMap,
	)
	resp.Diagnostics.Append(diags...)
	data.Version = versionObj

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
