package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"terraform-provider-authzed/internal/client"
	"terraform-provider-authzed/internal/models"
	"terraform-provider-authzed/internal/provider/deletelanes"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &serviceAccountResource{}
	_ resource.ResourceWithImportState = &serviceAccountResource{}
)

func NewServiceAccountResource() resource.Resource {
	return &serviceAccountResource{}
}

type serviceAccountResource struct {
	client      *client.CloudClient
	deleteLanes *deletelanes.DeleteLanes
}

type serviceAccountResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	Description         types.String   `tfsdk:"description"`
	PermissionsSystemID types.String   `tfsdk:"permission_system_id"`
	CreatedAt           types.String   `tfsdk:"created_at"`
	Creator             types.String   `tfsdk:"creator"`
	UpdatedAt           types.String   `tfsdk:"updated_at"`
	Updater             types.String   `tfsdk:"updater"`
	ETag                types.String   `tfsdk:"etag"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

func (r *serviceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *serviceAccountResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a service account",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"creator": schema.StringAttribute{
				Computed:    true,
				Description: "User who created the service account",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the service account was last updated",
			},
			"updater": schema.StringAttribute{
				Computed:    true,
				Description: "User who last updated the service account",
			},
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "Version identifier used to prevent conflicts from concurrent updates, ensuring safe resource modifications",
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *serviceAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*CloudProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *CloudProviderData, got: %T", req.ProviderData),
		)
		return
	}

	r.client = providerData.Client
	r.deleteLanes = providerData.DeleteLanes
}

func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var data serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get context with create timeout (default 10 minutes for service accounts)
	createTimeout, diags := data.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	// Create service account
	serviceAccount := &models.ServiceAccount{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		PermissionsSystemID: data.PermissionsSystemID.ValueString(),
	}

	createdServiceAccountWithETag, err := r.client.CreateServiceAccount(createCtx, serviceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service account, got error: %s", err))
		return
	}

	// Check if CREATE returned an ETag
	if createdServiceAccountWithETag.ETag != "" {
		// Use the ETag from CREATE operation
		data.ID = types.StringValue(createdServiceAccountWithETag.ServiceAccount.ID)
		data.CreatedAt = types.StringValue(createdServiceAccountWithETag.ServiceAccount.CreatedAt)
		data.Creator = types.StringValue(createdServiceAccountWithETag.ServiceAccount.Creator)
		if createdServiceAccountWithETag.ServiceAccount.UpdatedAt == "" {
			data.UpdatedAt = types.StringNull()
		} else {
			data.UpdatedAt = types.StringValue(createdServiceAccountWithETag.ServiceAccount.UpdatedAt)
		}
		if createdServiceAccountWithETag.ServiceAccount.Updater == "" {
			data.Updater = types.StringNull()
		} else {
			data.Updater = types.StringValue(createdServiceAccountWithETag.ServiceAccount.Updater)
		}
		data.ETag = types.StringValue(createdServiceAccountWithETag.ETag)

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// If CREATE didn't return ETag, set the basic fields and continue to GET retry logic
	data.ID = types.StringValue(createdServiceAccountWithETag.ServiceAccount.ID)
	data.CreatedAt = types.StringValue(createdServiceAccountWithETag.ServiceAccount.CreatedAt)
	data.Creator = types.StringValue(createdServiceAccountWithETag.ServiceAccount.Creator)
	if createdServiceAccountWithETag.ServiceAccount.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(createdServiceAccountWithETag.ServiceAccount.UpdatedAt)
	}
	if createdServiceAccountWithETag.ServiceAccount.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(createdServiceAccountWithETag.ServiceAccount.Updater)
	}
	// Post-create GET to stabilize metadata and ETag - retry until ETag is non-empty
	var fresh *client.ServiceAccountWithETag
	var gerr error
	for i := 0; i < 5; i++ {
		fresh, gerr = r.client.GetServiceAccount(createCtx, data.PermissionsSystemID.ValueString(), data.ID.ValueString())
		if gerr == nil && fresh.ETag != "" {
			break
		}
		if i < 4 { // Don't sleep on the last iteration
			time.Sleep(200 * time.Millisecond)
		}
	}

	if gerr != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account after creation for stabilization, got error: %s", gerr))
		return
	}

	if fresh.ETag == "" {
		// According to the OpenAPI spec, all service account operations should return ETags
		// If we don't get one, this indicates an API issue that needs to be resolved
		if fresh != nil && fresh.ServiceAccount != nil {
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Service account created successfully (ID: %s) but the API did not return the required ETag header. This violates the OpenAPI specification and indicates an API-side issue that needs to be resolved.", fresh.ServiceAccount.ID))
		} else {
			resp.Diagnostics.AddError("API Error", "Unable to obtain required ETag header from service account API response. This violates the OpenAPI specification and indicates an API-side issue that needs to be resolved.")
		}
		return
	}

	// Use stabilized data from the GET operation
	data.Name = types.StringValue(fresh.ServiceAccount.Name)
	data.Description = types.StringValue(fresh.ServiceAccount.Description)
	data.CreatedAt = types.StringValue(fresh.ServiceAccount.CreatedAt)
	data.Creator = types.StringValue(fresh.ServiceAccount.Creator)
	if fresh.ServiceAccount.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(fresh.ServiceAccount.UpdatedAt)
	}
	if fresh.ServiceAccount.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(fresh.ServiceAccount.Updater)
	}
	data.ETag = types.StringValue(fresh.ETag)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccountWithETag, err := r.client.GetServiceAccount(ctx, data.PermissionsSystemID.ValueString(), data.ID.ValueString())
	if err != nil {
		// Check if the resource was not found (404 error)
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "not found") {
			// Resource no longer exists, remove it from state
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account, got error: %s", err))
		return
	}

	// Map response to model
	data.Name = types.StringValue(serviceAccountWithETag.ServiceAccount.Name)
	data.Description = types.StringValue(serviceAccountWithETag.ServiceAccount.Description)
	data.CreatedAt = types.StringValue(serviceAccountWithETag.ServiceAccount.CreatedAt)
	data.Creator = types.StringValue(serviceAccountWithETag.ServiceAccount.Creator)
	if serviceAccountWithETag.ServiceAccount.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(serviceAccountWithETag.ServiceAccount.UpdatedAt)
	}
	if serviceAccountWithETag.ServiceAccount.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(serviceAccountWithETag.ServiceAccount.Updater)
	}
	data.ETag = types.StringValue(serviceAccountWithETag.ETag)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data serviceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current state to get the ETag
	var state serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create service account with updated data - use state values for immutable fields
	serviceAccount := &models.ServiceAccount{
		ID:                  state.ID.ValueString(), // Use state for immutable ID
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		PermissionsSystemID: data.PermissionsSystemID.ValueString(),
		CreatedAt:           state.CreatedAt.ValueString(), // Preserve immutable CreatedAt
	}

	// Handle Creator field - it might be null in state
	if !state.Creator.IsNull() {
		serviceAccount.Creator = state.Creator.ValueString()
	}

	// Use the ETag from state for optimistic concurrency control
	updateResult := r.client.UpdateServiceAccount(ctx, serviceAccount, state.ETag.ValueString())
	resp.Diagnostics.Append(updateResult.Diagnostics...)

	if updateResult.ServiceAccount == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to update service account")
		return
	}

	updatedServiceAccountWithETag := updateResult.ServiceAccount

	// Update resource data with the response - preserve immutable fields from state
	data.ID = state.ID
	data.CreatedAt = state.CreatedAt
	data.Creator = state.Creator
	if updatedServiceAccountWithETag.ServiceAccount.UpdatedAt == "" {
		data.UpdatedAt = types.StringNull()
	} else {
		data.UpdatedAt = types.StringValue(updatedServiceAccountWithETag.ServiceAccount.UpdatedAt)
	}
	if updatedServiceAccountWithETag.ServiceAccount.Updater == "" {
		data.Updater = types.StringNull()
	} else {
		data.Updater = types.StringValue(updatedServiceAccountWithETag.ServiceAccount.Updater)
	}
	data.ETag = types.StringValue(updatedServiceAccountWithETag.ETag)

	// Update mutable fields from the response
	data.Name = types.StringValue(updatedServiceAccountWithETag.ServiceAccount.Name)
	data.Description = types.StringValue(updatedServiceAccountWithETag.ServiceAccount.Description)
	data.PermissionsSystemID = types.StringValue(updatedServiceAccountWithETag.ServiceAccount.PermissionsSystemID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var data serviceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get context with delete timeout (default 10 minutes for deletes)
	deleteTimeout, diags := data.Timeouts.Delete(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	permissionSystemID := data.PermissionsSystemID.ValueString()

	// Serialize delete per Permission System with 409 retry
	err := r.deleteLanes.WithDelete(deleteCtx, permissionSystemID, func(ctx context.Context) error {
		return deletelanes.Retry409Delete(ctx, func(ctx context.Context) error {
			return r.client.DeleteServiceAccount(permissionSystemID, data.ID.ValueString())
		})
	})

	if err != nil {
		resp.Diagnostics.AddError("Error deleting service account", fmt.Sprintf("Unable to delete service account: %v", err))
		return
	}
}

// ImportState handles importing an existing service account into Terraform state
func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: permission_system_id:service_account_id
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import id in format 'permission_system_id:service_account_id', got: %s", req.ID),
		)
		return
	}

	permissionSystemID := idParts[0]
	serviceAccountID := idParts[1]

	// Set the main identifiers
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_system_id"), permissionSystemID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), serviceAccountID)...)

	// Terraform automatically calls Read to fetch the rest of the attributes
}
