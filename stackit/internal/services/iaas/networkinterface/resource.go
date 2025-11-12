package networkinterface

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkInterfaceResource{}
	_ resource.ResourceWithConfigure   = &networkInterfaceResource{}
	_ resource.ResourceWithImportState = &networkInterfaceResource{}
	_ resource.ResourceWithModifyPlan  = &networkInterfaceResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	ProjectId          types.String `tfsdk:"project_id"`
	NetworkId          types.String `tfsdk:"network_id"`
	NetworkInterfaceId types.String `tfsdk:"network_interface_id"`
	Name               types.String `tfsdk:"name"`
	AllowedAddresses   types.List   `tfsdk:"allowed_addresses"`
	IPv4               types.String `tfsdk:"ipv4"`
	Labels             types.Map    `tfsdk:"labels"`
	Security           types.Bool   `tfsdk:"security"`
	SecurityGroupIds   types.List   `tfsdk:"security_group_ids"`
	Device             types.String `tfsdk:"device"`
	Mac                types.String `tfsdk:"mac"`
	Type               types.String `tfsdk:"type"`
}

// NewNetworkInterfaceResource is a helper function to simplify the provider implementation.
func NewNetworkInterfaceResource() resource.Resource {
	return &networkInterfaceResource{}
}

// networkResource is the resource implementation.
type networkInterfaceResource struct {
	client *iaas.APIClient
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
func (r *networkInterfaceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	var configModel Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// If allowed_addresses were completly removed from the config this is not recognized by terraform
	// since this field is optional and computed therefore this plan modifier is needed.
	utils.CheckListRemoval(ctx, configModel.AllowedAddresses, planModel.AllowedAddresses, path.Root("allowed_addresses"), types.StringType, false, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// If security_group_ids were completly removed from the config this is not recognized by terraform
	// since this field is optional and computed therefore this plan modifier is needed.
	utils.CheckListRemoval(ctx, configModel.SecurityGroupIds, planModel.SecurityGroupIds, path.Root("security_group_ids"), types.StringType, true, resp)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Metadata returns the resource type name.
func (r *networkInterfaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_interface"
}

// Configure adds the provider configured client to the resource.
func (r *networkInterfaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *networkInterfaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	typeOptions := []string{"server", "metadata", "gateway"}
	description := "Network interface resource schema. Must have a `region` specified in the provider configuration."

	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
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
			"allowed_addresses": schema.ListAttribute{
				Description: "The list of CIDR (Classless Inter-Domain Routing) notations.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						validate.CIDR(),
					),
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
					validate.IP(false),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"security": schema.BoolAttribute{
				Description: "The Network Interface Security. If set to false, then no security groups will apply to this network interface.",
				Computed:    true,
				Optional:    true,
			},
			"security_group_ids": schema.ListAttribute{
				Description: "The list of security group UUIDs. If security is set to false, setting this field will lead to an error.",
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
				Description: "Type of network interface. Some of the possible values are: " + utils.FormatPossibleValues(typeOptions...),
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
	networkInterface, err := r.client.CreateNic(ctx, projectId, networkId).CreateNicPayload(*payload).Execute()
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

	networkInterfaceResp, err := r.client.GetNic(ctx, projectId, networkId, networkInterfaceId).Execute()
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
	nicResp, err := r.client.UpdateNic(ctx, projectId, networkId, networkInterfaceId).UpdateNicPayload(*payload).Execute()
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
	err := r.client.DeleteNic(ctx, projectId, networkId, networkInterfaceId).Execute()
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

func mapFields(ctx context.Context, networkInterfaceResp *iaas.NIC, model *Model) error {
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.NetworkId.ValueString(), networkInterfaceId)

	respAllowedAddresses := []string{}
	var diags diag.Diagnostics
	if networkInterfaceResp.AllowedAddresses == nil {
		// If we send an empty list, the API will send null in the response
		// We should handle this case and set the value to an empty list
		if !model.AllowedAddresses.IsNull() {
			model.AllowedAddresses, diags = types.ListValueFrom(ctx, types.StringType, []string{})
			if diags.HasError() {
				return fmt.Errorf("map network interface allowed addresses: %w", core.DiagsToError(diags))
			}
		} else {
			model.AllowedAddresses = types.ListNull(types.StringType)
		}
	} else {
		for _, n := range *networkInterfaceResp.AllowedAddresses {
			respAllowedAddresses = append(respAllowedAddresses, *n.String)
		}

		modelAllowedAddresses, err := utils.ListValuetoStringSlice(model.AllowedAddresses)
		if err != nil {
			return fmt.Errorf("get current network interface allowed addresses from model: %w", err)
		}

		reconciledAllowedAddresses := utils.ReconcileStringSlices(modelAllowedAddresses, respAllowedAddresses)

		allowedAddressesTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledAllowedAddresses)
		if diags.HasError() {
			return fmt.Errorf("map network interface allowed addresses: %w", core.DiagsToError(diags))
		}

		model.AllowedAddresses = allowedAddressesTF
	}

	if networkInterfaceResp.SecurityGroups == nil {
		model.SecurityGroupIds = types.ListNull(types.StringType)
	} else {
		respSecurityGroups := *networkInterfaceResp.SecurityGroups
		modelSecurityGroups, err := utils.ListValuetoStringSlice(model.SecurityGroupIds)
		if err != nil {
			return fmt.Errorf("get current network interface security groups from model: %w", err)
		}

		reconciledSecurityGroups := utils.ReconcileStringSlices(modelSecurityGroups, respSecurityGroups)

		securityGroupsTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledSecurityGroups)
		if diags.HasError() {
			return fmt.Errorf("map network interface security groups: %w", core.DiagsToError(diags))
		}

		model.SecurityGroupIds = securityGroupsTF
	}

	labels, err := iaasUtils.MapLabels(ctx, networkInterfaceResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	networkInterfaceName := types.StringNull()
	if networkInterfaceResp.Name != nil && *networkInterfaceResp.Name != "" {
		networkInterfaceName = types.StringPointerValue(networkInterfaceResp.Name)
	}

	model.NetworkInterfaceId = types.StringValue(networkInterfaceId)
	model.Name = networkInterfaceName
	model.IPv4 = types.StringPointerValue(networkInterfaceResp.Ipv4)
	model.Security = types.BoolPointerValue(networkInterfaceResp.NicSecurity)
	model.Device = types.StringPointerValue(networkInterfaceResp.Device)
	model.Mac = types.StringPointerValue(networkInterfaceResp.Mac)
	model.Type = types.StringPointerValue(networkInterfaceResp.Type)
	model.Labels = labels

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateNicPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var labelPayload *map[string]interface{}

	modelSecurityGroups := []string{}
	if !(model.SecurityGroupIds.IsNull() || model.SecurityGroupIds.IsUnknown()) {
		for _, ns := range model.SecurityGroupIds.Elements() {
			securityGroupString, ok := ns.(types.String)
			if !ok {
				return nil, fmt.Errorf("type assertion failed")
			}
			modelSecurityGroups = append(modelSecurityGroups, securityGroupString.ValueString())
		}
	}

	allowedAddressesPayload := &[]iaas.AllowedAddressesInner{}
	if !(model.AllowedAddresses.IsNull() || model.AllowedAddresses.IsUnknown()) {
		for _, allowedAddressModel := range model.AllowedAddresses.Elements() {
			allowedAddressString, ok := allowedAddressModel.(types.String)
			if !ok {
				return nil, fmt.Errorf("type assertion failed")
			}

			*allowedAddressesPayload = append(*allowedAddressesPayload, iaas.AllowedAddressesInner{
				String: conversion.StringValueToPointer(allowedAddressString),
			})
		}
	} else {
		allowedAddressesPayload = nil
	}

	if !model.Labels.IsNull() && !model.Labels.IsUnknown() {
		labelMap, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
		if err != nil {
			return nil, fmt.Errorf("mapping labels: %w", err)
		}
		labelPayload = &labelMap
	}

	return &iaas.CreateNicPayload{
		AllowedAddresses: allowedAddressesPayload,
		SecurityGroups:   &modelSecurityGroups,
		Labels:           labelPayload,
		Name:             conversion.StringValueToPointer(model.Name),
		Device:           conversion.StringValueToPointer(model.Device),
		Ipv4:             conversion.StringValueToPointer(model.IPv4),
		Mac:              conversion.StringValueToPointer(model.Mac),
		Type:             conversion.StringValueToPointer(model.Type),
		NicSecurity:      conversion.BoolValueToPointer(model.Security),
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.UpdateNicPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var labelPayload *map[string]interface{}

	modelSecurityGroups := []string{}
	for _, ns := range model.SecurityGroupIds.Elements() {
		securityGroupString, ok := ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelSecurityGroups = append(modelSecurityGroups, securityGroupString.ValueString())
	}

	allowedAddressesPayload := []iaas.AllowedAddressesInner{} // Even if null in the model, we need to send an empty list to the API since it's a PATCH endpoint
	if !(model.AllowedAddresses.IsNull() || model.AllowedAddresses.IsUnknown()) {
		for _, allowedAddressModel := range model.AllowedAddresses.Elements() {
			allowedAddressString, ok := allowedAddressModel.(types.String)
			if !ok {
				return nil, fmt.Errorf("type assertion failed")
			}

			allowedAddressesPayload = append(allowedAddressesPayload, iaas.AllowedAddressesInner{
				String: conversion.StringValueToPointer(allowedAddressString),
			})
		}
	}

	if !model.Labels.IsNull() && !model.Labels.IsUnknown() {
		labelMap, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
		if err != nil {
			return nil, fmt.Errorf("mapping labels: %w", err)
		}
		labelPayload = &labelMap
	}

	return &iaas.UpdateNicPayload{
		AllowedAddresses: &allowedAddressesPayload,
		SecurityGroups:   &modelSecurityGroups,
		Labels:           labelPayload,
		Name:             conversion.StringValueToPointer(model.Name),
		NicSecurity:      conversion.BoolValueToPointer(model.Security),
	}, nil
}
