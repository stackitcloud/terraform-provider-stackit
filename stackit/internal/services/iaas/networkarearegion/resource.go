package networkarearegion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	resourcemanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/utils"

	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkAreaRegionResource{}
	_ resource.ResourceWithConfigure   = &networkAreaRegionResource{}
	_ resource.ResourceWithImportState = &networkAreaRegionResource{}
	_ resource.ResourceWithModifyPlan  = &networkAreaRegionResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	OrganizationId types.String `tfsdk:"organization_id"`
	NetworkAreaId  types.String `tfsdk:"network_area_id"`
	Region         types.String `tfsdk:"region"`
	Ipv4           *ipv4Model   `tfsdk:"ipv4"`
}

// Struct corresponding to Model.Ipv4
type ipv4Model struct {
	DefaultNameservers  types.List          `tfsdk:"default_nameservers"`
	NetworkRanges       []networkRangeModel `tfsdk:"network_ranges"`
	TransferNetwork     types.String        `tfsdk:"transfer_network"`
	DefaultPrefixLength types.Int64         `tfsdk:"default_prefix_length"`
	MaxPrefixLength     types.Int64         `tfsdk:"max_prefix_length"`
	MinPrefixLength     types.Int64         `tfsdk:"min_prefix_length"`
}

// Struct corresponding to Model.NetworkRanges[i]
type networkRangeModel struct {
	Prefix         types.String `tfsdk:"prefix"`
	NetworkRangeId types.String `tfsdk:"network_range_id"`
}

// NewNetworkAreaRegionResource is a helper function to simplify the provider implementation.
func NewNetworkAreaRegionResource() resource.Resource {
	return &networkAreaRegionResource{}
}

// networkAreaRegionResource is the resource implementation.
type networkAreaRegionResource struct {
	client                *iaas.APIClient
	resourceManagerClient *resourcemanager.APIClient
	providerData          core.ProviderData
}

// Metadata returns the resource type name.
func (r *networkAreaRegionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_area_region"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *networkAreaRegionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *networkAreaRegionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.client = iaasUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.resourceManagerClient = resourcemanagerUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *networkAreaRegionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Network area region resource schema."

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`organization_id`,`network_area_id`,`region`\".",
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
				Description: "The resource region. If not defined, the provider region is used.",
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ipv4": schema.SingleNestedAttribute{
				Description: "The regional IPv4 config of a network area.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"default_nameservers": schema.ListAttribute{
						Description: "List of DNS Servers/Nameservers.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"network_ranges": schema.ListNestedAttribute{
						Description: "List of Network ranges.",
						Required:    true,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
							listvalidator.SizeAtMost(64),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"network_range_id": schema.StringAttribute{
									Computed: true,
									Validators: []validator.String{
										validate.UUID(),
										validate.NoSeparator(),
									},
								},
								"prefix": schema.StringAttribute{
									Description: "Classless Inter-Domain Routing (CIDR).",
									Required:    true,
								},
							},
						},
					},
					"transfer_network": schema.StringAttribute{
						Description: "IPv4 Classless Inter-Domain Routing (CIDR).",
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"default_prefix_length": schema.Int64Attribute{
						Description: "The default prefix length for networks in the network area.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(24),
							int64validator.AtMost(29),
						},
						Default: int64default.StaticInt64(25),
					},
					"max_prefix_length": schema.Int64Attribute{
						Description: "The maximal prefix length for networks in the network area.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(24),
							int64validator.AtMost(29),
						},
						Default: int64default.StaticInt64(29),
					},
					"min_prefix_length": schema.Int64Attribute{
						Description: "The minimal prefix length for networks in the network area.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(8),
							int64validator.AtMost(29),
						},
						Default: int64default.StaticInt64(24),
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *networkAreaRegionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area region", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network area region configuration
	networkAreaRegion, err := r.client.CreateNetworkAreaRegion(ctx, organizationId, networkAreaId, region).CreateNetworkAreaRegionPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area region", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"organization_id": organizationId,
		"network_area_id": networkAreaId,
		"region":          region,
	})

	// wait for creation of network area region to complete
	_, err = wait.CreateNetworkAreaRegionWaitHandler(ctx, r.client, organizationId, networkAreaId, region).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server", fmt.Sprintf("server creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, networkAreaRegion, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area region", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area region created")
}

