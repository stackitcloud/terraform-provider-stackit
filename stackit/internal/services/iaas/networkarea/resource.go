package networkarea

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	resourcemanager "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v0api"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	resourcemanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkAreaResource{}
	_ resource.ResourceWithConfigure   = &networkAreaResource{}
	_ resource.ResourceWithImportState = &networkAreaResource{}
	_ resource.ResourceWithModifyPlan  = &networkAreaResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	OrganizationId types.String `tfsdk:"organization_id"`
	NetworkAreaId  types.String `tfsdk:"network_area_id"`
	Name           types.String `tfsdk:"name"`
	ProjectCount   types.Int64  `tfsdk:"project_count"`
	Labels         types.Map    `tfsdk:"labels"`
}

// NewNetworkAreaResource is a helper function to simplify the provider implementation.
func NewNetworkAreaResource() resource.Resource {
	return &networkAreaResource{}
}

// networkResource is the resource implementation.
type networkAreaResource struct {
	client                *iaas.APIClient
	resourceManagerClient *resourcemanager.APIClient
}

func (r *networkAreaResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Metadata returns the resource type name.
func (r *networkAreaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_area"
}

// Configure adds the provider configured client to the resource.
func (r *networkAreaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.client = iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.resourceManagerClient = resourcemanagerUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the resource.
func (r *networkAreaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Network area resource schema.\n\n" +
		"This resource is for SNA, not VPC, networks."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`organization_id`,`network_area_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT organization ID to which the network area is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_area_id": schema.StringAttribute{
				Description: "The network area ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the network area.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"project_count": schema.Int64Attribute{
				Description: "The amount of projects currently referencing this area.",
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *networkAreaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network area
	networkArea, err := r.client.DefaultAPI.CreateNetworkArea(ctx, organizationId).CreateNetworkAreaPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	networkAreaId := *networkArea.Id
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	// Map response body to schema
	err = mapFields(ctx, networkArea, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area created")
}

// Read refreshes the Terraform state with the latest data.
func (r *networkAreaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	if networkAreaId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	networkAreaResp, err := r.client.DefaultAPI.GetNetworkArea(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, networkAreaResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkAreaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	// Retrieve values from state
	var stateModel Model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing network
	networkAreaUpdateResp, err := r.client.DefaultAPI.PartialUpdateNetworkArea(ctx, organizationId, networkAreaId).PartialUpdateNetworkAreaPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, networkAreaUpdateResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkAreaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	_, err := wait.ReadyForNetworkAreaDeletionWaitHandler(ctx, r.client.DefaultAPI, r.resourceManagerClient.DefaultAPI, organizationId, networkAreaId).WaitWithContext(ctx)
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area", fmt.Sprintf("Network area ready for deletion waiting: %v", err))
		return
	}

	// Get all configured regions so we can delete them one by one before deleting the network area
	regionsListResp, err := r.client.DefaultAPI.ListNetworkAreaRegions(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area region", fmt.Sprintf("Calling API to list configured regions: %v", err))
		return
	}

	// Delete network region configurations
	for region := range regionsListResp.Regions {
		err = r.client.DefaultAPI.DeleteNetworkAreaRegion(ctx, organizationId, networkAreaId, region).Execute()
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusBadRequest) { // TODO: iaas api returns http 400 in case network area region is not found
				continue
			}
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area region", fmt.Sprintf("Calling API: %v", err))
			return
		}

		_, err = wait.DeleteNetworkAreaRegionWaitHandler(ctx, r.client.DefaultAPI, organizationId, networkAreaId, region).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area region", fmt.Sprintf("Waiting for networea deletion: %v", err))
			return
		}
	}

	// Delete existing network area
	err = r.client.DefaultAPI.DeleteNetworkArea(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Network area deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,network_id
func (r *networkAreaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network area",
			fmt.Sprintf("Expected import identifier with format: [organization_id],[network_area_id]  Got: %q", req.ID),
		)
		return
	}

	organizationId := idParts[0]
	networkAreaId := idParts[1]
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_area_id"), networkAreaId)...)
	tflog.Info(ctx, "Network state imported")
}

func mapFields(ctx context.Context, networkAreaResp *iaas.NetworkArea, model *Model) error {
	if networkAreaResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkAreaId string
	if model.NetworkAreaId.ValueString() != "" {
		networkAreaId = model.NetworkAreaId.ValueString()
	} else if networkAreaResp.Id != nil {
		networkAreaId = *networkAreaResp.Id
	} else {
		return fmt.Errorf("network area id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.OrganizationId.ValueString(), networkAreaId)

	labels, err := iaasUtils.MapLabels(ctx, networkAreaResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.NetworkAreaId = types.StringValue(networkAreaId)
	model.Name = types.StringValue(networkAreaResp.Name)
	model.ProjectCount = types.Int64PointerValue(networkAreaResp.ProjectCount)
	model.Labels = labels

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateNetworkAreaPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.CreateNetworkAreaPayload{
		Name:   model.Name.ValueString(),
		Labels: labels,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.PartialUpdateNetworkAreaPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.PartialUpdateNetworkAreaPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: labels,
	}, nil
}
