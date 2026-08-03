package vpcnetworkrange

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vpcNetworkRangeResource{}
	_ resource.ResourceWithConfigure   = &vpcNetworkRangeResource{}
	_ resource.ResourceWithModifyPlan  = &vpcNetworkRangeResource{}
	_ resource.ResourceWithImportState = &vpcNetworkRangeResource{}
)

type SharedModel struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	VpcId          types.String `tfsdk:"vpc_id"`
	Region         types.String `tfsdk:"region"`
	NetworkRangeId types.String `tfsdk:"network_range_id"`

	Description         types.String `tfsdk:"description"`
	IpVersion           types.String `tfsdk:"ip_version"`
	DefaultPrefixLength types.Int64  `tfsdk:"default_prefix_length"`
	MaxPrefixLength     types.Int64  `tfsdk:"max_prefix_length"`
	MinPrefixLength     types.Int64  `tfsdk:"min_prefix_length"`
	Labels              types.Map    `tfsdk:"labels"`
	Nameservers         types.List   `tfsdk:"nameservers"`
	Prefix              types.String `tfsdk:"prefix"`
}

type ResourceModel struct {
	SharedModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// NewVpcNetworkRangeResource is a helper function to simplify the provider implementation.
func NewVpcNetworkRangeResource() resource.Resource {
	return &vpcNetworkRangeResource{}
}

// networkResource is the resource implementation.
type vpcNetworkRangeResource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *vpcNetworkRangeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_network_range"
}

