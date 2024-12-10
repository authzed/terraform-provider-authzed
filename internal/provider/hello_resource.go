package provider

import (
	"context"
	"fmt"

	"terraform-provider-platform-api/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type HelloResource struct {
	client *client.APIClient
}

type HelloResourceModel struct {
	Name     types.String `tfsdk:"name"`
	Response types.String `tfsdk:"response"`
	Id       types.String `tfsdk:"id"`
}

func NewHelloResource() resource.Resource {
	return &HelloResource{
		client: nil,
	}
}

func (r *HelloResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hello"
}

func (r *HelloResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Says hello to the provided name",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name to say hello to",
			},
			"response": schema.StringAttribute{
				Computed:    true,
				Description: "The hello response from the API",
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *HelloResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data HelloResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	apiReq := client.V20241017SayHelloRequest{
		Name: &name,
	}

	result, _, err := r.client.DefaultAPI.SayHello(ctx).V20241017SayHelloRequest(apiReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create hello, got error: %s", err))
		return
	}

	data.Response = types.StringValue(result.GetMessage())
	data.Id = types.StringValue(data.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HelloResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HelloResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	apiReq := client.V20241017SayHelloRequest{
		Name: &name,
	}

	result, _, err := r.client.DefaultAPI.SayHello(ctx).V20241017SayHelloRequest(apiReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read hello, got error: %s", err))
		return
	}

	data.Response = types.StringValue(result.GetMessage())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HelloResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data HelloResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	apiReq := client.V20241017SayHelloRequest{
		Name: &name,
	}

	result, _, err := r.client.DefaultAPI.SayHello(ctx).V20241017SayHelloRequest(apiReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update hello, got error: %s", err))
		return
	}

	data.Response = types.StringValue(result.GetMessage())
	data.Id = types.StringValue(data.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HelloResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Nothing to do for delete since the hello endpoint is stateless
}

func (r *HelloResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.APIClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}