// Read refreshes the Terraform state with the latest data.
func (r *networkAreaRegionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	networkAreaRegionResp, err := r.client.GetNetworkAreaRegion(ctx, organizationId, networkAreaId, region).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, networkAreaRegionResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area region", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area region read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkAreaRegionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	// Retrieve values from state
	var stateModel Model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area region", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Update existing network area region configuration
	_, err = r.client.UpdateNetworkAreaRegion(ctx, organizationId, networkAreaId, region).UpdateNetworkAreaRegionPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area region", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = updateIpv4NetworkRanges(ctx, organizationId, networkAreaId, model.Ipv4.NetworkRanges, r.client, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area region", fmt.Sprintf("Updating Network ranges: %v", err))
		return
	}

	updatedNetworkAreaRegion, err := r.client.GetNetworkAreaRegion(ctx, organizationId, networkAreaId, region).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area region", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, updatedNetworkAreaRegion, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area region", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "network area region updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkAreaRegionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "region", region)

	_, err := wait.ReadyForNetworkAreaDeletionWaitHandler(ctx, r.client, r.resourceManagerClient, organizationId, networkAreaId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area region", fmt.Sprintf("Network area ready for deletion waiting: %v", err))
		return
	}

	ctx = core.InitProviderContext(ctx)

	// Delete network area region configuration
	err = r.client.DeleteNetworkAreaRegion(ctx, organizationId, networkAreaId, region).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area region", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteNetworkAreaRegionWaitHandler(ctx, r.client, organizationId, networkAreaId, region).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area region", fmt.Sprintf("network area deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Network area region deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: organization_id,network_area_id,region
func (r *networkAreaRegionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network area region",
			fmt.Sprintf("Expected import identifier with format: [organization_id],[network_area_id],[region]  Got: %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"organization_id": idParts[0],
		"network_area_id": idParts[1],
		"region":          idParts[2],
	})

	tflog.Info(ctx, "Network area region state imported")
}

// mapFields maps the API response values to the Terraform resource model fields
func mapFields(ctx context.Context, networkAreaRegion *iaas.RegionalArea, model *Model, region string) error {
	if networkAreaRegion == nil {
		return fmt.Errorf("network are region input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.OrganizationId.ValueString(), model.NetworkAreaId.ValueString(), region)
	model.Region = types.StringValue(region)

	model.Ipv4 = &ipv4Model{}
	if networkAreaRegion.Ipv4 != nil {
		model.Ipv4.TransferNetwork = types.StringPointerValue(networkAreaRegion.Ipv4.TransferNetwork)
		model.Ipv4.DefaultPrefixLength = types.Int64PointerValue(networkAreaRegion.Ipv4.DefaultPrefixLen)
		model.Ipv4.MaxPrefixLength = types.Int64PointerValue(networkAreaRegion.Ipv4.MaxPrefixLen)
		model.Ipv4.MinPrefixLength = types.Int64PointerValue(networkAreaRegion.Ipv4.MinPrefixLen)
	}

	// map default nameservers
	if networkAreaRegion.Ipv4 == nil || networkAreaRegion.Ipv4.DefaultNameservers == nil {
		model.Ipv4.DefaultNameservers = types.ListNull(types.StringType)
	} else {
		respDefaultNameservers := *networkAreaRegion.Ipv4.DefaultNameservers
		modelDefaultNameservers, err := utils.ListValuetoStringSlice(model.Ipv4.DefaultNameservers)
		if err != nil {
			return fmt.Errorf("get current network area default nameservers from model: %w", err)
		}

		reconciledDefaultNameservers := utils.ReconcileStringSlices(modelDefaultNameservers, respDefaultNameservers)

		defaultNameserversTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledDefaultNameservers)
		if diags.HasError() {
			return fmt.Errorf("map network area default nameservers: %w", core.DiagsToError(diags))
		}

		model.Ipv4.DefaultNameservers = defaultNameserversTF
	}

	// map network ranges
	err := mapIpv4NetworkRanges(ctx, networkAreaRegion.Ipv4.NetworkRanges, model)
	if err != nil {
		return fmt.Errorf("mapping network ranges: %w", err)
	}

	return nil
}

// mapFields maps the API ipv4 network ranges response values to the Terraform resource model fields
func mapIpv4NetworkRanges(_ context.Context, networkAreaRangesList *[]iaas.NetworkRange, model *Model) error {
	if networkAreaRangesList == nil {
		return fmt.Errorf("nil network area ranges list")
	}
	if len(*networkAreaRangesList) == 0 {
		model.Ipv4.NetworkRanges = []networkRangeModel{}
		return nil
	}

	modelNetworkRangePrefixes := []string{}
	for _, m := range model.Ipv4.NetworkRanges {
		modelNetworkRangePrefixes = append(modelNetworkRangePrefixes, m.Prefix.ValueString())
	}

	apiNetworkRangePrefixes := []string{}
	for _, n := range *networkAreaRangesList {
		apiNetworkRangePrefixes = append(apiNetworkRangePrefixes, *n.Prefix)
	}

	reconciledRangePrefixes := utils.ReconcileStringSlices(modelNetworkRangePrefixes, apiNetworkRangePrefixes)

	model.Ipv4.NetworkRanges = []networkRangeModel{}
	for _, prefix := range reconciledRangePrefixes {
		var networkRangeId string
		for _, networkRangeElement := range *networkAreaRangesList {
			if *networkRangeElement.Prefix == prefix {
				networkRangeId = *networkRangeElement.Id
				break
			}
		}

		model.Ipv4.NetworkRanges = append(model.Ipv4.NetworkRanges, networkRangeModel{
			Prefix:         types.StringValue(prefix),
			NetworkRangeId: types.StringValue(networkRangeId),
		})
	}

	return nil
}

func toDefaultNameserversPayload(_ context.Context, model *Model) ([]string, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	modelDefaultNameservers := []string{}
	for _, ns := range model.Ipv4.DefaultNameservers.Elements() {
		nameserverString, ok := ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelDefaultNameservers = append(modelDefaultNameservers, nameserverString.ValueString())
	}

	return modelDefaultNameservers, nil
}

func toNetworkRangesPayload(_ context.Context, model *Model) (*[]iaas.NetworkRange, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	if len(model.Ipv4.NetworkRanges) == 0 {
		return nil, nil
	}

	payload := []iaas.NetworkRange{}
	for _, networkRange := range model.Ipv4.NetworkRanges {
		payload = append(payload, iaas.NetworkRange{
			Prefix: conversion.StringValueToPointer(networkRange.Prefix),
		})
	}

	return &payload, nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateNetworkAreaRegionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	} else if model.Ipv4 == nil {
		return nil, fmt.Errorf("nil model.Ipv4")
	}

	modelDefaultNameservers, err := toDefaultNameserversPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting default nameservers: %w", err)
	}

	networkRangesPayload, err := toNetworkRangesPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting network ranges: %w", err)
	}

	return &iaas.CreateNetworkAreaRegionPayload{
		Ipv4: &iaas.RegionalAreaIPv4{
			DefaultNameservers: &modelDefaultNameservers,
			DefaultPrefixLen:   conversion.Int64ValueToPointer(model.Ipv4.DefaultPrefixLength),
			MaxPrefixLen:       conversion.Int64ValueToPointer(model.Ipv4.MaxPrefixLength),
			MinPrefixLen:       conversion.Int64ValueToPointer(model.Ipv4.MinPrefixLength),
			TransferNetwork:    conversion.StringValueToPointer(model.Ipv4.TransferNetwork),
			NetworkRanges:      networkRangesPayload,
		},
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model) (*iaas.UpdateNetworkAreaRegionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelDefaultNameservers, err := toDefaultNameserversPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting default nameservers: %w", err)
	}

	return &iaas.UpdateNetworkAreaRegionPayload{
		Ipv4: &iaas.UpdateRegionalAreaIPv4{
			DefaultNameservers: &modelDefaultNameservers,
			DefaultPrefixLen:   conversion.Int64ValueToPointer(model.Ipv4.DefaultPrefixLength),
			MaxPrefixLen:       conversion.Int64ValueToPointer(model.Ipv4.MaxPrefixLength),
			MinPrefixLen:       conversion.Int64ValueToPointer(model.Ipv4.MinPrefixLength),
		},
	}, nil
}

