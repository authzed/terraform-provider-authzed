package provider

import (
	"context"
	"fmt"

	"terraform-provider-cloudapi/internal/client"
	"terraform-provider-cloudapi/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &policyResource{}

func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

type policyResource struct {
	client *client.CloudClient
}

type policyResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	PermissionSystemID types.String `tfsdk:"permission_system_id"`
	PrincipalID        types.String `tfsdk:"principal_id"`
	RoleIDs            types.List   `tfsdk:"role_ids"`
	CreatedAt          types.String `tfsdk:"created_at"`
	Creator            types.String `tfsdk:"creator"`
}

func (r *policyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *policyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a policy",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this resource",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the policy",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the policy",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system this policy belongs to",
			},
			"principal_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the principal this policy is associated with",
			},
			"role_ids": schema.ListAttribute{
				Required:    true,
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

func (r *policyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.CloudClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.CloudClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract role IDs from types.List
	var roleIDs []string
	resp.Diagnostics.Append(data.RoleIDs.ElementsAs(ctx, &roleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create policy
	policy := &models.Policy{
		Name:               data.Name.ValueString(),
		Description:        data.Description.ValueString(),
		PermissionSystemID: data.PermissionSystemID.ValueString(),
		PrincipalID:        data.PrincipalID.ValueString(),
		RoleIDs:            roleIDs,
	}

	createdPolicy, err := r.client.CreatePolicy(policy)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdPolicy.ID)
	data.CreatedAt = types.StringValue(createdPolicy.CreatedAt)
	data.Creator = types.StringValue(createdPolicy.Creator)

	// Update role IDs in case the order or values changed
	roleIDList, diags := types.ListValueFrom(ctx, types.StringType, createdPolicy.RoleIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RoleIDs = roleIDList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetPolicy(data.PermissionSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy, got error: %s", err))
		return
	}

	// Map response to model
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

func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Policy Update Not Supported",
		"Platform API does not support updating policies. To change a policy, you need to delete it and create a new one.",
	)
}

func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePolicy(data.PermissionSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete policy, got error: %s", err))
		return
	}
}
