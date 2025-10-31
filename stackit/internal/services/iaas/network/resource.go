package network

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/model"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/v1network"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/v2network"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkResource{}
	_ resource.ResourceWithConfigure   = &networkResource{}
	_ resource.ResourceWithImportState = &networkResource{}
)

const (
	ipv4BehaviorChangeTitle       = "Behavior of not configured `ipv4_nameservers` will change from January 2026"
	ipv4BehaviorChangeDescription = "When `ipv4_nameservers` is not set, it will be set to the network area's `default_nameservers`.\n" +
		"To prevent any nameserver configuration, the `ipv4_nameservers` attribute should be explicitly set to an empty list `[]`.\n" +
		"In cases where `ipv4_nameservers` are defined within the resource, the existing behavior will remain unchanged."
)

// NewNetworkResource is a helper function to simplify the provider implementation.
func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

// networkResource is the resource implementation.
type networkResource struct {
	client *iaas.APIClient
	// alphaClient will be used in case the experimental flag "network" is set
	alphaClient    *iaasalpha.APIClient
	isExperimental bool
	providerData   core.ProviderData
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

	r.isExperimental = features.CheckExperimentEnabledWithoutError(ctx, &r.providerData, features.NetworkExperiment, "stackit_network", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.isExperimental {
		alphaApiClient := iaasAlphaUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		r.alphaClient = alphaApiClient
	} else {
		apiClient := iaasUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		r.client = apiClient
	}
	tflog.Info(ctx, "IaaS client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *networkResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel model.Model
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel model.Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Warning should only be shown during the plan of the creation. This can be detected by checking if the ID is set.
	if utils.IsUndefined(planModel.Id) && utils.IsUndefined(planModel.IPv4Nameservers) {
		addIPv4Warning(&resp.Diagnostics)
	}

	// If the v1 api is used, it's not required to get the fallback region because it isn't used
	if !r.isExperimental {
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

func (r *networkResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var resourceModel model.Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &resourceModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !resourceModel.Nameservers.IsUnknown() && !resourceModel.IPv4Nameservers.IsUnknown() && !resourceModel.Nameservers.IsNull() && !resourceModel.IPv4Nameservers.IsNull() {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring network", "You cannot provide both the `nameservers` and `ipv4_nameservers` fields simultaneously. Please remove the deprecated `nameservers` field, and use `ipv4_nameservers` to configure nameservers for IPv4.")
	}
	if !r.isExperimental {
		if !utils.IsUndefined(resourceModel.Region) {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring network", "Setting the `region` is not supported yet. This can only be configured when the experiments `network` is set.")
		}
		if !utils.IsUndefined(resourceModel.RoutingTableID) {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring network", "Setting the field `routing_table_id` is not supported yet. This can only be configured when the experiments `network` is set.")
		}
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
		Description:         "Network resource schema. Must have a `region` specified in the provider configuration.",
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
					stringplanmodifier.RequiresReplace(),
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
				Description: "Can only be used when experimental \"network\" is set.\nThe ID of the routing table associated with the network.",
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
				Description: "Can only be used when experimental \"network\" is set.\nThe resource region. If not defined, the provider region is used.",
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
	var planModel model.Model
	diags := req.Plan.Get(ctx, &planModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// When IPv4Nameserver is not set, print warning that the behavior of ipv4_nameservers will change
	if utils.IsUndefined(planModel.IPv4Nameservers) {
		addIPv4Warning(&resp.Diagnostics)
	}

	if !r.isExperimental {
		v1network.Create(ctx, req, resp, r.client)
	} else {
		v2network.Create(ctx, req, resp, r.alphaClient)
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	if !r.isExperimental {
		v1network.Read(ctx, req, resp, r.client)
	} else {
		v2network.Read(ctx, req, resp, r.alphaClient, r.providerData)
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	if !r.isExperimental {
		v1network.Update(ctx, req, resp, r.client)
	} else {
		v2network.Update(ctx, req, resp, r.alphaClient)
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	if !r.isExperimental {
		v1network.Delete(ctx, req, resp, r.client)
	} else {
		v2network.Delete(ctx, req, resp, r.alphaClient)
	}
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,network_id
func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.isExperimental {
		v1network.ImportState(ctx, req, resp)
	} else {
		v2network.ImportState(ctx, req, resp)
	}
}

func addIPv4Warning(diags *diag.Diagnostics) {
	diags.AddAttributeWarning(path.Root("ipv4_nameservers"),
		ipv4BehaviorChangeTitle,
		ipv4BehaviorChangeDescription)
}
