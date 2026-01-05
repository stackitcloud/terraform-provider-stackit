package alb

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	albSdk "github.com/stackitcloud/stackit-sdk-go/services/alb"
	"github.com/stackitcloud/stackit-sdk-go/services/alb/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	albUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/alb/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &applicationLoadBalancerResource{}
	_ resource.ResourceWithConfigure   = &applicationLoadBalancerResource{}
	_ resource.ResourceWithImportState = &applicationLoadBalancerResource{}
	_ resource.ResourceWithModifyPlan  = &applicationLoadBalancerResource{}
)

type Model struct {
	Id                             types.String `tfsdk:"id"` // needed by TF
	ProjectId                      types.String `tfsdk:"project_id"`
	DisableSecurityGroupAssignment types.Bool   `tfsdk:"disable_target_security_group_assignment"`
	Errors                         types.Set    `tfsdk:"errors"`
	ExternalAddress                types.String `tfsdk:"external_address"`
	Labels                         types.Map    `tfsdk:"labels"`
	Listeners                      types.List   `tfsdk:"listeners"`
	LoadBalancerSecurityGroup      types.Object `tfsdk:"load_balancer_security_group"`
	Name                           types.String `tfsdk:"name"`
	Networks                       types.Set    `tfsdk:"networks"`
	Options                        types.Object `tfsdk:"options"`
	PlanId                         types.String `tfsdk:"plan_id"`
	PrivateAddress                 types.String `tfsdk:"private_address"`
	Region                         types.String `tfsdk:"region"`
	Status                         types.String `tfsdk:"status"`
	TargetPools                    types.List   `tfsdk:"target_pools"`
	TargetSecurityGroup            types.Object `tfsdk:"target_security_group"`
	Version                        types.String `tfsdk:"version"`
}

type errors struct {
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
}

var errorsType = map[string]attr.Type{
	"description": types.StringType,
	"type":        types.StringType,
}

type loadBalancerSecurityGroup struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

var loadBalancerSecurityGroupType = map[string]attr.Type{
	"id":   types.StringType,
	"name": types.StringType,
}

type targetSecurityGroup struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

var targetSecurityGroupType = map[string]attr.Type{
	"id":   types.StringType,
	"name": types.StringType,
}

// Struct corresponding to Model.Listeners[i]
type listener struct {
	Name          types.String `tfsdk:"name"`
	Port          types.Int64  `tfsdk:"port"`
	Protocol      types.String `tfsdk:"protocol"`
	Http          types.Object `tfsdk:"http"`
	Https         types.Object `tfsdk:"https"`
	WafConfigName types.String `tfsdk:"waf_config_name"`
}

// Types corresponding to listener
var listenerTypes = map[string]attr.Type{
	"name":            types.StringType,
	"port":            types.Int64Type,
	"protocol":        types.StringType,
	"http":            types.ObjectType{AttrTypes: httpTypes},
	"https":           types.ObjectType{AttrTypes: httpsTypes},
	"waf_config_name": types.StringType,
}

type httpALB struct {
	Hosts types.List `tfsdk:"hosts"`
}

var httpTypes = map[string]attr.Type{
	"hosts": types.ListType{ElemType: types.ObjectType{AttrTypes: hostConfigTypes}},
}

type hostConfig struct {
	Host  types.String `tfsdk:"host"`
	Rules types.List   `tfsdk:"rules"`
}

var hostConfigTypes = map[string]attr.Type{
	"host":  types.StringType,
	"rules": types.ListType{ElemType: types.ObjectType{AttrTypes: ruleTypes}},
}

type rule struct {
	Path              types.Object `tfsdk:"path"`
	Headers           types.Set    `tfsdk:"headers"`
	TargetPool        types.String `tfsdk:"target_pool"`
	WebSocket         types.Bool   `tfsdk:"web_socket"`
	QueryParameters   types.Set    `tfsdk:"query_parameters"`
	CookiePersistence types.Object `tfsdk:"cookie_persistence"`
}

var ruleTypes = map[string]attr.Type{
	"path":               types.ObjectType{AttrTypes: pathTypes},
	"headers":            types.SetType{ElemType: types.ObjectType{AttrTypes: headersTypes}},
	"target_pool":        types.StringType,
	"web_socket":         types.BoolType,
	"query_parameters":   types.SetType{ElemType: types.ObjectType{AttrTypes: queryParameterTypes}},
	"cookie_persistence": types.ObjectType{AttrTypes: cookiePersistenceTypes},
}

type pathALB struct {
	Exact  types.String `tfsdk:"exact_match"`
	Prefix types.String `tfsdk:"prefix"`
}

var pathTypes = map[string]attr.Type{
	"exact_match": types.StringType,
	"prefix":      types.StringType,
}

type headers struct {
	Name       types.String `tfsdk:"name"`
	ExactMatch types.String `tfsdk:"exact_match"`
}

var headersTypes = map[string]attr.Type{
	"name":        types.StringType,
	"exact_match": types.StringType,
}

type queryParameter struct {
	Name       types.String `tfsdk:"name"`
	ExactMatch types.String `tfsdk:"exact_match"`
}

var queryParameterTypes = map[string]attr.Type{
	"name":        types.StringType,
	"exact_match": types.StringType,
}

type cookiePersistence struct {
	Name types.String `tfsdk:"name"`
	Ttl  types.String `tfsdk:"ttl"`
}

var cookiePersistenceTypes = map[string]attr.Type{
	"name": types.StringType,
	"ttl":  types.StringType,
}

type https struct {
	CertificateConfig types.Object `tfsdk:"certificate_config"`
}

var httpsTypes = map[string]attr.Type{
	"certificate_config": types.ObjectType{AttrTypes: certificateConfigTypes},
}

type certificateConfig struct {
	CertificateConfigIDs types.Set `tfsdk:"certificate_ids"`
}

var certificateConfigTypes = map[string]attr.Type{
	"certificate_ids": types.SetType{ElemType: types.StringType},
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
	EphemeralAddress   types.Bool   `tfsdk:"ephemeral_address"`
}

// Types corresponding to options
var optionsTypes = map[string]attr.Type{
	"acl":                  types.SetType{ElemType: types.StringType},
	"private_network_only": types.BoolType,
	"observability":        types.ObjectType{AttrTypes: observabilityTypes},
	"ephemeral_address":    types.BoolType,
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
	ActiveHealthCheck types.Object `tfsdk:"active_health_check"`
	Name              types.String `tfsdk:"name"`
	TargetPort        types.Int64  `tfsdk:"target_port"`
	Targets           types.Set    `tfsdk:"targets"`
	TLSConfig         types.Object `tfsdk:"tls_config"`
}

// Types corresponding to targetPool
var targetPoolTypes = map[string]attr.Type{
	"active_health_check": types.ObjectType{AttrTypes: activeHealthCheckTypes},
	"name":                types.StringType,
	"target_port":         types.Int64Type,
	"targets":             types.SetType{ElemType: types.ObjectType{AttrTypes: targetTypes}},
	"tls_config":          types.ObjectType{AttrTypes: tlsConfigTypes},
}

// Struct corresponding to targetPool.ActiveHealthCheck
type activeHealthCheck struct {
	HealthyThreshold   types.Int64  `tfsdk:"healthy_threshold"`
	HttpHealthChecks   types.Object `tfsdk:"http_health_checks"`
	Interval           types.String `tfsdk:"interval"`
	IntervalJitter     types.String `tfsdk:"interval_jitter"`
	Timeout            types.String `tfsdk:"timeout"`
	UnhealthyThreshold types.Int64  `tfsdk:"unhealthy_threshold"`
}

// Types corresponding to activeHealthCheck
var activeHealthCheckTypes = map[string]attr.Type{
	"healthy_threshold":   types.Int64Type,
	"http_health_checks":  types.ObjectType{AttrTypes: httpHealthChecksTypes},
	"interval":            types.StringType,
	"interval_jitter":     types.StringType,
	"timeout":             types.StringType,
	"unhealthy_threshold": types.Int64Type,
}

type httpHealthChecks struct {
	OkStatus types.Set    `tfsdk:"ok_status"`
	Path     types.String `tfsdk:"path"`
}

