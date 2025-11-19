package httpcheck

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	observabilityUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &observabilityHttpCheckResource{}
	_ resource.ResourceWithConfigure   = &observabilityHttpCheckResource{}
	_ resource.ResourceWithImportState = &observabilityHttpCheckResource{}
)

type Model struct {
	Id          types.String `tfsdk:"id"`
	ProjectId   types.String `tfsdk:"project_id"`
	InstanceId  types.String `tfsdk:"instance_id"`
	HttpCheckId types.String `tfsdk:"http_check_id"`
	Url         types.String `tfsdk:"url"`
}

var descriptions = map[string]string{
	"id":            "Terraform resource ID in format `project_id,instance_id,http_check_id`.",
	"project_id":    "STACKIT project ID.",
	"instance_id":   "STACKIT Observability instance ID.",
	"http_check_id": "Unique ID of the HTTP-check.",
	"url":           "The URL to check, e.g. https://www.stackit.de. Must start with `http://` or `https://`.",
}

// NewHttpCheckResource is a helper function to simplify the provider implementation.
func NewHttpCheckResource() resource.Resource {
	return &observabilityHttpCheckResource{}
}

// observabilityHttpCheckResource is the resource implementation.
type observabilityHttpCheckResource struct {
	client *observability.APIClient
}

// Metadata returns the resource type name.
func (r *observabilityHttpCheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_http_check"
}

// Configure adds the provider configured client to the resource.
func (r *observabilityHttpCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_observability_http_check", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	r.client = observabilityUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability client configured")
}

// Schema defines the schema for the resource.
func (r *observabilityHttpCheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource for managing HTTP checks in STACKIT Observability. It ships Telegraf HTTP response metrics to the observability instance, as documented here: `https://docs.influxdata.com/telegraf/v1/input-plugins/http_response/`",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
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
				Description: descriptions["instance_id"],
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
				Description: descriptions["http_check_id"],
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: descriptions["url"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(5),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^https?://`),
						"The URL must start with http:// or https://.",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *observabilityHttpCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	httpCheckUrl := model.Url.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "http_check_url", httpCheckUrl)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	createHttpCheck, err := r.client.CreateHttpCheck(
		ctx,
		instanceId,
		projectId,
	).CreateHttpCheckPayload(observability.CreateHttpCheckPayload{Url: &httpCheckUrl}).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating http-check", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Log success message from API
	if createHttpCheck.Message != nil {
		tflog.Info(ctx, fmt.Sprintf("Create http-check response message: %s", *createHttpCheck.Message))
	}

	if createHttpCheck.HttpChecks == nil || len(*createHttpCheck.HttpChecks) == 0 {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating http-check", "Response is empty")
		return
	}

	for _, httpCheck := range *createHttpCheck.HttpChecks {
		if httpCheck.Url != nil && *httpCheck.Url == httpCheckUrl {
			if err := mapFields(ctx, &httpCheck, &model); err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating http-check", "Unable to map http-check model")
				return
			}
			break
		}
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "http-check created")
}

// Read refreshes the Terraform state with the latest data.
func (r *observabilityHttpCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	httpCheckUrl := model.Url.ValueString()
	httpCheckId := model.HttpCheckId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "http_check_url", httpCheckUrl)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "http_check_id", httpCheckId)

	listHttpCheck, err := r.client.ListHttpChecks(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing http-checks", fmt.Sprintf("Listing API payload: %v", err))
		return
	}

	if listHttpCheck.HttpChecks == nil || len(*listHttpCheck.HttpChecks) == 0 {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing http-checks", "Response is empty")
		return
	}

	for _, httpCheck := range *listHttpCheck.HttpChecks {
		// we also check if http-check-ids are matching to support import functionality
		if httpCheck.Url != nil && *httpCheck.Url == httpCheckUrl || (httpCheck.Id != nil && *httpCheck.Id == httpCheckId) {
			if err := mapFields(ctx, &httpCheck, &model); err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading http-check", "Unable to map http-check model")
				return
			}
			break
		}
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "http-check read")
}

// Update attempts to update the resource. In this case, http-checks cannot be updated.
// The Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (r *observabilityHttpCheckResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating http-check", "Observability http-checks can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *observabilityHttpCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	httpCheckId := model.HttpCheckId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "http_check_id", httpCheckId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	_, err := r.client.DeleteHttpCheck(ctx, instanceId, projectId, httpCheckId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting http-check", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	tflog.Info(ctx, "http-check deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,http_check_id
func (r *observabilityHttpCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing http-check",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id],[http_check_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("http_check_id"), idParts[2])...)
	tflog.Info(ctx, "Observability http-check state imported")
}

// mapFields maps httpCheck response to the model.
func mapFields(_ context.Context, httpCheck *observability.HttpCheckChildResponse, model *Model) error {
	if httpCheck == nil {
		return fmt.Errorf("http-check input is nil")
	}

	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if httpCheck.Id == nil {
		return fmt.Errorf("http-check id is nil")
	}

	if httpCheck.Url == nil {
		return fmt.Errorf("http-check url is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.InstanceId.ValueString(), *httpCheck.Id)
	model.HttpCheckId = types.StringValue(*httpCheck.Id)
	model.Url = types.StringValue(*httpCheck.Url)

	return nil
}
