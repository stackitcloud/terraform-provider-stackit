package network

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkResource{}
	_ resource.ResourceWithConfigure   = &networkResource{}
	_ resource.ResourceWithImportState = &networkResource{}
)

type Model struct {
	Id               types.String `tfsdk:"id"` // needed by TF
	ProjectId        types.String `tfsdk:"project_id"`
	NetworkId        types.String `tfsdk:"network_id"`
	Name             types.String `tfsdk:"name"`
	Nameservers      types.List   `tfsdk:"nameservers"`
	IPv4Gateway      types.String `tfsdk:"ipv4_gateway"`
	IPv4Nameservers  types.List   `tfsdk:"ipv4_nameservers"`
	IPv4Prefix       types.String `tfsdk:"ipv4_prefix"`
	IPv4PrefixLength types.Int64  `tfsdk:"ipv4_prefix_length"`
	Prefixes         types.List   `tfsdk:"prefixes"`
	IPv4Prefixes     types.List   `tfsdk:"ipv4_prefixes"`
	IPv6Gateway      types.String `tfsdk:"ipv6_gateway"`
	IPv6Nameservers  types.List   `tfsdk:"ipv6_nameservers"`
	IPv6Prefix       types.String `tfsdk:"ipv6_prefix"`
	IPv6PrefixLength types.Int64  `tfsdk:"ipv6_prefix_length"`
	IPv6Prefixes     types.List   `tfsdk:"ipv6_prefixes"`
	PublicIP         types.String `tfsdk:"public_ip"`
	Labels           types.Map    `tfsdk:"labels"`
	Routed           types.Bool   `tfsdk:"routed"`
}

// NewNetworkResource is a helper function to simplify the provider implementation.
func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

// networkResource is the resource implementation.
type networkResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *networkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

// Configure adds the provider configured client to the resource.
func (r *networkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *iaas.APIClient
	var err error
	if providerData.IaaSCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "iaas_custom_endpoint", providerData.IaaSCustomEndpoint)
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "IaaS client configured")
}

func (r networkResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !model.Nameservers.IsUnknown() && !model.IPv4Nameservers.IsUnknown() && !model.Nameservers.IsNull() && !model.IPv4Nameservers.IsNull() {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring network", "You cannot provide both the `nameservers` and `ipv4_nameservers` fields simultaneously. Please remove the deprecated `nameservers` field, and use `ipv4_nameservers` to configure nameservers for IPv4.")
	}
}

