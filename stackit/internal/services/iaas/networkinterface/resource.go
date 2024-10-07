package networkinterface

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &networkInterfaceResource{}
	_ resource.ResourceWithConfigure   = &networkInterfaceResource{}
	_ resource.ResourceWithImportState = &networkInterfaceResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	ProjectId          types.String `tfsdk:"project_id"`
	NetworkId          types.String `tfsdk:"network_id"`
	NetworkInterfaceId types.String `tfsdk:"network_interface_id"`
	Name               types.String `tfsdk:"name"`
	AllowedAddresses   types.List   `tfsdk:"allowed_addresses"`
	IPv4               types.String `tfsdk:"ipv4"`
	IPv6               types.String `tfsdk:"ipv6"`
	Labels             types.Map    `tfsdk:"labels"`
	Security           types.Bool   `tfsdk:"security"`
	SecurityGroups     types.List   `tfsdk:"security_groups"`
	Device             types.String `tfsdk:"device"`
	Mac                types.String `tfsdk:"mac"`
	Type               types.String `tfsdk:"type"`
}

// Struct corresponding to Model.AllowedAddresses[i]
type allowedAddresses struct {
	String types.String `tfsdk:"string"`
}

// Types corresponding to allowedAddresses
var allowedAddressesTypes = map[string]attr.Type{
	"string": types.StringType,
}

// NewNetworkInterfaceResource is a helper function to simplify the provider implementation.
func NewNetworkInterfaceResource() resource.Resource {
	return &networkInterfaceResource{}
}

// networkResource is the resource implementation.
type networkInterfaceResource struct {
	client *iaasalpha.APIClient
}

// Metadata returns the resource type name.
func (r *networkInterfaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_interface"
}

// Configure adds the provider configured client to the resource.
func (r *networkInterfaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_network_interface", "resource")
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
	tflog.Info(ctx, "IaaSalpha client configured")
}

// Schema defines the schema for the resource.
func (r *networkInterfaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	typeOptions := []string{"server", "metadata", "gateway"}

	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Network interface resource schema. Must have a `region` specified in the provider configuration."),
		Description:         "Network interface resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`network_id`,`network_interface_id`\".",
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
				Description: "The network ID to which the network interface is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_interface_id": schema.StringAttribute{
				Description: "The network interface ID.",
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
				Description: "The name of the network interface.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
			},
			"allowed_addresses": schema.ListNestedAttribute{
				Description: "The list of CIDR (Classless Inter-Domain Routing) notations.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"string": schema.StringAttribute{
							Optional: true,
							Validators: []validator.String{
								validate.CIDR(),
							},
						},
					},
				},
			},
			"device": schema.StringAttribute{
				Description: "The device UUID of the network interface.",
				Computed:    true,
			},
			"ipv4": schema.StringAttribute{
				Description: "The IPv4 address.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.IP(),
				},
			},
			"ipv6": schema.StringAttribute{
				Description: "The IPv6 address.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.IP(),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a network interface.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"mac": schema.StringAttribute{
				Description: "The MAC address of network interface.",
				Computed:    true,
			},
			"security": schema.BoolAttribute{
				Description: "The Network Interface Security. If set to false, then no security groups will apply to this network interface.",
				Computed:    true,
				Optional:    true,
			},
			"security_groups": schema.ListAttribute{
				Description: "The list of security group UUIDs.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
							"must match expression"),
					),
				},
			},
			"type": schema.StringAttribute{
				Description: "Type of network interface. Some of the possible values are: " + utils.SupportedValuesDocumentation(typeOptions),
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *networkInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "network_id", networkId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network interface", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network interface

	networkInterface, err := r.client.CreateNIC(ctx, projectId, networkId).CreateNICPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network interface", fmt.Sprintf("Calling API: %v", err))
		return
	}

	networkInterfaceId := *networkInterface.Id

	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	// Map response body to schema
	err = mapFields(ctx, networkInterface, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network interface", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network interface created")
}

