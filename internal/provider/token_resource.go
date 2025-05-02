package provider

import (
	"context"
	"fmt"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &TokenResource{}

func NewTokenResource() resource.Resource {
	return &TokenResource{}
}

type TokenResource struct {
	client *client.CloudClient
}

type TokenResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	PermissionsSystemID types.String `tfsdk:"permission_system_id"`
	ServiceAccountID    types.String `tfsdk:"service_account_id"`
	CreatedAt           types.String `tfsdk:"created_at"`
	Creator             types.String `tfsdk:"creator"`
	Secret              types.String `tfsdk:"secret"`
	ETag                types.String `tfsdk:"etag"` // Added for ETag support
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
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "Version identifier used to prevent conflicts from concurrent updates, ensuring safe resource modifications",
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
		Name:                plan.Name.ValueString(),
		Description:         plan.Description.ValueString(),
		PermissionsSystemID: plan.PermissionsSystemID.ValueString(),
		ServiceAccountID:    plan.ServiceAccountID.ValueString(),
	}

	tflog.Info(ctx, "Creating token", map[string]any{
		"name":                 token.Name,
		"permission_system_id": token.PermissionsSystemID,
		"service_account_id":   token.ServiceAccountID,
	})

	createdTokenWithETag, err := r.client.CreateToken(token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating token",
			fmt.Sprintf("Unable to create token: %v", err),
		)
		return
	}

	plan.ID = types.StringValue(createdTokenWithETag.Token.ID)
	plan.Name = types.StringValue(createdTokenWithETag.Token.Name)
	plan.Description = types.StringValue(createdTokenWithETag.Token.Description)
	plan.PermissionsSystemID = types.StringValue(createdTokenWithETag.Token.PermissionsSystemID)
	plan.ServiceAccountID = types.StringValue(createdTokenWithETag.Token.ServiceAccountID)
	plan.CreatedAt = types.StringValue(createdTokenWithETag.Token.CreatedAt)
	plan.Creator = types.StringValue(createdTokenWithETag.Token.Creator)
	plan.ETag = types.StringValue(createdTokenWithETag.ETag)

	// Set the secret (only available during creation)
	if createdTokenWithETag.Token.Hash != "" {
		plan.Secret = types.StringValue(createdTokenWithETag.Token.Hash)
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

	tokenWithETag, err := r.client.GetToken(
		state.PermissionsSystemID.ValueString(),
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
	state.ID = types.StringValue(tokenWithETag.Token.ID)
	state.Name = types.StringValue(tokenWithETag.Token.Name)
	state.Description = types.StringValue(tokenWithETag.Token.Description)
	state.PermissionsSystemID = types.StringValue(tokenWithETag.Token.PermissionsSystemID)
	state.ServiceAccountID = types.StringValue(tokenWithETag.Token.ServiceAccountID)
	state.CreatedAt = types.StringValue(tokenWithETag.Token.CreatedAt)
	state.Creator = types.StringValue(tokenWithETag.Token.Creator)
	state.ETag = types.StringValue(tokenWithETag.ETag)

	// The secret isn't returned during Read operations, but we want to keep it

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the token
func (r *TokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get planned changes
	var plan TokenResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state to retrieve ETag
	var state TokenResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create token with updated data
	token := &models.Token{
		ID:                  plan.ID.ValueString(),
		Name:                plan.Name.ValueString(),
		Description:         plan.Description.ValueString(),
		PermissionsSystemID: plan.PermissionsSystemID.ValueString(),
		ServiceAccountID:    plan.ServiceAccountID.ValueString(),
	}

	// Use the ETag from state for optimistic concurrency control
	updatedTokenWithETag, err := r.client.UpdateToken(token, state.ETag.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating token",
			fmt.Sprintf("Unable to update token, got error: %s", err),
		)
		return
	}

	// Update resource data with the response
	plan.CreatedAt = types.StringValue(updatedTokenWithETag.Token.CreatedAt)
	plan.Creator = types.StringValue(updatedTokenWithETag.Token.Creator)
	plan.ETag = types.StringValue(updatedTokenWithETag.ETag)

	// Preserve the secret value from state since it won't be returned in updates
	plan.Secret = state.Secret

	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
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
		state.PermissionsSystemID.ValueString(),
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
