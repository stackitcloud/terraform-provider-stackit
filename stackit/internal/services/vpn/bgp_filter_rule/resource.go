package bgpfilterrule

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/vpn/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &bgpFilterRuleResource{}
	_ resource.ResourceWithConfigure   = &bgpFilterRuleResource{}
	_ resource.ResourceWithImportState = &bgpFilterRuleResource{}
	_ resource.ResourceWithModifyPlan  = &bgpFilterRuleResource{}

	actionValues = sdkUtils.EnumSliceToStringSlice(vpn.AllowedBGPFilterRuleActionEnumValues)
)

type MatchModel struct {
	AsPathContainsAny types.List   `tfsdk:"as_path_contains_any"`
	Communities       types.List   `tfsdk:"communities"`
	FirstAsn          types.Int64  `tfsdk:"first_asn"`
	MaxPrefixLength   types.Int32  `tfsdk:"max_prefix_length"`
	MinPrefixLength   types.Int32  `tfsdk:"min_prefix_length"`
	Peer              types.String `tfsdk:"peer"`
	Prefixes          types.List   `tfsdk:"prefixes"`
}

type SetModel struct {
	LocalPreference types.Int32 `tfsdk:"local_preference"`
}

// Model is shared by the resource and the data source - the BGPFilterRule API only has plain,
// always-returned fields (no write-only values), so there is no need for a separate split
// like the one used for stackit_vpn_connection.
type Model struct {
	Id        types.String `tfsdk:"id"` // needed by TF
	ProjectId types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	GatewayId types.String `tfsdk:"gateway_id"`
	FilterId  types.String `tfsdk:"filter_id"`
	RuleId    types.String `tfsdk:"rule_id"`
	Action    types.String `tfsdk:"action"`
	Sequence  types.Int32  `tfsdk:"sequence"`
	Match     *MatchModel  `tfsdk:"match"`
	Set       *SetModel    `tfsdk:"set"`
}

type bgpFilterRuleResource struct {
	client       *vpn.APIClient
	providerData core.ProviderData
}

func NewVPNBGPFilterRuleResource() resource.Resource {
	return &bgpFilterRuleResource{}
}

func (r *bgpFilterRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "VPN client configured")
}

func (r *bgpFilterRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_bgp_filter_rule"
}