// updateIpv4NetworkRanges creates and deletes network ranges so that network area ranges are the ones in the model.
func updateIpv4NetworkRanges(ctx context.Context, organizationId, networkAreaId string, ranges []networkRangeModel, client *iaas.APIClient, region string) error {
	// Get network ranges current state
	currentNetworkRangesResp, err := client.ListNetworkAreaRanges(ctx, organizationId, networkAreaId, region).Execute()
	if err != nil {
		return fmt.Errorf("error reading network area ranges: %w", err)
	}

	type networkRangeState struct {
		isInModel bool
		isCreated bool
		id        string
	}

	networkRangesState := make(map[string]*networkRangeState)
	for _, nwRange := range ranges {
		networkRangesState[nwRange.Prefix.ValueString()] = &networkRangeState{
			isInModel: true,
		}
	}

	for _, networkRange := range *currentNetworkRangesResp.Items {
		prefix := *networkRange.Prefix
		if _, ok := networkRangesState[prefix]; !ok {
			networkRangesState[prefix] = &networkRangeState{}
		}
		networkRangesState[prefix].isCreated = true
		networkRangesState[prefix].id = *networkRange.Id
	}

	// Delete network ranges
	for prefix, state := range networkRangesState {
		if !state.isInModel && state.isCreated {
			err := client.DeleteNetworkAreaRange(ctx, organizationId, networkAreaId, region, state.id).Execute()
			if err != nil {
				return fmt.Errorf("deleting network area range '%v': %w", prefix, err)
			}
		}
	}

	// Create network ranges
	for prefix, state := range networkRangesState {
		if state.isInModel && !state.isCreated {
			payload := iaas.CreateNetworkAreaRangePayload{
				Ipv4: &[]iaas.NetworkRange{
					{
						Prefix: sdkUtils.Ptr(prefix),
					},
				},
			}

			_, err := client.CreateNetworkAreaRange(ctx, organizationId, networkAreaId, region).CreateNetworkAreaRangePayload(payload).Execute()
			if err != nil {
				return fmt.Errorf("creating network range '%v': %w", prefix, err)
			}
		}
	}

	return nil
}