var httpHealthChecksTypes = map[string]attr.Type{
	"path":      types.StringType,
	"ok_status": types.SetType{ElemType: types.StringType},
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

type tlsConfig struct {
	CustomCA           types.String `tfsdk:"custom_ca"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	SkipCertValidation types.Bool   `tfsdk:"skip_certificate_validation"`
}

var tlsConfigTypes = map[string]attr.Type{
	"custom_ca":                   types.StringType,
	"enabled":                     types.BoolType,
	"skip_certificate_validation": types.BoolType,
}

// NewApplicationLoadBalancerResource is a helper function to simplify the provider implementation.
func NewApplicationLoadBalancerResource() resource.Resource {
	return &applicationLoadBalancerResource{}
}

// applicationLoadBalancerResource is the resource implementation.
type applicationLoadBalancerResource struct {
	client       *albSdk.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *applicationLoadBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *applicationLoadBalancerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *applicationLoadBalancerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	// Do nothing!
	// We do not validate the TF input, because the API does that and
	// maintaining and syncing between them is not worth it, because
	// 400 Bad Request error gives all the details the user needs via the API in TF, which
	// also is the single source of truth.
}

// Configure adds the provider configured client to the resource.
func (r *applicationLoadBalancerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := albUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Application Load Balancer client configured")
}

// Schema defines the schema for the resource.
func (r *applicationLoadBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	protocolOptions := []string{"PROTOCOL_UNSPECIFIED", "PROTOCOL_HTTP", "PROTOCOL_HTTPS"}
	roleOptions := []string{"ROLE_UNSPECIFIED", "ROLE_LISTENERS_AND_TARGETS", "ROLE_LISTENERS", "ROLE_TARGETS"}
	servicePlanOptions := []string{"p10"}
	regionOptions := []string{"eu01", "eu02"}

	descriptions := map[string]string{
		"main":       "Application Load Balancer resource schema.",
		"id":         "Terraform's internal resource ID. It is structured as \"`project_id`\",\"region\",\"`name`\".",
		"project_id": "STACKIT project ID to which the Application Load Balancer is associated.",
		"region":     "The resource region. If not defined, the provider region is used. " + utils.FormatPossibleValues(regionOptions...),
		"disable_target_security_group_assignment": "Disable target security group assignemt to allow targets outside of the given network. Connectivity to targets need to be ensured by the customer, including routing and Security Groups (targetSecurityGroup can be assigned). Not changeable after creation.",
		"errors":                                 "Reports all errors a Application Load Balancer has.",
		"errors.type":                            "Enum: \"TYPE_UNSPECIFIED\" \"TYPE_INTERNAL\" \"TYPE_QUOTA_SECGROUP_EXCEEDED\" \"TYPE_QUOTA_SECGROUPRULE_EXCEEDED\" \"TYPE_PORT_NOT_CONFIGURED\" \"TYPE_FIP_NOT_CONFIGURED\" \"TYPE_TARGET_NOT_ACTIVE\" \"TYPE_METRICS_MISCONFIGURED\" \"TYPE_LOGS_MISCONFIGURED\"\nThe error type specifies which part of the Application Load Balancer encountered the error. I.e. the API will not check if a provided public IP is actually available in the project. Instead the Application Load Balancer with try to use the provided IP and if not available reports TYPE_FIP_NOT_CONFIGURED error.",
		"errors.description":                     "The error description contains additional helpful user information to fix the error state of the Application Load Balancer. For example the IP 45.135.247.139 does not exist in the project, then the description will report: Floating IP \"45.135.247.139\" could not be found.",
		"external_address":                       "The external IP address where this Application Load Balancer is exposed. Not changeable after creation.",
		"labels":                                 "Labels represent user-defined metadata as key-value pairs. Label count cannot exceed 64 per ALB.",
		"listeners":                              "List of all listeners which will accept traffic. Limited to 20.",
		"listeners.name":                         "Unique name for the listener",
		"http":                                   "Configuration for handling HTTP traffic on this listener.",
		"hosts":                                  "Defines routing rules grouped by hostname.",
		"host":                                   "Hostname to match. Supports wildcards (e.g. *.example.com).",
		"rules":                                  "Routing rules under the specified host, matched by path prefix.",
		"cookie_persistence":                     "Routing persistence via cookies.",
		"cookie_persistence.name":                "The name of the cookie to use.",
		"ttl":                                    "TTL specifies the time-to-live for the cookie. The default value is 0s, and it acts as a session cookie, expiring when the client session ends.",
		"headers":                                "Headers for the rule.",
		"headers.exact_match":                    "Exact match for the header value.",
		"headers.name":                           "Header name.",
		"path":                                   "Routing via path.",
		"path.exact_match":                       "Exact path match. Only a request path exactly equal to the value will match, e.g. '/foo' matches only '/foo', not '/foo/bar' or '/foobar'.",
		"path.prefix":                            "Prefix path match. Only matches on full segment boundaries, e.g. '/foo' matches '/foo' and '/foo/bar' but NOT '/foobar'.",
		"query_parameters":                       "Query parameters for the rule.",
		"query_parameters.exact_match":           "Exact match for the query parameters value.",
		"query_parameters.name":                  "Query parameter name.",
		"target_pool":                            "Reference target pool by target pool name.",
		"web_socket":                             "If enabled, when client sends an HTTP request with and Upgrade header, indicating the desire to establish a Websocket connection, if backend server supports WebSocket, it responds with HTTP 101 status code, switching protocols from HTTP to WebSocket. Hence the client and the server can exchange data in real-time using one long-lived TCP connection.",
		"https":                                  "Configuration for handling HTTPS traffic on this listener.",
		"certificate_config":                     "TLS termination certificate configuration.",
		"certificate_ids":                        "Certificate IDs for TLS termination.",
		"port":                                   "Port number on which the listener receives incoming traffic.",
		"protocol":                               "Protocol is the highest network protocol we understand to load balance. " + utils.FormatPossibleValues(protocolOptions...),
		"waf_config_name":                        "Enable Web Application Firewall (WAF), referenced by name. See \"Application Load Balancer - Web Application Firewall API\" for more information.",
		"load_balancer_security_group":           "Security Group permitting network traffic from the LoadBalancer to the targets. Useful when disableTargetSecurityGroupAssignment=true to manually assign target security groups to targets.",
		"load_balancer_security_group.id":        "ID of the security Group",
		"load_balancer_security_group.name":      "Name of the security Group",
		"name":                                   "Application Load balancer name.",
		"networks":                               "List of networks that listeners and targets reside in.",
		"network_id":                             "STACKIT network ID the Application Load Balancer and/or targets are in.",
		"role":                                   "The role defines how the Application Load Balancer is using the network. " + utils.FormatPossibleValues(roleOptions...),
		"options":                                "Defines any optional functionality you want to have enabled on your Application Load Balancer.",
		"acl":                                    "Use this option to limit the IP ranges that can use the Application Load Balancer.",
		"ephemeral_address":                      "This option automates the handling of the external IP address for an Application Load Balancer. If set to true a new IP address will be automatically created. It will also be automatically deleted when the Load Balancer is deleted.",
		"observability":                          "We offer Load Balancer observability via STACKIT Observability or external solutions.",
		"observability_logs":                     "Observability logs configuration.",
		"observability_logs_credentials_ref":     "Credentials reference for logging. This reference is created via the observability create endpoint and the credential needs to contain the basic auth username and password for the logging solution the push URL points to. Then this enables monitoring via remote write for the Application Load Balancer.",
		"observability_logs_push_url":            "The Observability(Logs)/Loki remote write Push URL you want the logs to be shipped to.",
		"observability_metrics":                  "Observability metrics configuration.",
		"observability_metrics_credentials_ref":  "Credentials reference for metrics. This reference is created via the observability create endpoint and the credential needs to contain the basic auth username and password for the metrics solution the push URL points to. Then this enables monitoring via remote write for the Application Load Balancer.",
		"observability_metrics_push_url":         "The Observability(Metrics)/Prometheus remote write push URL you want the metrics to be shipped to.",
		"plan_id":                                "Service Plan configures the size of the Application Load Balancer. " + utils.FormatPossibleValues(servicePlanOptions...) + ". This list can change in the future. Therefore, this is not an enum.",
		"private_network_only":                   "Application Load Balancer is accessible only via a private network ip address. Not changeable after creation.",
		"status":                                 "Enum: \"STATUS_UNSPECIFIED\" \"STATUS_PENDING\" \"STATUS_READY\" \"STATUS_ERROR\" \"STATUS_TERMINATING\"",
		"target_pools":                           "List of all target pools which will be used in the Application Load Balancer. Limited to 20.",
		"active_health_checks":                   "Set this to customize active health checks for targets in this pool.",
		"healthy_threshold":                      "Healthy threshold of the health checking.",
		"http_health_checks":                     "Options for the HTTP health checking.",
		"http_health_checks.ok_status":           "List of HTTP status codes that indicate a healthy response.",
		"http_health_checks.path":                "Path to send the health check request to.",
		"interval":                               "Interval duration of health checking in seconds.",
		"interval_jitter":                        "Interval duration threshold of the health checking in seconds.",
		"timeout":                                "Active health checking timeout duration in seconds.",
		"unhealthy_threshold":                    "Unhealthy threshold of the health checking.",
		"target_pools.name":                      "Target pool name.",
		"target_port":                            "The number identifying the port where each target listens for traffic.",
		"targets":                                "List of all targets which will be used in the pool. Limited to 250.",
		"targets.display_name":                   "Target display name",
		"ip":                                     "Private target IP, which must by unique within a target pool.",
		"tls_config":                             "Configuration for TLS bridging.",
		"tls_config.custom_ca":                   "Specifies a custom Certificate Authority (CA). When provided, the target pool will trust certificates signed by this CA, in addition to any system-trusted CAs. This is useful for scenarios where the target pool needs to communicate with servers using self-signed or internally-issued certificates. Enabled needs to be set to true and skip validation to false for this option.",
		"tls_config.enabled":                     "Enable TLS (Transport Layer Security) bridging for the connection between Application Load Balancer and targets in this pool. When enabled, public CAs are trusted. Can be used in tandem with the options either custom CA or skip validation or alone.",
		"tls_config.skip_certificate_validation": "Bypass certificate validation for TLS bridging in this target pool. This option is insecure and can only be used with public CAs by setting enabled true. Meant to be used for testing purposes only!",
		"target_security_group":                  "Security Group that allows the targets to receive traffic from the LoadBalancer. Useful when disableTargetSecurityGroupAssignment=true to manually assign target security groups to targets.",
		"target_security_group.id":               "ID of the security Group",
		"target_security_group.name":             "Name of the security Group",
		"version":                                "Application Load Balancer resource version. Used for concurrency safe updates.",
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
				Validators: []validator.String{
					validate.UUID(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(regionOptions...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disable_target_security_group_assignment": schema.BoolAttribute{
				Description: descriptions["disable_target_security_group_assignment"],
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"errors": schema.SetNestedAttribute{
				Description: descriptions["errors"],
				Computed:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: descriptions["errors.type"],
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"description": schema.StringAttribute{
							Description: descriptions["errors.description"],
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"external_address": schema.StringAttribute{
				Description: descriptions["external_address"],
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
				Description: descriptions["labels"],
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					mapvalidator.SizeBetween(1, 64),
					mapvalidator.KeysAre(stringvalidator.LengthBetween(1, 63)),
					mapvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 63)),
				},
			},
			"plan_id": schema.StringAttribute{
				Description: descriptions["plan_id"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(servicePlanOptions...),
				},
			},
			"listeners": schema.ListNestedAttribute{
				Description: descriptions["listeners"],
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 20),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: descriptions["listeners.name"],
							Computed:    true, // will be required in v2
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
									"1-63 characters [0-9] & [a-z] also [-] but not at the beginning or end",
								),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"port": schema.Int64Attribute{
							Description: descriptions["port"],
							Required:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"protocol": schema.StringAttribute{
							Description: descriptions["protocol"],
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(protocolOptions...),
							},
						},
						"waf_config_name": schema.StringAttribute{
							Description: descriptions["waf_config_name"],
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
									"1-63 characters [0-9] & [a-z] also [-] but not at the beginning or end",
								),
							},
						},
						"http": schema.SingleNestedAttribute{
							Description: "Configuration for HTTP traffic.",
							Required:    true,
							Attributes: map[string]schema.Attribute{
								"hosts": schema.ListNestedAttribute{
									Description: descriptions["hosts"],
									Required:    true,
									Validators: []validator.List{
										listvalidator.SizeBetween(1, 100),
									},
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"host": schema.StringAttribute{
												Description: descriptions["host"],
												Required:    true,
												Validators: []validator.String{
													stringvalidator.LengthBetween(1, 253),
												},
											},
											"rules": schema.ListNestedAttribute{ // This order matters and needs to be a list
												Description: descriptions["rules"],
												Required:    true,
												Validators: []validator.List{
													listvalidator.SizeBetween(1, 100),
												},
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"target_pool": schema.StringAttribute{
															Description: descriptions["target_pool"],
															Required:    true,
															Validators: []validator.String{
																stringvalidator.RegexMatches(
																	regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
																	"1-63 characters [0-9] & [a-z] also [-] but not at the beginning or end",
																),
															},
														},
														"web_socket": schema.BoolAttribute{
															Description: descriptions["web_socket"],
															Optional:    true,
															Computed:    true,
															Default:     booldefault.StaticBool(false),
														},
														"path": schema.SingleNestedAttribute{
															Description: descriptions["path"],
															Optional:    true,
															Attributes: map[string]schema.Attribute{
																"exact_match": schema.StringAttribute{
																	Description: descriptions["path.exact_match"],
																	Optional:    true,
																	Validators: []validator.String{
																		stringvalidator.LengthBetween(1, 253),
																	},
																},
																"prefix": schema.StringAttribute{
																	Description: descriptions["path.prefix"],
																	Optional:    true,
																	Validators: []validator.String{
																		stringvalidator.LengthBetween(1, 253),
																	},
																},
															},
														},
														"headers": schema.SetNestedAttribute{
															Description: descriptions["headers"],
															Optional:    true,
															Validators: []validator.Set{
																setvalidator.SizeBetween(1, 100),
															},
															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: descriptions["headers.name"],
																		Required:    true,
																		Validators: []validator.String{
																			stringvalidator.LengthBetween(1, 253),
																		},
																	},
																	"exact_match": schema.StringAttribute{
																		Description: descriptions["headers.exact_match"],
																		Optional:    true,
																		Validators: []validator.String{
																			stringvalidator.LengthBetween(1, 253),
																		},
																	},
																},
															},
														},
														"query_parameters": schema.SetNestedAttribute{
															Description: descriptions["query_parameters"],
															Optional:    true,
															Validators: []validator.Set{
																setvalidator.SizeBetween(1, 100),
															},
															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: descriptions["query_parameters.name"],
																		Required:    true,
																		Validators: []validator.String{
																			stringvalidator.LengthBetween(1, 253),
																		},
																	},
																	"exact_match": schema.StringAttribute{
																		Description: descriptions["query_parameters.exact_match"],
																		Optional:    true,
																		Validators: []validator.String{
																			stringvalidator.LengthBetween(1, 253),
																		},
																	},
																},
															},
														},
														"cookie_persistence": schema.SingleNestedAttribute{
															Description: descriptions["cookie_persistence"],
															Optional:    true,
															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	Description: descriptions["cookie_persistence.name"],
																	Required:    true,
																	Validators: []validator.String{
																		stringvalidator.RegexMatches(
																			regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
																			"1-63 characters [0-9] & [a-z] also [-] but not at the beginning or end",
																		),
																	},
																},
																"ttl": schema.StringAttribute{
																	Description: descriptions["ttl"],
																	Required:    true,
																	Validators: []validator.String{
																		stringvalidator.RegexMatches(
																			regexp.MustCompile(`^\d\d{0,7}s$`),
																			"The duration must be a whole number followed by 's' (for seconds), between 0s and 99999999s",
																		),
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"https": schema.SingleNestedAttribute{
							Description: descriptions["https"],
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"certificate_config": schema.SingleNestedAttribute{
									Description: descriptions["certificate_config"],
									Required:    true,
									Attributes: map[string]schema.Attribute{
										"certificate_ids": schema.SetAttribute{
											Description: descriptions["certificate_ids"],
											Required:    true,
											ElementType: types.StringType,
											Validators: []validator.Set{
												setvalidator.SizeBetween(1, 100),
												setvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 253)),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"load_balancer_security_group": schema.SingleNestedAttribute{
				Description: descriptions["load_balancer_security_group"],
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: descriptions["load_balancer_security_group.name"],
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"id": schema.StringAttribute{
						Description: descriptions["load_balancer_security_group.id"],
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
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
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
						"1-63 characters [0-9] & [a-z] also [-] but not at the beginning or end",
					),
				},
			},
			"networks": schema.SetNestedAttribute{
				Description: descriptions["networks"],
				Required:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Set{
					setvalidator.SizeBetween(1, 2),
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
							},
						},
						"role": schema.StringAttribute{
							Description: descriptions["role"],
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
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
				Attributes: map[string]schema.Attribute{
					"acl": schema.SetAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
						Optional:    true,
						Validators: []validator.Set{
							setvalidator.SizeBetween(1, 100),
							setvalidator.ValueStringsAre(
								validate.CIDR(),
							),
						},
					},
					"ephemeral_address": schema.BoolAttribute{
						Description: descriptions["ephemeral_address"],
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"private_network_only": schema.BoolAttribute{
						Description: descriptions["private_network_only"],
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"observability": schema.SingleNestedAttribute{
						Description: descriptions["observability"],
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"logs": schema.SingleNestedAttribute{
								Description: descriptions["observability_logs"],
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"credentials_ref": schema.StringAttribute{
										Description: descriptions["observability_logs_credentials_ref"],
										Required:    true,
										Validators: []validator.String{
											stringvalidator.LengthBetween(17, 17),
										},
									},
									"push_url": schema.StringAttribute{
										Description: descriptions["observability_logs_credentials_ref"],
										Required:    true,
										Validators: []validator.String{
											stringvalidator.LengthBetween(1, 1000),
										},
									},
								},
							},
							"metrics": schema.SingleNestedAttribute{
								Description: descriptions["observability_metrics"],
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"credentials_ref": schema.StringAttribute{
										Description: descriptions["observability_metrics_credentials_ref"],
										Required:    true,
										Validators: []validator.String{
											stringvalidator.LengthBetween(17, 17),
										},
									},
									"push_url": schema.StringAttribute{
										Description: descriptions["observability_metrics_credentials_ref"],
										Required:    true,
										Validators: []validator.String{
											stringvalidator.LengthBetween(1, 1000),
										},
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
			"status": schema.StringAttribute{
				Description: descriptions["status"],
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
							Attributes: map[string]schema.Attribute{
								"healthy_threshold": schema.Int64Attribute{
									Description: descriptions["healthy_threshold"],
									Required:    true,
									Validators: []validator.Int64{
										int64validator.Between(1, 999),
									},
								},
								"interval": schema.StringAttribute{
									Description: descriptions["interval"],
									Required:    true,
									Validators: []validator.String{
										stringvalidator.RegexMatches(
											regexp.MustCompile(`^[0-9]{1,3}s|[0-9]{1,3}\.(?:[0-9]{2}[1-9]|[0-9][1-9][0-9]|[1-9][0-9]{2})s$`),
											"The duration must be between 0s and 999.999s (e.g.: 1s or 0.100s or 12.345s)",
										),
									},
								},
								"interval_jitter": schema.StringAttribute{
									Description: descriptions["interval_jitter"],
									Required:    true,
									Validators: []validator.String{
										stringvalidator.RegexMatches(
											regexp.MustCompile(`^[0-9]{1,3}s|[0-9]{1,3}\.(?:[0-9]{2}[1-9]|[0-9][1-9][0-9]|[1-9][0-9]{2})s$`),
											"The duration must be between 0s and 999.999s (e.g.: 1s or 0.100s or 12.345s)",
										),
									},
								},
								"timeout": schema.StringAttribute{
									Description: descriptions["timeout"],
									Required:    true,
									Validators: []validator.String{
										stringvalidator.RegexMatches(
											regexp.MustCompile(`^[0-9]{1,3}s|[0-9]{1,3}\.(?:[0-9]{2}[1-9]|[0-9][1-9][0-9]|[1-9][0-9]{2})s$`),
											"The duration must be between 0s and 999.999s (e.g.: 1s or 0.100s or 12.345s)",
										),
									},
								},
								"unhealthy_threshold": schema.Int64Attribute{
									Description: descriptions["unhealthy_threshold"],
									Required:    true,
									Validators: []validator.Int64{
										int64validator.Between(1, 999),
									},
								},
								"http_health_checks": schema.SingleNestedAttribute{
									Description: descriptions["http_health_checks"],
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"path": schema.StringAttribute{
											Description: descriptions["http_health_checks.path"],
											Required:    true,
											Validators: []validator.String{
												stringvalidator.LengthBetween(1, 253),
											},
										},
										"ok_status": schema.SetAttribute{
											Description: descriptions["http_health_checks.ok_status"],
											Required:    true,
											ElementType: types.StringType,
											Validators: []validator.Set{
												setvalidator.SizeBetween(1, 100),
												setvalidator.ValueStringsAre(
													stringvalidator.RegexMatches(
														regexp.MustCompile(`\d{3}`),
														"must match expression",
													),
												),
											},
										},
									},
								},
							},
						},
						"name": schema.StringAttribute{
							Description: descriptions["target_pools.name"],
							Required:    true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
									"1-63 characters [0-9] & [a-z] also [-] but not at the beginning or end",
								),
							},
						},
						"target_port": schema.Int64Attribute{
							Description: descriptions["target_port"],
							Required:    true,
							Validators:  []validator.Int64{int64validator.Between(1, 65535)},
						},
						"targets": schema.SetNestedAttribute{
							Description: descriptions["targets"],
							Required:    true,
							Validators: []validator.Set{
								setvalidator.SizeBetween(1, 250),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"display_name": schema.StringAttribute{
										Description: descriptions["targets.display_name"],
										Optional:    true,
										Validators: []validator.String{
											stringvalidator.RegexMatches(
												regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
												"1-63 characters [0-9] & [a-z] also [-] but not at the beginning or end",
											),
										},
									},
									"ip": schema.StringAttribute{
										Description: descriptions["ip"],
										Required:    true,
										Validators: []validator.String{
											validate.IP(false),
										},
									},
								},
							},
						},
						"tls_config": schema.SingleNestedAttribute{
							Description: descriptions["tls_config"],
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"enabled": schema.BoolAttribute{
									Description: descriptions["tls_config.enabled"],
									Optional:    true,
									Computed:    true,
									Default:     booldefault.StaticBool(false),
								},
								"skip_certificate_validation": schema.BoolAttribute{
									Description: descriptions["tls_config.skip_certificate_validation"],
									Optional:    true,
									Computed:    true,
									Default:     booldefault.StaticBool(false),
								},
								"custom_ca": schema.StringAttribute{
									Description: descriptions["tls_config.custom_ca"],
									Optional:    true,
									Validators: []validator.String{
										stringvalidator.LengthBetween(1, 8192),
									},
								},
							},
						},
					},
				},
			},
			"target_security_group": schema.SingleNestedAttribute{
				Description: descriptions["target_security_group"],
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: descriptions["target_security_group.name"],
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"id": schema.StringAttribute{
						Description: descriptions["target_security_group.id"],
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *applicationLoadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Application Load Balancer", fmt.Sprintf("Payload for create: %v", err))
		return
	}

	// Create a new Application Load Balancer
	createResp, err := r.client.CreateLoadBalancer(ctx, projectId, region).CreateLoadBalancerPayload(*payload).XRequestID(uuid.NewString()).Execute()
	if err != nil {
		errStr := prettyApiErr(err)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Application Load Balancer", fmt.Sprintf("Calling API for create: %v", errStr))
		return
	}

	waitResp, err := wait.CreateOrUpdateLoadbalancerWaitHandler(ctx, r.client, projectId, region, *createResp.Name).SetTimeout(90 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Application Load Balancer", fmt.Sprintf("Application Load Balancer creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Application Load Balancer", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Application Load Balancer created")
}

// Read refreshes the Terraform state with the latest data.
func (r *applicationLoadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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
		errStr := prettyApiErr(err)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Application Load Balancer", fmt.Sprintf("Calling API: %v", errStr))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, lbResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Application Load Balancer", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Application Load Balancer read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *applicationLoadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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

	// get version (computed field) for update call via state
	var state Model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Version = state.Version

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Application Load Balancer", fmt.Sprintf("Payload for update: %s", err))
		return
	}

	// Update target pool
	updateResp, err := r.client.UpdateLoadBalancer(ctx, projectId, region, name).UpdateLoadBalancerPayload(*payload).Execute()
	if err != nil {
		errStr := prettyApiErr(err)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Application Load Balancer", fmt.Sprintf("Calling API for update: %v", errStr))
		return
	}

	waitResp, err := wait.CreateOrUpdateLoadbalancerWaitHandler(ctx, r.client, projectId, region, *updateResp.Name).SetTimeout(90 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Application Load Balancer", fmt.Sprintf("Application Load Balancer update waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Application Load Balancer", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Application Load Balancer updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *applicationLoadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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

	// Delete Application Load Balancer
	_, err := r.client.DeleteLoadBalancer(ctx, projectId, region, name).Execute()
	if err != nil {
		errStr := prettyApiErr(err)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Application Load Balancer", fmt.Sprintf("Calling API for delete: %v", errStr))
		return
	}

	_, err = wait.DeleteLoadbalancerWaitHandler(ctx, r.client, projectId, region, name).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Application Load Balancer", fmt.Sprintf("Application Load Balancer deleting waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Application Load Balancer deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,name
func (r *applicationLoadBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing Application Load Balancer",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[name]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[2])...)
	tflog.Info(ctx, "Application Load Balancer state imported")
}

func prettyApiErr(err error) string {
	oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
	if !ok {
		return err.Error()
	}
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, oapiErr.Body, "", "  "); err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%s, status code %d, Body:\n%s", oapiErr.ErrorMessage, oapiErr.StatusCode, prettyJSON.String())
}

// toCreatePayload and all other toX functions in this file turn a Terraform Application Load Balancer model into a createLoadBalancerPayload to be used with the Application Load Balancer API.
func toCreatePayload(ctx context.Context, model *Model) (*albSdk.CreateLoadBalancerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labelsPayload, err := toLabelPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting labels: %w", err)
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

	return &albSdk.CreateLoadBalancerPayload{
		DisableTargetSecurityGroupAssignment: conversion.BoolValueToPointer(model.DisableSecurityGroupAssignment),
		ExternalAddress:                      conversion.StringValueToPointer(model.ExternalAddress),
		Labels:                               labelsPayload,
		Listeners:                            listenersPayload,
		Name:                                 conversion.StringValueToPointer(model.Name),
		Networks:                             networksPayload,
		Options:                              optionsPayload,
		PlanId:                               conversion.StringValueToPointer(model.PlanId),
		TargetPools:                          targetPoolsPayload,
	}, nil
}

func toLabelPayload(ctx context.Context, model *Model) (albSdk.CreateLoadBalancerPayloadGetLabelsAttributeType, error) {
	if model.Labels.IsNull() || model.Labels.IsUnknown() {
		return nil, nil
	}
	var labels map[string]string
	// Unpack types.Map -> map[string]string
	diags := model.Labels.ElementsAs(ctx, &labels, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting labels: %w", core.DiagsToError(diags))
	}

	payload := albSdk.CreateLoadBalancerPayloadGetLabelsArgType{}
	for key, value := range labels {
		payload[key] = value
	}

	return &payload, nil
}

func toListenersPayload(ctx context.Context, model *Model) (*[]albSdk.Listener, error) {
	if model.Listeners.IsNull() || model.Listeners.IsUnknown() {
		return nil, nil
	}

	listenersModel := []listener{}
	diags := model.Listeners.ElementsAs(ctx, &listenersModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting listeners: %w", core.DiagsToError(diags))
	}

	payload := []albSdk.Listener{}
	for i := range listenersModel {
		listenerModel := listenersModel[i]
		httpPayload, err := toHttpPayload(ctx, &listenerModel)
		if err != nil {
			return nil, fmt.Errorf("converting http payload: %w", err)
		}
		httpsPayload, err := toHttpsPayload(ctx, &listenerModel)
		if err != nil {
			return nil, fmt.Errorf("converting https payload: %w", err)
		}
		payload = append(payload, albSdk.Listener{
			Http:  httpPayload,
			Https: httpsPayload,
			//Name:          conversion.StringValueToPointer(listenerModel.Name), will be added in v2
			Port:          conversion.Int64ValueToPointer(listenerModel.Port),
			Protocol:      albSdk.ListenerGetProtocolAttributeType(conversion.StringValueToPointer(listenerModel.Protocol)),
			WafConfigName: conversion.StringValueToPointer(listenerModel.WafConfigName),
		})
	}

	return &payload, nil
}

func toHttpPayload(ctx context.Context, listenerModel *listener) (albSdk.ListenerGetHttpAttributeType, error) {
	if listenerModel.Http.IsNull() || listenerModel.Http.IsUnknown() {
		return nil, nil
	}

	httpModel := httpALB{}
	diags := listenerModel.Http.As(ctx, &httpModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting http: %w", core.DiagsToError(diags))
	}

	hostsPayload, err := toHostsPayload(ctx, &httpModel)
	if err != nil {
		return nil, fmt.Errorf("converting host payload: %w", err)
	}

	payload := albSdk.ListenerGetHttpArgType{
		Hosts: hostsPayload,
	}
	return &payload, nil
}

func toHostsPayload(ctx context.Context, httpModel *httpALB) (albSdk.ProtocolOptionsHTTPGetHostsAttributeType, error) {
	if httpModel.Hosts.IsNull() || httpModel.Hosts.IsUnknown() {
		return nil, nil
	}

	hostsModel := []hostConfig{}
	diags := httpModel.Hosts.ElementsAs(ctx, &hostsModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting hosts: %w", core.DiagsToError(diags))
	}

	payload := albSdk.ProtocolOptionsHTTPGetHostsArgType{}
	for i := range hostsModel {
		hostModel := hostsModel[i]
		if hostModel.Host.IsNull() || hostModel.Host.IsUnknown() {
			return nil, fmt.Errorf("no hosts specified")
		}
		rulesPayload, err := toRulesPayload(ctx, &hostModel)
		if err != nil {
			return nil, fmt.Errorf("converting host payload: %w", err)
		}
		payload = append(payload, albSdk.HostConfig{
			Host:  conversion.StringValueToPointer(hostModel.Host),
			Rules: rulesPayload,
		})
	}
	return &payload, nil
}

func toRulesPayload(ctx context.Context, hostConfigModel *hostConfig) (albSdk.HostConfigGetRulesAttributeType, error) {
	if hostConfigModel.Rules.IsNull() || hostConfigModel.Rules.IsUnknown() {
		return nil, nil
	}

	rulesModel := []rule{}
	diags := hostConfigModel.Rules.ElementsAs(ctx, &rulesModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting rules: %w", core.DiagsToError(diags))
	}

	payload := []albSdk.Rule{}
	for i := range rulesModel {
		ruleModel := rulesModel[i]
		cookiePersistencePayload, err := toCookiePersistencePayload(ctx, &ruleModel)
		if err != nil {
			return nil, fmt.Errorf("converting rule payload: %w", err)
		}
		headersPayload, err := toHeadersPayload(ctx, &ruleModel)
		if err != nil {
			return nil, fmt.Errorf("converting rule payload: %w", err)
		}
		pathPayload, err := toPathPayload(ctx, &ruleModel)
		if err != nil {
			return nil, fmt.Errorf("converting rule payload: %w", err)
		}
		queryParametersPayload, err := toQueryParametersPayload(ctx, &ruleModel)
		if err != nil {
			return nil, fmt.Errorf("converting rule payload: %w", err)
		}
		payload = append(payload, albSdk.Rule{
			CookiePersistence: cookiePersistencePayload,
			Headers:           headersPayload,
			Path:              pathPayload,
			PathPrefix:        nil, // will be removed in v2
			QueryParameters:   queryParametersPayload,
			TargetPool:        conversion.StringValueToPointer(ruleModel.TargetPool),
			WebSocket:         conversion.BoolValueToPointer(ruleModel.WebSocket),
		})
	}
	return &payload, nil
}

func toQueryParametersPayload(ctx context.Context, ruleModel *rule) (albSdk.RuleGetQueryParametersAttributeType, error) {
	if ruleModel.QueryParameters.IsNull() || ruleModel.QueryParameters.IsUnknown() {
		return nil, nil
	}

	queryParametersModel := []queryParameter{}
	diags := ruleModel.QueryParameters.ElementsAs(ctx, &queryParametersModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting query parameter payload: %w", core.DiagsToError(diags))
	}

	payload := albSdk.RuleGetQueryParametersArgType{}
	for i := range queryParametersModel {
		queryParameterModel := queryParametersModel[i]
		payload = append(payload, albSdk.QueryParameter{
			ExactMatch: conversion.StringValueToPointer(queryParameterModel.ExactMatch),
			Name:       conversion.StringValueToPointer(queryParameterModel.Name),
		})
	}

	return &payload, nil
}

func toPathPayload(ctx context.Context, ruleModel *rule) (albSdk.RuleGetPathAttributeType, error) {
	if ruleModel.Path.IsNull() || ruleModel.Path.IsUnknown() {
		return nil, nil
	}

	pathModel := pathALB{}
	diags := ruleModel.Path.As(ctx, &pathModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting path: %w", core.DiagsToError(diags))
	}

	if (pathModel.Exact.IsNull() || pathModel.Exact.IsUnknown()) && (pathModel.Prefix.IsNull() || pathModel.Prefix.IsUnknown()) {
		return nil, fmt.Errorf("no path prefix or exact match specified")
	}
	if !(pathModel.Exact.IsNull() || pathModel.Exact.IsUnknown()) && !(pathModel.Prefix.IsNull() || pathModel.Prefix.IsUnknown()) {
		return nil, fmt.Errorf("path prefix and exact match are specified at the same time")
	}

	payload := albSdk.RuleGetPathArgType{
		Exact:  conversion.StringValueToPointer(pathModel.Exact),
		Prefix: conversion.StringValueToPointer(pathModel.Prefix),
	}
	return &payload, nil
}

func toCookiePersistencePayload(ctx context.Context, ruleModel *rule) (albSdk.RuleGetCookiePersistenceAttributeType, error) {
	if ruleModel.CookiePersistence.IsNull() || ruleModel.CookiePersistence.IsUnknown() {
		return nil, nil
	}

	cookieModel := cookiePersistence{}
	diags := ruleModel.CookiePersistence.As(ctx, &cookieModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting cookie persistence config: %w", core.DiagsToError(diags))
	}

	payload := albSdk.RuleGetCookiePersistenceArgType{
		Name: conversion.StringValueToPointer(cookieModel.Name),
		Ttl:  conversion.StringValueToPointer(cookieModel.Ttl),
	}

	return &payload, nil
}

func toHeadersPayload(ctx context.Context, ruleModel *rule) (albSdk.RuleGetHeadersAttributeType, error) {
	if ruleModel.Headers.IsNull() || ruleModel.Headers.IsUnknown() {
		return nil, nil
	}

	headersModel := []headers{}
	diags := ruleModel.Headers.ElementsAs(ctx, &headersModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting headers: %w", core.DiagsToError(diags))
	}

	payload := albSdk.RuleGetHeadersArgType{}
	for i := range headersModel {
		header := headersModel[i]
		payload = append(payload, albSdk.HttpHeader{
			ExactMatch: conversion.StringValueToPointer(header.ExactMatch),
			Name:       conversion.StringValueToPointer(header.Name),
		})
	}
	return &payload, nil
}

func toHttpsPayload(ctx context.Context, listenerModel *listener) (albSdk.ListenerGetHttpsAttributeType, error) {
	if listenerModel.Https.IsNull() || listenerModel.Https.IsUnknown() {
		return nil, nil
	}

	httpsModel := https{}
	diags := listenerModel.Https.As(ctx, &httpsModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting https: %w", core.DiagsToError(diags))
	}

	certificateConfigPayload, err := toCertificateConfigPayload(ctx, &httpsModel)
	if err != nil {
		return nil, fmt.Errorf("converting certificate config: %w", err)
	}

	payload := albSdk.ListenerGetHttpsArgType{
		CertificateConfig: certificateConfigPayload,
	}

	return &payload, nil
}

func toCertificateConfigPayload(ctx context.Context, https *https) (*albSdk.CertificateConfig, error) {
	if https.CertificateConfig.IsNull() || https.CertificateConfig.IsUnknown() {
		return nil, nil
	}

	certificateConfigModel := certificateConfig{}
	diags := https.CertificateConfig.As(ctx, &certificateConfigModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting certificate config: %w", core.DiagsToError(diags))
	}
	if certificateConfigModel.CertificateConfigIDs.IsNull() || certificateConfigModel.CertificateConfigIDs.IsUnknown() {
		return nil, fmt.Errorf("converting certificate config: no certificate config found")
	}

	certificateConfigSet, err := conversion.StringSetToPointer(certificateConfigModel.CertificateConfigIDs)
	if err != nil {
		return nil, fmt.Errorf("converting certificate config list: %w", err)
	}

	payload := albSdk.CertificateConfig{
		CertificateIds: certificateConfigSet,
	}
	return &payload, nil
}

func toNetworksPayload(ctx context.Context, model *Model) (*[]albSdk.Network, error) {
	if model.Networks.IsNull() || model.Networks.IsUnknown() {
		return nil, nil
	}

	networksModel := []network{}
	diags := model.Networks.ElementsAs(ctx, &networksModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting networks: %w", core.DiagsToError(diags))
	}

	payload := []albSdk.Network{}
	for i := range networksModel {
		networkModel := networksModel[i]
		payload = append(payload, albSdk.Network{
			NetworkId: conversion.StringValueToPointer(networkModel.NetworkId),
			Role:      albSdk.NetworkGetRoleAttributeType(conversion.StringValueToPointer(networkModel.Role)),
		})
	}

	return &payload, nil
}

func toOptionsPayload(ctx context.Context, model *Model) (*albSdk.LoadBalancerOptions, error) {
	if model.Options.IsNull() || model.Options.IsUnknown() {
		return nil, nil
	}

	optionsModel := options{}
	diags := model.Options.As(ctx, &optionsModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting options: %w", core.DiagsToError(diags))
	}

	accessControlPayload := &albSdk.LoadbalancerOptionAccessControl{}
	if !(optionsModel.ACL.IsNull() || optionsModel.ACL.IsUnknown()) {
		var aclModel []string
		diags := optionsModel.ACL.ElementsAs(ctx, &aclModel, false)
		if diags.HasError() {
			return nil, fmt.Errorf("converting acl: %w", core.DiagsToError(diags))
		}
		accessControlPayload.AllowedSourceRanges = &aclModel
	}

	observabilityPayload := &albSdk.LoadbalancerOptionObservability{}
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
		observabilityPayload.Logs = &albSdk.LoadbalancerOptionLogs{
			CredentialsRef: observabilityLogsModel.CredentialsRef.ValueStringPointer(),
			PushUrl:        observabilityLogsModel.PushUrl.ValueStringPointer(),
		}

		// observability metrics
		observabilityMetricsModel := observabilityOption{}
		diags = observabilityModel.Metrics.As(ctx, &observabilityMetricsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting observability metrics: %w", core.DiagsToError(diags))
		}
		observabilityPayload.Metrics = &albSdk.LoadbalancerOptionMetrics{
			CredentialsRef: observabilityMetricsModel.CredentialsRef.ValueStringPointer(),
			PushUrl:        observabilityMetricsModel.PushUrl.ValueStringPointer(),
		}
	}

	payload := albSdk.LoadBalancerOptions{
		AccessControl:      accessControlPayload,
		Observability:      observabilityPayload,
		PrivateNetworkOnly: conversion.BoolValueToPointer(optionsModel.PrivateNetworkOnly),
		EphemeralAddress:   conversion.BoolValueToPointer(optionsModel.EphemeralAddress),
	}

	return &payload, nil
}

func toTargetPoolsPayload(ctx context.Context, model *Model) (*[]albSdk.TargetPool, error) {
	if model.TargetPools.IsNull() || model.TargetPools.IsUnknown() {
		return nil, nil
	}

	targetPoolsModel := []targetPool{}
	diags := model.TargetPools.ElementsAs(ctx, &targetPoolsModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting targetPools: %w", core.DiagsToError(diags))
	}

	payload := []albSdk.TargetPool{}
	for i := range targetPoolsModel {
		targetPoolModel := targetPoolsModel[i]

		activeHealthCheckPayload, err := toActiveHealthCheckPayload(ctx, &targetPoolModel)
		if err != nil {
			return nil, fmt.Errorf("converting index %d: converting active_health_check: %w", i, err)
		}
		targetsPayload, err := toTargetsPayload(ctx, &targetPoolModel)
		if err != nil {
			return nil, fmt.Errorf("converting index %d: converting targets: %w", i, err)
		}
		tlsConfigPayload, err := toTlsConfigPayload(ctx, &targetPoolModel)
		if err != nil {
			return nil, fmt.Errorf("converting index %d: converting tls_config: %w", i, err)
		}

		payload = append(payload, albSdk.TargetPool{
			ActiveHealthCheck: activeHealthCheckPayload,
			Name:              conversion.StringValueToPointer(targetPoolModel.Name),
			TargetPort:        conversion.Int64ValueToPointer(targetPoolModel.TargetPort),
			Targets:           targetsPayload,
			TlsConfig:         tlsConfigPayload,
		})
	}

	return &payload, nil
}

func toTlsConfigPayload(ctx context.Context, tp *targetPool) (albSdk.TargetPoolGetTlsConfigAttributeType, error) {
	if tp.TLSConfig.IsNull() || tp.TLSConfig.IsUnknown() {
		return nil, nil
	}

	tlsConfigModel := tlsConfig{}
	diags := tp.TLSConfig.As(ctx, &tlsConfigModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting target pool TLS config: %w", core.DiagsToError(diags))
	}

	payload := albSdk.TargetPoolGetTlsConfigArgType{
		Enabled:                   conversion.BoolValueToPointer(tlsConfigModel.Enabled),
		SkipCertificateValidation: conversion.BoolValueToPointer(tlsConfigModel.SkipCertValidation),
	}

	if !tlsConfigModel.CustomCA.IsNull() && !tlsConfigModel.CustomCA.IsUnknown() {
		customCa := base64.StdEncoding.EncodeToString([]byte(tlsConfigModel.CustomCA.ValueString()))
		payload.CustomCa = &customCa
	}

	return &payload, nil
}

func toUpdatePayload(ctx context.Context, model *Model) (*albSdk.UpdateLoadBalancerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labelsPayload, err := toLabelPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting labels: %w", err)
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
	externalAddressPayload, err := toExternalAddress(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting external_address: %w", err)
	}

	return &albSdk.UpdateLoadBalancerPayload{
		DisableTargetSecurityGroupAssignment: conversion.BoolValueToPointer(model.DisableSecurityGroupAssignment),
		ExternalAddress:                      externalAddressPayload,
		Labels:                               labelsPayload,
		Listeners:                            listenersPayload,
		Name:                                 conversion.StringValueToPointer(model.Name),
		Networks:                             networksPayload,
		Options:                              optionsPayload,
		PlanId:                               conversion.StringValueToPointer(model.PlanId),
		TargetPools:                          targetPoolsPayload,
		Version:                              conversion.StringValueToPointer(model.Version),
	}, nil
}

// toExternalAddress needs to exist because during UPDATE the model will always have it, but
// we do not send it if ephemeral_address or private_network_only options are set.
func toExternalAddress(ctx context.Context, m *Model) (albSdk.UpdateLoadBalancerPayloadGetExternalAddressAttributeType, error) {
	if m.Options.IsNull() || m.Options.IsUnknown() {
		// no ephemeral or private option are set
		return conversion.StringValueToPointer(m.ExternalAddress), nil
	}
	o := &options{}
	diags := m.Options.As(ctx, o, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting options: %w", core.DiagsToError(diags))
	}
	if (o.EphemeralAddress.IsNull() || o.EphemeralAddress.IsUnknown()) && (o.PrivateNetworkOnly.IsNull() || o.PrivateNetworkOnly.IsUnknown()) {
		// options exist but ephemeral or private option are set (default false) so use external address
		return conversion.StringValueToPointer(m.ExternalAddress), nil
	}
	if !o.EphemeralAddress.IsNull() && !o.PrivateNetworkOnly.IsNull() && o.EphemeralAddress.ValueBool() && o.PrivateNetworkOnly.ValueBool() {
		// options exist but both ephemeral or private option are set true so error for impossible combination
		return nil, fmt.Errorf("ephemeral_address and private_network_only cannot both be true")
	}
	if !o.EphemeralAddress.IsNull() && o.EphemeralAddress.ValueBool() {
		// ephemeral exist and true so send no external address
		return nil, nil
	}
	if !o.PrivateNetworkOnly.IsNull() && o.PrivateNetworkOnly.ValueBool() {
		// private exist and true so send no external address
		return nil, nil
	}
	// ephemeral and private exist, but is false so use external address
	return conversion.StringValueToPointer(m.ExternalAddress), nil
}

func toActiveHealthCheckPayload(ctx context.Context, tp *targetPool) (*albSdk.ActiveHealthCheck, error) {
	if tp.ActiveHealthCheck.IsNull() || tp.ActiveHealthCheck.IsUnknown() {
		return nil, nil
	}

	activeHealthCheckModel := activeHealthCheck{}
	diags := tp.ActiveHealthCheck.As(ctx, &activeHealthCheckModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting active health check: %w", core.DiagsToError(diags))
	}

	httpHealthChecksPayload, err := toHttpHealthChecksPayload(ctx, &activeHealthCheckModel)
	if err != nil {
		return nil, fmt.Errorf("converting http health check: %w", err)
	}

	return &albSdk.ActiveHealthCheck{
		HealthyThreshold:   conversion.Int64ValueToPointer(activeHealthCheckModel.HealthyThreshold),
		Interval:           conversion.StringValueToPointer(activeHealthCheckModel.Interval),
		IntervalJitter:     conversion.StringValueToPointer(activeHealthCheckModel.IntervalJitter),
		Timeout:            conversion.StringValueToPointer(activeHealthCheckModel.Timeout),
		UnhealthyThreshold: conversion.Int64ValueToPointer(activeHealthCheckModel.UnhealthyThreshold),
		HttpHealthChecks:   httpHealthChecksPayload,
	}, nil
}

func toHttpHealthChecksPayload(ctx context.Context, check *activeHealthCheck) (albSdk.ActiveHealthCheckGetHttpHealthChecksAttributeType, error) {
	if check.HttpHealthChecks.IsNull() || check.HttpHealthChecks.IsUnknown() {
		return nil, nil
	}

	httpHealthChecksModel := httpHealthChecks{}
	diags := check.HttpHealthChecks.As(ctx, &httpHealthChecksModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting active health check: %w", core.DiagsToError(diags))
	}

	okStatus, err := conversion.StringSetToPointer(httpHealthChecksModel.OkStatus)
	if err != nil {
		return nil, fmt.Errorf("converting active health check ok status: %w", err)
	}

	payload := albSdk.HttpHealthChecks{
		OkStatuses: okStatus,
		Path:       conversion.StringValueToPointer(httpHealthChecksModel.Path),
	}
	return &payload, nil
}

func toTargetsPayload(ctx context.Context, tp *targetPool) (*[]albSdk.Target, error) {
	if tp.Targets.IsNull() || tp.Targets.IsUnknown() {
		return nil, nil
	}

	targetsModel := []target{}
	diags := tp.Targets.ElementsAs(ctx, &targetsModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting Targets list: %w", core.DiagsToError(diags))
	}

	payload := []albSdk.Target{}
	for i := range targetsModel {
		targetModel := targetsModel[i]
		payload = append(payload, albSdk.Target{
			DisplayName: conversion.StringValueToPointer(targetModel.DisplayName),
			Ip:          conversion.StringValueToPointer(targetModel.Ip),
		})
	}

	return &payload, nil
}

// mapFields and all other map functions in this file translate an API resource into a Terraform model.
func mapFields(ctx context.Context, alb *albSdk.LoadBalancer, m *Model, region string) error {
	if alb == nil {
		return fmt.Errorf("response input is nil")
	}
	if m == nil {
		return fmt.Errorf("model input is nil")
	}

	var name string
	if m.Name.ValueString() != "" {
		name = m.Name.ValueString()
	} else if alb.Name != nil {
		name = *alb.Name
	} else {
		return fmt.Errorf("name not present")
	}
	m.Region = types.StringValue(region)
	m.Name = types.StringValue(name)
	m.Id = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), m.Region.ValueString(), name)

	m.PlanId = types.StringPointerValue(alb.PlanId)
	m.PrivateAddress = types.StringPointerValue(alb.PrivateAddress)
	m.ExternalAddress = types.StringPointerValue(alb.ExternalAddress)
	m.Version = types.StringPointerValue(alb.Version)
	m.Status = types.StringPointerValue((*string)(alb.Status))
	mapDisableSecurityGroupAssignment(alb, m)
	err := mapErrors(alb, m)
	if err != nil {
		return fmt.Errorf("mapping errors: %w", err)
	}
	err = mapTargetSecurityGroup(alb, m)
	if err != nil {
		return fmt.Errorf("mapping target security group: %w", err)
	}
	err = mapLoadBalancerSecurityGroup(alb, m)
	if err != nil {
		return fmt.Errorf("mapping load balancer security group: %w", err)
	}
	err = mapLabels(alb, m)
	if err != nil {
		return fmt.Errorf("mapping labels: %w", err)
	}
	err = mapListeners(ctx, alb, m)
	if err != nil {
		return fmt.Errorf("mapping listeners: %w", err)
	}
	err = mapNetworks(alb, m)
	if err != nil {
		return fmt.Errorf("mapping network: %w", err)
	}
	err = mapOptions(ctx, alb, m)
	if err != nil {
		return fmt.Errorf("mapping options: %w", err)
	}
	err = mapTargetPools(ctx, alb, m)
	if err != nil {
		return fmt.Errorf("mapping target pools: %w", err)
	}

	return nil
}

func mapDisableSecurityGroupAssignment(applicationLoadBalancerResp *albSdk.LoadBalancer, m *Model) {
	m.DisableSecurityGroupAssignment = types.BoolValue(false)
	// If the disable target security group assignment field is nil in the response we set it to false in the TF state
	// to prevent an inconsistent result after apply error
	if applicationLoadBalancerResp.DisableTargetSecurityGroupAssignment != nil && *applicationLoadBalancerResp.DisableTargetSecurityGroupAssignment {
		m.DisableSecurityGroupAssignment = types.BoolValue(true)
	}
	return
}

func mapErrors(applicationLoadBalancerResp *albSdk.LoadBalancer, m *Model) error {
	if applicationLoadBalancerResp.Errors == nil {
		m.Errors = types.SetNull(types.ObjectType{AttrTypes: errorsType})
		return nil
	}

	errorsSet := []attr.Value{}
	for i, errorsResp := range *applicationLoadBalancerResp.Errors {
		errorMap := map[string]attr.Value{
			"description": types.StringPointerValue(errorsResp.Description),
			"type":        types.StringPointerValue((*string)(errorsResp.Type)),
		}

		errorTF, diags := types.ObjectValue(errorsType, errorMap)
		if diags.HasError() {
			return fmt.Errorf("mapping error %d: %w", i, core.DiagsToError(diags))
		}

		errorsSet = append(errorsSet, errorTF)
	}

	errorsTF, diags := types.SetValue(
		types.ObjectType{AttrTypes: errorsType},
		errorsSet,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping errors: %w", core.DiagsToError(diags))
	}

	m.Errors = errorsTF
	return nil
}

func mapLoadBalancerSecurityGroup(applicationLoadBalancerResp *albSdk.LoadBalancer, m *Model) error {
	if applicationLoadBalancerResp.LoadBalancerSecurityGroup == nil {
		m.LoadBalancerSecurityGroup = types.ObjectNull(loadBalancerSecurityGroupType)
		return nil
	}

	lbSecurityGroupMap := map[string]attr.Value{
		"id":   types.StringPointerValue(applicationLoadBalancerResp.LoadBalancerSecurityGroup.Id),
		"name": types.StringPointerValue(applicationLoadBalancerResp.LoadBalancerSecurityGroup.Name),
	}

	lbSecurityGroupTF, diags := types.ObjectValue(loadBalancerSecurityGroupType, lbSecurityGroupMap)
	if diags.HasError() {
		return fmt.Errorf("mapping loadBalancerSecurityGroup: %w", core.DiagsToError(diags))
	}

	m.LoadBalancerSecurityGroup = lbSecurityGroupTF
	return nil
}

func mapTargetSecurityGroup(applicationLoadBalancerResp *albSdk.LoadBalancer, m *Model) error {
	if applicationLoadBalancerResp.TargetSecurityGroup == nil {
		m.TargetSecurityGroup = types.ObjectNull(targetSecurityGroupType)
		return nil
	}

	tSecurityGroupMap := map[string]attr.Value{
		"id":   types.StringPointerValue(applicationLoadBalancerResp.TargetSecurityGroup.Id),
		"name": types.StringPointerValue(applicationLoadBalancerResp.TargetSecurityGroup.Name),
	}

	tSecurityGroupTF, diags := types.ObjectValue(targetSecurityGroupType, tSecurityGroupMap)
	if diags.HasError() {
		return fmt.Errorf("mapping targetSecurityGroup: %w", core.DiagsToError(diags))
	}

	m.TargetSecurityGroup = tSecurityGroupTF
	return nil
}

func mapLabels(applicationLoadBalancerResp *albSdk.LoadBalancer, m *Model) error {
	if applicationLoadBalancerResp.Labels == nil {
		m.Labels = types.MapNull(types.StringType)
		return nil
	}

	labelsMap := map[string]attr.Value{}
	for key, value := range *applicationLoadBalancerResp.Labels {
		labelsMap[key] = types.StringValue(value)
	}

	labelsTF, diags := types.MapValue(
		types.StringType,
		labelsMap,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping labels: %w", core.DiagsToError(diags))
	}

	m.Labels = labelsTF
	return nil
}

func mapListeners(ctx context.Context, applicationLoadBalancerResp *albSdk.LoadBalancer, m *Model) error {
	if applicationLoadBalancerResp.Listeners == nil {
		m.Listeners = types.ListNull(types.ObjectType{AttrTypes: listenerTypes})
		return nil
	}

	var configListeners []listener
	if !m.Listeners.IsNull() && !m.Listeners.IsUnknown() {
		diags := m.Listeners.ElementsAs(ctx, &configListeners, false)
		if diags.HasError() {
			return fmt.Errorf("unpacking listeners from model: %w", core.DiagsToError(diags))
		}
	}

	listenersSet := []attr.Value{}
	for i, listenerResp := range *applicationLoadBalancerResp.Listeners {
		var configMatch *listener
		for _, cl := range configListeners {
			if !cl.Name.IsNull() && cl.Name.ValueString() == *listenerResp.Name {
				configMatch = &cl
				break
			}
		}
		var httpModel = types.ObjectNull(httpTypes)
		if configMatch != nil {
			httpModel = configMatch.Http
		}

		listenerMap := map[string]attr.Value{
			"name":            types.StringPointerValue(listenerResp.Name),
			"port":            types.Int64PointerValue(listenerResp.Port),
			"protocol":        types.StringValue(string(listenerResp.GetProtocol())),
			"waf_config_name": types.StringPointerValue(listenerResp.WafConfigName),
		}

		err := mapHttp(ctx, listenerResp.Http, listenerMap, httpModel)
		if err != nil {
			return fmt.Errorf("mapping http %d: %w", i, err)
		}

		err = mapHttps(ctx, listenerResp.Https, listenerMap)
		if err != nil {
			return fmt.Errorf("mapping https %d: %w", i, err)
		}

		listenerTF, diags := types.ObjectValue(listenerTypes, listenerMap)
		if diags.HasError() {
			return fmt.Errorf("mapping listener %d: %w", i, core.DiagsToError(diags))
		}

		listenersSet = append(listenersSet, listenerTF)
	}

	listenersTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: listenerTypes},
		listenersSet,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping listeners: %w", core.DiagsToError(diags))
	}

	m.Listeners = listenersTF
	return nil
}

func mapHttp(ctx context.Context, httpResp albSdk.ListenerGetHttpAttributeType, l map[string]attr.Value, httpModel basetypes.ObjectValue) error {
	if httpResp == nil {
		l["http"] = types.ObjectNull(httpTypes)
		return nil
	}

	var configHttp *httpALB
	diags := httpModel.As(ctx, &configHttp, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return fmt.Errorf("unpacking http from model: %w", core.DiagsToError(diags))
	}
	var hostsModel = types.ListNull(types.ObjectType{AttrTypes: hostConfigTypes})
	if configHttp != nil {
		hostsModel = configHttp.Hosts
	}

	httpMap := map[string]attr.Value{}
	err := mapHosts(ctx, httpResp.Hosts, httpMap, hostsModel)
	if err != nil {
		return fmt.Errorf("mapping hosts: %w", err)
	}

	httpTF, diags := types.ObjectValue(httpTypes, httpMap)
	if diags.HasError() {
		return fmt.Errorf("mapping http: %w", core.DiagsToError(diags))
	}

	l["http"] = httpTF
	return nil
}

func mapHosts(ctx context.Context, hostsResp albSdk.ProtocolOptionsHTTPGetHostsAttributeType, h map[string]attr.Value, hostsModel types.List) error {
	if hostsResp == nil {
		h["hosts"] = types.ListNull(types.ObjectType{AttrTypes: hostConfigTypes})
		return nil
	}

	var configHosts []hostConfig
	diags := hostsModel.ElementsAs(ctx, &configHosts, false)
	if diags.HasError() {
		return fmt.Errorf("unpacking hosts from model: %w", core.DiagsToError(diags))
	}

	hostsSet := []attr.Value{}
	for i, hostResp := range *hostsResp {
		var configMatch *hostConfig
		for _, ch := range configHosts {
			if !ch.Host.IsNull() && ch.Host.ValueString() == *hostResp.Host {
				configMatch = &ch
				break
			}
		}
		var rulesModel = types.ListNull(types.ObjectType{AttrTypes: ruleTypes})
		if configMatch != nil {
			rulesModel = configMatch.Rules
		}

		hostMap := map[string]attr.Value{
			"host": types.StringPointerValue(hostResp.Host),
		}

		err := mapRules(ctx, hostResp.Rules, hostMap, rulesModel)
		if err != nil {
			return fmt.Errorf("mapping rules %d: %w", i, err)
		}

		hostTF, diags := types.ObjectValue(hostConfigTypes, hostMap)
		if diags.HasError() {
			return fmt.Errorf("mapping host %d: %w", i, core.DiagsToError(diags))
		}

		hostsSet = append(hostsSet, hostTF)
	}

	hostsTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: hostConfigTypes},
		hostsSet,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping hosts: %w", core.DiagsToError(diags))
	}

	h["hosts"] = hostsTF
	return nil
}

func mapRules(ctx context.Context, rulesResp albSdk.HostConfigGetRulesAttributeType, h map[string]attr.Value, rulesModel types.List) error {
	if rulesResp == nil {
		h["rules"] = types.ListNull(types.ObjectType{AttrTypes: ruleTypes})
		return nil
	}

	var configRules []rule
	diags := rulesModel.ElementsAs(ctx, &configRules, false)
	if diags.HasError() {
		return fmt.Errorf("unpacking rules from model: %w", core.DiagsToError(diags))
	}

	rulesList := []attr.Value{}
	for i, ruleResp := range *rulesResp {
		webSocket := types.BoolValue(false)
		// If the webSocket is nil in the response we set it to false in the TF state to
		// prevent an inconsistent result after apply error
		if ruleResp.WebSocket != nil && *ruleResp.WebSocket {
			webSocket = types.BoolValue(true)
		}

		ruleMap := map[string]attr.Value{
			"target_pool": types.StringPointerValue(ruleResp.TargetPool),
			"web_socket":  webSocket,
		}

		err := mapPath(ruleResp.Path, ruleMap)
		if err != nil {
			return fmt.Errorf("mapping Path %d: %w", i, err)
		}

		err = mapHeaders(ruleResp.Headers, ruleMap)
		if err != nil {
			return fmt.Errorf("mapping Headers %d: %w", i, err)
		}

		err = mapQueryParameters(ruleResp.QueryParameters, ruleMap)
		if err != nil {
			return fmt.Errorf("mapping Query Parameters %d: %w", i, err)
		}

		err = mapCookiePersistence(ruleResp.CookiePersistence, ruleMap)
		if err != nil {
			return fmt.Errorf("mapping Cookie Persistence %d: %w", i, err)
		}

		ruleTF, diags := types.ObjectValue(ruleTypes, ruleMap)
		if diags.HasError() {
			return fmt.Errorf("mapping Rule %d: %w", i, core.DiagsToError(diags))
		}

		rulesList = append(rulesList, ruleTF)
	}

	rulesTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: ruleTypes},
		rulesList,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping rules: %w", core.DiagsToError(diags))
	}

	h["rules"] = rulesTF
	return nil
}

func mapPath(pathResp albSdk.RuleGetPathAttributeType, r map[string]attr.Value) error {
	if pathResp == nil {
		r["path"] = types.ObjectNull(pathTypes)
		return nil
	}

	pathMap := map[string]attr.Value{
		"exact_match": types.StringPointerValue(pathResp.Exact),
		"prefix":      types.StringPointerValue(pathResp.Prefix),
	}

	pathTF, diags := types.ObjectValue(pathTypes, pathMap)
	if diags.HasError() {
		return fmt.Errorf("mapping path: %w", core.DiagsToError(diags))
	}

	r["path"] = pathTF
	return nil
}

func mapQueryParameters(queryParamsResp albSdk.RuleGetQueryParametersAttributeType, r map[string]attr.Value) error {
	if queryParamsResp == nil {
		r["query_parameters"] = types.SetNull(types.ObjectType{AttrTypes: queryParameterTypes})
		return nil
	}

	queryParamsSet := []attr.Value{}
	for i, queryParamResp := range *queryParamsResp {
		queryParamMap := map[string]attr.Value{
			"name":        types.StringPointerValue(queryParamResp.Name),
			"exact_match": types.StringPointerValue(queryParamResp.ExactMatch),
		}

		queryParamTF, diags := types.ObjectValue(queryParameterTypes, queryParamMap)
		if diags.HasError() {
			return fmt.Errorf("mapping queryParameter %d: %w", i, core.DiagsToError(diags))
		}

		queryParamsSet = append(queryParamsSet, queryParamTF)
	}

	queryParamTF, diags := types.SetValue(
		types.ObjectType{AttrTypes: queryParameterTypes},
		queryParamsSet,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping queryParameters: %w", core.DiagsToError(diags))
	}

	r["query_parameters"] = queryParamTF
	return nil
}

func mapHeaders(headersResp albSdk.RuleGetHeadersAttributeType, r map[string]attr.Value) error {
	if headersResp == nil {
		r["headers"] = types.SetNull(types.ObjectType{AttrTypes: headersTypes})
		return nil
	}

	headersSet := []attr.Value{}
	for i, headerResp := range *headersResp {
		headerMap := map[string]attr.Value{
			"name":        types.StringPointerValue(headerResp.Name),
			"exact_match": types.StringPointerValue(headerResp.ExactMatch),
		}

		headerTF, diags := types.ObjectValue(headersTypes, headerMap)
		if diags.HasError() {
			return fmt.Errorf("mapping header %d: %w", i, core.DiagsToError(diags))
		}

		headersSet = append(headersSet, headerTF)
	}

	headersTF, diags := types.SetValue(
		types.ObjectType{AttrTypes: headersTypes},
		headersSet,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping headers: %w", core.DiagsToError(diags))
	}

	r["headers"] = headersTF
	return nil
}

func mapCookiePersistence(cookiePersistResp albSdk.RuleGetCookiePersistenceAttributeType, r map[string]attr.Value) error {
	if cookiePersistResp == nil {
		r["cookie_persistence"] = types.ObjectNull(cookiePersistenceTypes)
		return nil
	}

	cookiePersistMap := map[string]attr.Value{
		"name": types.StringPointerValue(cookiePersistResp.Name),
		"ttl":  types.StringPointerValue(cookiePersistResp.Ttl),
	}

	cookiePersistTF, diags := types.ObjectValue(cookiePersistenceTypes, cookiePersistMap)
	if diags.HasError() {
		return fmt.Errorf("mapping cookiePersistence: %w", core.DiagsToError(diags))
	}

	r["cookie_persistence"] = cookiePersistTF
	return nil
}

func mapHttps(ctx context.Context, httpsResp albSdk.ListenerGetHttpsAttributeType, l map[string]attr.Value) error {
	if httpsResp == nil {
		l["https"] = types.ObjectNull(httpsTypes)
		return nil
	}

	httpsMap := map[string]attr.Value{}

	err := mapCertificates(ctx, httpsResp.CertificateConfig, httpsMap)
	if err != nil {
		return fmt.Errorf("mapping certificates: %w", err)
	}

	httpsTF, diags := types.ObjectValue(httpsTypes, httpsMap)
	if diags.HasError() {
		return fmt.Errorf("mapping https: %w", core.DiagsToError(diags))
	}

	l["https"] = httpsTF
	return nil
}

func mapCertificates(ctx context.Context, certResp albSdk.ProtocolOptionsHTTPSGetCertificateConfigAttributeType, h map[string]attr.Value) error {
	if certResp == nil {
		h["certificate_config"] = types.ObjectNull(certificateConfigTypes)
		return nil
	}

	certificateIDsTF, diags := types.SetValueFrom(ctx, types.StringType, certResp.CertificateIds)
	if diags.HasError() {
		return fmt.Errorf("mapping certificateIDs: %w", core.DiagsToError(diags))
	}
	certMap := map[string]attr.Value{
		"certificate_ids": certificateIDsTF,
	}

	certTF, diags := types.ObjectValue(certificateConfigTypes, certMap)
	if diags.HasError() {
		return fmt.Errorf("mapping certificates: %w", core.DiagsToError(diags))
	}

	h["certificate_config"] = certTF
	return nil
}

func mapNetworks(applicationLoadBalancerResp *albSdk.LoadBalancer, m *Model) error {
	if applicationLoadBalancerResp.Networks == nil {
		m.Networks = types.SetNull(types.ObjectType{AttrTypes: networkTypes})
		return nil
	}

	networksSet := []attr.Value{}
	for i, networkResp := range *applicationLoadBalancerResp.Networks {
		networkMap := map[string]attr.Value{
			"network_id": types.StringPointerValue(networkResp.NetworkId),
			"role":       types.StringValue(string(networkResp.GetRole())),
		}

		networkTF, diags := types.ObjectValue(networkTypes, networkMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		networksSet = append(networksSet, networkTF)
	}

	networksTF, diags := types.SetValue(
		types.ObjectType{AttrTypes: networkTypes},
		networksSet,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping networks: %w", core.DiagsToError(diags))
	}

	m.Networks = networksTF
	return nil
}

func mapOptions(ctx context.Context, applicationLoadBalancerResp *albSdk.LoadBalancer, m *Model) error {
	if applicationLoadBalancerResp.Options == nil {
		m.Options = types.ObjectNull(optionsTypes)
		return nil
	}

	opt := applicationLoadBalancerResp.Options
	// If no options are set in the model and the response has no fields filed,
	// leave the option out of the model to prevent an inconsistent result after apply error
	if (m.Options.IsNull() || m.Options.IsUnknown()) && !opt.HasEphemeralAddress() && !opt.HasPrivateNetworkOnly() && !opt.HasAccessControl() && !opt.HasObservability() {
		return nil
	}

	privateNetworkOnlyTF := types.BoolValue(false)
	ephemeralAddressTF := types.BoolValue(false)
	// If the private_network_only and/or ephemeral_address field is nil in the response we set it to
	// false in the TF state to prevent an inconsistent result after apply error
	if opt.PrivateNetworkOnly != nil && *opt.PrivateNetworkOnly {
		privateNetworkOnlyTF = types.BoolValue(true)
	}
	if opt.EphemeralAddress != nil && *opt.EphemeralAddress {
		ephemeralAddressTF = types.BoolValue(true)
	}

	optionsMap := map[string]attr.Value{
		"private_network_only": privateNetworkOnlyTF,
		"ephemeral_address":    ephemeralAddressTF,
	}

	err := mapACL(opt.AccessControl, optionsMap)
	if err != nil {
		return fmt.Errorf("mapping field ACL: %w", err)
	}

	err = mapObservability(opt.Observability, optionsMap)
	if err != nil {
		return fmt.Errorf("mapping field Observability: %w", err)
	}

	optionsTF, diags := types.ObjectValue(optionsTypes, optionsMap)
	if diags.HasError() {
		return fmt.Errorf("mapping options: %w", core.DiagsToError(diags))
	}

	m.Options = optionsTF
	return nil
}

func mapObservability(observabilityResp *albSdk.LoadbalancerOptionObservability, o map[string]attr.Value) error {
	if observabilityResp == nil {
		o["observability"] = types.ObjectNull(observabilityTypes)
		return nil
	}

	observabilityLogsMap := map[string]attr.Value{
		"credentials_ref": types.StringNull(),
		"push_url":        types.StringNull(),
	}
	if observabilityResp.HasLogs() {
		observabilityLogsMap["credentials_ref"] = types.StringPointerValue(observabilityResp.Logs.CredentialsRef)
		observabilityLogsMap["push_url"] = types.StringPointerValue(observabilityResp.Logs.PushUrl)
	}
	observabilityLogsTF, diags := types.ObjectValue(observabilityOptionTypes, observabilityLogsMap)
	if diags.HasError() {
		return fmt.Errorf("mapping logs: %w", core.DiagsToError(diags))
	}

	observabilityMetricsMap := map[string]attr.Value{
		"credentials_ref": types.StringNull(),
		"push_url":        types.StringNull(),
	}
	if observabilityResp.HasMetrics() {
		observabilityMetricsMap["credentials_ref"] = types.StringPointerValue(observabilityResp.Metrics.CredentialsRef)
		observabilityMetricsMap["push_url"] = types.StringPointerValue(observabilityResp.Metrics.PushUrl)
	}
	observabilityMetricsTF, diags := types.ObjectValue(observabilityOptionTypes, observabilityMetricsMap)
	if diags.HasError() {
		return fmt.Errorf("mapping metrics: %w", core.DiagsToError(diags))
	}

	observabilityMap := map[string]attr.Value{
		"logs":    observabilityLogsTF,
		"metrics": observabilityMetricsTF,
	}
	observabilityTF, diags := types.ObjectValue(observabilityTypes, observabilityMap)
	if diags.HasError() {
		return fmt.Errorf("mapping observability: %w", core.DiagsToError(diags))
	}

	o["observability"] = observabilityTF
	return nil
}

func mapACL(accessControlResp *albSdk.LoadbalancerOptionAccessControl, o map[string]attr.Value) error {
	if accessControlResp == nil || accessControlResp.AllowedSourceRanges == nil {
		o["acl"] = types.SetNull(types.StringType)
		return nil
	}

	aclSet := []attr.Value{}
	for _, rangeResp := range *accessControlResp.AllowedSourceRanges {
		rangeTF := types.StringValue(rangeResp)
		aclSet = append(aclSet, rangeTF)
	}

	aclTF, diags := types.SetValue(types.StringType, aclSet)
	if diags.HasError() {
		return fmt.Errorf("mapping ALC: %w", core.DiagsToError(diags))
	}

	o["acl"] = aclTF
	return nil
}

func mapTargetPools(ctx context.Context, applicationLoadBalancerResp *albSdk.LoadBalancer, m *Model) error {
	if applicationLoadBalancerResp.TargetPools == nil {
		m.TargetPools = types.ListNull(types.ObjectType{AttrTypes: targetPoolTypes})
		return nil
	}

	var configTargetPools []targetPool
	if !m.TargetPools.IsNull() && !m.TargetPools.IsUnknown() {
		diags := m.TargetPools.ElementsAs(ctx, &configTargetPools, false)
		if diags.HasError() {
			return fmt.Errorf("unpacking target pools from model: %w", core.DiagsToError(diags))
		}
	}

	targetPoolsSet := []attr.Value{}
	for i, targetPoolResp := range *applicationLoadBalancerResp.TargetPools {
		var configMatch *targetPool
		for _, ctp := range configTargetPools {
			if !ctp.Name.IsNull() && ctp.Name.ValueString() == *targetPoolResp.Name {
				configMatch = &ctp
				break
			}
		}
		var tlsModel = types.ObjectNull(tlsConfigTypes)
		if configMatch != nil {
			tlsModel = configMatch.TLSConfig
		}

		targetPoolMap := map[string]attr.Value{
			"name":        types.StringPointerValue(targetPoolResp.Name),
			"target_port": types.Int64PointerValue(targetPoolResp.TargetPort),
		}

		err := mapActiveHealthCheck(ctx, targetPoolResp.ActiveHealthCheck, targetPoolMap)
		if err != nil {
			return fmt.Errorf("mapping index %d, field ActiveHealthCheck: %w", i, err)
		}

		err = mapTLSConfig(ctx, targetPoolResp.TlsConfig, targetPoolMap, tlsModel)
		if err != nil {
			return fmt.Errorf("mapping index %d, field TLSConfig: %w", i, err)
		}

		err = mapTargets(targetPoolResp.Targets, targetPoolMap)
		if err != nil {
			return fmt.Errorf("mapping index %d, field Targets: %w", i, err)
		}

		targetPoolTF, diags := types.ObjectValue(targetPoolTypes, targetPoolMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}
		targetPoolsSet = append(targetPoolsSet, targetPoolTF)
	}

	targetPoolsTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: targetPoolTypes},
		targetPoolsSet,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping targetPools: %w", core.DiagsToError(diags))
	}

	m.TargetPools = targetPoolsTF
	return nil
}

func mapActiveHealthCheck(ctx context.Context, activeHealthCheckResp *albSdk.ActiveHealthCheck, tp map[string]attr.Value) error {
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

	err := mapHttpHealthChecks(ctx, activeHealthCheckResp.HttpHealthChecks, activeHealthCheckMap)
	if err != nil {
		return fmt.Errorf("map HttpHealthChecks: %w", err)
	}

	activeHealthCheckTF, diags := types.ObjectValue(activeHealthCheckTypes, activeHealthCheckMap)
	if diags.HasError() {
		return fmt.Errorf("mapping activeHealthChecks: %w", core.DiagsToError(diags))
	}

	tp["active_health_check"] = activeHealthCheckTF
	return nil
}

func mapHttpHealthChecks(ctx context.Context, httpHealthChecksResp *albSdk.HttpHealthChecks, ahc map[string]attr.Value) error {
	if httpHealthChecksResp == nil {
		ahc["http_health_checks"] = types.ObjectNull(httpHealthChecksTypes)
		return nil
	}

	okStatusesTF, diags := types.SetValueFrom(ctx, types.StringType, httpHealthChecksResp.OkStatuses)
	if diags.HasError() {
		return fmt.Errorf("map OkStatuses list: %w", core.DiagsToError(diags))
	}
	httpHealthChecksMap := map[string]attr.Value{
		"ok_status": okStatusesTF,
		"path":      types.StringPointerValue(httpHealthChecksResp.Path),
	}

	httpHealthChecksTF, diags := types.ObjectValue(httpHealthChecksTypes, httpHealthChecksMap)
	if diags.HasError() {
		return fmt.Errorf("mapping httpHealthChecks: %w", core.DiagsToError(diags))
	}

	ahc["http_health_checks"] = httpHealthChecksTF
	return nil
}

func mapTLSConfig(ctx context.Context, targetPoolTLSConfigResp *albSdk.TargetPoolTlsConfig, tp map[string]attr.Value, tlsModel basetypes.ObjectValue) error {
	if targetPoolTLSConfigResp == nil {
		tp["tls_config"] = types.ObjectNull(tlsConfigTypes)
		return nil
	}

	var configTLS = &tlsConfig{}
	if !tlsModel.IsNull() && !tlsModel.IsUnknown() {
		diags := tlsModel.As(ctx, configTLS, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("unpacking tls config from model: %w", core.DiagsToError(diags))
		}
	}

	enabled := types.BoolValue(false)
	skipCertificateValidation := types.BoolValue(false)
	// If the enabled or skip field is nil in the response we set it to false in the TF state to
	// prevent an inconsistent result after apply error
	if targetPoolTLSConfigResp.Enabled != nil && *targetPoolTLSConfigResp.Enabled {
		enabled = types.BoolValue(true)
	}
	if targetPoolTLSConfigResp.SkipCertificateValidation != nil && *targetPoolTLSConfigResp.SkipCertificateValidation {
		skipCertificateValidation = types.BoolValue(true)
	}

	tlsConfigMap := map[string]attr.Value{
		"custom_ca":                   types.StringNull(),
		"enabled":                     enabled,
		"skip_certificate_validation": skipCertificateValidation,
	}

	if targetPoolTLSConfigResp.CustomCa != nil {
		pemBytes, err := base64.StdEncoding.DecodeString(*targetPoolTLSConfigResp.CustomCa)
		if err != nil {
			return fmt.Errorf("base64 decoding custom ca: %w", err)
		}
		tlsConfigMap["custom_ca"] = types.StringValue(string(pemBytes))
	}

	targetPoolTLSConfigTF, diags := types.ObjectValue(tlsConfigTypes, tlsConfigMap)
	if diags.HasError() {
		return fmt.Errorf("mapping TLSConfig: %w", core.DiagsToError(diags))
	}

	tp["tls_config"] = targetPoolTLSConfigTF
	return nil
}

func mapTargets(targetsResp *[]albSdk.Target, tp map[string]attr.Value) error {
	if targetsResp == nil || *targetsResp == nil {
		tp["targets"] = types.SetNull(types.ObjectType{AttrTypes: targetTypes})
		return nil
	}

	targetsSet := []attr.Value{}
	for i, targetResp := range *targetsResp {
		targetMap := map[string]attr.Value{
			"display_name": types.StringPointerValue(targetResp.DisplayName),
			"ip":           types.StringPointerValue(targetResp.Ip),
		}

		targetTF, diags := types.ObjectValue(targetTypes, targetMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		targetsSet = append(targetsSet, targetTF)
	}

	targetsTF, diags := types.SetValue(
		types.ObjectType{AttrTypes: targetTypes},
		targetsSet,
	)
	if diags.HasError() {
		return fmt.Errorf("mapping targets: %w", core.DiagsToError(diags))
	}

	tp["targets"] = targetsTF
	return nil
}