// Configure adds the provider configured client to the resource.
func (r *vpcNetworkRangeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &r.providerData, features.VpcExperiment, "stackit_vpc_network_range", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.client = iaasAlphaUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "IaaS v2alpha client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *vpcNetworkRangeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Schema defines the schema for the resource.
func (r *vpcNetworkRangeResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_range_id": schema.StringAttribute{
				Description: descriptions["network_range_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_version": schema.StringAttribute{
				Description: descriptions["ip_version"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(sdkUtils.EnumSliceToStringSlice(iaas.AllowedNetworkRangeIPv4RequestIpVersionEnumValues)...),
				},
			},
			"prefix": schema.StringAttribute{
				Description: descriptions["prefix"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.CIDR(),
				},
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"default_prefix_length": schema.Int64Attribute{
				Description: descriptions["default_prefix_length"],
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_prefix_length": schema.Int64Attribute{
				Description: descriptions["max_prefix_length"],
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_prefix_length": schema.Int64Attribute{
				Description: descriptions["min_prefix_length"],
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"nameservers": schema.ListAttribute{
				Description: descriptions["nameservers"],
				ElementType: types.StringType,
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseNonNullStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				ElementType: types.StringType,
				Optional:    true,
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

// Create creates the resource and sets the initial Terraform state.
func (r *vpcNetworkRangeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	waiterTimeout := wait.CreateVPCNetworkRangeWaitHandler(ctx, r.client.DefaultAPI, "", "", "", "").GetTimeout()
	createTimeout, diags := model.Timeouts.Create(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model.SharedModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network range", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network range
	networkRange, err := r.client.DefaultAPI.CreateVPCNetworkRange(ctx, projectId, vpcId, region).CreateVPCNetworkRangePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network range", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if networkRange == nil || networkRange.VPCNetworkRangeIPv4 == nil || networkRange.VPCNetworkRangeIPv4.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network range", fmt.Sprintf("Missing Network Range ID in response: %+v", networkRange))
		return
	}
	ctx = tflog.SetField(ctx, "network_range_id", *networkRange.VPCNetworkRangeIPv4.Id)

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":       projectId,
		"vpc_id":           vpcId,
		"region":           region,
		"network_range_id": *networkRange.VPCNetworkRangeIPv4.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := wait.CreateVPCNetworkRangeWaitHandler(ctx, r.client.DefaultAPI, projectId, vpcId, region, *networkRange.VPCNetworkRangeIPv4.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network range", fmt.Sprintf("Waiting for network range become ready: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network range", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC Network range created")
}

// Read refreshes the Terraform state with the latest data.
func (r *vpcNetworkRangeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	networkRangeId := model.NetworkRangeId.ValueString()
	if networkRangeId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_range_id", networkRangeId)

	networkRangeResp, err := r.client.DefaultAPI.GetVPCNetworkRange(ctx, projectId, vpcId, region, networkRangeId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network range", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, networkRangeResp, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network range", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC Network range read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *vpcNetworkRangeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	waiterTimeout := wait.UpdateVPCNetworkRangeWaitHandler(ctx, r.client.DefaultAPI, "", "", "", "").GetTimeout()
	updateTimeout, diags := model.Timeouts.Update(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	networkRangeId := model.NetworkRangeId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_range_id", networkRangeId)

	// Retrieve values from state
	var stateModel ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model.SharedModel, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network range", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing network
	_, err = r.client.DefaultAPI.UpdateVPCNetworkRange(ctx, projectId, vpcId, region, networkRangeId).UpdateVPCNetworkRangePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network range", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.UpdateVPCNetworkRangeWaitHandler(ctx, r.client.DefaultAPI, projectId, vpcId, region, networkRangeId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error update network range", fmt.Sprintf("Waiting for network range become ready: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network range", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC Network range updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *vpcNetworkRangeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model ResourceModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	waiterTimeout := wait.DeleteVPCNetworkRangeWaitHandler(ctx, r.client.DefaultAPI, "", "", "", "").GetTimeout()
	updateTimeout, diags := model.Timeouts.Delete(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	networkRangeId := model.NetworkRangeId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_range_id", networkRangeId)

	// Delete existing vpc
	err := r.client.DefaultAPI.DeleteVPCNetworkRange(ctx, projectId, vpcId, region, networkRangeId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network range", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteVPCNetworkRangeWaitHandler(ctx, r.client.DefaultAPI, projectId, vpcId, region, networkRangeId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network range", fmt.Sprintf("Waiting for network range become deleted: %v", err))
		return
	}

	tflog.Info(ctx, "VPC Network range deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,vpc_id,region,network_range_id
func (r *vpcNetworkRangeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing vpc network range",
			fmt.Sprintf("Expected import identifier with format: [project_id],[vpc_id],[region],[network_range_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":       idParts[0],
		"vpc_id":           idParts[1],
		"region":           idParts[2],
		"network_range_id": idParts[3],
	})
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC Network range state imported")
}

func mapFields(ctx context.Context, networkRangeResp *iaas.VPCNetworkRange, model *SharedModel, region string) error {
	if networkRangeResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if networkRangeResp.VPCNetworkRangeIPv4 != nil {
		return mapIpv4NetworkRange(ctx, networkRangeResp.VPCNetworkRangeIPv4, model, region)
	} else if networkRangeResp.VPCNetworkRangeIPv6 != nil {
		return mapIpv6NetworkRange(ctx, networkRangeResp.VPCNetworkRangeIPv6, model, region)
	}

	return fmt.Errorf("VPC Network range is nil")
}

func mapIpv4NetworkRange(ctx context.Context, ipv4Resp *iaas.VPCNetworkRangeIPv4, model *SharedModel, region string) error {
	if ipv4Resp == nil {
		return fmt.Errorf("response network range ipv4 is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkRangeId string
	if model.NetworkRangeId.ValueString() != "" {
		networkRangeId = model.NetworkRangeId.ValueString()
	} else if ipv4Resp.Id != nil {
		networkRangeId = *ipv4Resp.Id
	} else {
		return fmt.Errorf("VPC id not present")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		model.VpcId.ValueString(),
		region,
		networkRangeId,
	)

	labels, err := iaasUtils.MapLabels(ctx, ipv4Resp.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.Labels = labels
	model.NetworkRangeId = types.StringValue(networkRangeId)
	model.Region = types.StringValue(region)
	model.Description = types.StringPointerValue(ipv4Resp.Description)
	model.IpVersion = types.StringValue(string(ipv4Resp.IpVersion))
	model.DefaultPrefixLength = types.Int64PointerValue(ipv4Resp.DefaultPrefixLen)
	model.MaxPrefixLength = types.Int64PointerValue(ipv4Resp.MaxPrefixLen)
	model.MinPrefixLength = types.Int64PointerValue(ipv4Resp.MinPrefixLen)
	model.Prefix = types.StringValue(ipv4Resp.Prefix)

	if ipv4Resp.Nameservers != nil {
		modelNameservers, err := conversion.StringListToSlice(model.Nameservers)
		if err != nil {
			return fmt.Errorf("error converting nameservers to slice: %w", err)
		}

		reconciledRangePrefixes := utils.ReconcileStringSlices(modelNameservers, ipv4Resp.Nameservers)

		var diags diag.Diagnostics
		model.Nameservers, diags = types.ListValueFrom(ctx, types.StringType, reconciledRangePrefixes)
		if diags.HasError() {
			return fmt.Errorf("failed to map nameservers: %w", core.DiagsToError(diags))
		}
	}

	return nil
}

func mapIpv6NetworkRange(ctx context.Context, ipv6Resp *iaas.VPCNetworkRangeIPv6, model *SharedModel, region string) error {
	if ipv6Resp == nil {
		return fmt.Errorf("response network range ipv6 is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkRangeId string
	if model.NetworkRangeId.ValueString() != "" {
		networkRangeId = model.NetworkRangeId.ValueString()
	} else if ipv6Resp.Id != nil {
		networkRangeId = *ipv6Resp.Id
	} else {
		return fmt.Errorf("VPC id not present")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		model.VpcId.ValueString(),
		region,
		networkRangeId,
	)

	labels, err := iaasUtils.MapLabels(ctx, ipv6Resp.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.Labels = labels
	model.NetworkRangeId = types.StringValue(networkRangeId)
	model.Region = types.StringValue(region)
	model.Description = types.StringPointerValue(ipv6Resp.Description)
	model.IpVersion = types.StringValue(string(ipv6Resp.IpVersion))
	model.DefaultPrefixLength = types.Int64PointerValue(ipv6Resp.DefaultPrefixLen)
	model.MaxPrefixLength = types.Int64PointerValue(ipv6Resp.MaxPrefixLen)
	model.MinPrefixLength = types.Int64PointerValue(ipv6Resp.MinPrefixLen)
	model.Prefix = types.StringValue(ipv6Resp.Prefix)

	if ipv6Resp.Nameservers != nil {
		modelNameservers, err := conversion.StringListToSlice(model.Nameservers)
		if err != nil {
			return fmt.Errorf("error converting nameservers to slice: %w", err)
		}

		reconciledRangePrefixes := utils.ReconcileStringSlices(modelNameservers, ipv6Resp.Nameservers)

		var diags diag.Diagnostics
		model.Nameservers, diags = types.ListValueFrom(ctx, types.StringType, reconciledRangePrefixes)
		if diags.HasError() {
			return fmt.Errorf("failed to map nameservers: %w", core.DiagsToError(diags))
		}
	}

	return nil
}

func toCreatePayload(ctx context.Context, model *SharedModel) (*iaas.CreateVPCNetworkRangePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting model labels: %w", err)
	}

	modelNameservers, err := conversion.StringListToSlice(model.Nameservers)
	if err != nil {
		return nil, fmt.Errorf("converting model nameservers: %w", err)
	}

	return &iaas.CreateVPCNetworkRangePayload{
		DefaultPrefixLen: conversion.Int64ValueToPointer(model.DefaultPrefixLength),
		Description:      model.Description.ValueStringPointer(),
		IpVersion:        iaas.NetworkRangeIPv4RequestIpVersion(model.IpVersion.ValueString()),
		Labels:           labels,
		MaxPrefixLen:     conversion.Int64ValueToPointer(model.MaxPrefixLength),
		MinPrefixLen:     conversion.Int64ValueToPointer(model.MinPrefixLength),
		Nameservers:      modelNameservers,
		Prefix:           model.Prefix.ValueString(),
	}, nil
}

func toUpdatePayload(ctx context.Context, model *SharedModel, currentLabels types.Map) (*iaas.UpdateVPCNetworkRangePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to go map: %w", err)
	}

	modelNameservers, err := conversion.StringListToSlice(model.Nameservers)
	if err != nil {
		return nil, fmt.Errorf("converting model nameservers: %w", err)
	}

	return &iaas.UpdateVPCNetworkRangePayload{
		DefaultPrefixLen: model.DefaultPrefixLength.ValueInt64Pointer(),
		Description:      model.Description.ValueStringPointer(),
		IpVersion:        iaas.V1UpdateVPCNetworkRangeIPv4IpVersion(model.IpVersion.ValueString()),
		Labels:           labels,
		MaxPrefixLen:     model.MaxPrefixLength.ValueInt64Pointer(),
		MinPrefixLen:     model.MinPrefixLength.ValueInt64Pointer(),
		Nameservers:      modelNameservers,
	}, nil
}
