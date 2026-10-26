package enablement

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
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
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	serviceenablementUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceenablement/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &serviceEnablementResource{}
	_ resource.ResourceWithConfigure   = &serviceEnablementResource{}
	_ resource.ResourceWithImportState = &serviceEnablementResource{}
	_ resource.ResourceWithModifyPlan  = &serviceEnablementResource{}
)

const (
	deletionWarning = "Services enabled via this resource will **not** be disabled by Terraform during a `terraform destroy`. They will just be thrown out of the Terraform state and not disabled on API side."
)

// IMPORTANT: Only services should be allowed here which match *all* of the following criteria for
// PROD env for *all* regions (eu01, eu02):
// - Enablement: REQUEST
// - Lifecycle: FLEX
// - enable: EVENT
// - disable: EVENT
// When in doubt whether a service can be added here please get in touch with the STACKIT DevTools Core
// Platform integration team and/or the STACKIT service enablement team.
var allowedServiceIdsByRegion = map[string][]string{
	"eu01": {
		"cloud.stackit.ske",
	},
	"eu02": {
		"cloud.stackit.ske",
	},
}

// allowedServiceIds returns all services which are part of the allowedServiceIdsByRegion map
func allowedServiceIds() []string {
	var serviceIds []string

	for _, v := range allowedServiceIdsByRegion {
		serviceIds = append(serviceIds, v...)
	}

	// remove duplicates
	return slices.Compact(serviceIds)
}

type ResourceModel struct {
	Id        types.String `tfsdk:"id"` // needed by TF
	ProjectId types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	ServiceId types.String `tfsdk:"service_id"`
}

// NewServiceEnablementResource is a helper function to simplify the provider implementation.
func NewServiceEnablementResource() resource.Resource {
	return &serviceEnablementResource{}
}

// serviceEnablementResource is the resource implementation.
type serviceEnablementResource struct {
	client       serviceenablement.DefaultAPI
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *serviceEnablementResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_enablement"
}

// Configure adds the provider configured client to the resource.
func (r *serviceEnablementResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.providerData = providerData
	r.client = clients.ServiceEnablementV2Client

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_service_enablement", core.Resource)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "service enablement client configured")
}

// Schema defines the schema for the resource.
func (r *serviceEnablementResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Service enablement resource schema."
	resp.Schema = schema.Schema{
		Description:         features.AddBetaDescription(description, core.Resource),
		MarkdownDescription: features.AddBetaDescription(fmt.Sprintf("%s\n\n ~> %s", description, deletionWarning), core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`service_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			// IMPORTANT: see note above at definition of 'allowedServiceIdsByRegion' variable
			"service_id": schema.StringAttribute{
				Description: fmt.Sprintf("The identifier of the STACKIT service. %s", utils.FormatPossibleValues(allowedServiceIds()...)),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(allowedServiceIds()...),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *serviceEnablementResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	// add a warning that destroying the service enablement resource doesn't mean the service will be disabled on API side
	if req.Plan.Raw.IsNull() && !req.State.Raw.IsNull() {
		core.LogAndAddWarning(ctx, &resp.Diagnostics, "stackit_service_enablement", "Note that destroying this resource will only remove it from the Terraform state. It will not disable the service on API side.")
		return
	}

	var configModel ResourceModel
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// post-modification validation
	errorDiag := serviceenablementUtils.ValidateServiceIdAndRegionCombination(allowedServiceIdsByRegion, planModel.ServiceId.ValueString(), planModel.Region.ValueString())
	if errorDiag != nil {
		resp.Diagnostics.Append(errorDiag)
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *serviceEnablementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	serviceId := model.ServiceId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", projectId)
	ctx = tflog.SetField(ctx, "service_id", serviceId)

	ctx = core.InitProviderContext(ctx)

	// enable service
	err := r.client.EnableServiceRegional(ctx, region, projectId, serviceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service enablement", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"service_id": serviceId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := wait.EnableServiceWaitHandler(ctx, r.client, region, projectId, serviceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service enablement", fmt.Sprintf("Waiting for service enablement: %v", err))
	}

	// Map response body to schema
	err = mapFields(&model, waitResp, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service enablement", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "service enablement created")
}

// Read refreshes the Terraform state with the latest data.
func (r *serviceEnablementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	serviceId := model.ServiceId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", projectId)
	ctx = tflog.SetField(ctx, "service_id", serviceId)

	ctx = core.InitProviderContext(ctx)

	serviceEnablementResp, err := r.client.GetServiceStatusRegional(ctx, region, projectId, serviceId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service enablement", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	enabled, err := serviceenablementUtils.CheckServiceEnabled(serviceEnablementResp)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service enablement", fmt.Sprintf("Processing API payload: %v", err))
	}

	// in case the service got disabled in the meantime - remove resource from state so it gets recreated (which means the service gets enabled again)
	if !enabled {
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(&model, serviceEnablementResp, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service enablement", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "service enablement read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *serviceEnablementResource) Update(ctx context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// update shouldn't be possible
	tflog.Info(ctx, "service enablement updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *serviceEnablementResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Service enablement as of now doesn't allow for consistent disablement of services.
	// Therefore, we leave the services enabled and show a warning to make users aware.
	core.LogAndAddWarning(ctx, &resp.Diagnostics, "Service not disabled on API side", deletionWarning)

	tflog.Info(ctx, "service enablement deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,region,service
func (r *serviceEnablementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing service enablement",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[service_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"service_id": idParts[2],
	})

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "service enablement state imported")
}

func mapFields(model *ResourceModel, _ *serviceenablement.ServiceStatus, region string) error {
	if model == nil {
		return fmt.Errorf("nil model")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, model.ServiceId.ValueString())
	model.Region = types.StringValue(region)

	return nil
}
