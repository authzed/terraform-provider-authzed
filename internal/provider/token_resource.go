package provider

import (
	"context"
	"fmt"
	"terraform-provider-cloud-api/internal/client"
	"terraform-provider-cloud-api/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &TokenResource{}

func NewTokenResource() resource.Resource {
	return &TokenResource{}
}

type TokenResource struct {
	client *client.CloudClient
}

type TokenResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	PermissionSystemID types.String `tfsdk:"permission_system_id"`
	ServiceAccountID   types.String `tfsdk:"service_account_id"`
	CreatedAt          types.String `tfsdk:"created_at"`
	Creator            types.String `tfsdk:"creator"`
	Secret             types.String `tfsdk:"secret"`
}

func (r *TokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (r *TokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a service account token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The globally unique ID for this token",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the token",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The human-supplied description of the token",
				Optional:    true,
			},
			"permission_system_id": schema.StringAttribute{
				Description: "The globally unique ID for the permission system",
				Required:    true,
			},
			"service_account_id": schema.StringAttribute{
				Description: "The globally unique ID for the containing service account",
				Required:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the token was created",
				Computed:    true,
			},
			"creator": schema.StringAttribute{
				Description: "The name of the user that created this token",
				Computed:    true,
			},
			"secret": schema.StringAttribute{
				Description: "The secret value of the token. Only available after creation.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *TokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan TokenResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new token
	token := &models.Token{
		Name:               plan.Name.ValueString(),
		Description:        plan.Description.ValueString(),
		PermissionSystemID: plan.PermissionSystemID.ValueString(),
		ServiceAccountID:   plan.ServiceAccountID.ValueString(),
	}

	tflog.Info(ctx, "Creating token", map[string]interface{}{
		"name":                 token.Name,
		"permission_system_id": token.PermissionSystemID,
		"service_account_id":   token.ServiceAccountID,
	})

	createdToken, err := r.client.CreateToken(token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating token",
			fmt.Sprintf("Unable to create token: %v", err),
		)
		return
	}

	plan.ID = types.StringValue(createdToken.ID)
	plan.Name = types.StringValue(createdToken.Name)
	plan.Description = types.StringValue(createdToken.Description)
	plan.PermissionSystemID = types.StringValue(createdToken.PermissionSystemID)
	plan.ServiceAccountID = types.StringValue(createdToken.ServiceAccountID)
	plan.CreatedAt = types.StringValue(createdToken.CreatedAt)
	plan.Creator = types.StringValue(createdToken.Creator)

	// Set the secret (only available during creation)
	if createdToken.Hash != "" {
		plan.Secret = types.StringValue(createdToken.Hash)
	}

	// Save data into Terraform state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the token state
func (r *TokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state TokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.GetToken(
		state.PermissionSystemID.ValueString(),
		state.ServiceAccountID.ValueString(),
		state.ID.ValueString(),
	)
	if err != nil {
		apiErr, ok := err.(*client.APIError)
		if ok && apiErr.StatusCode == 404 {
			// Token was deleted outside of Terraform
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading token",
			fmt.Sprintf("Unable to read token: %v", err),
		)
		return
	}

	// Update state
	state.ID = types.StringValue(token.ID)
	state.Name = types.StringValue(token.Name)
	state.Description = types.StringValue(token.Description)
	state.PermissionSystemID = types.StringValue(token.PermissionSystemID)
	state.ServiceAccountID = types.StringValue(token.ServiceAccountID)
	state.CreatedAt = types.StringValue(token.CreatedAt)
	state.Creator = types.StringValue(token.Creator)

	// The secret isn't returned during Read operations, but we want to keep it

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the token (not supported for tokens)
func (r *TokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Tokens cannot be updated, only created and deleted
	resp.Diagnostics.AddError(
		"Error updating token",
		"Tokens cannot be updated. Delete and create a new token instead.",
	)
}

func (r *TokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var state TokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteToken(
		state.PermissionSystemID.ValueString(),
		state.ServiceAccountID.ValueString(),
		state.ID.ValueString(),
	)
	if err != nil {
		apiErr, ok := err.(*client.APIError)
		if ok && apiErr.StatusCode == 404 {
			// Token already deleted, ignore
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting token",
			fmt.Sprintf("Unable to delete token: %v", err),
		)
		return
	}
}
