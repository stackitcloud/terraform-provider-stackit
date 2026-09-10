package vpcregion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api/wait"
)

var (
	_ resource.Resource                = &vpcRegion{}
	_ resource.ResourceWithConfigure   = &vpcRegion{}
	_ resource.ResourceWithImportState = &vpcRegion{}
	_ resource.ResourceWithModifyPlan  = &vpcRegion{}
)

type Model struct {
	SharedModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type SharedModel struct {
	Id        types.String `tfsdk:"id"` // needed by TF
	ProjectId types.String `tfsdk:"project_id"`
	VPCId     types.String `tfsdk:"vpc_id"`
	Region    types.String `tfsdk:"region"`
}

type ipv4Model struct {
	DefaultNameservers types.List `tfsdk:"default_nameservers"`
}

func NewVPCRegion() resource.Resource {
	return &vpcRegion{}
}

type vpcRegion struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

func (v *vpcRegion) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_vpc_region"
}

func (v *vpcRegion) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	v.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &v.providerData, features.VpcExperiment, "stackit_vpc_region", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	v.client = iaasAlphaUtils.ConfigureClient(ctx, &v.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "IaaS v2alpha client configured")
}

func (v *vpcRegion) Schema(ctx context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description:         resourceDescription,
		MarkdownDescription: descriptions["resource.markdown"],
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: descriptions["vpc_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"timeouts": timeouts.AttributesAll(ctx),
		},
	}
}

func (v *vpcRegion) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) { // nolint:gocritic // signature required by TF
	var configModel Model
	// skip initial empty configuration to avoid follow-up errors
	if request.Config.Raw.IsNull() {
		return
	}
	response.Diagnostics.Append(request.Config.Get(ctx, &configModel)...)
	if response.Diagnostics.HasError() {
		return
	}

	var planModel Model
	response.Diagnostics.Append(request.Plan.Get(ctx, &planModel)...)
	if response.Diagnostics.HasError() {
		return
	}

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, v.providerData.GetRegion(), response)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.Plan.Set(ctx, planModel)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (v *vpcRegion) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) { // nolint:gocritic // signature required by TF
	var model Model
	response.Diagnostics.Append(request.Plan.Get(ctx, &model)...)
	if response.Diagnostics.HasError() {
		return
	}

	waiterTimeout := wait.CreateVPCRegionWaitHandler(ctx, v.client.DefaultAPI, "", "", "").GetTimeout()
	createTimeout, diags := model.Timeouts.Create(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VPCId.ValueString()
	region := v.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(ctx, &model.SharedModel)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating VPC region", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	regionalVPC, err := v.client.DefaultAPI.CreateVPCRegion(ctx, projectId, vpcId, region).CreateVPCRegionPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating VPC region", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ctx = utils.SetAndLogStateFields(ctx, &response.Diagnostics, &response.State, map[string]interface{}{
		"project_id": projectId,
		"vpc_id":     vpcId,
		"region":     region,
	})

	_, err = wait.CreateVPCRegionWaitHandler(ctx, v.client.DefaultAPI, projectId, vpcId, region).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating VPC region", fmt.Sprintf("VPC region creation waiting: %v", err))
		return
	}

	err = mapFields(ctx, regionalVPC, &model.SharedModel)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating VPC region", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	response.Diagnostics.Append(response.State.Set(ctx, model)...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC Region created")
}

func (v *vpcRegion) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) { // nolint:gocritic // signature required by TF
	var model Model
	response.Diagnostics.Append(request.State.Get(ctx, &model)...)
	if response.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, core.DefaultOperationTimeout)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VPCId.ValueString()
	region := v.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)

	regionalVPC, err := v.client.DefaultAPI.GetVPCRegion(ctx, projectId, vpcId, region).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading VPC region", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, regionalVPC, &model.SharedModel)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading VPC region", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, model)...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC Region read")
}

func (v *vpcRegion) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) { // nolint:gocritic // signature required by TF
	var model, state Model
	response.Diagnostics.Append(request.Plan.Get(ctx, &model)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := model.Timeouts.Update(ctx, core.DefaultOperationTimeout)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VPCId.ValueString()
	region := v.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(ctx, &model.SharedModel)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error updating VPC region", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	regionalVPC, err := v.client.DefaultAPI.UpdateVPCRegion(ctx, projectId, vpcId, region).UpdateVPCRegionPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error updating VPC region", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, regionalVPC, &model.SharedModel)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error updating VPC region", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC Region updated")
}

func (v *vpcRegion) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) { // nolint:gocritic // signature required by TF
	var model Model
	response.Diagnostics.Append(request.State.Get(ctx, &model)...)
	if response.Diagnostics.HasError() {
		return
	}

	waiterTimeout := wait.DeleteVPCRegionWaitHandler(ctx, v.client.DefaultAPI, "", "", "").GetTimeout()
	deleteTimeout, diags := model.Timeouts.Delete(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VPCId.ValueString()
	region := v.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)

	err := v.client.DefaultAPI.DeleteVPCRegion(ctx, projectId, vpcId, region).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &response.Diagnostics, "Error deleting VPC region", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteVPCRegionWaitHandler(ctx, v.client.DefaultAPI, projectId, vpcId, region).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error deleting VPC region", fmt.Sprintf("VPC region deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "VPC Region deleted")
}

func (v *vpcRegion) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	idParts := strings.Split(request.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &response.Diagnostics,
			"Error importing vpc",
			fmt.Sprintf("Expected import identifier with format: [project_id],[vpc_id],[region]  Got: %q", request.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &response.Diagnostics, &response.State, map[string]interface{}{
		"project_id": idParts[0],
		"vpc_id":     idParts[1],
		"region":     idParts[2],
	})
	if response.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "VPC Region imported")
}

func toCreatePayload(_ context.Context, model *SharedModel) (*iaas.CreateVPCRegionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var payload iaas.CreateVPCRegionPayload

	return &payload, nil
}

func toUpdatePayload(_ context.Context, model *SharedModel) (iaas.UpdateVPCRegionPayload, error) {
	var payload iaas.UpdateVPCRegionPayload
	if model == nil {
		return payload, fmt.Errorf("nil model")
	}

	return payload, nil
}

func mapFields(_ context.Context, regionalVPC *iaas.RegionalVPC, model *SharedModel) error {
	if regionalVPC == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.VPCId.ValueString(), model.Region.ValueString())

	return nil
}