func (r *bgpFilterRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("VPN BGP filter rule resource schema. A single rule within a `stackit_vpn_bgp_filter`. "+
			"All non-empty fields within `match` are AND-combined. Rules within a filter are evaluated in `sequence` "+
			"order (lower first); the first matching rule decides the outcome. An implicit deny follows the last rule. "+
			"A filter may hold at most 10 rules. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`gateway_id`,`filter_id`,`rule_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rule_id": schema.StringAttribute{
				Description: "The server-generated UUID of the rule.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID associated with the BGP filter rule.",
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
				Description: "STACKIT region name the resource is located in. If not defined, the provider region is used.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gateway_id": schema.StringAttribute{
				Description: "The UUID of the parent VPN gateway.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"filter_id": schema.StringAttribute{
				Description: "The UUID of the parent `stackit_vpn_bgp_filter`.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"action": schema.StringAttribute{
				Description: fmt.Sprintf("The action to take if the route matches all criteria. %s", tfutils.FormatPossibleValues(actionValues...)),
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(actionValues...),
				},
			},
			"sequence": schema.Int32Attribute{
				Description: "The evaluation order of the rule. Lower numbers are evaluated first. Must be unique within a filter. If omitted on creation, the server auto-assigns the next value.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"match": schema.SingleNestedAttribute{
				Description: "Optional matching criteria. If omitted entirely, the rule acts as match-all. All non-empty fields in this block must match (logical AND).",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"as_path_contains_any": schema.ListAttribute{
						Description: "Matches if the AS-PATH contains any one of the listed ASNs (logical OR within the list).",
						Optional:    true,
						ElementType: types.Int64Type,
					},
					"communities": schema.ListAttribute{
						Description: "Matches if the route carries any one of these BGP standard communities. Format is `asn:value` per RFC 1997.",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.RegexMatches(regexp.MustCompile(`^\d+:\d+$`), "must be in the format \"asn:value\""),
							),
						},
					},
					"first_asn": schema.Int64Attribute{
						Description: "Matches if the first ASN (immediate neighbor) in the AS-PATH equals this ASN.",
						Optional:    true,
					},
					"max_prefix_length": schema.Int32Attribute{
						Description: "Maximum subnet mask length for matched prefixes.",
						Optional:    true,
						Validators: []validator.Int32{
							int32validator.Between(0, 32),
						},
					},
					"min_prefix_length": schema.Int32Attribute{
						Description: "Minimum subnet mask length for matched prefixes.",
						Optional:    true,
						Validators: []validator.Int32{
							int32validator.Between(0, 32),
						},
					},
					"peer": schema.StringAttribute{
						Description: "Matches the exact IPv4 address of the BGP neighbor that advertised the route.",
						Optional:    true,
						Validators: []validator.String{
							validate.IP(false),
						},
					},
					"prefixes": schema.ListAttribute{
						Description: "List of IPv4 networks to match. A route's prefix matches if it equals one of these (subject to min/max prefix length refinement).",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(validate.CIDR()),
						},
					},
				},
			},
			"set": schema.SingleNestedAttribute{
				Description: "Optional BGP attributes to apply when `action` is `PERMIT`. Ignored for `DENY` rules.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"local_preference": schema.Int32Attribute{
						Description: "BGP LOCAL_PREF to set on the route. Higher values are preferred during best-path selection. Default BGP LOCAL_PREF is 100.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (r *bgpFilterRuleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
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

	tfutils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *bgpFilterRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 5 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" || idParts[4] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing VPN BGP filter rule",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[gateway_id],[filter_id],[rule_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"gateway_id": idParts[2],
		"filter_id":  idParts[3],
		"rule_id":    idParts[4],
	})
	tflog.Info(ctx, "VPN BGP filter rule state imported")
}

func (r *bgpFilterRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	gatewayId := model.GatewayId.ValueString()
	filterId := model.FilterId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "filter_id", filterId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN BGP filter rule", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.DefaultAPI.CreateGatewayBGPFilterRule(ctx, projectId, region, gatewayId, filterId).CreateGatewayBGPFilterRulePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN BGP filter rule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, createResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN BGP filter rule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN BGP filter rule created")
}

func (r *bgpFilterRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	gatewayId := model.GatewayId.ValueString()
	filterId := model.FilterId.ValueString()
	ruleId := model.RuleId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "filter_id", filterId)
	ctx = tflog.SetField(ctx, "rule_id", ruleId)
	ctx = tflog.SetField(ctx, "region", region)

	ruleResp, err := r.client.DefaultAPI.GetGatewayBGPFilterRule(ctx, projectId, region, gatewayId, filterId, ruleId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN BGP filter rule", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, ruleResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN BGP filter rule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN BGP filter rule read")
}

func (r *bgpFilterRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	gatewayId := model.GatewayId.ValueString()
	filterId := model.FilterId.ValueString()
	ruleId := model.RuleId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "filter_id", filterId)
	ctx = tflog.SetField(ctx, "rule_id", ruleId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN BGP filter rule", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateResp, err := r.client.DefaultAPI.UpdateGatewayBGPFilterRule(ctx, projectId, region, gatewayId, filterId, ruleId).UpdateGatewayBGPFilterRulePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN BGP filter rule", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, updateResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN BGP filter rule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN BGP filter rule updated")
}

func (r *bgpFilterRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	gatewayId := model.GatewayId.ValueString()
	filterId := model.FilterId.ValueString()
	ruleId := model.RuleId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "filter_id", filterId)
	ctx = tflog.SetField(ctx, "rule_id", ruleId)
	ctx = tflog.SetField(ctx, "region", region)

	err := r.client.DefaultAPI.DeleteGatewayBGPFilterRule(ctx, projectId, region, gatewayId, filterId, ruleId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting VPN BGP filter rule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "VPN BGP filter rule deleted")
}

