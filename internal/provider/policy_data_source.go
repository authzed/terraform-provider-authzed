package provider

import (
	"context"
	"fmt"

	"terraform-provider-authzed/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &policyDataSource{}

func NewPolicyDataSource() datasource.DataSource {
	return &policyDataSource{}
}

type policyDataSource struct {
	client *client.CloudClient
}

type policyDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	PolicyID            types.String `tfsdk:"policy_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	PermissionsSystemID types.String `tfsdk:"permission_system_id"`
	PrincipalID         types.String `tfsdk:"principal_id"`
	RoleIDs             types.List   `tfsdk:"role_ids"`
	CreatedAt           types.String `tfsdk:"created_at"`
	Creator             types.String `tfsdk:"creator"`
}

func (d *policyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (d *policyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a policy",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Terraform identifier",
			},
			"policy_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the policy to fetch",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the policy",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the policy",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system this policy belongs to",
			},
			"principal_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the principal this policy is associated with",
			},
			"role_ids": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "IDs of the roles this policy is associated with",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the policy was created",
			},
			"creator": schema.StringAttribute{
				Computed:    true,
				Description: "User who created the policy",
			},
		},
	}
}

func (d *policyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.CloudClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.CloudClient, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *policyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data policyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := d.client.GetPolicy(data.PermissionsSystemID.ValueString(), data.PolicyID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy, got error: %s", err))
		return
	}

	// Map response to model
	data.ID = types.StringValue(policy.ID)
	data.Name = types.StringValue(policy.Name)
	data.Description = types.StringValue(policy.Description)
	data.PrincipalID = types.StringValue(policy.PrincipalID)
	data.CreatedAt = types.StringValue(policy.CreatedAt)
	data.Creator = types.StringValue(policy.Creator)

	// Map role IDs
	roleIDList, diags := types.ListValueFrom(ctx, types.StringType, policy.RoleIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RoleIDs = roleIDList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
