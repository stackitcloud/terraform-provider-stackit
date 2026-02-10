package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkResource{}
	_ resource.ResourceWithConfigure   = &networkResource{}
	_ resource.ResourceWithImportState = &networkResource{}
	_ resource.ResourceWithModifyPlan  = &networkResource{}
)

const (
	ipv4BehaviorChangeTitle       = "Behavior of not configured `ipv4_nameservers` will change from January 2026"
	ipv4BehaviorChangeDescription = "When `ipv4_nameservers` is not set, it will be set to the network area's `default_nameservers`.\n" +
		"To prevent any nameserver configuration, the `ipv4_nameservers` attribute should be explicitly set to an empty list `[]`.\n" +
		"In cases where `ipv4_nameservers` are defined within the resource, the existing behavior will remain unchanged."
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
	NoIPv4Gateway    types.Bool   `tfsdk:"no_ipv4_gateway"`
	NoIPv6Gateway    types.Bool   `tfsdk:"no_ipv6_gateway"`
	Region           types.String `tfsdk:"region"`
	RoutingTableID   types.String `tfsdk:"routing_table_id"`
}

// NewNetworkResource is a helper function to simplify the provider implementation.
func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

// networkResource is the resource implementation.
type networkResource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *networkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

// Configure adds the provider configured client to the resource.
func (r *networkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "IaaS client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *networkResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	// Warning should only be shown during the plan of the creation. This can be detected by checking if the ID is set.
	if utils.IsUndefined(planModel.Id) && utils.IsUndefined(planModel.IPv4Nameservers) {
		addIPv4Warning(&resp.Diagnostics)
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

func (r *networkResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var resourceModel Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &resourceModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !resourceModel.Nameservers.IsUnknown() && !resourceModel.IPv4Nameservers.IsUnknown() && !resourceModel.Nameservers.IsNull() && !resourceModel.IPv4Nameservers.IsNull() {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring network", "You cannot provide both the `nameservers` and `ipv4_nameservers` fields simultaneously. Please remove the deprecated `nameservers` field, and use `ipv4_nameservers` to configure nameservers for IPv4.")
	}
}

// ConfigValidators validates the resource configuration
func (r *networkResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("no_ipv4_gateway"),
			path.MatchRoot("ipv4_gateway"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("no_ipv6_gateway"),
			path.MatchRoot("ipv6_gateway"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("ipv4_prefix"),
			path.MatchRoot("ipv4_prefix_length"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("ipv6_prefix"),
			path.MatchRoot("ipv6_prefix_length"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("ipv4_prefix_length"),
			path.MatchRoot("ipv4_gateway"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("ipv6_prefix_length"),
			path.MatchRoot("ipv6_gateway"),
		),
	}
}

// Schema defines the schema for the resource.
func (r *networkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Network resource schema. Must have a `region` specified in the provider configuration."
	descriptionNote := fmt.Sprintf("~> %s. %s", ipv4BehaviorChangeTitle, ipv4BehaviorChangeDescription)
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("%s\n%s", description, descriptionNote),
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`network_id`\".",
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
				Description:        "The nameservers of the network. This field is deprecated and will be removed in January 2026, use `ipv4_nameservers` to configure the nameservers for IPv4.",
				DeprecationMessage: "Use `ipv4_nameservers` to configure the nameservers for IPv4.",
				Optional:           true,
				Computed:           true,
				ElementType:        types.StringType,
			},
			"no_ipv4_gateway": schema.BoolAttribute{
				Description: "If set to `true`, the network doesn't have a gateway.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv4_gateway": schema.StringAttribute{
				Description: "The IPv4 gateway of a network. If not specified, the first IP of the network will be assigned as the gateway.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.IP(false),
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
				Computed:    true,
				Validators: []validator.String{
					validate.CIDR(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"ipv4_prefix_length": schema.Int64Attribute{
				Description: "The IPv4 prefix length of the network.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"prefixes": schema.ListAttribute{
				Description:        "The prefixes of the network. This field is deprecated and will be removed in January 2026, use `ipv4_prefixes` to read the prefixes of the IPv4 networks.",
				DeprecationMessage: "Use `ipv4_prefixes` to read the prefixes of the IPv4 networks.",
				Computed:           true,
				ElementType:        types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv4_prefixes": schema.ListAttribute{
				Description: "The IPv4 prefixes of the network.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"no_ipv6_gateway": schema.BoolAttribute{
				Description: "If set to `true`, the network doesn't have a gateway.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv6_gateway": schema.StringAttribute{
				Description: "The IPv6 gateway of a network. If not specified, the first IP of the network will be assigned as the gateway.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.IP(false),
				},
			},
			"ipv6_nameservers": schema.ListAttribute{
				Description: "The IPv6 nameservers of the network.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipv6_prefix": schema.StringAttribute{
				Description: "The IPv6 prefix of the network (CIDR).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.CIDR(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"ipv6_prefix_length": schema.Int64Attribute{
				Description: "The IPv6 prefix length of the network.",
				Optional:    true,
				Computed:    true,
			},
			"ipv6_prefixes": schema.ListAttribute{
				Description: "The IPv6 prefixes of the network.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"public_ip": schema.StringAttribute{
				Description: "The public IP of the network.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
					boolplanmodifier.RequiresReplace(),
				},
			},
			"routing_table_id": schema.StringAttribute{
				Description: "The ID of the routing table associated with the network.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
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

	// When IPv4Nameserver is not set, print warning that the behavior of ipv4_nameservers will change
	if utils.IsUndefined(model.IPv4Nameservers) {
		addIPv4Warning(&resp.Diagnostics)
	}

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network

	network, err := r.client.CreateNetwork(ctx, projectId, region).CreateNetworkPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Calling API: %v", err))
		return
	}

	networkId := *network.Id
	ctx = tflog.SetField(ctx, "network_id", networkId)

	network, err = wait.CreateNetworkWaitHandler(ctx, r.client, projectId, region, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Network creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, network, &model, region)
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
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "region", region)

	networkResp, err := r.client.GetNetwork(ctx, projectId, region, networkId).Execute()
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
	err = mapFields(ctx, networkResp, &model, region)
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
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "region", region)

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
	err = r.client.PartialUpdateNetwork(ctx, projectId, region, networkId).PartialUpdateNetworkPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.UpdateNetworkWaitHandler(ctx, r.client, projectId, region, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Network update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model, region)
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
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing network
	err := r.client.DeleteNetwork(ctx, projectId, region, networkId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteNetworkWaitHandler(ctx, r.client, projectId, region, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network", fmt.Sprintf("Network deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Network deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,region,network_id
func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[network_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	region := idParts[1]
	networkId := idParts[2]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), region)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_id"), networkId)...)
	tflog.Info(ctx, "Network state imported")
}

func mapFields(ctx context.Context, networkResp *iaas.Network, model *Model, region string) error {
	if networkResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkId string
	if model.NetworkId.ValueString() != "" {
		networkId = model.NetworkId.ValueString()
	} else if networkResp.Id != nil {
		networkId = *networkResp.Id
	} else {
		return fmt.Errorf("network id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, networkId)

	labels, err := iaasUtils.MapLabels(ctx, networkResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	// IPv4

	if networkResp.Ipv4 == nil || networkResp.Ipv4.Nameservers == nil {
		model.Nameservers = types.ListNull(types.StringType)
		model.IPv4Nameservers = types.ListNull(types.StringType)
	} else {
		respNameservers := *networkResp.Ipv4.Nameservers
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

	model.IPv4PrefixLength = types.Int64Null()
	if networkResp.Ipv4 == nil || networkResp.Ipv4.Prefixes == nil {
		model.Prefixes = types.ListNull(types.StringType)
		model.IPv4Prefixes = types.ListNull(types.StringType)
	} else {
		respPrefixes := *networkResp.Ipv4.Prefixes
		prefixesTF, diags := types.ListValueFrom(ctx, types.StringType, respPrefixes)
		if diags.HasError() {
			return fmt.Errorf("map network prefixes: %w", core.DiagsToError(diags))
		}
		if len(respPrefixes) > 0 {
			model.IPv4Prefix = types.StringValue(respPrefixes[0])
			_, netmask, err := net.ParseCIDR(respPrefixes[0])
			if err != nil {
				tflog.Error(ctx, fmt.Sprintf("ipv4_prefix_length: %+v", err))
				// silently ignore parsing error for the netmask
				model.IPv4PrefixLength = types.Int64Null()
			} else {
				ones, _ := netmask.Mask.Size()
				model.IPv4PrefixLength = types.Int64Value(int64(ones))
			}
		}

		model.Prefixes = prefixesTF
		model.IPv4Prefixes = prefixesTF
	}

	if networkResp.Ipv4 == nil || networkResp.Ipv4.Gateway == nil {
		model.IPv4Gateway = types.StringNull()
	} else {
		model.IPv4Gateway = types.StringPointerValue(networkResp.Ipv4.GetGateway())
	}

	if networkResp.Ipv4 == nil || networkResp.Ipv4.PublicIp == nil {
		model.PublicIP = types.StringNull()
	} else {
		model.PublicIP = types.StringPointerValue(networkResp.Ipv4.PublicIp)
	}

	// IPv6

	if networkResp.Ipv6 == nil || networkResp.Ipv6.Nameservers == nil {
		model.IPv6Nameservers = types.ListNull(types.StringType)
	} else {
		respIPv6Nameservers := *networkResp.Ipv6.Nameservers
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

	model.IPv6PrefixLength = types.Int64Null()
	model.IPv6Prefix = types.StringNull()
	if networkResp.Ipv6 == nil || networkResp.Ipv6.Prefixes == nil {
		model.IPv6Prefixes = types.ListNull(types.StringType)
	} else {
		respPrefixesV6 := *networkResp.Ipv6.Prefixes
		prefixesV6TF, diags := types.ListValueFrom(ctx, types.StringType, respPrefixesV6)
		if diags.HasError() {
			return fmt.Errorf("map network IPv6 prefixes: %w", core.DiagsToError(diags))
		}
		if len(respPrefixesV6) > 0 {
			model.IPv6Prefix = types.StringValue(respPrefixesV6[0])
			_, netmask, err := net.ParseCIDR(respPrefixesV6[0])
			if err != nil {
				// silently ignore parsing error for the netmask
				model.IPv6PrefixLength = types.Int64Null()
			} else {
				ones, _ := netmask.Mask.Size()
				model.IPv6PrefixLength = types.Int64Value(int64(ones))
			}
		}
		model.IPv6Prefixes = prefixesV6TF
	}

	if networkResp.Ipv6 == nil || networkResp.Ipv6.Gateway == nil {
		model.IPv6Gateway = types.StringNull()
	} else {
		model.IPv6Gateway = types.StringPointerValue(networkResp.Ipv6.GetGateway())
	}

	model.RoutingTableID = types.StringPointerValue(networkResp.RoutingTableId)
	model.NetworkId = types.StringValue(networkId)
	model.Name = types.StringPointerValue(networkResp.Name)
	model.Labels = labels
	model.Routed = types.BoolPointerValue(networkResp.Routed)
	model.Region = types.StringValue(region)

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateNetworkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var modelIPv6Nameservers []string
	// Is true when IPv6Nameservers is not null or unset
	if !utils.IsUndefined(model.IPv6Nameservers) {
		// If ipv6Nameservers is empty, modelIPv6Nameservers will be set to an empty slice.
		// empty slice != nil slice. Empty slice will result in an empty list in the payload []. Nil slice will result in a payload without the property set
		modelIPv6Nameservers = []string{}
		for _, ipv6ns := range model.IPv6Nameservers.Elements() {
			ipv6NameserverString, ok := ipv6ns.(types.String)
			if !ok {
				return nil, fmt.Errorf("type assertion failed")
			}
			modelIPv6Nameservers = append(modelIPv6Nameservers, ipv6NameserverString.ValueString())
		}
	}

	var ipv6Body *iaas.CreateNetworkIPv6
	if !utils.IsUndefined(model.IPv6PrefixLength) {
		ipv6Body = &iaas.CreateNetworkIPv6{
			CreateNetworkIPv6WithPrefixLength: &iaas.CreateNetworkIPv6WithPrefixLength{
				PrefixLength: conversion.Int64ValueToPointer(model.IPv6PrefixLength),
			},
		}

		// IPv6 nameservers should only be set, if it contains any value. If the slice is nil, it should NOT be set.
		// Setting it to a nil slice would result in a payload, where nameservers is set to null in the json payload,
		// but it should actually be unset. Setting it to "null" will result in an error, because it's NOT nullable.
		if modelIPv6Nameservers != nil {
			ipv6Body.CreateNetworkIPv6WithPrefixLength.Nameservers = &modelIPv6Nameservers
		}
	} else if !utils.IsUndefined(model.IPv6Prefix) {
		var gateway *iaas.NullableString
		if model.NoIPv6Gateway.ValueBool() {
			gateway = iaas.NewNullableString(nil)
		} else if !(model.IPv6Gateway.IsUnknown() || model.IPv6Gateway.IsNull()) {
			gateway = iaas.NewNullableString(conversion.StringValueToPointer(model.IPv6Gateway))
		}

		ipv6Body = &iaas.CreateNetworkIPv6{
			CreateNetworkIPv6WithPrefix: &iaas.CreateNetworkIPv6WithPrefix{
				Gateway: gateway,
				Prefix:  conversion.StringValueToPointer(model.IPv6Prefix),
			},
		}

		// IPv6 nameservers should only be set, if it contains any value. If the slice is nil, it should NOT be set.
		// Setting it to a nil slice would result in a payload, where nameservers is set to null in the json payload,
		// but it should actually be unset. Setting it to "null" will result in an error, because it's NOT nullable.
		if modelIPv6Nameservers != nil {
			ipv6Body.CreateNetworkIPv6WithPrefix.Nameservers = &modelIPv6Nameservers
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

	var ipv4Body *iaas.CreateNetworkIPv4
	if !utils.IsUndefined(model.IPv4PrefixLength) {
		ipv4Body = &iaas.CreateNetworkIPv4{
			CreateNetworkIPv4WithPrefixLength: &iaas.CreateNetworkIPv4WithPrefixLength{
				Nameservers:  &modelIPv4Nameservers,
				PrefixLength: conversion.Int64ValueToPointer(model.IPv4PrefixLength),
			},
		}
	} else if !utils.IsUndefined(model.IPv4Prefix) {
		var gateway *iaas.NullableString
		if model.NoIPv4Gateway.ValueBool() {
			gateway = iaas.NewNullableString(nil)
		} else if !(model.IPv4Gateway.IsUnknown() || model.IPv4Gateway.IsNull()) {
			gateway = iaas.NewNullableString(conversion.StringValueToPointer(model.IPv4Gateway))
		}

		ipv4Body = &iaas.CreateNetworkIPv4{
			CreateNetworkIPv4WithPrefix: &iaas.CreateNetworkIPv4WithPrefix{
				Nameservers: &modelIPv4Nameservers,
				Prefix:      conversion.StringValueToPointer(model.IPv4Prefix),
				Gateway:     gateway,
			},
		}
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	payload := iaas.CreateNetworkPayload{
		Name:           conversion.StringValueToPointer(model.Name),
		Labels:         &labels,
		Routed:         conversion.BoolValueToPointer(model.Routed),
		Ipv4:           ipv4Body,
		Ipv6:           ipv6Body,
		RoutingTableId: conversion.StringValueToPointer(model.RoutingTableID),
	}

	return &payload, nil
}

func toUpdatePayload(ctx context.Context, model, stateModel *Model) (*iaas.PartialUpdateNetworkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var modelIPv6Nameservers []string
	// Is true when IPv6Nameservers is not null or unset
	if !utils.IsUndefined(model.IPv6Nameservers) {
		// If ipv6Nameservers is empty, modelIPv6Nameservers will be set to an empty slice.
		// empty slice != nil slice. Empty slice will result in an empty list in the payload []. Nil slice will result in a payload without the property set
		modelIPv6Nameservers = []string{}
		for _, ipv6ns := range model.IPv6Nameservers.Elements() {
			ipv6NameserverString, ok := ipv6ns.(types.String)
			if !ok {
				return nil, fmt.Errorf("type assertion failed")
			}
			modelIPv6Nameservers = append(modelIPv6Nameservers, ipv6NameserverString.ValueString())
		}
	}

	var ipv6Body *iaas.UpdateNetworkIPv6Body
	if modelIPv6Nameservers != nil || !utils.IsUndefined(model.NoIPv6Gateway) || !utils.IsUndefined(model.IPv6Gateway) {
		ipv6Body = &iaas.UpdateNetworkIPv6Body{}
		// IPv6 nameservers should only be set, if it contains any value. If the slice is nil, it should NOT be set.
		// Setting it to a nil slice would result in a payload, where nameservers is set to null in the json payload,
		// but it should actually be unset. Setting it to "null" will result in an error, because it's NOT nullable.
		if modelIPv6Nameservers != nil {
			ipv6Body.Nameservers = &modelIPv6Nameservers
		}

		if model.NoIPv6Gateway.ValueBool() {
			ipv6Body.Gateway = iaas.NewNullableString(nil)
		} else if !(model.IPv6Gateway.IsUnknown() || model.IPv6Gateway.IsNull()) {
			ipv6Body.Gateway = iaas.NewNullableString(conversion.StringValueToPointer(model.IPv6Gateway))
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

	var ipv4Body *iaas.UpdateNetworkIPv4Body
	if !model.IPv4Nameservers.IsNull() || !model.Nameservers.IsNull() {
		ipv4Body = &iaas.UpdateNetworkIPv4Body{
			Nameservers: &modelIPv4Nameservers,
		}

		if model.NoIPv4Gateway.ValueBool() {
			ipv4Body.Gateway = iaas.NewNullableString(nil)
		} else if !(model.IPv4Gateway.IsUnknown() || model.IPv4Gateway.IsNull()) {
			ipv4Body.Gateway = iaas.NewNullableString(conversion.StringValueToPointer(model.IPv4Gateway))
		}
	}
	currentLabels := stateModel.Labels
	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	payload := iaas.PartialUpdateNetworkPayload{
		Name:           conversion.StringValueToPointer(model.Name),
		Labels:         &labels,
		Ipv4:           ipv4Body,
		Ipv6:           ipv6Body,
		RoutingTableId: conversion.StringValueToPointer(model.RoutingTableID),
	}

	return &payload, nil
}

func addIPv4Warning(diags *diag.Diagnostics) {
	diags.AddAttributeWarning(path.Root("ipv4_nameservers"),
		ipv4BehaviorChangeTitle,
		ipv4BehaviorChangeDescription)
}
