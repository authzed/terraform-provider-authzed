package provider

import (
	"context"
	"fmt"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &serviceAccountResource{}

func NewServiceAccountResource() resource.Resource {
	return &serviceAccountResource{}
}

type serviceAccountResource struct {
	client *client.CloudClient
}

type serviceAccountResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	PermissionsSystemID types.String `tfsdk:"permission_system_id"`
	CreatedAt           types.String `tfsdk:"created_at"`
	Creator             types.String `tfsdk:"creator"`
}

func (r *serviceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *serviceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a service account",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this resource",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the service account",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the service account",
			},
			"permission_system_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the permission system this service account belongs to",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the service account was created",
			},
			"creator": schema.StringAttribute{
				Computed:    true,
				Description: "User who created the service account",
			},
		},
	}
}

func (r *serviceAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// More detailed debug output
	fmt.Printf("DEBUG TERRAFORM: Creating service account with Permissions System ID: %s\n", data.PermissionsSystemID.ValueString())
	fmt.Printf("DEBUG TERRAFORM: Service account name: %s\n", data.Name.ValueString())
	fmt.Printf("DEBUG TERRAFORM: Service account description: %s\n", data.Description.ValueString())

	// Create service account
	serviceAccount := &models.ServiceAccount{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		PermissionsSystemID: data.PermissionsSystemID.ValueString(),
	}

	// Try force setting the permission system ID for request building
	// This is for API path construction only, not the JSON body
	fmt.Printf("DEBUG: Force setting permissionsSystemID to %s\n", data.PermissionsSystemID.ValueString())

	createdServiceAccount, err := r.client.CreateServiceAccount(serviceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service account, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdServiceAccount.ID)
	data.CreatedAt = types.StringValue(createdServiceAccount.CreatedAt)
	data.Creator = types.StringValue(createdServiceAccount.Creator)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccount, err := r.client.GetServiceAccount(data.PermissionsSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account, got error: %s", err))
		return
	}

	// Map response to model
	data.Name = types.StringValue(serviceAccount.Name)
	data.Description = types.StringValue(serviceAccount.Description)
	data.CreatedAt = types.StringValue(serviceAccount.CreatedAt)
	data.Creator = types.StringValue(serviceAccount.Creator)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The API  doesn't support direct updates, so we need to delete and recreate
	err := r.client.DeleteServiceAccount(data.PermissionsSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service account during update, got error: %s", err))
		return
	}

	// Create service account with updated data
	serviceAccount := &models.ServiceAccount{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		PermissionsSystemID: data.PermissionsSystemID.ValueString(),
	}

	createdServiceAccount, err := r.client.CreateServiceAccount(serviceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to recreate service account during update, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdServiceAccount.ID)
	data.CreatedAt = types.StringValue(createdServiceAccount.CreatedAt)
	data.Creator = types.StringValue(createdServiceAccount.Creator)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteServiceAccount(data.PermissionsSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service account, got error: %s", err))
		return
	}
}
