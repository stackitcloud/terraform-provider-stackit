package networkarea

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	resourcemanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
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
)

type Model struct {
	Id                  types.String `tfsdk:"id"` // needed by TF
	OrganizationId      types.String `tfsdk:"organization_id"`
	NetworkAreaId       types.String `tfsdk:"network_area_id"`
	Name                types.String `tfsdk:"name"`
	ProjectCount        types.Int64  `tfsdk:"project_count"`
	DefaultNameservers  types.List   `tfsdk:"default_nameservers"`
	NetworkRanges       types.List   `tfsdk:"network_ranges"`
	TransferNetwork     types.String `tfsdk:"transfer_network"`
	DefaultPrefixLength types.Int64  `tfsdk:"default_prefix_length"`
	MaxPrefixLength     types.Int64  `tfsdk:"max_prefix_length"`
	MinPrefixLength     types.Int64  `tfsdk:"min_prefix_length"`
	Labels              types.Map    `tfsdk:"labels"`
}

// Struct corresponding to Model.NetworkRanges[i]
type networkRange struct {
	Prefix         types.String `tfsdk:"prefix"`
	NetworkRangeId types.String `tfsdk:"network_range_id"`
}

// Types corresponding to networkRanges
var networkRangeTypes = map[string]attr.Type{
	"prefix":           types.StringType,
	"network_range_id": types.StringType,
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

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	resourceManagerClient := resourcemanagerUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.resourceManagerClient = resourceManagerClient
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the resource.
func (r *networkAreaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Network area resource schema. Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on resource level."
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
				Description: "Classless Inter-Domain Routing (CIDR).",
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
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network area
	area, err := r.client.CreateNetworkArea(ctx, organizationId).CreateNetworkAreaPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	networkArea, err := wait.CreateNetworkAreaWaitHandler(ctx, r.client, organizationId, *area.AreaId).WaitWithContext(context.Background())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Network area creation waiting: %v", err))
		return
	}
	networkAreaId := *networkArea.AreaId
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	networkAreaRanges := networkArea.Ipv4.NetworkRanges

	// Map response body to schema
	err = mapFields(ctx, networkArea, networkAreaRanges, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area created")
}

// Read refreshes the Terraform state with the latest data.
func (r *networkAreaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	networkAreaResp, err := r.client.GetNetworkArea(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	networkAreaRanges := networkAreaResp.Ipv4.NetworkRanges

	// Map response body to schema
	err = mapFields(ctx, networkAreaResp, networkAreaRanges, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkAreaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	ranges := []networkRange{}
	if !(model.NetworkRanges.IsNull() || model.NetworkRanges.IsUnknown()) {
		diags = model.NetworkRanges.ElementsAs(ctx, &ranges, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Retrieve values from state
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
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
	_, err = r.client.PartialUpdateNetworkArea(ctx, organizationId, networkAreaId).PartialUpdateNetworkAreaPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.UpdateNetworkAreaWaitHandler(ctx, r.client, organizationId, networkAreaId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Network area update waiting: %v", err))
		return
	}

	// Update network ranges
	err = updateNetworkRanges(ctx, organizationId, networkAreaId, ranges, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Updating Network ranges: %v", err))
		return
	}

	networkAreaResp, err := r.client.GetNetworkArea(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	networkAreaRanges := networkAreaResp.Ipv4.NetworkRanges

	err = mapFields(ctx, waitResp, networkAreaRanges, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
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
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	_, err := wait.ReadyForNetworkAreaDeletionWaitHandler(ctx, r.client, r.resourceManagerClient, organizationId, networkAreaId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area", fmt.Sprintf("Network area ready for deletion waiting: %v", err))
		return
	}

	// Delete existing network
	err = r.client.DeleteNetworkArea(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteNetworkAreaWaitHandler(ctx, r.client, organizationId, networkAreaId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area", fmt.Sprintf("Network area deletion waiting: %v", err))
		return
	}

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

func mapFields(ctx context.Context, networkAreaResp *iaas.NetworkArea, networkAreaRangesResp *[]iaas.NetworkRange, model *Model) error {
	if networkAreaResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkAreaId string
	if model.NetworkAreaId.ValueString() != "" {
		networkAreaId = model.NetworkAreaId.ValueString()
	} else if networkAreaResp.AreaId != nil {
		networkAreaId = *networkAreaResp.AreaId
	} else {
		return fmt.Errorf("network area id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.OrganizationId.ValueString(), networkAreaId)

	if networkAreaResp.Ipv4 == nil || networkAreaResp.Ipv4.DefaultNameservers == nil {
		model.DefaultNameservers = types.ListNull(types.StringType)
	} else {
		respDefaultNameservers := *networkAreaResp.Ipv4.DefaultNameservers
		modelDefaultNameservers, err := utils.ListValuetoStringSlice(model.DefaultNameservers)
		if err != nil {
			return fmt.Errorf("get current network area default nameservers from model: %w", err)
		}

		reconciledDefaultNameservers := utils.ReconcileStringSlices(modelDefaultNameservers, respDefaultNameservers)

		defaultNameserversTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledDefaultNameservers)
		if diags.HasError() {
			return fmt.Errorf("map network area default nameservers: %w", core.DiagsToError(diags))
		}

		model.DefaultNameservers = defaultNameserversTF
	}

	err := mapNetworkRanges(ctx, networkAreaRangesResp, model)
	if err != nil {
		return fmt.Errorf("mapping network ranges: %w", err)
	}

	labels, err := iaasUtils.MapLabels(ctx, networkAreaResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.NetworkAreaId = types.StringValue(networkAreaId)
	model.Name = types.StringPointerValue(networkAreaResp.Name)
	model.ProjectCount = types.Int64PointerValue(networkAreaResp.ProjectCount)
	model.Labels = labels

	if networkAreaResp.Ipv4 != nil {
		model.TransferNetwork = types.StringPointerValue(networkAreaResp.Ipv4.TransferNetwork)
		model.DefaultPrefixLength = types.Int64PointerValue(networkAreaResp.Ipv4.DefaultPrefixLen)
		model.MaxPrefixLength = types.Int64PointerValue(networkAreaResp.Ipv4.MaxPrefixLen)
		model.MinPrefixLength = types.Int64PointerValue(networkAreaResp.Ipv4.MinPrefixLen)
	}

	return nil
}

func mapNetworkRanges(ctx context.Context, networkAreaRangesList *[]iaas.NetworkRange, model *Model) error {
	var diags diag.Diagnostics

	if networkAreaRangesList == nil {
		return fmt.Errorf("nil network area ranges list")
	}
	if len(*networkAreaRangesList) == 0 {
		model.NetworkRanges = types.ListNull(types.ObjectType{AttrTypes: networkRangeTypes})
		return nil
	}

	ranges := []networkRange{}
	if !(model.NetworkRanges.IsNull() || model.NetworkRanges.IsUnknown()) {
		diags = model.NetworkRanges.ElementsAs(ctx, &ranges, false)
		if diags.HasError() {
			return fmt.Errorf("map network ranges: %w", core.DiagsToError(diags))
		}
	}

	modelNetworkRangePrefixes := []string{}
	for _, m := range ranges {
		modelNetworkRangePrefixes = append(modelNetworkRangePrefixes, m.Prefix.ValueString())
	}

	apiNetworkRangePrefixes := []string{}
	for _, n := range *networkAreaRangesList {
		apiNetworkRangePrefixes = append(apiNetworkRangePrefixes, *n.Prefix)
	}

	reconciledRangePrefixes := utils.ReconcileStringSlices(modelNetworkRangePrefixes, apiNetworkRangePrefixes)

	networkRangesList := []attr.Value{}
	for i, prefix := range reconciledRangePrefixes {
		var networkRangeId string
		for _, networkRangeElement := range *networkAreaRangesList {
			if *networkRangeElement.Prefix == prefix {
				networkRangeId = *networkRangeElement.NetworkRangeId
				break
			}
		}
		networkRangeMap := map[string]attr.Value{
			"prefix":           types.StringValue(prefix),
			"network_range_id": types.StringValue(networkRangeId),
		}

		networkRangeTF, diags := types.ObjectValue(networkRangeTypes, networkRangeMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		networkRangesList = append(networkRangesList, networkRangeTF)
	}

	networkRangesTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: networkRangeTypes},
		networkRangesList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	model.NetworkRanges = networkRangesTF
	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateNetworkAreaPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelDefaultNameservers := []string{}
	for _, ns := range model.DefaultNameservers.Elements() {
		nameserverString, ok := ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelDefaultNameservers = append(modelDefaultNameservers, nameserverString.ValueString())
	}

	networkRangesPayload, err := toNetworkRangesPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting network ranges: %w", err)
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.CreateNetworkAreaPayload{
		Name: conversion.StringValueToPointer(model.Name),
		AddressFamily: &iaas.CreateAreaAddressFamily{
			Ipv4: &iaas.CreateAreaIPv4{
				DefaultNameservers: &modelDefaultNameservers,
				NetworkRanges:      networkRangesPayload,
				TransferNetwork:    conversion.StringValueToPointer(model.TransferNetwork),
				DefaultPrefixLen:   conversion.Int64ValueToPointer(model.DefaultPrefixLength),
				MaxPrefixLen:       conversion.Int64ValueToPointer(model.MaxPrefixLength),
				MinPrefixLen:       conversion.Int64ValueToPointer(model.MinPrefixLength),
			},
		},
		Labels: &labels,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.PartialUpdateNetworkAreaPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelDefaultNameservers := []string{}
	for _, ns := range model.DefaultNameservers.Elements() {
		nameserverString, ok := ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelDefaultNameservers = append(modelDefaultNameservers, nameserverString.ValueString())
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.PartialUpdateNetworkAreaPayload{
		Name: conversion.StringValueToPointer(model.Name),
		AddressFamily: &iaas.UpdateAreaAddressFamily{
			Ipv4: &iaas.UpdateAreaIPv4{
				DefaultNameservers: &modelDefaultNameservers,
				DefaultPrefixLen:   conversion.Int64ValueToPointer(model.DefaultPrefixLength),
				MaxPrefixLen:       conversion.Int64ValueToPointer(model.MaxPrefixLength),
				MinPrefixLen:       conversion.Int64ValueToPointer(model.MinPrefixLength),
			},
		},
		Labels: &labels,
	}, nil
}

func toNetworkRangesPayload(ctx context.Context, model *Model) (*[]iaas.NetworkRange, error) {
	if model.NetworkRanges.IsNull() || model.NetworkRanges.IsUnknown() {
		return nil, nil
	}

	networkRangesModel := []networkRange{}
	diags := model.NetworkRanges.ElementsAs(ctx, &networkRangesModel, false)
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	if len(networkRangesModel) == 0 {
		return nil, nil
	}

	payload := []iaas.NetworkRange{}
	for i := range networkRangesModel {
		networkRangeModel := networkRangesModel[i]
		payload = append(payload, iaas.NetworkRange{
			Prefix: conversion.StringValueToPointer(networkRangeModel.Prefix),
		})
	}

	return &payload, nil
}

// updateNetworkRanges creates and deletes network ranges so that network area ranges are the ones in the model
func updateNetworkRanges(ctx context.Context, organizationId, networkAreaId string, ranges []networkRange, client *iaas.APIClient) error {
	// Get network ranges current state
	currentNetworkRangesResp, err := client.ListNetworkAreaRanges(ctx, organizationId, networkAreaId).Execute()
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
		networkRangesState[prefix].id = *networkRange.NetworkRangeId
	}

	// Delete network ranges
	for prefix, state := range networkRangesState {
		if !state.isInModel && state.isCreated {
			err := client.DeleteNetworkAreaRange(ctx, organizationId, networkAreaId, state.id).Execute()
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

			_, err := client.CreateNetworkAreaRange(ctx, organizationId, networkAreaId).CreateNetworkAreaRangePayload(payload).Execute()
			if err != nil {
				return fmt.Errorf("creating network range '%v': %w", prefix, err)
			}
		}
	}

	return nil
}