// Read refreshes the Terraform state with the latest data.
func (r *networkInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	networkInterfaceResp, err := r.client.GetNIC(ctx, projectId, networkId, networkInterfaceId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network interface", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, networkInterfaceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network interface", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network interface read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network interface", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing network
	nicResp, err := r.client.UpdateNIC(ctx, projectId, networkId, networkInterfaceId).UpdateNICPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network interface", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, nicResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network interface", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network interface updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	// Delete existing network interface
	err := r.client.DeleteNIC(ctx, projectId, networkId, networkInterfaceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network interface", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Network interface deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,network_id,network_interface_id
func (r *networkInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network interface",
			fmt.Sprintf("Expected import identifier with format: [project_id],[network_id],[network_interface_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	networkId := idParts[1]
	networkInterfaceId := idParts[2]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_id"), networkId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_interface_id"), networkInterfaceId)...)
	tflog.Info(ctx, "Network interface state imported")
}

func mapFields(ctx context.Context, networkInterfaceResp *iaasalpha.NIC, model *Model) error {
	if networkInterfaceResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkInterfaceId string
	if model.NetworkInterfaceId.ValueString() != "" {
		networkInterfaceId = model.NetworkInterfaceId.ValueString()
	} else if networkInterfaceResp.NetworkId != nil {
		networkInterfaceId = *networkInterfaceResp.Id
	} else {
		return fmt.Errorf("network interface id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		model.NetworkId.ValueString(),
		networkInterfaceId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	respAllowedAddresses := []allowedAddresses{}
	var diags diag.Diagnostics
	if networkInterfaceResp.AllowedAddresses == nil {
		model.AllowedAddresses = types.ListNull(types.ObjectType{AttrTypes: allowedAddressesTypes})
	} else {
		if !(model.AllowedAddresses.IsNull() || model.AllowedAddresses.IsUnknown()) {
			diags = model.AllowedAddresses.ElementsAs(ctx, &respAllowedAddresses, false)
			if diags.HasError() {
				return fmt.Errorf("map allowed addresses: %w", core.DiagsToError(diags))
			}
		}

		modelAllowedAddressesStrings := []string{}
		for _, m := range respAllowedAddresses {
			modelAllowedAddressesStrings = append(modelAllowedAddressesStrings, m.String.ValueString())
		}

		apiAllowedAddressesStrings := []string{}
		for _, n := range *networkInterfaceResp.AllowedAddresses {
			apiAllowedAddressesStrings = append(apiAllowedAddressesStrings, *n.String)
		}

		reconciledAllowedAddresses := utils.ReconcileStringSlices(modelAllowedAddressesStrings, apiAllowedAddressesStrings)

		allowedAddressList := []attr.Value{}
		for i, allowedAddress := range reconciledAllowedAddresses {
			allowedAddressMap := map[string]attr.Value{
				"string": types.StringValue(allowedAddress),
			}

			reconciledAllowedAddressesTF, diags := types.ObjectValue(allowedAddressesTypes, allowedAddressMap)
			if diags.HasError() {
				return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
			}

			allowedAddressList = append(allowedAddressList, reconciledAllowedAddressesTF)
		}

		allowedAddressesTF, diags := types.ListValue(
			types.ObjectType{AttrTypes: allowedAddressesTypes},
			allowedAddressList,
		)
		if diags.HasError() {
			return fmt.Errorf("failed to map allowed addresses: %w", core.DiagsToError(diags))
		}

		model.AllowedAddresses = allowedAddressesTF
	}

	if networkInterfaceResp.SecurityGroups == nil {
		model.SecurityGroups = types.ListNull(types.StringType)
	} else {
		respSecurityGroups := *networkInterfaceResp.SecurityGroups
		modelSecurityGroups, err := utils.ListValuetoStringSlice(model.SecurityGroups)
		if err != nil {
			return fmt.Errorf("get current network interface security groups from model: %w", err)
		}

		reconciledSecurityGroups := utils.ReconcileStringSlices(modelSecurityGroups, respSecurityGroups)

		securityGroupsTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledSecurityGroups)
		if diags.HasError() {
			return fmt.Errorf("map network interface security groups: %w", core.DiagsToError(diags))
		}

		model.SecurityGroups = securityGroupsTF
	}

	var labels basetypes.MapValue
	if networkInterfaceResp.Labels != nil && len(*networkInterfaceResp.Labels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *networkInterfaceResp.Labels)
		if diags.HasError() {
			return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else {
		labels = types.MapNull(types.StringType)
	}

	model.NetworkInterfaceId = types.StringValue(networkInterfaceId)
	model.Name = types.StringPointerValue(networkInterfaceResp.Name)
	model.IPv4 = types.StringPointerValue(networkInterfaceResp.Ipv4)
	model.IPv6 = types.StringPointerValue(networkInterfaceResp.Ipv6)
	model.Security = types.BoolPointerValue(networkInterfaceResp.NicSecurity)
	model.Device = types.StringPointerValue(networkInterfaceResp.Device)
	model.Mac = types.StringPointerValue(networkInterfaceResp.Mac)
	model.Type = types.StringPointerValue(networkInterfaceResp.Type)
	model.Labels = labels

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaasalpha.CreateNICPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var labelPayload *map[string]interface{}

	modelSecurityGroups := []string{}
	if !(model.SecurityGroups.IsNull() || model.SecurityGroups.IsUnknown()) {
		for _, ns := range model.SecurityGroups.Elements() {
			securityGroupString, ok := ns.(types.String)
			if !ok {
				return nil, fmt.Errorf("type assertion failed")
			}
			modelSecurityGroups = append(modelSecurityGroups, securityGroupString.ValueString())
		}
	}

	allowedAddressesPayload := []iaasalpha.AllowedAddressesInner{}

	if !(model.AllowedAddresses.IsNull() || model.AllowedAddresses.IsUnknown()) {
		allowedAddressesModel := []allowedAddresses{}
		diags := model.AllowedAddresses.ElementsAs(ctx, &allowedAddressesModel, false)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping allowed addresses: %w", core.DiagsToError(diags))
		}

		for i := range allowedAddressesModel {
			allowedAddressModel := allowedAddressesModel[i]
			allowedAddressesPayload = append(allowedAddressesPayload, iaasalpha.AllowedAddressesInner{
				String: conversion.StringValueToPointer(allowedAddressModel.String),
			})
		}
	}

	if !model.Labels.IsNull() && !model.Labels.IsUnknown() {
		labelMap, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
		if err != nil {
			return nil, fmt.Errorf("mapping labels: %w", err)
		}
		labelPayload = &labelMap
	}

	return &iaasalpha.CreateNICPayload{
		AllowedAddresses: &allowedAddressesPayload,
		SecurityGroups:   &modelSecurityGroups,
		Labels:           labelPayload,
		Name:             conversion.StringValueToPointer(model.Name),
		Device:           conversion.StringValueToPointer(model.Device),
		Ipv4:             conversion.StringValueToPointer(model.IPv4),
		Ipv6:             conversion.StringValueToPointer(model.IPv6),
		Mac:              conversion.StringValueToPointer(model.Mac),
		Type:             conversion.StringValueToPointer(model.Type),
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaasalpha.UpdateNICPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var labelPayload *map[string]interface{}

	modelSecurityGroups := []string{}
	for _, ns := range model.SecurityGroups.Elements() {
		securityGroupString, ok := ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelSecurityGroups = append(modelSecurityGroups, securityGroupString.ValueString())
	}

	allowedAddressesPayload := []iaasalpha.AllowedAddressesInner{}

	if !(model.AllowedAddresses.IsNull() || model.AllowedAddresses.IsUnknown()) {
		allowedAddressesModel := []allowedAddresses{}
		diags := model.AllowedAddresses.ElementsAs(ctx, &allowedAddressesModel, false)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping allowed addresses: %w", core.DiagsToError(diags))
		}

		for i := range allowedAddressesModel {
			allowedAddressModel := allowedAddressesModel[i]
			allowedAddressesPayload = append(allowedAddressesPayload, iaasalpha.AllowedAddressesInner{
				String: conversion.StringValueToPointer(allowedAddressModel.String),
			})
		}
	}

	if !model.Labels.IsNull() && !model.Labels.IsUnknown() {
		labelMap, err := utils.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
		if err != nil {
			return nil, fmt.Errorf("mapping labels: %w", err)
		}
		labelPayload = &labelMap
	}

	return &iaasalpha.UpdateNICPayload{
		AllowedAddresses: &allowedAddressesPayload,
		SecurityGroups:   &modelSecurityGroups,
		Labels:           labelPayload,
		Name:             conversion.StringValueToPointer(model.Name),
		Device:           conversion.StringValueToPointer(model.Device),
		Ipv4:             conversion.StringValueToPointer(model.IPv4),
		Ipv6:             conversion.StringValueToPointer(model.IPv6),
		Mac:              conversion.StringValueToPointer(model.Mac),
		Type:             conversion.StringValueToPointer(model.Type),
	}, nil
}
