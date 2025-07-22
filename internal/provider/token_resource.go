package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &TokenResource{}
	_ resource.ResourceWithImportState = &TokenResource{}
)

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
	Hash                types.String `tfsdk:"hash"`
	PlainText           types.String `tfsdk:"plain_text"`
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
			"hash": schema.StringAttribute{
				Description: "The SHA256 hash of the token",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"plain_text": schema.StringAttribute{
				Description: "One-time token value (returned only at creation).",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
	token := &models.TokenRequest{
		Name:                plan.Name.ValueString(),
		Description:         plan.Description.ValueString(),
		PermissionsSystemID: plan.PermissionsSystemID.ValueString(),
		ServiceAccountID:    plan.ServiceAccountID.ValueString(),
		ReturnPlainText:     true, // Always request plain text during creation
	}

	createdTokenWithETag, err := r.client.CreateToken(token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating token",
			fmt.Sprintf("Unable to create token: %v", err),
		)
		return
	}

	// Set all fields in the plan
	plan.ID = types.StringValue(createdTokenWithETag.Token.ID)
	plan.Name = types.StringValue(createdTokenWithETag.Token.Name)
	plan.Description = types.StringValue(createdTokenWithETag.Token.Description)
	plan.PermissionsSystemID = types.StringValue(createdTokenWithETag.Token.PermissionsSystemID)
	plan.ServiceAccountID = types.StringValue(createdTokenWithETag.Token.ServiceAccountID)
	plan.CreatedAt = types.StringValue(createdTokenWithETag.Token.CreatedAt)
	plan.Creator = types.StringValue(createdTokenWithETag.Token.Creator)

	// Set the one-time plain text value and hash during creation
	if createdTokenWithETag.Token.Secret != "" {
		plan.PlainText = types.StringValue(createdTokenWithETag.Token.Secret)
	}
	if createdTokenWithETag.Token.Hash != "" {
		plan.Hash = types.StringValue(createdTokenWithETag.Token.Hash)
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

	// Update state with metadata only
	state.ID = types.StringValue(tokenWithETag.Token.ID)
	state.Name = types.StringValue(tokenWithETag.Token.Name)
	state.Description = types.StringValue(tokenWithETag.Token.Description)
	state.PermissionsSystemID = types.StringValue(tokenWithETag.Token.PermissionsSystemID)
	state.ServiceAccountID = types.StringValue(tokenWithETag.Token.ServiceAccountID)
	state.CreatedAt = types.StringValue(tokenWithETag.Token.CreatedAt)
	state.Creator = types.StringValue(tokenWithETag.Token.Creator)

	// Only set the hash, never reset plain_text
	if tokenWithETag.Token.Hash != "" {
		state.Hash = types.StringValue(tokenWithETag.Token.Hash)
	}

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
	token := &models.TokenRequest{
		ID:                  plan.ID.ValueString(),
		Name:                plan.Name.ValueString(),
		Description:         plan.Description.ValueString(),
		PermissionsSystemID: plan.PermissionsSystemID.ValueString(),
		ServiceAccountID:    plan.ServiceAccountID.ValueString(),
	}

	// Update token without ETag (tokens don't support optimistic concurrency control)
	updatedTokenWithETag, err := r.client.UpdateToken(token, "")
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

	// Preserve the token value and hash from state since they won't be returned in updates
	plan.PlainText = state.PlainText
	plan.Hash = state.Hash

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

// ImportState handles importing an existing token into Terraform state
func (r *TokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import id in format 'permission_system_id:service_account_id:token_id', got: %s", req.ID),
		)
		return
	}

	permissionSystemID := idParts[0]
	serviceAccountID := idParts[1]
	tokenID := idParts[2]

	// Set the main identifiers
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_system_id"), permissionSystemID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_account_id"), serviceAccountID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), tokenID)...)

	// Terraform automatically calls Read to fetch the rest of the attributes
}
