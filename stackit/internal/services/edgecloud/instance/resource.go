package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/edge"
	edgewait "github.com/stackitcloud/stackit-sdk-go/services/edge/wait"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceenablement"
	enablementWait "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	edgeutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/edgecloud/utils"
	serviceenablementUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceenablement/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
	_ resource.ResourceWithModifyPlan  = &instanceResource{}
)

// Model represents the schema for the Edge Cloud instance resource.
type Model struct {
	Id          types.String `tfsdk:"id"` // Resource ID for Terraform
	Created     types.String `tfsdk:"created"`
	InstanceId  types.String `tfsdk:"instance_id"`
	Region      types.String `tfsdk:"region"`
	DisplayName types.String `tfsdk:"display_name"`
	ProjectId   types.String `tfsdk:"project_id"`
	PlanID      types.String `tfsdk:"plan_id"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
	FrontendUrl types.String `tfsdk:"frontend_url"`
}

// NewInstanceResource is a helper function to create a new edge resource instance.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource implements the resource interface for Edge Cloud instances.
type instanceResource struct {
	client           *edge.APIClient
	enablementClient *serviceenablement.APIClient
	providerData     core.ProviderData
}

func (i *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, i.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// descriptions for the attributes in the Schema
var descriptions = map[string]string{
	"id":           "Terraform's internal resource ID, structured as \"`project_id`,`region`,`instance_id`\".",
	"instance_id":  "<displayName>-<projectIDHash>",
	"display_name": fmt.Sprintf("Display name shown for the Edge Cloud instance. Has to be a valid hostname, with a length between %d and %d characters.", edgeutils.DisplayNameMinimumChars, edgeutils.DisplayNameMaximumChars),
	"created":      "The date and time the creation of the instance was triggered.",
	"frontend_url": "Frontend URL for the Edge Cloud instance.",
	"region":       "STACKIT region to use for the instance, providers default_region will be used if unset.",
	"project_id":   "STACKIT project ID to which the Edge Cloud instance is associated.",
	"plan_id":      "STACKIT Edge Plan ID for the Edge Cloud instance, has to be the UUID of an existing plan.",
	"description":  fmt.Sprintf("Description for your STACKIT Edge Cloud instance. Max length is %d characters", edgeutils.DescriptionMaxLength),
	"status":       "instance status",
}

// Configure sets up the API client for the Edge Cloud instance resource.
func (i *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	i.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	features.CheckBetaResourcesEnabled(ctx, &i.providerData, &resp.Diagnostics, "stackit_edgecloud_instance", "resource")
	if resp.Diagnostics.HasError() {
		return
	}
	apiClient := edgeutils.ConfigureClient(ctx, &i.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	serviceEnablementClient := serviceenablementUtils.ConfigureClient(ctx, &i.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	i.client = apiClient
	i.enablementClient = serviceEnablementClient
	tflog.Info(ctx, "edge client configured")
}

// Metadata sets the resource type name for the Edge Cloud instance resource.
func (i *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edgecloud_instance"
}

// Schema defines the schema for the resource.
func (i *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Edge Cloud is in private Beta and not generally available.\n You can contact support if you are interested in trying it out.", core.Resource),

		Description: "edge cloud instance resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created": schema.StringAttribute{
				Description: descriptions["created"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"frontend_url": schema.StringAttribute{
				Description: descriptions["frontend_url"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: descriptions["status"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(edgeutils.DisplayNameMinimumChars, edgeutils.DisplayNameMaximumChars),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`),
						"must be a valid hostname label, starting with a letter and containing only letters, numbers, or hyphens",
					),
				},
			},
			"plan_id": schema.StringAttribute{
				Description: descriptions["plan_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(edgeutils.DescriptionMaxLength),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial state for the Edge Cloud instance.
func (i *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	displayName := model.DisplayName.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "display_name", displayName)
	ctx = tflog.SetField(ctx, "region", region)

	// If the service edge-cloud is not enabled, enable it
	err := i.enablementClient.EnableServiceRegional(ctx, region, projectId, utils.EdgecloudServiceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API to enable edge-cloud: %v", err))
		return
	}

	_, err = enablementWait.EnableServiceWaitHandler(ctx, i.enablementClient, region, projectId, utils.EdgecloudServiceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Wait for edge-cloud enablement: %v", err))
		return
	}

	tflog.Info(ctx, "Creating new Edge Cloud instance")
	payload := toCreatePayload(&model)
	createResp, err := i.client.CreateInstance(ctx, projectId, region).CreateInstancePayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", "API returned nil response")
		return
	}
	if createResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", "API returned nil Instance ID")
		return
	}
	edgeCloudInstanceId := *createResp.Id
	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"instance_id": edgeCloudInstanceId,
		"region":      region,
	})

	waitResp, err := edgewait.CreateOrUpdateInstanceWaitHandler(ctx, i.client, projectId, region, edgeCloudInstanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Instance waiting: %v", err))
		return
	}

	err = mapFields(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Mapping API response fields to model: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "edge cloud instance created successfully")
}

// Read refreshes the state with the latest Edge Cloud instance data.
func (i *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := i.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	edgeCloudInstanceResp, err := i.client.GetInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(edgeCloudInstanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (i *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	tflog.Info(ctx, "Updating Edge Cloud instance", map[string]any{"instance_id": instanceId})
	payload := toUpdatePayload(&model)
	err := i.client.UpdateInstance(ctx, projectId, region, instanceId).UpdateInstancePayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := edgewait.CreateOrUpdateInstanceWaitHandler(ctx, i.client, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance waiting: %v", err))
		return
	}

	err = mapFields(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Mapping fields: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "edge cloud instance successfully updated")
}

// Delete deletes the Edge Cloud instance.
func (i *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	err := i.client.DeleteInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = edgewait.DeleteInstanceWaitHandler(ctx, i.client, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "edge cloud instance deleted")
}

// ImportState imports a resource into the state.
func (i *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[instance_id]  Got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
}

// mapFields maps the API response to the Terraform model.
func mapFields(resp *edge.Instance, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	// Build the ID by combining the project id, region and instance id and assign the model's fields.
	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if resp.Id != nil {
		instanceId = *resp.Id
	}
	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.Region.ValueString(), instanceId)
	model.InstanceId = types.StringValue(instanceId)
	if resp.Created.String() != "" {
		model.Created = types.StringValue(resp.Created.String())
	} else {
		model.Created = types.StringNull()
	}
	model.FrontendUrl = types.StringPointerValue(resp.FrontendUrl)
	model.DisplayName = types.StringPointerValue(resp.DisplayName)
	model.PlanID = types.StringPointerValue(resp.PlanId)
	model.Status = types.StringValue(string(*resp.Status))

	if resp.Description != nil {
		model.Description = types.StringValue(*resp.Description)
	} else {
		model.Description = types.StringValue("")
	}

	return nil
}

// toCreatePayload creates the payload for creating an Edge Cloud instance.
func toCreatePayload(model *Model) edge.CreateInstancePayload {
	return edge.CreateInstancePayload{
		DisplayName: model.DisplayName.ValueStringPointer(),
		Description: model.Description.ValueStringPointer(),
		PlanId:      model.PlanID.ValueStringPointer(),
	}
}

// toUpdatePayload creates the payload for updating an Edge Cloud instance using the correct struct.
func toUpdatePayload(model *Model) edge.UpdateInstancePayload {
	return edge.UpdateInstancePayload{
		Description: model.Description.ValueStringPointer(),
		PlanId:      model.PlanID.ValueStringPointer(),
	}
}