// Schema defines the schema for the resource.
func (r *networkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`network_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the network is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_id": schema.StringAttribute{
				Description: "The network ID.",
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
				Description: "The name of the network.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"nameservers": schema.ListAttribute{
				Description:        "The nameservers of the network. This field is deprecated and will be removed after April 28th 2025, use `ipv4_nameservers` to configure the nameservers for IPv4.",
				DeprecationMessage: "Use `ipv4_nameservers` to configure the nameservers for IPv4.",
				Optional:           true,
				Computed:           true,
				ElementType:        types.StringType,
			},
			"ipv4_gateway": schema.StringAttribute{
				Description: "The IPv4 gateway of a network. If not specified, the first IP of the network will be assigned as the gateway.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.IP(),
				},
			},
			"ipv4_nameservers": schema.ListAttribute{
				Description: "The IPv4 nameservers of the network.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipv4_prefix": schema.StringAttribute{
				Description: "The IPv4 prefix of the network (CIDR).",
				Optional:    true,
				Validators: []validator.String{
					validate.CIDR(),
				},
			},
			"ipv4_prefix_length": schema.Int64Attribute{
				Description: "The IPv4 prefix length of the network.",
				Optional:    true,
			},
			"prefixes": schema.ListAttribute{
				Description:        "The prefixes of the network. This field is deprecated and will be removed after April 28th 2025, use `ipv4_prefixes` to read the prefixes of the IPv4 networks.",
				DeprecationMessage: "Use `ipv4_prefixes` to read the prefixes of the IPv4 networks.",
				Computed:           true,
				ElementType:        types.StringType,
			},
			"ipv4_prefixes": schema.ListAttribute{
				Description: "The IPv4 prefixes of the network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipv6_gateway": schema.StringAttribute{
				Description: "The IPv6 gateway of a network. If not specified, the first IP of the network will be assigned as the gateway.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.IP(),
				},
			},
			"ipv6_nameservers": schema.ListAttribute{
				Description: "The IPv6 nameservers of the network.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				ElementType: types.StringType,
			},
			"ipv6_prefix": schema.StringAttribute{
				Description: "The IPv6 prefix of the network (CIDR).",
				Optional:    true,
				Validators: []validator.String{
					validate.CIDR(),
				},
			},
			"ipv6_prefix_length": schema.Int64Attribute{
				Description: "The IPv6 prefix length of the network.",
				Optional:    true,
			},
			"ipv6_prefixes": schema.ListAttribute{
				Description: "The IPv6 prefixes of the network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"public_ip": schema.StringAttribute{
				Description: "The public IP of the network.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
			"routed": schema.BoolAttribute{
				Description: "If set to `true`, the network is routed and therefore accessible from other networks.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network

	network, err := r.client.CreateNetwork(ctx, projectId).CreateNetworkPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Calling API: %v", err))
		return
	}

	networkId := *network.NetworkId
	network, err = wait.CreateNetworkWaitHandler(ctx, r.client, projectId, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Network creation waiting: %v", err))
		return
	}

	ctx = tflog.SetField(ctx, "network_id", networkId)

	// Map response body to schema
	err = mapFields(ctx, network, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network created")
}

// Read refreshes the Terraform state with the latest data.
func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	networkResp, err := r.client.GetNetwork(ctx, projectId, networkId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, networkResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	// Retrieve values from state
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, &stateModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing network
	err = r.client.PartialUpdateNetwork(ctx, projectId, networkId).PartialUpdateNetworkPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.UpdateNetworkWaitHandler(ctx, r.client, projectId, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Network update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	// Delete existing network
	err := r.client.DeleteNetwork(ctx, projectId, networkId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteNetworkWaitHandler(ctx, r.client, projectId, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network", fmt.Sprintf("Network deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Network deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,network_id
func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network",
			fmt.Sprintf("Expected import identifier with format: [project_id],[network_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	networkId := idParts[1]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_id"), networkId)...)
	tflog.Info(ctx, "Network state imported")
}

func mapFields(ctx context.Context, networkResp *iaas.Network, model *Model) error {
	if networkResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkId string
	if model.NetworkId.ValueString() != "" {
		networkId = model.NetworkId.ValueString()
	} else if networkResp.NetworkId != nil {
		networkId = *networkResp.NetworkId
	} else {
		return fmt.Errorf("network id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		networkId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	labels, diags := types.MapValueFrom(ctx, types.StringType, map[string]interface{}{})
	if diags.HasError() {
		return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
	}
	if networkResp.Labels != nil && len(*networkResp.Labels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *networkResp.Labels)
		if diags.HasError() {
			return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else if model.Labels.IsNull() {
		labels = types.MapNull(types.StringType)
	}

	// IPv4

	if networkResp.Nameservers == nil {
		model.Nameservers = types.ListNull(types.StringType)
		model.IPv4Nameservers = types.ListNull(types.StringType)
	} else {
		respNameservers := *networkResp.Nameservers
		modelNameservers, err := utils.ListValuetoStringSlice(model.Nameservers)
		modelIPv4Nameservers, errIpv4 := utils.ListValuetoStringSlice(model.IPv4Nameservers)
		if err != nil {
			return fmt.Errorf("get current network nameservers from model: %w", err)
		}
		if errIpv4 != nil {
			return fmt.Errorf("get current IPv4 network nameservers from model: %w", errIpv4)
		}

		reconciledNameservers := utils.ReconcileStringSlices(modelNameservers, respNameservers)
		reconciledIPv4Nameservers := utils.ReconcileStringSlices(modelIPv4Nameservers, respNameservers)

		nameserversTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledNameservers)
		ipv4NameserversTF, ipv4Diags := types.ListValueFrom(ctx, types.StringType, reconciledIPv4Nameservers)
		if diags.HasError() {
			return fmt.Errorf("map network nameservers: %w", core.DiagsToError(diags))
		}
		if ipv4Diags.HasError() {
			return fmt.Errorf("map IPv4 network nameservers: %w", core.DiagsToError(ipv4Diags))
		}

		model.Nameservers = nameserversTF
		model.IPv4Nameservers = ipv4NameserversTF
	}

	if networkResp.Prefixes == nil {
		model.Prefixes = types.ListNull(types.StringType)
		model.IPv4Prefixes = types.ListNull(types.StringType)
	} else {
		respPrefixes := *networkResp.Prefixes
		prefixesTF, diags := types.ListValueFrom(ctx, types.StringType, respPrefixes)
		if diags.HasError() {
			return fmt.Errorf("map network prefixes: %w", core.DiagsToError(diags))
		}

		model.Prefixes = prefixesTF
		model.IPv4Prefixes = prefixesTF
	}

	if networkResp.Gateway != nil {
		model.IPv4Gateway = types.StringPointerValue(networkResp.GetGateway())
	} else {
		model.IPv4Gateway = types.StringNull()
	}

	// IPv6

	if networkResp.NameserversV6 == nil {
		model.IPv6Nameservers = types.ListNull(types.StringType)
	} else {
		respIPv6Nameservers := *networkResp.NameserversV6
		modelIPv6Nameservers, errIpv6 := utils.ListValuetoStringSlice(model.IPv6Nameservers)
		if errIpv6 != nil {
			return fmt.Errorf("get current IPv6 network nameservers from model: %w", errIpv6)
		}

		reconciledIPv6Nameservers := utils.ReconcileStringSlices(modelIPv6Nameservers, respIPv6Nameservers)

		ipv6NameserversTF, ipv6Diags := types.ListValueFrom(ctx, types.StringType, reconciledIPv6Nameservers)
		if ipv6Diags.HasError() {
			return fmt.Errorf("map IPv6 network nameservers: %w", core.DiagsToError(ipv6Diags))
		}

		model.IPv6Nameservers = ipv6NameserversTF
	}

	if networkResp.PrefixesV6 == nil {
		model.IPv6Prefixes = types.ListNull(types.StringType)
	} else {
		respPrefixesV6 := *networkResp.PrefixesV6
		prefixesV6TF, diags := types.ListValueFrom(ctx, types.StringType, respPrefixesV6)
		if diags.HasError() {
			return fmt.Errorf("map network IPv6 prefixes: %w", core.DiagsToError(diags))
		}

		model.IPv6Prefixes = prefixesV6TF
	}

	if networkResp.Gatewayv6 != nil {
		model.IPv6Gateway = types.StringPointerValue(networkResp.GetGatewayv6())
	} else {
		model.IPv6Gateway = types.StringNull()
	}

	model.NetworkId = types.StringValue(networkId)
	model.Name = types.StringPointerValue(networkResp.Name)
	model.PublicIP = types.StringPointerValue(networkResp.PublicIp)
	model.Labels = labels
	model.Routed = types.BoolPointerValue(networkResp.Routed)

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateNetworkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	addressFamily := &iaas.CreateNetworkAddressFamily{}

	modelIPv6Nameservers := []string{}
	for _, ipv6ns := range model.IPv6Nameservers.Elements() {
		ipv6NameserverString, ok := ipv6ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelIPv6Nameservers = append(modelIPv6Nameservers, ipv6NameserverString.ValueString())
	}

	if !(model.IPv6Prefix.IsNull() || model.IPv6PrefixLength.IsNull() || model.IPv6Nameservers.IsNull()) {
		addressFamily.Ipv6 = &iaas.CreateNetworkIPv6Body{
			Nameservers:  &modelIPv6Nameservers,
			Gateway:      iaas.NewNullableString(conversion.StringValueToPointer(model.IPv6Gateway)),
			Prefix:       conversion.StringValueToPointer(model.IPv6Prefix),
			PrefixLength: conversion.Int64ValueToPointer(model.IPv6PrefixLength),
		}
	}

	modelIPv4Nameservers := []string{}
	var modelIPv4List []attr.Value

	if !(model.IPv4Nameservers.IsNull() || model.IPv4Nameservers.IsUnknown()) {
		modelIPv4List = model.IPv4Nameservers.Elements()
	} else {
		modelIPv4List = model.Nameservers.Elements()
	}

	for _, ipv4ns := range modelIPv4List {
		ipv4NameserverString, ok := ipv4ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelIPv4Nameservers = append(modelIPv4Nameservers, ipv4NameserverString.ValueString())
	}

	if !model.IPv4Prefix.IsNull() || !model.IPv4PrefixLength.IsNull() || (!model.IPv4Nameservers.IsNull() && len(model.IPv4Nameservers.Elements()) > 0) || (!model.Nameservers.IsNull() && len(model.Nameservers.Elements()) > 0) {
		addressFamily.Ipv4 = &iaas.CreateNetworkIPv4Body{
			Nameservers:  &modelIPv4Nameservers,
			Gateway:      iaas.NewNullableString(conversion.StringValueToPointer(model.IPv4Gateway)),
			Prefix:       conversion.StringValueToPointer(model.IPv4Prefix),
			PrefixLength: conversion.Int64ValueToPointer(model.IPv4PrefixLength),
		}
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	payload := iaas.CreateNetworkPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: &labels,
		Routed: conversion.BoolValueToPointer(model.Routed),
	}

	if addressFamily.Ipv6 != nil || addressFamily.Ipv4 != nil {
		payload.AddressFamily = addressFamily
	}

	return &payload, nil
}

func toUpdatePayload(ctx context.Context, model, stateModel *Model) (*iaas.PartialUpdateNetworkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	addressFamily := &iaas.UpdateNetworkAddressFamily{}

	modelIPv6Nameservers := []string{}
	for _, ipv6ns := range model.IPv6Nameservers.Elements() {
		ipv6NameserverString, ok := ipv6ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelIPv6Nameservers = append(modelIPv6Nameservers, ipv6NameserverString.ValueString())
	}

	if !model.IPv6Nameservers.IsNull() && len(model.IPv6Nameservers.Elements()) > 0 {
		addressFamily.Ipv6 = &iaas.UpdateNetworkIPv6Body{
			Nameservers: &modelIPv6Nameservers,
			Gateway:     iaas.NewNullableString(conversion.StringValueToPointer(model.IPv6Gateway)),
		}
	}

	modelIPv4Nameservers := []string{}
	var modelIPv4List []attr.Value

	if !(model.IPv4Nameservers.IsNull() || model.IPv4Nameservers.IsUnknown() || (model.IPv4Nameservers.Equal(stateModel.IPv4Nameservers) && !model.Nameservers.Equal(stateModel.Nameservers))) {
		modelIPv4List = model.IPv4Nameservers.Elements()
	} else {
		modelIPv4List = model.Nameservers.Elements()
	}
	for _, ipv4ns := range modelIPv4List {
		ipv4NameserverString, ok := ipv4ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelIPv4Nameservers = append(modelIPv4Nameservers, ipv4NameserverString.ValueString())
	}

	if (!model.IPv4Nameservers.IsNull() && len(model.IPv4Nameservers.Elements()) > 0) || (!model.Nameservers.IsNull() && len(model.Nameservers.Elements()) > 0) {
		addressFamily.Ipv4 = &iaas.UpdateNetworkIPv4Body{
			Nameservers: &modelIPv4Nameservers,
			Gateway:     iaas.NewNullableString(conversion.StringValueToPointer(model.IPv4Gateway)),
		}
	}

	currentLabels := stateModel.Labels
	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	payload := iaas.PartialUpdateNetworkPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: &labels,
	}

	if addressFamily.Ipv6 != nil || addressFamily.Ipv4 != nil {
		payload.AddressFamily = addressFamily
	}

	return &payload, nil
}
