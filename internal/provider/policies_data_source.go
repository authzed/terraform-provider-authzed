package provider

import (
	"context"
	"fmt"

	"terraform-provider-cloudapi/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &policiesDataSource{}

func NewPoliciesDataSource() datasource.DataSource {
	return &policiesDataSource{}
}

type policiesDataSource struct {
	client *client.CloudClient
}

type policiesDataSourceModel struct {
	ID                 types.String      `tfsdk:"id"`
	PermissionSystemID types.String      `tfsdk:"permission_system_id"`
	Policies           []policyListModel `tfsdk:"policies"`
}

type policyListModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	PermissionSystemID types.String `tfsdk:"permission_system_id"`
	PrincipalID        types.String `tfsdk:"principal_id"`
	RoleIDs            types.List   `tfsdk:"role_ids"`
	CreatedAt          types.String `tfsdk:"created_at"`
	Creator            types.String `tfsdk:"creator"`
}

func (d *policiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policies"
}

func (d *policiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of policies",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Terraform identifier",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system to list policies for",
			},
			"policies": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of policies",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of the policy",
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
							Computed:    true,
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
				},
			},
		},
	}
}

func (d *policiesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *policiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data policiesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use permission system ID as the data source ID
	data.ID = data.PermissionSystemID

	policies, err := d.client.ListPolicies(data.PermissionSystemID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list policies, got error: %s", err))
		return
	}

	// Map response to model
	policyList := make([]policyListModel, 0, len(policies))
	for _, policy := range policies {
		// Map role IDs
		roleIDList, diags := types.ListValueFrom(ctx, types.StringType, policy.RoleIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		policyList = append(policyList, policyListModel{
			ID:                 types.StringValue(policy.ID),
			Name:               types.StringValue(policy.Name),
			Description:        types.StringValue(policy.Description),
			PermissionSystemID: types.StringValue(policy.PermissionSystemID),
			PrincipalID:        types.StringValue(policy.PrincipalID),
			RoleIDs:            roleIDList,
			CreatedAt:          types.StringValue(policy.CreatedAt),
			Creator:            types.StringValue(policy.Creator),
		})
	}

	data.Policies = policyList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