// matchPayload is implemented (via pointer receiver) by CreateGatewayBGPFilterRulePayloadMatch and
// UpdateGatewayBGPFilterRulePayloadMatch - the generated SDK creates a distinct match type per
// payload even though their shape is identical, so a shared interface lets fillMatchPayload work
// for both without duplicating the field-by-field conversion.
type matchPayload interface {
	SetAsPathContainsAny([]int64)
	SetCommunities([]string)
	SetFirstASN(int64)
	SetMaxPrefixLength(int32)
	SetMinPrefixLength(int32)
	SetPeer(string)
	SetPrefixes([]string)
}

// setPayload is implemented (via pointer receiver) by CreateGatewayBGPFilterRulePayloadSet and
// UpdateGatewayBGPFilterRulePayloadSet, mirroring matchPayload above.
type setPayload interface {
	SetLocalPreference(int32)
}

func fillMatchPayload(model *MatchModel, payload matchPayload) error {
	if !tfutils.IsUndefined(model.AsPathContainsAny) {
		asns, err := int64ListValueToSlice(model.AsPathContainsAny)
		if err != nil {
			return fmt.Errorf("converting match.as_path_contains_any: %w", err)
		}
		payload.SetAsPathContainsAny(asns)
	}

	if !tfutils.IsUndefined(model.Communities) {
		communities, err := tfutils.ListValueToStringSlice(model.Communities)
		if err != nil {
			return fmt.Errorf("converting match.communities: %w", err)
		}
		payload.SetCommunities(communities)
	}

	if !tfutils.IsUndefined(model.FirstAsn) {
		payload.SetFirstASN(model.FirstAsn.ValueInt64())
	}

	if !tfutils.IsUndefined(model.MaxPrefixLength) {
		payload.SetMaxPrefixLength(model.MaxPrefixLength.ValueInt32())
	}

	if !tfutils.IsUndefined(model.MinPrefixLength) {
		payload.SetMinPrefixLength(model.MinPrefixLength.ValueInt32())
	}

	if !tfutils.IsUndefined(model.Peer) {
		payload.SetPeer(model.Peer.ValueString())
	}

	if !tfutils.IsUndefined(model.Prefixes) {
		prefixes, err := tfutils.ListValueToStringSlice(model.Prefixes)
		if err != nil {
			return fmt.Errorf("converting match.prefixes: %w", err)
		}
		payload.SetPrefixes(prefixes)
	}

	return nil
}

func fillSetPayload(model *SetModel, payload setPayload) {
	if !tfutils.IsUndefined(model.LocalPreference) {
		payload.SetLocalPreference(model.LocalPreference.ValueInt32())
	}
}

func int64ListValueToSlice(list types.List) ([]int64, error) {
	result := []int64{}
	for _, el := range list.Elements() {
		elInt, ok := el.(types.Int64)
		if !ok {
			return result, fmt.Errorf("expected element to be of type %T, got %T", types.Int64{}, el)
		}
		result = append(result, elInt.ValueInt64())
	}
	return result, nil
}

func toCreatePayload(model *Model) (*vpn.CreateGatewayBGPFilterRulePayload, error) {
	payload := &vpn.CreateGatewayBGPFilterRulePayload{
		Action: vpn.CreateGatewayBGPFilterRulePayloadAction(model.Action.ValueString()),
	}

	// sequence is optional on create - if the user didn't set it, let the server auto-assign it.
	if !tfutils.IsUndefined(model.Sequence) {
		payload.Sequence = conversion.Int32ValueToPointer(model.Sequence)
	}

	if model.Match != nil {
		match := &vpn.CreateGatewayBGPFilterRulePayloadMatch{}
		if err := fillMatchPayload(model.Match, match); err != nil {
			return nil, err
		}
		payload.Match = match
	}

	if model.Set != nil {
		set := &vpn.CreateGatewayBGPFilterRulePayloadSet{}
		fillSetPayload(model.Set, set)
		payload.Set = set
	}

	return payload, nil
}

