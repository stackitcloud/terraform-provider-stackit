package securitygrouprule

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &securityGroupRuleResource{}
	_ resource.ResourceWithConfigure   = &securityGroupRuleResource{}
	_ resource.ResourceWithImportState = &securityGroupRuleResource{}
)

type Model struct {
	Id                    types.String `tfsdk:"id"` // needed by TF
	ProjectId             types.String `tfsdk:"project_id"`
	SecurityGroupId       types.String `tfsdk:"security_group_id"`
	SecurityGroupRuleId   types.String `tfsdk:"security_group_rule_id"`
	Direction             types.String `tfsdk:"direction"`
	Description           types.String `tfsdk:"description"`
	EtherType             types.String `tfsdk:"ether_type"`
	IcmpParameters        types.Object `tfsdk:"icmp_parameters"`
	IpRange               types.String `tfsdk:"ip_range"`
	PortRange             types.Object `tfsdk:"port_range"`
	Protocol              types.Object `tfsdk:"protocol"`
	RemoteSecurityGroupId types.String `tfsdk:"remote_security_group_id"`
}

type icmpParametersModel struct {
	Code types.Int64 `tfsdk:"code"`
	Type types.Int64 `tfsdk:"type"`
}

// Types corresponding to icmpParameters
var icmpParametersTypes = map[string]attr.Type{
	"code": basetypes.Int64Type{},
	"type": basetypes.Int64Type{},
}

type portRangeModel struct {
	Max types.Int64 `tfsdk:"max"`
	Min types.Int64 `tfsdk:"min"`
}

// Types corresponding to portRange
var portRangeTypes = map[string]attr.Type{
	"max": basetypes.Int64Type{},
	"min": basetypes.Int64Type{},
}

type protocolModel struct {
	Name     types.String `tfsdk:"name"`
	Protocol types.Int64  `tfsdk:"protocol"`
}

// Types corresponding to protocol
var protocolTypes = map[string]attr.Type{
	"name":     basetypes.StringType{},
	"protocol": basetypes.Int64Type{},
}

// NewSecurityGroupRuleResource is a helper function to simplify the provider implementation.
func NewSecurityGroupRuleResource() resource.Resource {
	return &securityGroupRuleResource{}
}

// securityGroupRuleResource is the resource implementation.
type securityGroupRuleResource struct {
	client *iaasalpha.APIClient
}

// Metadata returns the resource type name.
func (r *securityGroupRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group_rule"
}

// Configure adds the provider configured client to the resource.
func (r *securityGroupRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !resourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_security_group_rule", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	var apiClient *iaasalpha.APIClient
	var err error
	if providerData.IaaSCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "iaas_custom_endpoint", providerData.IaaSCustomEndpoint)
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "iaasalpha client configured")
}

