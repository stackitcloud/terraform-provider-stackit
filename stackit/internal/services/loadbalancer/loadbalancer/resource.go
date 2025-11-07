package loadbalancer

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"

	loadbalancerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/loadbalancer/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &loadBalancerResource{}
	_ resource.ResourceWithConfigure   = &loadBalancerResource{}
	_ resource.ResourceWithImportState = &loadBalancerResource{}
	_ resource.ResourceWithModifyPlan  = &loadBalancerResource{}
)

type Model struct {
	Id                             types.String `tfsdk:"id"` // needed by TF
	ProjectId                      types.String `tfsdk:"project_id"`
	ExternalAddress                types.String `tfsdk:"external_address"`
	DisableSecurityGroupAssignment types.Bool   `tfsdk:"disable_security_group_assignment"`
	Listeners                      types.List   `tfsdk:"listeners"`
	Name                           types.String `tfsdk:"name"`
	PlanId                         types.String `tfsdk:"plan_id"`
	Networks                       types.List   `tfsdk:"networks"`
	Options                        types.Object `tfsdk:"options"`
	PrivateAddress                 types.String `tfsdk:"private_address"`
	TargetPools                    types.List   `tfsdk:"target_pools"`
	Region                         types.String `tfsdk:"region"`
	SecurityGroupId                types.String `tfsdk:"security_group_id"`
}

// Struct corresponding to Model.Listeners[i]
type listener struct {
	DisplayName          types.String `tfsdk:"display_name"`
	Port                 types.Int64  `tfsdk:"port"`
	Protocol             types.String `tfsdk:"protocol"`
	ServerNameIndicators types.List   `tfsdk:"server_name_indicators"`
	TargetPool           types.String `tfsdk:"target_pool"`
	TCP                  types.Object `tfsdk:"tcp"`
	UDP                  types.Object `tfsdk:"udp"`
}

// Types corresponding to listener
var listenerTypes = map[string]attr.Type{
	"display_name":           types.StringType,
	"port":                   types.Int64Type,
	"protocol":               types.StringType,
	"server_name_indicators": types.ListType{ElemType: types.ObjectType{AttrTypes: serverNameIndicatorTypes}},
	"target_pool":            types.StringType,
	"tcp":                    types.ObjectType{AttrTypes: tcpTypes},
	"udp":                    types.ObjectType{AttrTypes: udpTypes},
}

// Struct corresponding to listener.ServerNameIndicators[i]
type serverNameIndicator struct {
	Name types.String `tfsdk:"name"`
}

// Types corresponding to serverNameIndicator
var serverNameIndicatorTypes = map[string]attr.Type{
	"name": types.StringType,
}

type tcp struct {
	IdleTimeout types.String `tfsdk:"idle_timeout"`
}

var tcpTypes = map[string]attr.Type{
	"idle_timeout": types.StringType,
}

type udp struct {
	IdleTimeout types.String `tfsdk:"idle_timeout"`
}

var udpTypes = map[string]attr.Type{
	"idle_timeout": types.StringType,
}

// Struct corresponding to Model.Networks[i]
type network struct {
	NetworkId types.String `tfsdk:"network_id"`
	Role      types.String `tfsdk:"role"`
}

// Types corresponding to network
var networkTypes = map[string]attr.Type{
	"network_id": types.StringType,
	"role":       types.StringType,
}

// Struct corresponding to Model.Options
type options struct {
	ACL                types.Set    `tfsdk:"acl"`
	PrivateNetworkOnly types.Bool   `tfsdk:"private_network_only"`
	Observability      types.Object `tfsdk:"observability"`
}

// Types corresponding to options
var optionsTypes = map[string]attr.Type{
	"acl":                  types.SetType{ElemType: types.StringType},
	"private_network_only": types.BoolType,
	"observability":        types.ObjectType{AttrTypes: observabilityTypes},
}

type observability struct {
	Logs    types.Object `tfsdk:"logs"`
	Metrics types.Object `tfsdk:"metrics"`
}

var observabilityTypes = map[string]attr.Type{
	"logs":    types.ObjectType{AttrTypes: observabilityOptionTypes},
	"metrics": types.ObjectType{AttrTypes: observabilityOptionTypes},
}

type observabilityOption struct {
	CredentialsRef types.String `tfsdk:"credentials_ref"`
	PushUrl        types.String `tfsdk:"push_url"`
}

var observabilityOptionTypes = map[string]attr.Type{
	"credentials_ref": types.StringType,
	"push_url":        types.StringType,
}

// Struct corresponding to Model.TargetPools[i]
type targetPool struct {
	ActiveHealthCheck  types.Object `tfsdk:"active_health_check"`
	Name               types.String `tfsdk:"name"`
	TargetPort         types.Int64  `tfsdk:"target_port"`
	Targets            types.List   `tfsdk:"targets"`
	SessionPersistence types.Object `tfsdk:"session_persistence"`
}

// Types corresponding to targetPool
var targetPoolTypes = map[string]attr.Type{
	"active_health_check": types.ObjectType{AttrTypes: activeHealthCheckTypes},
	"name":                types.StringType,
	"target_port":         types.Int64Type,
	"targets":             types.ListType{ElemType: types.ObjectType{AttrTypes: targetTypes}},
	"session_persistence": types.ObjectType{AttrTypes: sessionPersistenceTypes},
}

// Struct corresponding to targetPool.ActiveHealthCheck
type activeHealthCheck struct {
	HealthyThreshold   types.Int64  `tfsdk:"healthy_threshold"`
	Interval           types.String `tfsdk:"interval"`
	IntervalJitter     types.String `tfsdk:"interval_jitter"`
	Timeout            types.String `tfsdk:"timeout"`
	UnhealthyThreshold types.Int64  `tfsdk:"unhealthy_threshold"`
}

// Types corresponding to activeHealthCheck
var activeHealthCheckTypes = map[string]attr.Type{
	"healthy_threshold":   types.Int64Type,
	"interval":            types.StringType,
	"interval_jitter":     types.StringType,
	"timeout":             types.StringType,
	"unhealthy_threshold": types.Int64Type,
}