func toUpdatePayload(model *Model) (*vpn.UpdateGatewayBGPFilterRulePayload, error) {
	payload := &vpn.UpdateGatewayBGPFilterRulePayload{
		Action: vpn.UpdateGatewayBGPFilterRulePayloadAction(model.Action.ValueString()),
		// sequence is required on every PUT by the API - model.Sequence is Optional+Computed with
		// UseStateForUnknown, so it always carries a known value by the time an update runs.
		Sequence: conversion.Int32ValueToPointer(model.Sequence),
	}

	if model.Match != nil {
		match := &vpn.UpdateGatewayBGPFilterRulePayloadMatch{}
		if err := fillMatchPayload(model.Match, match); err != nil {
			return nil, err
		}
		payload.Match = match
	}

	if model.Set != nil {
		set := &vpn.UpdateGatewayBGPFilterRulePayloadSet{}
		fillSetPayload(model.Set, set)
		payload.Set = set
	}

	return payload, nil
}

func mapFields(ctx context.Context, rule *vpn.BGPFilterRule, model *Model, region string) error {
	if rule == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var ruleId string
	if model.RuleId.ValueString() != "" {
		ruleId = model.RuleId.ValueString()
	} else if rule.Id != nil {
		ruleId = *rule.Id
	} else {
		return fmt.Errorf("rule id not present")
	}

	model.Id = tfutils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, model.GatewayId.ValueString(), model.FilterId.ValueString(), ruleId)
	model.RuleId = types.StringValue(ruleId)
	model.Region = types.StringValue(region)
	model.Action = types.StringValue(string(rule.Action))

	model.Sequence = types.Int32Null()
	if sequence, ok := rule.GetSequenceOk(); ok && sequence != nil {
		model.Sequence = types.Int32Value(*sequence)
	}

	model.Match = nil
	if match, ok := rule.GetMatchOk(); ok && match != nil {
		matchModel := &MatchModel{
			FirstAsn: types.Int64Null(),
			Peer:     types.StringNull(),
		}

		matchModel.AsPathContainsAny = types.ListNull(types.Int64Type)
		if asns, ok := match.GetAsPathContainsAnyOk(); ok && asns != nil {
			listVal, diags := types.ListValueFrom(ctx, types.Int64Type, asns)
			if diags.HasError() {
				return fmt.Errorf("mapping match.as_path_contains_any: %w", core.DiagsToError(diags))
			}
			matchModel.AsPathContainsAny = listVal
		}

		matchModel.Communities = types.ListNull(types.StringType)
		if communities, ok := match.GetCommunitiesOk(); ok && communities != nil {
			listVal, diags := types.ListValueFrom(ctx, types.StringType, communities)
			if diags.HasError() {
				return fmt.Errorf("mapping match.communities: %w", core.DiagsToError(diags))
			}
			matchModel.Communities = listVal
		}

		if firstAsn, ok := match.GetFirstASNOk(); ok && firstAsn != nil {
			matchModel.FirstAsn = types.Int64Value(*firstAsn)
		}

		matchModel.MaxPrefixLength = types.Int32PointerValue(func() *int32 { v, _ := match.GetMaxPrefixLengthOk(); return v }())
		matchModel.MinPrefixLength = types.Int32PointerValue(func() *int32 { v, _ := match.GetMinPrefixLengthOk(); return v }())

		if peer, ok := match.GetPeerOk(); ok && peer != nil {
			matchModel.Peer = types.StringValue(*peer)
		}

		matchModel.Prefixes = types.ListNull(types.StringType)
		if prefixes, ok := match.GetPrefixesOk(); ok && prefixes != nil {
			listVal, diags := types.ListValueFrom(ctx, types.StringType, prefixes)
			if diags.HasError() {
				return fmt.Errorf("mapping match.prefixes: %w", core.DiagsToError(diags))
			}
			matchModel.Prefixes = listVal
		}

		model.Match = matchModel
	}

	model.Set = nil
	if set, ok := rule.GetSetOk(); ok && set != nil {
		setModel := &SetModel{}
		setModel.LocalPreference = types.Int32PointerValue(func() *int32 { v, _ := set.GetLocalPreferenceOk(); return v }())
		model.Set = setModel
	}

	return nil
}
