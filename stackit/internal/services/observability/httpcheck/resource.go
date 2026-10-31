package httpcheck

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	observabilitySdk "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &httpCheckResource{}
	_ resource.ResourceWithConfigure   = &httpCheckResource{}
	_ resource.ResourceWithImportState = &httpCheckResource{}
)

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	InstanceId  types.String `tfsdk:"instance_id"`
	HttpCheckId types.String `tfsdk:"http_check_id"`
	Url         types.String `tfsdk:"url"`
}

// NewHttpCheckResource is a helper function to simplify the provider implementation.
func NewHttpCheckResource() resource.Resource {
	return &httpCheckResource{}
}

// httpCheckResource is the resource implementation.
type httpCheckResource struct {
	client observabilitySdk.DefaultAPI
}

// Metadata returns the resource type name.
func (r *httpCheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_http_check"
}

// Configure adds the provider configured client to the resource.
func (r *httpCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !core.ValidateServiceRegion(ctx, &providerData, "Observability", &resp.Diagnostics) {
		return
	}

	r.client = clients.ObservabilityV1Client

	tflog.Info(ctx, "Observability HTTP check client configured")
}

// Schema defines the schema for the resource.
func (r *httpCheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Observability HTTP check resource schema. HTTP checks are unique per URL within an instance: creating a check for a URL that already exists returns the existing check. " +
			"Two resources with the same `url` therefore manage the same check, and destroying one of them deletes it. " +
			"Must have a `region` specified in the provider configuration. This service is only available when the provider is configured with `default_region = \"eu01\"`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`,`http_check_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the HTTP check is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "The Observability instance ID to which the HTTP check is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"http_check_id": schema.StringAttribute{
				Description: "The HTTP check ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Description: "The URL to check.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 400),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *httpCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	createResp, err := r.client.CreateHttpCheck(ctx, instanceId, projectId).CreateHttpCheckPayload(toCreatePayload(&model)).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating HTTP check", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	httpCheck, err := createdHttpCheck(createResp, model.Url.ValueString())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating HTTP check", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":    projectId,
		"instance_id":   instanceId,
		"http_check_id": httpCheck.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	err = mapFields(httpCheck, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating HTTP check", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability HTTP check created")
}

// Read refreshes the Terraform state with the latest data.
func (r *httpCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	httpCheckId := model.HttpCheckId.ValueString()
	if httpCheckId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "http_check_id", httpCheckId)

	// The API has no endpoint to get a single HTTP check
	listResp, err := r.client.ListHttpChecks(ctx, instanceId, projectId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading HTTP check", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if listResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading HTTP check", "Empty response")
		return
	}
	httpCheck := findHttpCheck(listResp.HttpChecks, httpCheckId)
	if httpCheck == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapFields(httpCheck, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading HTTP check", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability HTTP check read")
}

// Update is not supported, all attributes require a replacement.
func (r *httpCheckResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating HTTP check", "HTTP check can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *httpCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	httpCheckId := model.HttpCheckId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "http_check_id", httpCheckId)

	_, err := r.client.DeleteHttpCheck(ctx, instanceId, projectId, httpCheckId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting HTTP check", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Observability HTTP check deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,http_check_id
func (r *httpCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing HTTP check",
			fmt.Sprintf("Expected import identifier with format [project_id],[instance_id],[http_check_id], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":    idParts[0],
		"instance_id":   idParts[1],
		"http_check_id": idParts[2],
	})
	tflog.Info(ctx, "Observability HTTP check state imported")
}

func toCreatePayload(model *Model) observabilitySdk.CreateHttpCheckPayload {
	return observabilitySdk.CreateHttpCheckPayload{
		Url: model.Url.ValueString(),
	}
}

// createdHttpCheck returns the HTTP check created by a CreateHttpCheck call.
// The response contains the created check, and the list of all HTTP checks of the instance as a fallback.
// The API does not create duplicates for the same URL, so the fallback is expected to find at most one match.
func createdHttpCheck(createResp *observabilitySdk.HttpCheckResponse, url string) (*observabilitySdk.HttpCheckChildResponse, error) {
	if createResp == nil {
		return nil, fmt.Errorf("response is nil")
	}
	if createResp.HttpCheck != nil && createResp.HttpCheck.Id != "" {
		return createResp.HttpCheck, nil
	}

	var match *observabilitySdk.HttpCheckChildResponse
	for i := range createResp.HttpChecks {
		if createResp.HttpChecks[i].Url != url {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("created HTTP check not in response and more than one HTTP check with URL %q found", url)
		}
		match = &createResp.HttpChecks[i]
	}
	if match == nil {
		return nil, fmt.Errorf("created HTTP check with URL %q not found in response", url)
	}
	return match, nil
}

func findHttpCheck(httpChecks []observabilitySdk.HttpCheckChildResponse, id string) *observabilitySdk.HttpCheckChildResponse {
	for i := range httpChecks {
		if httpChecks[i].Id == id {
			return &httpChecks[i]
		}
	}
	return nil
}

func mapFields(httpCheck *observabilitySdk.HttpCheckChildResponse, model *Model) error {
	if httpCheck == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	if httpCheck.Id == "" {
		return fmt.Errorf("HTTP check id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.InstanceId.ValueString(), httpCheck.Id)
	model.HttpCheckId = types.StringValue(httpCheck.Id)
	model.Url = types.StringValue(httpCheck.Url)
	return nil
}