// Schema defines the schema for the resource.
func (r *securityGroupRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	directionOptions := []string{"ingress", "egress"}

	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Security group rule resource schema. Must have a `region` specified in the provider configuration."),
		Description:         "Security group rule resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`security_group_id`,`security_group_rule_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the security group rule is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"security_group_id": schema.StringAttribute{
				Description: "The security group ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"security_group_rule_id": schema.StringAttribute{
				Description: "The security group rule ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The rule description.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(127),
				},
			},
			"direction": schema.StringAttribute{
				Description: "The direction of the traffic which the rule should match. Some of the possible values are: " + utils.SupportedValuesDocumentation(directionOptions),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ether_type": schema.StringAttribute{
				Description: "The ethertype which the rule should match.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"icmp_parameters": schema.SingleNestedAttribute{
				Description: "ICMP Parameters.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"code": schema.Int64Attribute{
						Description: "ICMP code. Can be set if the protocol is ICMP.",
						Optional:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
							int64validator.AtMost(255),
						},
					},
					"type": schema.Int64Attribute{
						Description: "ICMP type. Can be set if the protocol is ICMP.",
						Optional:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
							int64validator.AtMost(255),
						},
					},
				},
			},
			"ip_range": schema.StringAttribute{
				Description: "The remote IP range which the rule should match.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.IP(),
				},
			},
			"port_range": schema.SingleNestedAttribute{
				Description: "The range of ports.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"max": schema.Int64Attribute{
						Description: "The maximum port number. Should be greater or equal to the minimum.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
							int64planmodifier.RequiresReplace(),
						},
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
							int64validator.AtMost(65535),
						},
					},
					"min": schema.Int64Attribute{
						Description: "The minimum port number. Should be less or equal to the minimum.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
							int64planmodifier.RequiresReplace(),
						},
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
							int64validator.AtMost(65535),
						},
					},
				},
			},
			"protocol": schema.SingleNestedAttribute{
				Description: "The internet protocol which the rule should match.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "The protocol name which the rule should match.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"protocol": schema.Int64Attribute{
						Description: "The protocol number which the rule should match.",
						Optional:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
							int64validator.AtMost(255),
						},
					},
				},
			},
			"remote_security_group_id": schema.StringAttribute{
				Description: "The remote security group which the rule should match.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *securityGroupRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	securityGroupId := model.SecurityGroupId.ValueString()
	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)

	var icmpParameters *icmpParametersModel
	if !(model.IcmpParameters.IsNull() || model.IcmpParameters.IsUnknown()) {
		icmpParameters = &icmpParametersModel{}
		diags = model.IcmpParameters.As(ctx, icmpParameters, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var portRange *portRangeModel
	if !(model.PortRange.IsNull() || model.PortRange.IsUnknown()) {
		portRange = &portRangeModel{}
		diags = model.PortRange.As(ctx, portRange, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var protocol *protocolModel
	if !(model.Protocol.IsNull() || model.Protocol.IsUnknown()) {
		protocol = &protocolModel{}
		diags = model.Protocol.As(ctx, protocol, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, icmpParameters, portRange, protocol)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating security group rule", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new security group rule
	securityGroupRule, err := r.client.CreateSecurityGroupRule(ctx, projectId, securityGroupId).CreateSecurityGroupRulePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating security group rule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = tflog.SetField(ctx, "security_group_rule_id", *securityGroupRule.Id)

	// Map response body to schema
	err = mapFields(securityGroupRule, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating security group rule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Security group rule created")
}

// Read refreshes the Terraform state with the latest data.
func (r *securityGroupRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	securityGroupId := model.SecurityGroupId.ValueString()
	securityGroupRuleId := model.SecurityGroupRuleId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)
	ctx = tflog.SetField(ctx, "security_group_rule_id", securityGroupRuleId)

	securityGroupRuleResp, err := r.client.GetSecurityGroupRule(ctx, projectId, securityGroupId, securityGroupRuleId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading security group rule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(securityGroupRuleResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading security group rule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "security group rule read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *securityGroupRuleResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating security group rule", "Security group rule can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *securityGroupRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	securityGroupId := model.SecurityGroupId.ValueString()
	securityGroupRuleId := model.SecurityGroupRuleId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)
	ctx = tflog.SetField(ctx, "security_group_rule_id", securityGroupRuleId)

	// Delete existing security group rule
	err := r.client.DeleteSecurityGroupRule(ctx, projectId, securityGroupId, securityGroupRuleId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting security group rule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "security group rule deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,security_group_id, security_group_rule_id
func (r *securityGroupRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing security group rule",
			fmt.Sprintf("Expected import identifier with format: [project_id],[security_group_id],[security_group_rule_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	securityGroupId := idParts[1]
	securityGroupRuleId := idParts[2]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)
	ctx = tflog.SetField(ctx, "security_group_rule_id", securityGroupRuleId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("security_group_id"), securityGroupId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("security_group_rule_id"), securityGroupRuleId)...)
	tflog.Info(ctx, "security group rule state imported")
}

func mapFields(securityGroupRuleResp *iaasalpha.SecurityGroupRule, model *Model) error {
	if securityGroupRuleResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var securityGroupRuleId string
	if model.SecurityGroupRuleId.ValueString() != "" {
		securityGroupRuleId = model.SecurityGroupRuleId.ValueString()
	} else if securityGroupRuleResp.Id != nil {
		securityGroupRuleId = *securityGroupRuleResp.Id
	} else {
		return fmt.Errorf("security group rule id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		model.SecurityGroupId.ValueString(),
		securityGroupRuleId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	model.SecurityGroupRuleId = types.StringValue(securityGroupRuleId)
	model.Direction = types.StringPointerValue(securityGroupRuleResp.Direction)
	model.Description = types.StringPointerValue(securityGroupRuleResp.Description)
	model.EtherType = types.StringPointerValue(securityGroupRuleResp.Ethertype)
	model.IpRange = types.StringPointerValue(securityGroupRuleResp.IpRange)
	model.RemoteSecurityGroupId = types.StringPointerValue(securityGroupRuleResp.RemoteSecurityGroupId)

	err := mapIcmpParameters(securityGroupRuleResp, model)
	if err != nil {
		return fmt.Errorf("map icmp_parameters: %w", err)
	}
	err = mapPortRange(securityGroupRuleResp, model)
	if err != nil {
		return fmt.Errorf("map port_range: %w", err)
	}
	err = mapProtocol(securityGroupRuleResp, model)
	if err != nil {
		return fmt.Errorf("map protocol: %w", err)
	}

	return nil
}

func mapIcmpParameters(securityGroupRuleResp *iaasalpha.SecurityGroupRule, m *Model) error {
	if securityGroupRuleResp.IcmpParameters == nil {
		m.IcmpParameters = types.ObjectNull(icmpParametersTypes)
		return nil
	}

	icmpParametersValues := map[string]attr.Value{
		"type": types.Int64Value(*securityGroupRuleResp.IcmpParameters.Type),
		"code": types.Int64Value(*securityGroupRuleResp.IcmpParameters.Code),
	}

	icmpParametersObject, diags := types.ObjectValue(icmpParametersTypes, icmpParametersValues)
	if diags.HasError() {
		return fmt.Errorf("create icmpParameters object: %w", core.DiagsToError(diags))
	}
	m.IcmpParameters = icmpParametersObject
	return nil
}

func mapPortRange(securityGroupRuleResp *iaasalpha.SecurityGroupRule, m *Model) error {
	if securityGroupRuleResp.PortRange == nil {
		m.PortRange = types.ObjectNull(portRangeTypes)
		return nil
	}

	portRangeValues := map[string]attr.Value{
		"max": types.Int64Value(*securityGroupRuleResp.PortRange.Max),
		"min": types.Int64Value(*securityGroupRuleResp.PortRange.Min),
	}

	portRangeObject, diags := types.ObjectValue(portRangeTypes, portRangeValues)
	if diags.HasError() {
		return fmt.Errorf("create portRange object: %w", core.DiagsToError(diags))
	}
	m.PortRange = portRangeObject
	return nil
}

func mapProtocol(securityGroupRuleResp *iaasalpha.SecurityGroupRule, m *Model) error {
	if securityGroupRuleResp.Protocol == nil {
		m.Protocol = types.ObjectNull(protocolTypes)
		return nil
	}

	protocolValue := types.Int64Null()
	if securityGroupRuleResp.Protocol.Protocol != nil {
		protocolValue = types.Int64Value(*securityGroupRuleResp.Protocol.Protocol)
	}

	protocolValues := map[string]attr.Value{
		"name":     types.StringValue(*securityGroupRuleResp.Protocol.Name),
		"protocol": protocolValue,
	}
	protocolObject, diags := types.ObjectValue(protocolTypes, protocolValues)
	if diags.HasError() {
		return fmt.Errorf("create protocol object: %w", core.DiagsToError(diags))
	}
	m.Protocol = protocolObject
	return nil
}

func toCreatePayload(model *Model, icmpParameters *icmpParametersModel, portRange *portRangeModel, protocol *protocolModel) (*iaasalpha.CreateSecurityGroupRulePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payloadIcmpParameters, err := toIcmpParametersPayload(icmpParameters)
	if err != nil {
		return nil, fmt.Errorf("converting icmp parameters: %w", err)
	}

	payloadPortRange, err := toPortRangePayload(portRange)
	if err != nil {
		return nil, fmt.Errorf("converting port range: %w", err)
	}

	payloadProtocol, err := toProtocolPayload(protocol)
	if err != nil {
		return nil, fmt.Errorf("converting protocol: %w", err)
	}

	return &iaasalpha.CreateSecurityGroupRulePayload{
		Description:           conversion.StringValueToPointer(model.Description),
		Direction:             conversion.StringValueToPointer(model.Direction),
		Ethertype:             conversion.StringValueToPointer(model.EtherType),
		IpRange:               conversion.StringValueToPointer(model.IpRange),
		RemoteSecurityGroupId: conversion.StringValueToPointer(model.RemoteSecurityGroupId),
		IcmpParameters:        payloadIcmpParameters,
		PortRange:             payloadPortRange,
		Protocol:              payloadProtocol,
	}, nil
}

func toIcmpParametersPayload(icmpParameters *icmpParametersModel) (*iaasalpha.ICMPParameters, error) {
	if icmpParameters == nil {
		return nil, nil
	}
	payloadParams := &iaasalpha.ICMPParameters{}

	payloadParams.Code = conversion.Int64ValueToPointer(icmpParameters.Code)
	payloadParams.Type = conversion.Int64ValueToPointer(icmpParameters.Type)

	return payloadParams, nil
}

func toPortRangePayload(portRange *portRangeModel) (*iaasalpha.PortRange, error) {
	if portRange == nil {
		return nil, nil
	}
	payloadPortRange := &iaasalpha.PortRange{}

	payloadPortRange.Max = conversion.Int64ValueToPointer(portRange.Max)
	payloadPortRange.Min = conversion.Int64ValueToPointer(portRange.Min)

	return payloadPortRange, nil
}

func toProtocolPayload(protocol *protocolModel) (*iaasalpha.V1SecurityGroupRuleProtocol, error) {
	if protocol == nil {
		return nil, nil
	}
	payloadProtocol := &iaasalpha.V1SecurityGroupRuleProtocol{}

	payloadProtocol.Name = conversion.StringValueToPointer(protocol.Name)
	payloadProtocol.Protocol = conversion.Int64ValueToPointer(protocol.Protocol)

	return payloadProtocol, nil
}