// Struct corresponding to targetPool.Targets[i]
type target struct {
	DisplayName types.String `tfsdk:"display_name"`
	Ip          types.String `tfsdk:"ip"`
}

// Types corresponding to target
var targetTypes = map[string]attr.Type{
	"display_name": types.StringType,
	"ip":           types.StringType,
}

// Struct corresponding to targetPool.SessionPersistence
type sessionPersistence struct {
	UseSourceIPAddress types.Bool `tfsdk:"use_source_ip_address"`
}

// Types corresponding to SessionPersistence
var sessionPersistenceTypes = map[string]attr.Type{
	"use_source_ip_address": types.BoolType,
}

// NewLoadBalancerResource is a helper function to simplify the provider implementation.
func NewLoadBalancerResource() resource.Resource {
	return &loadBalancerResource{}
}

// loadBalancerResource is the resource implementation.
type loadBalancerResource struct {
	client       *loadbalancer.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *loadBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancer"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *loadBalancerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

// ConfigValidators validates the resource configuration
func (r *loadBalancerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// validation is done in extracted func so it's easier to unit-test it
	validateConfig(ctx, &resp.Diagnostics, &model)
}

func validateConfig(ctx context.Context, diags *diag.Diagnostics, model *Model) {
	externalAddressIsSet := !model.ExternalAddress.IsNull()

	lbOptions, err := toOptionsPayload(ctx, model)
	if err != nil || lbOptions == nil {
		// private_network_only is not set and external_address is not set
		if !externalAddressIsSet {
			core.LogAndAddError(ctx, diags, "Error configuring load balancer", fmt.Sprintf("You need to provide either the `options.private_network_only = true` or `external_address` field. %v", err))
		}
		return
	}
	if lbOptions.PrivateNetworkOnly == nil || !*lbOptions.PrivateNetworkOnly {
		// private_network_only is not set or false and external_address is not set
		if !externalAddressIsSet {
			core.LogAndAddError(ctx, diags, "Error configuring load balancer", "You need to provide either the `options.private_network_only = true` or `external_address` field.")
		}
		return
	}

	// Both are set
	if *lbOptions.PrivateNetworkOnly && externalAddressIsSet {
		core.LogAndAddError(ctx, diags, "Error configuring load balancer", "You need to provide either the `options.private_network_only = true` or `external_address` field.")
	}
}

// Configure adds the provider configured client to the resource.
func (r *loadBalancerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := loadbalancerUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Load Balancer client configured")
}

// Schema defines the schema for the resource.
func (r *loadBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	protocolOptions := []string{"PROTOCOL_UNSPECIFIED", "PROTOCOL_TCP", "PROTOCOL_UDP", "PROTOCOL_TCP_PROXY", "PROTOCOL_TLS_PASSTHROUGH"}
	roleOptions := []string{"ROLE_UNSPECIFIED", "ROLE_LISTENERS_AND_TARGETS", "ROLE_LISTENERS", "ROLE_TARGETS"}
	servicePlanOptions := []string{"p10", "p50", "p250", "p750"}

	descriptions := map[string]string{
		"main":                                  "Load Balancer resource schema.",
		"id":                                    "Terraform's internal resource ID. It is structured as \"`project_id`\",\"region\",\"`name`\".",
		"project_id":                            "STACKIT project ID to which the Load Balancer is associated.",
		"external_address":                      "External Load Balancer IP address where this Load Balancer is exposed.",
		"disable_security_group_assignment":     "If set to true, this will disable the automatic assignment of a security group to the load balancer's targets. This option is primarily used to allow targets that are not within the load balancer's own network or SNA (STACKIT network area). When this is enabled, you are fully responsible for ensuring network connectivity to the targets, including managing all routing and security group rules manually. This setting cannot be changed after the load balancer is created.",
		"listeners":                             "List of all listeners which will accept traffic. Limited to 20.",
		"port":                                  "Port number where we listen for traffic.",
		"protocol":                              "Protocol is the highest network protocol we understand to load balance. " + utils.FormatPossibleValues(protocolOptions...),
		"target_pool":                           "Reference target pool by target pool name.",
		"name":                                  "Load balancer name.",
		"plan_id":                               "The service plan ID. If not defined, the default service plan is `p10`. " + utils.FormatPossibleValues(servicePlanOptions...),
		"networks":                              "List of networks that listeners and targets reside in.",
		"network_id":                            "Openstack network ID.",
		"role":                                  "The role defines how the load balancer is using the network. " + utils.FormatPossibleValues(roleOptions...),
		"observability":                         "We offer Load Balancer metrics observability via ARGUS or external solutions. Not changeable after creation.",
		"observability_logs":                    "Observability logs configuration. Not changeable after creation.",
		"observability_logs_credentials_ref":    "Credentials reference for logs. Not changeable after creation.",
		"observability_logs_push_url":           "The ARGUS/Loki remote write Push URL to ship the logs to. Not changeable after creation.",
		"observability_metrics":                 "Observability metrics configuration. Not changeable after creation.",
		"observability_metrics_credentials_ref": "Credentials reference for metrics. Not changeable after creation.",
		"observability_metrics_push_url":        "The ARGUS/Prometheus remote write Push URL to ship the metrics to. Not changeable after creation.",
		"options":                               "Defines any optional functionality you want to have enabled on your load balancer.",
		"acl":                                   "Load Balancer is accessible only from an IP address in this range.",
		"private_network_only":                  "If true, Load Balancer is accessible only via a private network IP address.",
		"session_persistence":                   "Here you can setup various session persistence options, so far only \"`use_source_ip_address`\" is supported.",
		"use_source_ip_address":                 "If true then all connections from one source IP address are redirected to the same target. This setting changes the load balancing algorithm to Maglev.",
		"server_name_indicators":                "A list of domain names to match in order to pass TLS traffic to the target pool in the current listener",
		"server_name_indicators.name":           "A domain name to match in order to pass TLS traffic to the target pool in the current listener",
		"private_address":                       "Transient private Load Balancer IP address. It can change any time.",
		"target_pools":                          "List of all target pools which will be used in the Load Balancer. Limited to 20.",
		"healthy_threshold":                     "Healthy threshold of the health checking.",
		"interval":                              "Interval duration of health checking in seconds.",
		"interval_jitter":                       "Interval duration threshold of the health checking in seconds.",
		"timeout":                               "Active health checking timeout duration in seconds.",
		"unhealthy_threshold":                   "Unhealthy threshold of the health checking.",
		"target_pools.name":                     "Target pool name.",
		"target_port":                           "Identical port number where each target listens for traffic.",
		"targets":                               "List of all targets which will be used in the pool. Limited to 1000.",
		"targets.display_name":                  "Target display name",
		"ip":                                    "Target IP",
		"region":                                "The resource region. If not defined, the provider region is used.",
		"security_group_id":                     "The ID of the egress security group assigned to the Load Balancer's internal machines. This ID is essential for allowing traffic from the Load Balancer to targets in different networks or STACKIT network areas (SNA). To enable this, create a security group rule for your target VMs and set the `remote_security_group_id` of that rule to this value. This is typically used when `disable_security_group_assignment` is set to `true`.",
		"tcp_options":                           "Options that are specific to the TCP protocol.",
		"tcp_options_idle_timeout":              "Time after which an idle connection is closed. The default value is set to 300 seconds, and the maximum value is 3600 seconds. The format is a duration and the unit must be seconds. Example: 30s",
		"udp_options":                           "Options that are specific to the UDP protocol.",
		"udp_options_idle_timeout":              "Time after which an idle session is closed. The default value is set to 1 minute, and the maximum value is 2 minutes. The format is a duration and the unit must be seconds. Example: 30s",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		MarkdownDescription: `
## Setting up supporting infrastructure` + "\n" + `

The example below creates the supporting infrastructure using the STACKIT Terraform provider, including the network, network interface, a public IP address and server resources.
`,
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
				},
			},
			"external_address": schema.StringAttribute{
				Description: descriptions["external_address"],
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disable_security_group_assignment": schema.BoolAttribute{
				Description: descriptions["disable_security_group_assignment"],
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"plan_id": schema.StringAttribute{
				Description: descriptions["plan_id"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"listeners": schema.ListNestedAttribute{
				Description: descriptions["listeners"],
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 20),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"display_name": schema.StringAttribute{
							Description: descriptions["listeners.display_name"],
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"port": schema.Int64Attribute{
							Description: descriptions["port"],
							Required:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.RequiresReplace(),
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"protocol": schema.StringAttribute{
							Description: descriptions["protocol"],
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
								stringplanmodifier.UseStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf(protocolOptions...),
							},
						},
						"server_name_indicators": schema.ListNestedAttribute{
							Description: descriptions["server_name_indicators"],
							Optional:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Description: descriptions["server_name_indicators.name"],
										Optional:    true,
									},
								},
							},
						},
						"target_pool": schema.StringAttribute{
							Description: descriptions["target_pool"],
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"tcp": schema.SingleNestedAttribute{
							Description: descriptions["tcp_options"],
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"idle_timeout": schema.StringAttribute{
									Description: descriptions["tcp_options_idle_timeout"],
									Optional:    true,
								},
							},
						},
						"udp": schema.SingleNestedAttribute{
							Description: descriptions["udp_options"],
							Optional:    true,
							Computed:    false,
							Attributes: map[string]schema.Attribute{
								"idle_timeout": schema.StringAttribute{
									Description: descriptions["udp_options_idle_timeout"],
									Optional:    true,
								},
							},
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					validate.NoSeparator(),
				},
			},
			"networks": schema.ListNestedAttribute{
				Description: descriptions["networks"],
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{
							Description: descriptions["network_id"],
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								validate.UUID(),
								validate.NoSeparator(),
							},
						},
						"role": schema.StringAttribute{
							Description: descriptions["role"],
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
								stringplanmodifier.UseStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf(roleOptions...),
							},
						},
					},
				},
			},
			"options": schema.SingleNestedAttribute{
				Description: descriptions["options"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"acl": schema.SetAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.RequiresReplace(),
							setplanmodifier.UseStateForUnknown(),
						},
						Validators: []validator.Set{
							setvalidator.ValueStringsAre(
								validate.CIDR(),
							),
						},
					},
					"private_network_only": schema.BoolAttribute{
						Description: descriptions["private_network_only"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"observability": schema.SingleNestedAttribute{
						Description: descriptions["observability"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Object{
							// API docs says observability options are not changeable after creation
							objectplanmodifier.RequiresReplace(),
						},
						Attributes: map[string]schema.Attribute{
							"logs": schema.SingleNestedAttribute{
								Description: descriptions["observability_logs"],
								Optional:    true,
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"credentials_ref": schema.StringAttribute{
										Description: descriptions["observability_logs_credentials_ref"],
										Optional:    true,
										Computed:    true,
									},
									"push_url": schema.StringAttribute{
										Description: descriptions["observability_logs_credentials_ref"],
										Optional:    true,
										Computed:    true,
									},
								},
							},
							"metrics": schema.SingleNestedAttribute{
								Description: descriptions["observability_metrics"],
								Optional:    true,
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"credentials_ref": schema.StringAttribute{
										Description: descriptions["observability_metrics_credentials_ref"],
										Optional:    true,
										Computed:    true,
									},
									"push_url": schema.StringAttribute{
										Description: descriptions["observability_metrics_credentials_ref"],
										Optional:    true,
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
			"private_address": schema.StringAttribute{
				Description: descriptions["private_address"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"target_pools": schema.ListNestedAttribute{
				Description: descriptions["target_pools"],
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 20),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"active_health_check": schema.SingleNestedAttribute{
							Description: descriptions["active_health_check"],
							Optional:    true,
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"healthy_threshold": schema.Int64Attribute{
									Description: descriptions["healthy_threshold"],
									Optional:    true,
									Computed:    true,
								},
								"interval": schema.StringAttribute{
									Description: descriptions["interval"],
									Optional:    true,
									Computed:    true,
								},
								"interval_jitter": schema.StringAttribute{
									Description: descriptions["interval_jitter"],
									Optional:    true,
									Computed:    true,
								},
								"timeout": schema.StringAttribute{
									Description: descriptions["timeout"],
									Optional:    true,
									Computed:    true,
								},
								"unhealthy_threshold": schema.Int64Attribute{
									Description: descriptions["unhealthy_threshold"],
									Optional:    true,
									Computed:    true,
								},
							},
						},
						"name": schema.StringAttribute{
							Description: descriptions["target_pools.name"],
							Required:    true,
						},
						"target_port": schema.Int64Attribute{
							Description: descriptions["target_port"],
							Required:    true,
						},
						"session_persistence": schema.SingleNestedAttribute{
							Description: descriptions["session_persistence"],
							Optional:    true,
							Computed:    false,
							Attributes: map[string]schema.Attribute{
								"use_source_ip_address": schema.BoolAttribute{
									Description: descriptions["use_source_ip_address"],
									Optional:    true,
									Computed:    false,
								},
							},
						},
						"targets": schema.ListNestedAttribute{
							Description: descriptions["targets"],
							Required:    true,
							Validators: []validator.List{
								listvalidator.SizeBetween(1, 1000),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"display_name": schema.StringAttribute{
										Description: descriptions["targets.display_name"],
										Required:    true,
									},
									"ip": schema.StringAttribute{
										Description: descriptions["ip"],
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"security_group_id": schema.StringAttribute{
				Description: descriptions["security_group_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *loadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create a new load balancer
	createResp, err := r.client.CreateLoadBalancer(ctx, projectId, region).CreateLoadBalancerPayload(*payload).XRequestID(uuid.NewString()).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Calling API: %v", err))
		return
	}

	waitResp, err := wait.CreateLoadBalancerWaitHandler(ctx, r.client, projectId, region, *createResp.Name).SetTimeout(90 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Load balancer creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Load balancer created")
}

// Read refreshes the Terraform state with the latest data.
func (r *loadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)
	ctx = tflog.SetField(ctx, "region", region)

	lbResp, err := r.client.GetLoadBalancer(ctx, projectId, region, name).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading load balancer", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, lbResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading load balancer", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Load balancer read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *loadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)
	ctx = tflog.SetField(ctx, "region", region)

	targetPoolsModel := []targetPool{}
	diags = model.TargetPools.ElementsAs(ctx, &targetPoolsModel, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	for i := range targetPoolsModel {
		targetPoolModel := targetPoolsModel[i]
		targetPoolName := targetPoolModel.Name.ValueString()
		ctx = tflog.SetField(ctx, "target_pool_name", targetPoolName)

		// Generate API request body from model
		payload, err := toTargetPoolUpdatePayload(ctx, sdkUtils.Ptr(targetPoolModel))
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating load balancer", fmt.Sprintf("Creating API payload for target pool: %v", err))
			return
		}

		// Update target pool
		_, err = r.client.UpdateTargetPool(ctx, projectId, region, name, targetPoolName).UpdateTargetPoolPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating load balancer", fmt.Sprintf("Calling API for target pool: %v", err))
			return
		}
	}
	ctx = tflog.SetField(ctx, "target_pool_name", nil)

	// Get updated load balancer
	getResp, err := r.client.GetLoadBalancer(ctx, projectId, region, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating load balancer", fmt.Sprintf("Calling API after update: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, getResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Load balancer updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *loadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete load balancer
	_, err := r.client.DeleteLoadBalancer(ctx, projectId, region, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting load balancer", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.DeleteLoadBalancerWaitHandler(ctx, r.client, projectId, region, name).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting load balancer", fmt.Sprintf("Load balancer deleting waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Load balancer deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,name
func (r *loadBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing load balancer",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[name]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[2])...)
	tflog.Info(ctx, "Load balancer state imported")
}

// toCreatePayload and all other toX functions in this file turn a Terraform load balancer model into a createLoadBalancerPayload to be used with the load balancer API.
func toCreatePayload(ctx context.Context, model *Model) (*loadbalancer.CreateLoadBalancerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	listenersPayload, err := toListenersPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting listeners: %w", err)
	}
	networksPayload, err := toNetworksPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting networks: %w", err)
	}
	optionsPayload, err := toOptionsPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting options: %w", err)
	}
	targetPoolsPayload, err := toTargetPoolsPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting target_pools: %w", err)
	}

	return &loadbalancer.CreateLoadBalancerPayload{
		ExternalAddress:                      conversion.StringValueToPointer(model.ExternalAddress),
		DisableTargetSecurityGroupAssignment: conversion.BoolValueToPointer(model.DisableSecurityGroupAssignment),
		Listeners:                            listenersPayload,
		Name:                                 conversion.StringValueToPointer(model.Name),
		PlanId:                               conversion.StringValueToPointer(model.PlanId),
		Networks:                             networksPayload,
		Options:                              optionsPayload,
		TargetPools:                          targetPoolsPayload,
	}, nil
}

func toListenersPayload(ctx context.Context, model *Model) (*[]loadbalancer.Listener, error) {
	if model.Listeners.IsNull() || model.Listeners.IsUnknown() {
		return nil, nil
	}

	listenersModel := []listener{}
	diags := model.Listeners.ElementsAs(ctx, &listenersModel, false)
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	if len(listenersModel) == 0 {
		return nil, nil
	}

	payload := []loadbalancer.Listener{}
	for i := range listenersModel {
		listenerModel := listenersModel[i]
		serverNameIndicatorsPayload, err := toServerNameIndicatorsPayload(ctx, &listenerModel)
		if err != nil {
			return nil, fmt.Errorf("converting index %d: converting server_name_indicator: %w", i, err)
		}
		tcp, err := toTCP(ctx, &listenerModel)
		if err != nil {
			return nil, fmt.Errorf("converting index %d: converting tcp: %w", i, err)
		}
		udp, err := toUDP(ctx, &listenerModel)
		if err != nil {
			return nil, fmt.Errorf("converting index %d: converting udp: %w", i, err)
		}
		payload = append(payload, loadbalancer.Listener{
			DisplayName:          conversion.StringValueToPointer(listenerModel.DisplayName),
			Port:                 conversion.Int64ValueToPointer(listenerModel.Port),
			Protocol:             loadbalancer.ListenerGetProtocolAttributeType(conversion.StringValueToPointer(listenerModel.Protocol)),
			ServerNameIndicators: serverNameIndicatorsPayload,
			TargetPool:           conversion.StringValueToPointer(listenerModel.TargetPool),
			Tcp:                  tcp,
			Udp:                  udp,
		})
	}

	return &payload, nil
}

func toServerNameIndicatorsPayload(ctx context.Context, l *listener) (*[]loadbalancer.ServerNameIndicator, error) {
	if l.ServerNameIndicators.IsNull() || l.ServerNameIndicators.IsUnknown() {
		return nil, nil
	}

	serverNameIndicatorsModel := []serverNameIndicator{}
	diags := l.ServerNameIndicators.ElementsAs(ctx, &serverNameIndicatorsModel, false)
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	payload := []loadbalancer.ServerNameIndicator{}
	for i := range serverNameIndicatorsModel {
		indicatorModel := serverNameIndicatorsModel[i]
		payload = append(payload, loadbalancer.ServerNameIndicator{
			Name: conversion.StringValueToPointer(indicatorModel.Name),
		})
	}

	return &payload, nil
}

func toTCP(ctx context.Context, listener *listener) (*loadbalancer.OptionsTCP, error) {
	if listener.TCP.IsNull() || listener.TCP.IsUnknown() {
		return nil, nil
	}

	tcp := tcp{}
	diags := listener.TCP.As(ctx, &tcp, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}
	if tcp.IdleTimeout.IsNull() || tcp.IdleTimeout.IsUnknown() {
		return nil, nil
	}

	return &loadbalancer.OptionsTCP{
		IdleTimeout: tcp.IdleTimeout.ValueStringPointer(),
	}, nil
}

func toUDP(ctx context.Context, listener *listener) (*loadbalancer.OptionsUDP, error) {
	if listener.UDP.IsNull() || listener.UDP.IsUnknown() {
		return nil, nil
	}

	udp := udp{}
	diags := listener.UDP.As(ctx, &udp, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}
	if udp.IdleTimeout.IsNull() || udp.IdleTimeout.IsUnknown() {
		return nil, nil
	}

	return &loadbalancer.OptionsUDP{
		IdleTimeout: udp.IdleTimeout.ValueStringPointer(),
	}, nil
}

func toNetworksPayload(ctx context.Context, model *Model) (*[]loadbalancer.Network, error) {
	if model.Networks.IsNull() || model.Networks.IsUnknown() {
		return nil, nil
	}

	networksModel := []network{}
	diags := model.Networks.ElementsAs(ctx, &networksModel, false)
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	if len(networksModel) == 0 {
		return nil, nil
	}

	payload := []loadbalancer.Network{}
	for i := range networksModel {
		networkModel := networksModel[i]
		payload = append(payload, loadbalancer.Network{
			NetworkId: conversion.StringValueToPointer(networkModel.NetworkId),
			Role:      loadbalancer.NetworkGetRoleAttributeType(conversion.StringValueToPointer(networkModel.Role)),
		})
	}

	return &payload, nil
}

func toOptionsPayload(ctx context.Context, model *Model) (*loadbalancer.LoadBalancerOptions, error) {
	if model.Options.IsNull() || model.Options.IsUnknown() {
		return &loadbalancer.LoadBalancerOptions{
			AccessControl: &loadbalancer.LoadbalancerOptionAccessControl{},
			Observability: &loadbalancer.LoadbalancerOptionObservability{},
		}, nil
	}

	optionsModel := options{}
	diags := model.Options.As(ctx, &optionsModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	accessControlPayload := &loadbalancer.LoadbalancerOptionAccessControl{}
	if !(optionsModel.ACL.IsNull() || optionsModel.ACL.IsUnknown()) {
		var aclModel []string
		diags := optionsModel.ACL.ElementsAs(ctx, &aclModel, false)
		if diags.HasError() {
			return nil, fmt.Errorf("converting acl: %w", core.DiagsToError(diags))
		}
		accessControlPayload.AllowedSourceRanges = &aclModel
	}

	observabilityPayload := &loadbalancer.LoadbalancerOptionObservability{}
	if !(optionsModel.Observability.IsNull() || optionsModel.Observability.IsUnknown()) {
		observabilityModel := observability{}
		diags := optionsModel.Observability.As(ctx, &observabilityModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting observability: %w", core.DiagsToError(diags))
		}

		// observability logs
		observabilityLogsModel := observabilityOption{}
		diags = observabilityModel.Logs.As(ctx, &observabilityLogsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting observability logs: %w", core.DiagsToError(diags))
		}
		observabilityPayload.Logs = &loadbalancer.LoadbalancerOptionLogs{
			CredentialsRef: observabilityLogsModel.CredentialsRef.ValueStringPointer(),
			PushUrl:        observabilityLogsModel.PushUrl.ValueStringPointer(),
		}

		// observability metrics
		observabilityMetricsModel := observabilityOption{}
		diags = observabilityModel.Metrics.As(ctx, &observabilityMetricsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting observability metrics: %w", core.DiagsToError(diags))
		}
		observabilityPayload.Metrics = &loadbalancer.LoadbalancerOptionMetrics{
			CredentialsRef: observabilityMetricsModel.CredentialsRef.ValueStringPointer(),
			PushUrl:        observabilityMetricsModel.PushUrl.ValueStringPointer(),
		}
	}

	payload := loadbalancer.LoadBalancerOptions{
		AccessControl:      accessControlPayload,
		Observability:      observabilityPayload,
		PrivateNetworkOnly: conversion.BoolValueToPointer(optionsModel.PrivateNetworkOnly),
	}

	return &payload, nil
}

func toTargetPoolsPayload(ctx context.Context, model *Model) (*[]loadbalancer.TargetPool, error) {
	if model.TargetPools.IsNull() || model.TargetPools.IsUnknown() {
		return nil, nil
	}

	targetPoolsModel := []targetPool{}
	diags := model.TargetPools.ElementsAs(ctx, &targetPoolsModel, false)
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	if len(targetPoolsModel) == 0 {
		return nil, nil
	}

	payload := []loadbalancer.TargetPool{}
	for i := range targetPoolsModel {
		targetPoolModel := targetPoolsModel[i]

		activeHealthCheckPayload, err := toActiveHealthCheckPayload(ctx, &targetPoolModel)
		if err != nil {
			return nil, fmt.Errorf("converting index %d: converting active_health_check: %w", i, err)
		}
		sessionPersistencePayload, err := toSessionPersistencePayload(ctx, &targetPoolModel)
		if err != nil {
			return nil, fmt.Errorf("converting index %d: converting session_persistence: %w", i, err)
		}
		targetsPayload, err := toTargetsPayload(ctx, &targetPoolModel)
		if err != nil {
			return nil, fmt.Errorf("converting index %d: converting targets: %w", i, err)
		}

		payload = append(payload, loadbalancer.TargetPool{
			ActiveHealthCheck:  activeHealthCheckPayload,
			Name:               conversion.StringValueToPointer(targetPoolModel.Name),
			SessionPersistence: sessionPersistencePayload,
			TargetPort:         conversion.Int64ValueToPointer(targetPoolModel.TargetPort),
			Targets:            targetsPayload,
		})
	}

	return &payload, nil
}

func toTargetPoolUpdatePayload(ctx context.Context, tp *targetPool) (*loadbalancer.UpdateTargetPoolPayload, error) {
	if tp == nil {
		return nil, fmt.Errorf("nil target pool")
	}

	activeHealthCheckPayload, err := toActiveHealthCheckPayload(ctx, tp)
	if err != nil {
		return nil, fmt.Errorf("converting active_health_check: %w", err)
	}
	sessionPersistencePayload, err := toSessionPersistencePayload(ctx, tp)
	if err != nil {
		return nil, fmt.Errorf("converting session_persistence: %w", err)
	}
	targetsPayload, err := toTargetsPayload(ctx, tp)
	if err != nil {
		return nil, fmt.Errorf("converting targets: %w", err)
	}

	return &loadbalancer.UpdateTargetPoolPayload{
		ActiveHealthCheck:  activeHealthCheckPayload,
		Name:               conversion.StringValueToPointer(tp.Name),
		SessionPersistence: sessionPersistencePayload,
		TargetPort:         conversion.Int64ValueToPointer(tp.TargetPort),
		Targets:            targetsPayload,
	}, nil
}

func toSessionPersistencePayload(ctx context.Context, tp *targetPool) (*loadbalancer.SessionPersistence, error) {
	if tp.SessionPersistence.IsNull() || tp.ActiveHealthCheck.IsUnknown() {
		return nil, nil
	}

	sessionPersistenceModel := sessionPersistence{}
	diags := tp.SessionPersistence.As(ctx, &sessionPersistenceModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	return &loadbalancer.SessionPersistence{
		UseSourceIpAddress: conversion.BoolValueToPointer(sessionPersistenceModel.UseSourceIPAddress),
	}, nil
}

func toActiveHealthCheckPayload(ctx context.Context, tp *targetPool) (*loadbalancer.ActiveHealthCheck, error) {
	if tp.ActiveHealthCheck.IsNull() || tp.ActiveHealthCheck.IsUnknown() {
		return nil, nil
	}

	activeHealthCheckModel := activeHealthCheck{}
	diags := tp.ActiveHealthCheck.As(ctx, &activeHealthCheckModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting active health check: %w", core.DiagsToError(diags))
	}

	return &loadbalancer.ActiveHealthCheck{
		HealthyThreshold:   conversion.Int64ValueToPointer(activeHealthCheckModel.HealthyThreshold),
		Interval:           conversion.StringValueToPointer(activeHealthCheckModel.Interval),
		IntervalJitter:     conversion.StringValueToPointer(activeHealthCheckModel.IntervalJitter),
		Timeout:            conversion.StringValueToPointer(activeHealthCheckModel.Timeout),
		UnhealthyThreshold: conversion.Int64ValueToPointer(activeHealthCheckModel.UnhealthyThreshold),
	}, nil
}

func toTargetsPayload(ctx context.Context, tp *targetPool) (*[]loadbalancer.Target, error) {
	if tp.Targets.IsNull() || tp.Targets.IsUnknown() {
		return nil, nil
	}

	targetsModel := []target{}
	diags := tp.Targets.ElementsAs(ctx, &targetsModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting Targets list: %w", core.DiagsToError(diags))
	}

	if len(targetsModel) == 0 {
		return nil, nil
	}

	payload := []loadbalancer.Target{}
	for i := range targetsModel {
		targetModel := targetsModel[i]
		payload = append(payload, loadbalancer.Target{
			DisplayName: conversion.StringValueToPointer(targetModel.DisplayName),
			Ip:          conversion.StringValueToPointer(targetModel.Ip),
		})
	}

	return &payload, nil
}

// mapFields and all other map functions in this file translate an API resource into a Terraform model.
func mapFields(ctx context.Context, lb *loadbalancer.LoadBalancer, m *Model, region string) error {
	if lb == nil {
		return fmt.Errorf("response input is nil")
	}
	if m == nil {
		return fmt.Errorf("model input is nil")
	}

	var name string
	if m.Name.ValueString() != "" {
		name = m.Name.ValueString()
	} else if lb.Name != nil {
		name = *lb.Name
	} else {
		return fmt.Errorf("name not present")
	}
	m.Region = types.StringValue(region)
	m.Name = types.StringValue(name)
	m.Id = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), m.Region.ValueString(), name)

	m.PlanId = types.StringPointerValue(lb.PlanId)
	m.ExternalAddress = types.StringPointerValue(lb.ExternalAddress)
	m.PrivateAddress = types.StringPointerValue(lb.PrivateAddress)
	m.DisableSecurityGroupAssignment = types.BoolPointerValue(lb.DisableTargetSecurityGroupAssignment)

	if lb.TargetSecurityGroup != nil {
		m.SecurityGroupId = types.StringPointerValue(lb.TargetSecurityGroup.Id)
	} else {
		m.SecurityGroupId = types.StringNull()
	}
	err := mapListeners(lb, m)
	if err != nil {
		return fmt.Errorf("mapping listeners: %w", err)
	}
	err = mapNetworks(lb, m)
	if err != nil {
		return fmt.Errorf("mapping network: %w", err)
	}
	err = mapOptions(ctx, lb, m)
	if err != nil {
		return fmt.Errorf("mapping options: %w", err)
	}
	err = mapTargetPools(lb, m)
	if err != nil {
		return fmt.Errorf("mapping target pools: %w", err)
	}

	return nil
}

func mapListeners(loadBalancerResp *loadbalancer.LoadBalancer, m *Model) error {
	if loadBalancerResp.Listeners == nil {
		m.Listeners = types.ListNull(types.ObjectType{AttrTypes: listenerTypes})
		return nil
	}

	listenersList := []attr.Value{}
	for i, listenerResp := range *loadBalancerResp.Listeners {
		listenerMap := map[string]attr.Value{
			"display_name": types.StringPointerValue(listenerResp.DisplayName),
			"port":         types.Int64PointerValue(listenerResp.Port),
			"protocol":     types.StringValue(string(listenerResp.GetProtocol())),
			"target_pool":  types.StringPointerValue(listenerResp.TargetPool),
		}

		err := mapServerNameIndicators(listenerResp.ServerNameIndicators, listenerMap)
		if err != nil {
			return fmt.Errorf("mapping index %d, field serverNameIndicators: %w", i, err)
		}

		err = mapTCP(listenerResp.Tcp, listenerMap)
		if err != nil {
			return fmt.Errorf("mapping index %d, field tcp: %w", i, err)
		}

		err = mapUDP(listenerResp.Udp, listenerMap)
		if err != nil {
			return fmt.Errorf("mapping index %d, field udp: %w", i, err)
		}

		listenerTF, diags := types.ObjectValue(listenerTypes, listenerMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		listenersList = append(listenersList, listenerTF)
	}

	listenersTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: listenerTypes},
		listenersList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	m.Listeners = listenersTF
	return nil
}

func mapServerNameIndicators(serverNameIndicatorsResp *[]loadbalancer.ServerNameIndicator, l map[string]attr.Value) error {
	if serverNameIndicatorsResp == nil || *serverNameIndicatorsResp == nil {
		l["server_name_indicators"] = types.ListNull(types.ObjectType{AttrTypes: serverNameIndicatorTypes})
		return nil
	}

	serverNameIndicatorsList := []attr.Value{}
	for i, serverNameIndicatorResp := range *serverNameIndicatorsResp {
		serverNameIndicatorMap := map[string]attr.Value{
			"name": types.StringPointerValue(serverNameIndicatorResp.Name),
		}

		serverNameIndicatorTF, diags := types.ObjectValue(serverNameIndicatorTypes, serverNameIndicatorMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		serverNameIndicatorsList = append(serverNameIndicatorsList, serverNameIndicatorTF)
	}

	serverNameIndicatorsTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: serverNameIndicatorTypes},
		serverNameIndicatorsList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	l["server_name_indicators"] = serverNameIndicatorsTF
	return nil
}

func mapTCP(tcp *loadbalancer.OptionsTCP, listener map[string]attr.Value) error {
	if tcp == nil || tcp.IdleTimeout == nil || *tcp.IdleTimeout == "" {
		listener["tcp"] = types.ObjectNull(tcpTypes)
		return nil
	}

	tcpAttr, diags := types.ObjectValue(tcpTypes, map[string]attr.Value{
		"idle_timeout": types.StringValue(*tcp.IdleTimeout),
	})
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	listener["tcp"] = tcpAttr
	return nil
}

func mapUDP(udp *loadbalancer.OptionsUDP, listener map[string]attr.Value) error {
	if udp == nil || udp.IdleTimeout == nil || *udp.IdleTimeout == "" {
		listener["udp"] = types.ObjectNull(udpTypes)
		return nil
	}

	udpAttr, diags := types.ObjectValue(udpTypes, map[string]attr.Value{
		"idle_timeout": types.StringValue(*udp.IdleTimeout),
	})
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	listener["udp"] = udpAttr
	return nil
}

func mapNetworks(loadBalancerResp *loadbalancer.LoadBalancer, m *Model) error {
	if loadBalancerResp.Networks == nil {
		m.Networks = types.ListNull(types.ObjectType{AttrTypes: networkTypes})
		return nil
	}

	networksList := []attr.Value{}
	for i, networkResp := range *loadBalancerResp.Networks {
		networkMap := map[string]attr.Value{
			"network_id": types.StringPointerValue(networkResp.NetworkId),
			"role":       types.StringValue(string(networkResp.GetRole())),
		}

		networkTF, diags := types.ObjectValue(networkTypes, networkMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		networksList = append(networksList, networkTF)
	}

	networksTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: networkTypes},
		networksList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	m.Networks = networksTF
	return nil
}

func mapOptions(ctx context.Context, loadBalancerResp *loadbalancer.LoadBalancer, m *Model) error {
	if loadBalancerResp.Options == nil {
		m.Options = types.ObjectNull(optionsTypes)
		return nil
	}

	privateNetworkOnlyTF := types.BoolPointerValue(loadBalancerResp.Options.PrivateNetworkOnly)

	// If the private_network_only field is nil in the response but is explicitly set to false in the model,
	// we set it to false in the TF state to prevent an inconsistent result after apply error
	if !m.Options.IsNull() && !m.Options.IsUnknown() {
		optionsModel := options{}
		diags := m.Options.As(ctx, &optionsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("convert options: %w", core.DiagsToError(diags))
		}
		if loadBalancerResp.Options.PrivateNetworkOnly == nil && !optionsModel.PrivateNetworkOnly.IsNull() && !optionsModel.PrivateNetworkOnly.IsUnknown() && !optionsModel.PrivateNetworkOnly.ValueBool() {
			privateNetworkOnlyTF = types.BoolValue(false)
		}
	}

	optionsMap := map[string]attr.Value{
		"private_network_only": privateNetworkOnlyTF,
	}

	err := mapACL(loadBalancerResp.Options.AccessControl, optionsMap)
	if err != nil {
		return fmt.Errorf("mapping field ACL: %w", err)
	}

	observabilityLogsMap := map[string]attr.Value{
		"credentials_ref": types.StringNull(),
		"push_url":        types.StringNull(),
	}
	if loadBalancerResp.Options.HasObservability() && loadBalancerResp.Options.Observability.HasLogs() {
		observabilityLogsMap["credentials_ref"] = types.StringPointerValue(loadBalancerResp.Options.Observability.Logs.CredentialsRef)
		observabilityLogsMap["push_url"] = types.StringPointerValue(loadBalancerResp.Options.Observability.Logs.PushUrl)
	}
	observabilityLogsTF, diags := types.ObjectValue(observabilityOptionTypes, observabilityLogsMap)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	observabilityMetricsMap := map[string]attr.Value{
		"credentials_ref": types.StringNull(),
		"push_url":        types.StringNull(),
	}
	if loadBalancerResp.Options.HasObservability() && loadBalancerResp.Options.Observability.HasMetrics() {
		observabilityMetricsMap["credentials_ref"] = types.StringPointerValue(loadBalancerResp.Options.Observability.Metrics.CredentialsRef)
		observabilityMetricsMap["push_url"] = types.StringPointerValue(loadBalancerResp.Options.Observability.Metrics.PushUrl)
	}
	observabilityMetricsTF, diags := types.ObjectValue(observabilityOptionTypes, observabilityMetricsMap)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	observabilityMap := map[string]attr.Value{
		"logs":    observabilityLogsTF,
		"metrics": observabilityMetricsTF,
	}
	observabilityTF, diags := types.ObjectValue(observabilityTypes, observabilityMap)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	optionsMap["observability"] = observabilityTF

	optionsTF, diags := types.ObjectValue(optionsTypes, optionsMap)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	m.Options = optionsTF
	return nil
}

func mapACL(accessControlResp *loadbalancer.LoadbalancerOptionAccessControl, o map[string]attr.Value) error {
	if accessControlResp == nil || accessControlResp.AllowedSourceRanges == nil {
		o["acl"] = types.SetNull(types.StringType)
		return nil
	}

	aclList := []attr.Value{}
	for _, rangeResp := range *accessControlResp.AllowedSourceRanges {
		rangeTF := types.StringValue(rangeResp)
		aclList = append(aclList, rangeTF)
	}

	aclTF, diags := types.SetValue(types.StringType, aclList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	o["acl"] = aclTF
	return nil
}

func mapTargetPools(loadBalancerResp *loadbalancer.LoadBalancer, m *Model) error {
	if loadBalancerResp.TargetPools == nil {
		m.TargetPools = types.ListNull(types.ObjectType{AttrTypes: targetPoolTypes})
		return nil
	}

	targetPoolsList := []attr.Value{}
	for i, targetPoolResp := range *loadBalancerResp.TargetPools {
		targetPoolMap := map[string]attr.Value{
			"name":        types.StringPointerValue(targetPoolResp.Name),
			"target_port": types.Int64PointerValue(targetPoolResp.TargetPort),
		}

		err := mapActiveHealthCheck(targetPoolResp.ActiveHealthCheck, targetPoolMap)
		if err != nil {
			return fmt.Errorf("mapping index %d, field ActiveHealthCheck: %w", i, err)
		}

		err = mapTargets(targetPoolResp.Targets, targetPoolMap)
		if err != nil {
			return fmt.Errorf("mapping index %d, field Targets: %w", i, err)
		}

		err = mapSessionPersistence(targetPoolResp.SessionPersistence, targetPoolMap)
		if err != nil {
			return fmt.Errorf("mapping index %d, field SessionPersistence: %w", i, err)
		}

		targetPoolTF, diags := types.ObjectValue(targetPoolTypes, targetPoolMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		targetPoolsList = append(targetPoolsList, targetPoolTF)
	}

	targetPoolsTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: targetPoolTypes},
		targetPoolsList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	m.TargetPools = targetPoolsTF
	return nil
}

func mapActiveHealthCheck(activeHealthCheckResp *loadbalancer.ActiveHealthCheck, tp map[string]attr.Value) error {
	if activeHealthCheckResp == nil {
		tp["active_health_check"] = types.ObjectNull(activeHealthCheckTypes)
		return nil
	}

	activeHealthCheckMap := map[string]attr.Value{
		"healthy_threshold":   types.Int64PointerValue(activeHealthCheckResp.HealthyThreshold),
		"interval":            types.StringPointerValue(activeHealthCheckResp.Interval),
		"interval_jitter":     types.StringPointerValue(activeHealthCheckResp.IntervalJitter),
		"timeout":             types.StringPointerValue(activeHealthCheckResp.Timeout),
		"unhealthy_threshold": types.Int64PointerValue(activeHealthCheckResp.UnhealthyThreshold),
	}

	activeHealthCheckTF, diags := types.ObjectValue(activeHealthCheckTypes, activeHealthCheckMap)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	tp["active_health_check"] = activeHealthCheckTF
	return nil
}

func mapTargets(targetsResp *[]loadbalancer.Target, tp map[string]attr.Value) error {
	if targetsResp == nil || *targetsResp == nil {
		tp["targets"] = types.ListNull(types.ObjectType{AttrTypes: targetTypes})
		return nil
	}

	targetsList := []attr.Value{}
	for i, targetResp := range *targetsResp {
		targetMap := map[string]attr.Value{
			"display_name": types.StringPointerValue(targetResp.DisplayName),
			"ip":           types.StringPointerValue(targetResp.Ip),
		}

		targetTF, diags := types.ObjectValue(targetTypes, targetMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		targetsList = append(targetsList, targetTF)
	}

	targetsTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: targetTypes},
		targetsList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	tp["targets"] = targetsTF
	return nil
}

func mapSessionPersistence(sessionPersistenceResp *loadbalancer.SessionPersistence, tp map[string]attr.Value) error {
	if sessionPersistenceResp == nil {
		tp["session_persistence"] = types.ObjectNull(sessionPersistenceTypes)
		return nil
	}

	sessionPersistenceMap := map[string]attr.Value{
		"use_source_ip_address": types.BoolPointerValue(sessionPersistenceResp.UseSourceIpAddress),
	}

	sessionPersistenceTF, diags := types.ObjectValue(sessionPersistenceTypes, sessionPersistenceMap)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	tp["session_persistence"] = sessionPersistenceTF
	return nil
}
