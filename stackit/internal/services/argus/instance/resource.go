package argus

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
	"github.com/stackitcloud/stackit-sdk-go/services/argus/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Currently, due to incorrect types in the API, the maximum recursion level for child routes is set to 1.
// Once this is fixed, the value should be set to 10.
const childRouteMaxRecursionLevel = 1

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
)

type Model struct {
	Id                                 types.String `tfsdk:"id"` // needed by TF
	ProjectId                          types.String `tfsdk:"project_id"`
	InstanceId                         types.String `tfsdk:"instance_id"`
	Name                               types.String `tfsdk:"name"`
	PlanName                           types.String `tfsdk:"plan_name"`
	PlanId                             types.String `tfsdk:"plan_id"`
	Parameters                         types.Map    `tfsdk:"parameters"`
	DashboardURL                       types.String `tfsdk:"dashboard_url"`
	IsUpdatable                        types.Bool   `tfsdk:"is_updatable"`
	GrafanaURL                         types.String `tfsdk:"grafana_url"`
	GrafanaPublicReadAccess            types.Bool   `tfsdk:"grafana_public_read_access"`
	GrafanaInitialAdminPassword        types.String `tfsdk:"grafana_initial_admin_password"`
	GrafanaInitialAdminUser            types.String `tfsdk:"grafana_initial_admin_user"`
	MetricsRetentionDays               types.Int64  `tfsdk:"metrics_retention_days"`
	MetricsRetentionDays5mDownsampling types.Int64  `tfsdk:"metrics_retention_days_5m_downsampling"`
	MetricsRetentionDays1hDownsampling types.Int64  `tfsdk:"metrics_retention_days_1h_downsampling"`
	MetricsURL                         types.String `tfsdk:"metrics_url"`
	MetricsPushURL                     types.String `tfsdk:"metrics_push_url"`
	TargetsURL                         types.String `tfsdk:"targets_url"`
	AlertingURL                        types.String `tfsdk:"alerting_url"`
	LogsURL                            types.String `tfsdk:"logs_url"`
	LogsPushURL                        types.String `tfsdk:"logs_push_url"`
	JaegerTracesURL                    types.String `tfsdk:"jaeger_traces_url"`
	JaegerUIURL                        types.String `tfsdk:"jaeger_ui_url"`
	OtlpTracesURL                      types.String `tfsdk:"otlp_traces_url"`
	ZipkinSpansURL                     types.String `tfsdk:"zipkin_spans_url"`
	ACL                                types.Set    `tfsdk:"acl"`
	AlertConfig                        types.Object `tfsdk:"alert_config"`
}

// Struct corresponding to Model.AlertConfig
type alertConfigModel struct {
	GlobalConfiguration types.Object `tfsdk:"global"`
	Receivers           types.List   `tfsdk:"receivers"`
	Route               types.Object `tfsdk:"route"`
}

var alertConfigTypes = map[string]attr.Type{
	"receivers": types.ListType{ElemType: types.ObjectType{AttrTypes: receiversTypes}},
	"route":     types.ObjectType{AttrTypes: routeTypes},
	"global":    types.ObjectType{AttrTypes: globalConfigurationTypes},
}

// Struct corresponding to Model.AlertConfig.global
type globalConfigurationModel struct {
	OpsgenieApiKey   types.String `tfsdk:"opsgenie_api_key"`
	OpsgenieApiUrl   types.String `tfsdk:"opsgenie_api_url"`
	ResolveTimeout   types.String `tfsdk:"resolve_timeout"`
	SmtpAuthIdentity types.String `tfsdk:"smtp_auth_identity"`
	SmtpAuthPassword types.String `tfsdk:"smtp_auth_password"`
	SmtpAuthUsername types.String `tfsdk:"smtp_auth_username"`
	SmtpFrom         types.String `tfsdk:"smtp_from"`
	SmtpSmartHost    types.String `tfsdk:"smtp_smart_host"`
}

var globalConfigurationTypes = map[string]attr.Type{
	"opsgenie_api_key":   types.StringType,
	"opsgenie_api_url":   types.StringType,
	"resolve_timeout":    types.StringType,
	"smtp_auth_identity": types.StringType,
	"smtp_auth_password": types.StringType,
	"smtp_auth_username": types.StringType,
	"smtp_from":          types.StringType,
	"smtp_smart_host":    types.StringType,
}

// Struct corresponding to Model.AlertConfig.route
type routeModel struct {
	GroupBy        types.List   `tfsdk:"group_by"`
	GroupInterval  types.String `tfsdk:"group_interval"`
	GroupWait      types.String `tfsdk:"group_wait"`
	Match          types.Map    `tfsdk:"match"`
	MatchRegex     types.Map    `tfsdk:"match_regex"`
	Receiver       types.String `tfsdk:"receiver"`
	RepeatInterval types.String `tfsdk:"repeat_interval"`
	Routes         types.List   `tfsdk:"routes"`
}

// Struct corresponding to Model.AlertConfig.route but without the recursive routes field
// This is used to map the last level of recursion of the routes field
type routeModelNoRoutes struct {
	GroupBy        types.List   `tfsdk:"group_by"`
	GroupInterval  types.String `tfsdk:"group_interval"`
	GroupWait      types.String `tfsdk:"group_wait"`
	Match          types.Map    `tfsdk:"match"`
	MatchRegex     types.Map    `tfsdk:"match_regex"`
	Receiver       types.String `tfsdk:"receiver"`
	RepeatInterval types.String `tfsdk:"repeat_interval"`
}

var routeTypes = map[string]attr.Type{
	"group_by":        types.ListType{ElemType: types.StringType},
	"group_interval":  types.StringType,
	"group_wait":      types.StringType,
	"match":           types.MapType{ElemType: types.StringType},
	"match_regex":     types.MapType{ElemType: types.StringType},
	"receiver":        types.StringType,
	"repeat_interval": types.StringType,
	"routes":          types.ListType{ElemType: getRouteListType()},
}

// Struct corresponding to Model.AlertConfig.receivers
type receiversModel struct {
	Name            types.String `tfsdk:"name"`
	EmailConfigs    types.List   `tfsdk:"email_configs"`
	OpsGenieConfigs types.List   `tfsdk:"opsgenie_configs"`
	WebHooksConfigs types.List   `tfsdk:"webhooks_configs"`
}

var receiversTypes = map[string]attr.Type{
	"name":             types.StringType,
	"email_configs":    types.ListType{ElemType: types.ObjectType{AttrTypes: emailConfigsTypes}},
	"opsgenie_configs": types.ListType{ElemType: types.ObjectType{AttrTypes: opsgenieConfigsTypes}},
	"webhooks_configs": types.ListType{ElemType: types.ObjectType{AttrTypes: webHooksConfigsTypes}},
}

// Struct corresponding to Model.AlertConfig.receivers.emailConfigs
type emailConfigsModel struct {
	AuthIdentity types.String `tfsdk:"auth_identity"`
	AuthPassword types.String `tfsdk:"auth_password"`
	AuthUsername types.String `tfsdk:"auth_username"`
	From         types.String `tfsdk:"from"`
	Smarthost    types.String `tfsdk:"smart_host"`
	To           types.String `tfsdk:"to"`
}

var emailConfigsTypes = map[string]attr.Type{
	"auth_identity": types.StringType,
	"auth_password": types.StringType,
	"auth_username": types.StringType,
	"from":          types.StringType,
	"smart_host":    types.StringType,
	"to":            types.StringType,
}

// Struct corresponding to Model.AlertConfig.receivers.opsGenieConfigs
type opsgenieConfigsModel struct {
	ApiKey types.String `tfsdk:"api_key"`
	ApiUrl types.String `tfsdk:"api_url"`
	Tags   types.String `tfsdk:"tags"`
}

var opsgenieConfigsTypes = map[string]attr.Type{
	"api_key": types.StringType,
	"api_url": types.StringType,
	"tags":    types.StringType,
}

// Struct corresponding to Model.AlertConfig.receivers.webHooksConfigs
type webHooksConfigsModel struct {
	Url     types.String `tfsdk:"url"`
	MsTeams types.Bool   `tfsdk:"ms_teams"`
}

var webHooksConfigsTypes = map[string]attr.Type{
	"url":      types.StringType,
	"ms_teams": types.BoolType,
}

var routeDescriptions = map[string]string{
	"group_by":        "The labels by which incoming alerts are grouped together. For example, multiple alerts coming in for cluster=A and alertname=LatencyHigh would be batched into a single group. To aggregate by all possible labels use the special value '...' as the sole label name, for example: group_by: ['...']. This effectively disables aggregation entirely, passing through all alerts as-is. This is unlikely to be what you want, unless you have a very low alert volume or your upstream notification system performs its own grouping.",
	"group_interval":  "How long to wait before sending a notification about new alerts that are added to a group of alerts for which an initial notification has already been sent. (Usually ~5m or more.)",
	"group_wait":      "How long to initially wait to send a notification for a group of alerts. Allows to wait for an inhibiting alert to arrive or collect more initial alerts for the same group. (Usually ~0s to few minutes.)",
	"match":           "A set of equality matchers an alert has to fulfill to match the node.",
	"match_regex":     "A set of regex-matchers an alert has to fulfill to match the node.",
	"receiver":        "The name of the receiver to route the alerts to.",
	"repeat_interval": "How long to wait before sending a notification again if it has already been sent successfully for an alert. (Usually ~3h or more).",
	"routes":          "List of child routes.",
}

// getRouteListType is a helper function to return the route list attribute type.
func getRouteListType() types.ObjectType {
	return getRouteListTypeAux(1, childRouteMaxRecursionLevel)
}

// getRouteListTypeAux returns the type of the route list attribute with the given level of child routes recursion.
// The level is used to determine the current depth of the nested object.
// The limit is used to determine the maximum depth of the nested object.
// The level should be lower or equal to the limit, if higher, the function will produce a stack overflow.
func getRouteListTypeAux(level, limit int) types.ObjectType {
	attributeTypes := map[string]attr.Type{
		"group_by":        types.ListType{ElemType: types.StringType},
		"group_interval":  types.StringType,
		"group_wait":      types.StringType,
		"match":           types.MapType{ElemType: types.StringType},
		"match_regex":     types.MapType{ElemType: types.StringType},
		"receiver":        types.StringType,
		"repeat_interval": types.StringType,
	}

	if level != limit {
		attributeTypes["routes"] = types.ListType{ElemType: getRouteListTypeAux(level+1, limit)}
	}

	return types.ObjectType{AttrTypes: attributeTypes}
}

func getRouteNestedObject() schema.ListNestedAttribute {
	return getRouteNestedObjectAux(false, 1, childRouteMaxRecursionLevel)
}

func getDatasourceRouteNestedObject() schema.ListNestedAttribute {
	return getRouteNestedObjectAux(true, 1, childRouteMaxRecursionLevel)
}

// getRouteNestedObjectAux returns the nested object for the route attribute with the given level of child routes recursion.
// The isDatasource is used to determine if the route is used in a datasource schema or not. If it is a datasource, all fields are computed.
// The level is used to determine the current depth of the nested object.
// The limit is used to determine the maximum depth of the nested object.
// The level should be lower or equal to the limit, if higher, the function will produce a stack overflow.
func getRouteNestedObjectAux(isDatasource bool, level, limit int) schema.ListNestedAttribute {
	attributesMap := map[string]schema.Attribute{
		"group_by": schema.ListAttribute{
			Description: routeDescriptions["group_by"],
			Optional:    !isDatasource,
			Computed:    isDatasource,
			ElementType: types.StringType,
		},
		"group_interval": schema.StringAttribute{
			Description: routeDescriptions["group_interval"],
			Optional:    !isDatasource,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"group_wait": schema.StringAttribute{
			Description: routeDescriptions["group_wait"],
			Optional:    !isDatasource,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"match": schema.MapAttribute{
			Description: routeDescriptions["match"],
			Optional:    !isDatasource,
			Computed:    isDatasource,
			ElementType: types.StringType,
		},
		"match_regex": schema.MapAttribute{
			Description: routeDescriptions["match_regex"],
			Optional:    !isDatasource,
			Computed:    isDatasource,
			ElementType: types.StringType,
		},
		"receiver": schema.StringAttribute{
			Description: routeDescriptions["receiver"],
			Required:    !isDatasource,
			Computed:    isDatasource,
		},
		"repeat_interval": schema.StringAttribute{
			Description: routeDescriptions["repeat_interval"],
			Optional:    !isDatasource,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}

	if level != limit {
		attributesMap["routes"] = getRouteNestedObjectAux(isDatasource, level+1, limit)
	}

	return schema.ListNestedAttribute{
		Description: routeDescriptions["routes"],
		Optional:    !isDatasource,
		Computed:    isDatasource,
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: attributesMap,
		},
	}
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client *argus.APIClient
}

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_argus_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *argus.APIClient
	var err error
	if providerData.ArgusCustomEndpoint != "" {
		apiClient, err = argus.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ArgusCustomEndpoint),
		)
	} else {
		apiClient, err = argus.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Argus instance client configured")
}

var (
	descriptions = map[string]string{
		"main": "Argus instance resource schema. Must have a `region` specified in the provider configuration.",
		"deprecation_message": "The `stackit_argus_instance` resource has been deprecated and will be removed after February 26th 2025. " +
			"Please use `stackit_observability_instance` instead, which offers the exact same functionality.",
	}
	Schema = schema.Schema{
		Description:         fmt.Sprintf("%s\n%s", descriptions["main"], descriptions["deprecation_message"]),
		MarkdownDescription: fmt.Sprintf("%s\n\n!> %s", descriptions["main"], descriptions["deprecation_message"]),
		DeprecationMessage:  descriptions["deprecation_message"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the instance is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "The Argus instance ID.",
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
				Description: "The name of the Argus instance.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(200),
				},
			},
			"plan_name": schema.StringAttribute{
				Description: "Specifies the Argus plan. E.g. `Monitoring-Medium-EU01`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(200),
				},
			},
			"plan_id": schema.StringAttribute{
				Description: "The Argus plan ID.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"parameters": schema.MapAttribute{
				Description: "Additional parameters.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_url": schema.StringAttribute{
				Description: "Specifies Argus instance dashboard URL.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_updatable": schema.BoolAttribute{
				Description: "Specifies if the instance can be updated.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"grafana_public_read_access": schema.BoolAttribute{
				Description: "If true, anyone can access Grafana dashboards without logging in.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"grafana_url": schema.StringAttribute{
				Description: "Specifies Grafana URL.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"grafana_initial_admin_user": schema.StringAttribute{
				Description: "Specifies an initial Grafana admin username.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"grafana_initial_admin_password": schema.StringAttribute{
				Description: "Specifies an initial Grafana admin password.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"metrics_retention_days": schema.Int64Attribute{
				Description: "Specifies for how many days the raw metrics are kept.",
				Optional:    true,
				Computed:    true,
			},
			"metrics_retention_days_5m_downsampling": schema.Int64Attribute{
				Description: "Specifies for how many days the 5m downsampled metrics are kept. must be less than the value of the general retention. Default is set to `0` (disabled).",
				Optional:    true,
				Computed:    true,
			},
			"metrics_retention_days_1h_downsampling": schema.Int64Attribute{
				Description: "Specifies for how many days the 1h downsampled metrics are kept. must be less than the value of the 5m downsampling retention. Default is set to `0` (disabled).",
				Optional:    true,
				Computed:    true,
			},
			"metrics_url": schema.StringAttribute{
				Description: "Specifies metrics URL.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"metrics_push_url": schema.StringAttribute{
				Description: "Specifies URL for pushing metrics.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"targets_url": schema.StringAttribute{
				Description: "Specifies Targets URL.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"alerting_url": schema.StringAttribute{
				Description: "Specifies Alerting URL.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"logs_url": schema.StringAttribute{
				Description: "Specifies Logs URL.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"logs_push_url": schema.StringAttribute{
				Description: "Specifies URL for pushing logs.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"jaeger_traces_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"jaeger_ui_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"otlp_traces_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zipkin_spans_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"acl": schema.SetAttribute{
				Description: "The access control list for this instance. Each entry is an IP address range that is permitted to access, in CIDR notation.",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						validate.CIDR(),
					),
				},
			},
			"alert_config": schema.SingleNestedAttribute{
				Description: "Alert configuration for the instance.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"receivers": schema.ListNestedAttribute{
						Description: "List of alert receivers.",
						Required:    true,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Description: "Name of the receiver.",
									Required:    true,
								},
								"email_configs": schema.ListNestedAttribute{
									Description: "List of email configurations.",
									Optional:    true,
									Validators: []validator.List{
										listvalidator.SizeAtLeast(1),
									},
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"auth_identity": schema.StringAttribute{
												Description: "SMTP authentication information. Must be a valid email address",
												Optional:    true,
											},
											"auth_password": schema.StringAttribute{
												Description: "SMTP authentication password.",
												Optional:    true,
											},
											"auth_username": schema.StringAttribute{
												Description: "SMTP authentication username.",
												Optional:    true,
											},
											"from": schema.StringAttribute{
												Description: "The sender email address. Must be a valid email address",
												Optional:    true,
											},
											"smart_host": schema.StringAttribute{
												Description: "The SMTP host through which emails are sent.",
												Optional:    true,
											},
											"to": schema.StringAttribute{
												Description: "The email address to send notifications to. Must be a valid email address",
												Optional:    true,
											},
										},
									},
								},
								"opsgenie_configs": schema.ListNestedAttribute{
									Description: "List of OpsGenie configurations.",
									Optional:    true,
									Validators: []validator.List{
										listvalidator.SizeAtLeast(1),
									},
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"api_key": schema.StringAttribute{
												Description: "The API key for OpsGenie.",
												Optional:    true,
											},
											"api_url": schema.StringAttribute{
												Description: "The host to send OpsGenie API requests to. Must be a valid URL",
												Optional:    true,
											},
											"tags": schema.StringAttribute{
												Description: "Comma separated list of tags attached to the notifications.",
												Optional:    true,
											},
										},
									},
								},
								"webhooks_configs": schema.ListNestedAttribute{
									Description: "List of Webhooks configurations.",
									Optional:    true,
									Validators: []validator.List{
										listvalidator.SizeAtLeast(1),
									},
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"url": schema.StringAttribute{
												Description: "The endpoint to send HTTP POST requests to. Must be a valid URL",
												Optional:    true,
											},
											"ms_teams": schema.BoolAttribute{
												Description: "Microsoft Teams webhooks require special handling, set this to true if the webhook is for Microsoft Teams.",
												Optional:    true,
											},
										},
									},
								},
							},
						},
					},
					"route": schema.SingleNestedAttribute{
						Description: "Route configuration for the alerts.",
						Required:    true,
						Attributes: map[string]schema.Attribute{
							"group_by": schema.ListAttribute{
								Description: routeDescriptions["group_by"],
								Optional:    true,
								ElementType: types.StringType,
							},
							"group_interval": schema.StringAttribute{
								Description: routeDescriptions["group_interval"],
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"group_wait": schema.StringAttribute{
								Description: routeDescriptions["group_wait"],
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"match": schema.MapAttribute{
								Description: routeDescriptions["match"],
								Optional:    true,
								ElementType: types.StringType,
							},
							"match_regex": schema.MapAttribute{
								Description: routeDescriptions["match_regex"],
								Optional:    true,
								ElementType: types.StringType,
							},
							"receiver": schema.StringAttribute{
								Description: routeDescriptions["receiver"],
								Required:    true,
							},
							"repeat_interval": schema.StringAttribute{
								Description: routeDescriptions["repeat_interval"],
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"routes": getRouteNestedObject(),
						},
					},
					"global": schema.SingleNestedAttribute{
						Description: "Global configuration for the alerts.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"opsgenie_api_key": schema.StringAttribute{
								Description: "The API key for OpsGenie.",
								Optional:    true,
								Sensitive:   true,
							},
							"opsgenie_api_url": schema.StringAttribute{
								Description: "The host to send OpsGenie API requests to. Must be a valid URL",
								Optional:    true,
							},
							"resolve_timeout": schema.StringAttribute{
								Description: "The default value used by alertmanager if the alert does not include EndsAt. After this time passes, it can declare the alert as resolved if it has not been updated. This has no impact on alerts from Prometheus, as they always include EndsAt.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"smtp_auth_identity": schema.StringAttribute{
								Description: "SMTP authentication information. Must be a valid email address",
								Optional:    true,
							},
							"smtp_auth_password": schema.StringAttribute{
								Description: "SMTP Auth using LOGIN and PLAIN.",
								Optional:    true,
								Sensitive:   true,
							},
							"smtp_auth_username": schema.StringAttribute{
								Description: "SMTP Auth using CRAM-MD5, LOGIN and PLAIN. If empty, Alertmanager doesn't authenticate to the SMTP server.",
								Optional:    true,
							},
							"smtp_from": schema.StringAttribute{
								Description: "The default SMTP From header field. Must be a valid email address",
								Optional:    true,
								Computed:    true,
							},
							"smtp_smart_host": schema.StringAttribute{
								Description: "The default SMTP smarthost used for sending emails, including port number in format `host:port` (eg. `smtp.example.com:587`). Port number usually is 25, or 587 for SMTP over TLS (sometimes referred to as STARTTLS).",
								Optional:    true,
							},
						},
					},
				},
			},
		},
	}
)

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	acl := []string{}
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	metricsRetentionDays := conversion.Int64ValueToPointer(model.MetricsRetentionDays)
	metricsRetentionDays5mDownsampling := conversion.Int64ValueToPointer(model.MetricsRetentionDays5mDownsampling)
	metricsRetentionDays1hDownsampling := conversion.Int64ValueToPointer(model.MetricsRetentionDays1hDownsampling)

	alertConfig := alertConfigModel{}
	if !(model.AlertConfig.IsNull() || model.AlertConfig.IsUnknown()) {
		diags = model.AlertConfig.As(ctx, &alertConfig, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	err := r.loadPlanId(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Loading service plan: %v", err))
		return
	}
	// Generate API request body from model
	createPayload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	createResp, err := r.client.CreateInstance(ctx, projectId).CreateInstancePayload(*createPayload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	instanceId := createResp.InstanceId
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client, *instanceId, projectId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to instance populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create ACL
	err = updateACL(ctx, projectId, *instanceId, acl, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating ACL: %v", err))
		return
	}
	aclList, err := r.client.ListACL(ctx, *instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API to list ACL data: %v", err))
		return
	}

	// Map response body to schema
	err = mapACLField(aclList, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response for the ACL: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If any of the metrics retention days are set, set the metrics retention policy
	if metricsRetentionDays != nil || metricsRetentionDays5mDownsampling != nil || metricsRetentionDays1hDownsampling != nil {
		// Need to get the metrics retention policy because update endpoint is a PUT and we need to send all fields
		metricsResp, err := r.client.GetMetricsStorageRetentionExecute(ctx, *instanceId, projectId)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Getting metrics retention policy: %v", err))
			return
		}

		metricsRetentionPayload, err := toUpdateMetricsStorageRetentionPayload(metricsRetentionDays, metricsRetentionDays5mDownsampling, metricsRetentionDays1hDownsampling, metricsResp)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Building metrics retention policy payload: %v", err))
			return
		}

		_, err = r.client.UpdateMetricsStorageRetention(ctx, *instanceId, projectId).UpdateMetricsStorageRetentionPayload(*metricsRetentionPayload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Setting metrics retention policy: %v", err))
			return
		}
	}

	// Get metrics retention policy after update
	metricsResp, err := r.client.GetMetricsStorageRetentionExecute(ctx, *instanceId, projectId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Getting metrics retention policy: %v", err))
		return
	}
	// Map response body to schema
	err = mapMetricsRetentionField(metricsResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response for the metrics retention: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Alert Config
	if model.AlertConfig.IsUnknown() || model.AlertConfig.IsNull() {
		alertConfig, err = getMockAlertConfig(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Getting mock alert config: %v", err))
			return
		}
	}

	alertConfigPayload, err := toUpdateAlertConfigPayload(ctx, &alertConfig)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Building alert config payload: %v", err))
		return
	}

	if alertConfigPayload != nil {
		_, err = r.client.UpdateAlertConfigs(ctx, *instanceId, projectId).UpdateAlertConfigsPayload(*alertConfigPayload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Setting alert config: %v", err))
			return
		}
	}

	// Get alert config after update
	alertConfigResp, err := r.client.GetAlertConfigs(ctx, *instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Getting alert config: %v", err))
		return
	}

	// Map response body to schema
	err = mapAlertConfigField(ctx, alertConfigResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response for the alert config: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Argus instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	instanceResp, err := r.client.GetInstance(ctx, instanceId, projectId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if instanceResp != nil && instanceResp.Status != nil && *instanceResp.Status == wait.DeleteSuccess {
		resp.State.RemoveResource(ctx)
		return
	}

	aclListResp, err := r.client.ListACL(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API for ACL data: %v", err))
		return
	}

	metricsRetentionResp, err := r.client.GetMetricsStorageRetention(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API to get metrics retention: %v", err))
		return
	}

	alertConfigResp, err := r.client.GetAlertConfigs(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API to get alert config: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, instanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Map response body to schema
	err = mapACLField(aclListResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API response for the ACL: %v", err))
		return
	}

	// Map response body to schema
	err = mapMetricsRetentionField(metricsRetentionResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API response for the metrics retention: %v", err))
		return
	}

	// Map response body to schema
	err = mapAlertConfigField(ctx, alertConfigResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response for the alert config: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Argus instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()

	acl := []string{}
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	metricsRetentionDays := conversion.Int64ValueToPointer(model.MetricsRetentionDays)
	metricsRetentionDays5mDownsampling := conversion.Int64ValueToPointer(model.MetricsRetentionDays5mDownsampling)
	metricsRetentionDays1hDownsampling := conversion.Int64ValueToPointer(model.MetricsRetentionDays1hDownsampling)

	alertConfig := alertConfigModel{}
	if !(model.AlertConfig.IsNull() || model.AlertConfig.IsUnknown()) {
		diags = model.AlertConfig.As(ctx, &alertConfig, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	err := r.loadPlanId(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Loading service plan: %v", err))
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing instance
	_, err = r.client.UpdateInstance(ctx, instanceId, projectId).UpdateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.UpdateInstanceWaitHandler(ctx, r.client, instanceId, projectId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update ACL
	err = updateACL(ctx, projectId, instanceId, acl, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Updating ACL: %v", err))
		return
	}
	aclList, err := r.client.ListACL(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Calling API to list ACL data: %v", err))
		return
	}

	// Map response body to schema
	err = mapACLField(aclList, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API response for the ACL: %v", err))
		return
	}

	// Set state to ACL populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If any of the metrics retention days are set, set the metrics retention policy
	if metricsRetentionDays != nil || metricsRetentionDays5mDownsampling != nil || metricsRetentionDays1hDownsampling != nil {
		// Need to get the metrics retention policy because update endpoint is a PUT and we need to send all fields
		metricsResp, err := r.client.GetMetricsStorageRetentionExecute(ctx, instanceId, projectId)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Getting metrics retention policy: %v", err))
			return
		}

		metricsRetentionPayload, err := toUpdateMetricsStorageRetentionPayload(metricsRetentionDays, metricsRetentionDays5mDownsampling, metricsRetentionDays1hDownsampling, metricsResp)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Building metrics retention policy payload: %v", err))
			return
		}
		_, err = r.client.UpdateMetricsStorageRetention(ctx, instanceId, projectId).UpdateMetricsStorageRetentionPayload(*metricsRetentionPayload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Setting metrics retention policy: %v", err))
			return
		}
	}

	// Get metrics retention policy after update
	metricsResp, err := r.client.GetMetricsStorageRetentionExecute(ctx, instanceId, projectId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Getting metrics retention policy: %v", err))
		return
	}

	// Map response body to schema
	err = mapMetricsRetentionField(metricsResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API response for the metrics retention %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Alert Config
	if model.AlertConfig.IsUnknown() || model.AlertConfig.IsNull() {
		alertConfig, err = getMockAlertConfig(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Getting mock alert config: %v", err))
			return
		}
	}

	alertConfigPayload, err := toUpdateAlertConfigPayload(ctx, &alertConfig)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Building alert config payload: %v", err))
		return
	}

	if alertConfigPayload != nil {
		_, err = r.client.UpdateAlertConfigs(ctx, instanceId, projectId).UpdateAlertConfigsPayload(*alertConfigPayload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Setting alert config: %v", err))
			return
		}
	}

	// Get updated alert config
	alertConfigResp, err := r.client.GetAlertConfigs(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Calling API to get alert config: %v", err))
		return
	}

	// Map response body to schema
	err = mapAlertConfigField(ctx, alertConfigResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response for the alert config: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Argus instance updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()

	// Delete existing instance
	_, err := r.client.DeleteInstance(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client, instanceId, projectId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Argus instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	tflog.Info(ctx, "Argus instance state imported")
}

func mapFields(ctx context.Context, r *argus.GetInstanceResponse, model *Model) error {
	if r == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if r.Id != nil {
		instanceId = *r.Id
	} else {
		return fmt.Errorf("instance id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		instanceId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.InstanceId = types.StringValue(instanceId)
	model.PlanName = types.StringPointerValue(r.PlanName)
	model.PlanId = types.StringPointerValue(r.PlanId)
	model.Name = types.StringPointerValue(r.Name)

	ps := r.Parameters
	if ps == nil {
		model.Parameters = types.MapNull(types.StringType)
	} else {
		params := make(map[string]attr.Value, len(*ps))
		for k, v := range *ps {
			params[k] = types.StringValue(v)
		}
		res, diags := types.MapValueFrom(ctx, types.StringType, params)
		if diags.HasError() {
			return fmt.Errorf("parameter mapping %s", diags.Errors())
		}
		model.Parameters = res
	}

	model.IsUpdatable = types.BoolPointerValue(r.IsUpdatable)
	model.DashboardURL = types.StringPointerValue(r.DashboardUrl)
	if r.Instance != nil {
		i := *r.Instance
		model.GrafanaURL = types.StringPointerValue(i.GrafanaUrl)
		model.GrafanaPublicReadAccess = types.BoolPointerValue(i.GrafanaPublicReadAccess)
		model.GrafanaInitialAdminPassword = types.StringPointerValue(i.GrafanaAdminPassword)
		model.GrafanaInitialAdminUser = types.StringPointerValue(i.GrafanaAdminUser)
		model.MetricsRetentionDays = types.Int64Value(int64(*i.MetricsRetentionTimeRaw))
		model.MetricsRetentionDays5mDownsampling = types.Int64Value(int64(*i.MetricsRetentionTime5m))
		model.MetricsRetentionDays1hDownsampling = types.Int64Value(int64(*i.MetricsRetentionTime1h))
		model.MetricsURL = types.StringPointerValue(i.MetricsUrl)
		model.MetricsPushURL = types.StringPointerValue(i.PushMetricsUrl)
		model.TargetsURL = types.StringPointerValue(i.TargetsUrl)
		model.AlertingURL = types.StringPointerValue(i.AlertingUrl)
		model.LogsURL = types.StringPointerValue(i.LogsUrl)
		model.LogsPushURL = types.StringPointerValue(i.LogsPushUrl)
		model.JaegerTracesURL = types.StringPointerValue(i.JaegerTracesUrl)
		model.JaegerUIURL = types.StringPointerValue(i.JaegerUiUrl)
		model.OtlpTracesURL = types.StringPointerValue(i.OtlpTracesUrl)
		model.ZipkinSpansURL = types.StringPointerValue(i.ZipkinSpansUrl)
	}

	return nil
}

func mapACLField(aclList *argus.ListACLResponse, model *Model) error {
	if aclList == nil {
		return fmt.Errorf("mapping ACL: nil API response")
	}

	if aclList.Acl == nil || len(*aclList.Acl) == 0 {
		if !(model.ACL.IsNull() || model.ACL.IsUnknown() || model.ACL.Equal(types.SetValueMust(types.StringType, []attr.Value{}))) {
			model.ACL = types.SetNull(types.StringType)
		}
		return nil
	}

	acl := []attr.Value{}
	for _, cidr := range *aclList.Acl {
		acl = append(acl, types.StringValue(cidr))
	}
	aclTF, diags := types.SetValue(types.StringType, acl)
	if diags.HasError() {
		return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
	}
	model.ACL = aclTF
	return nil
}

func mapMetricsRetentionField(r *argus.GetMetricsStorageRetentionResponse, model *Model) error {
	if r == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if r.MetricsRetentionTimeRaw == nil || r.MetricsRetentionTime5m == nil || r.MetricsRetentionTime1h == nil {
		return fmt.Errorf("metrics retention time is nil")
	}

	stripedMetricsRetentionDays := strings.TrimSuffix(*r.MetricsRetentionTimeRaw, "d")
	metricsRetentionDays, err := strconv.ParseInt(stripedMetricsRetentionDays, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing metrics retention days: %w", err)
	}
	model.MetricsRetentionDays = types.Int64Value(metricsRetentionDays)

	stripedMetricsRetentionDays5m := strings.TrimSuffix(*r.MetricsRetentionTime5m, "d")
	metricsRetentionDays5m, err := strconv.ParseInt(stripedMetricsRetentionDays5m, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing metrics retention days 5m: %w", err)
	}
	model.MetricsRetentionDays5mDownsampling = types.Int64Value(metricsRetentionDays5m)

	stripedMetricsRetentionDays1h := strings.TrimSuffix(*r.MetricsRetentionTime1h, "d")
	metricsRetentionDays1h, err := strconv.ParseInt(stripedMetricsRetentionDays1h, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing metrics retention days 1h: %w", err)
	}
	model.MetricsRetentionDays1hDownsampling = types.Int64Value(metricsRetentionDays1h)

	return nil
}

func mapAlertConfigField(ctx context.Context, resp *argus.GetAlertConfigsResponse, model *Model) error {
	if resp == nil || resp.Data == nil {
		model.AlertConfig = types.ObjectNull(alertConfigTypes)
		return nil
	}

	if model == nil {
		return fmt.Errorf("nil model")
	}

	var alertConfigTF *alertConfigModel
	if !(model.AlertConfig.IsNull() || model.AlertConfig.IsUnknown()) {
		alertConfigTF = &alertConfigModel{}
		diags := model.AlertConfig.As(ctx, &alertConfigTF, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("mapping alert config: %w", core.DiagsToError(diags))
		}
	}

	respReceivers := resp.Data.Receivers
	respRoute := resp.Data.Route
	respGlobalConfigs := resp.Data.Global

	receiversList, err := mapReceiversToAttributes(ctx, respReceivers)
	if err != nil {
		return fmt.Errorf("mapping alert config receivers: %w", err)
	}

	route, err := mapRouteToAttributes(ctx, respRoute)
	if err != nil {
		return fmt.Errorf("mapping alert config route: %w", err)
	}

	var globalConfigModel *globalConfigurationModel
	if alertConfigTF != nil && !alertConfigTF.GlobalConfiguration.IsNull() && !alertConfigTF.GlobalConfiguration.IsUnknown() {
		globalConfigModel = &globalConfigurationModel{}
		diags := alertConfigTF.GlobalConfiguration.As(ctx, globalConfigModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("mapping alert config: %w", core.DiagsToError(diags))
		}
	}

	globalConfig, err := mapGlobalConfigToAttributes(respGlobalConfigs, globalConfigModel)
	if err != nil {
		return fmt.Errorf("mapping alert config global config: %w", err)
	}

	alertConfig, diags := types.ObjectValue(alertConfigTypes, map[string]attr.Value{
		"receivers": receiversList,
		"route":     route,
		"global":    globalConfig,
	})
	if diags.HasError() {
		return fmt.Errorf("converting alert config to TF type: %w", core.DiagsToError(diags))
	}

	// Check if the alert config is equal to the mock alert config
	// This is done because the Alert Config cannot be removed from the instance, but can be unset by the user in the Terraform configuration
	// If the alert config is equal to the mock alert config, we will map the Alert Config to an empty object in the Terraform state
	// This is done to avoid inconsistent applies or non-empty plans after applying
	mockAlertConfig, err := getMockAlertConfig(ctx)
	if err != nil {
		return fmt.Errorf("getting mock alert config: %w", err)
	}
	modelMockAlertConfig, diags := types.ObjectValueFrom(ctx, alertConfigTypes, mockAlertConfig)
	if diags.HasError() {
		return fmt.Errorf("converting mock alert config to TF type: %w", core.DiagsToError(diags))
	}
	if alertConfig.Equal(modelMockAlertConfig) {
		alertConfig = types.ObjectNull(alertConfigTypes)
	}

	model.AlertConfig = alertConfig
	return nil
}

// getMockAlertConfig returns a default alert config to be set in the instance if the alert config is unset in the Terraform configuration
//
// This is done because the Alert Config cannot be removed from the instance, but can be unset by the user in the Terraform configuration.
// So, we set the Alert Config in the instance to our mock configuration and
// map the Alert Config to an empty object in the Terraform state if it matches the mock alert config
func getMockAlertConfig(ctx context.Context) (alertConfigModel, error) {
	mockEmailConfig, diags := types.ObjectValue(emailConfigsTypes, map[string]attr.Value{
		"to":            types.StringValue("123@gmail.com"),
		"smart_host":    types.StringValue("smtp.gmail.com:587"),
		"from":          types.StringValue("xxxx@gmail.com"),
		"auth_username": types.StringValue("xxxx@gmail.com"),
		"auth_password": types.StringValue("xxxxxxxxx"),
		"auth_identity": types.StringValue("xxxx@gmail.com"),
	})
	if diags.HasError() {
		return alertConfigModel{}, fmt.Errorf("mapping email config: %w", core.DiagsToError(diags))
	}

	mockEmailConfigs, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: emailConfigsTypes}, []attr.Value{
		mockEmailConfig,
	})
	if diags.HasError() {
		return alertConfigModel{}, fmt.Errorf("mapping email configs: %w", core.DiagsToError(diags))
	}

	mockReceiver, diags := types.ObjectValue(receiversTypes, map[string]attr.Value{
		"name":             types.StringValue("email-me"),
		"email_configs":    mockEmailConfigs,
		"opsgenie_configs": types.ListNull(types.ObjectType{AttrTypes: opsgenieConfigsTypes}),
		"webhooks_configs": types.ListNull(types.ObjectType{AttrTypes: webHooksConfigsTypes}),
	})
	if diags.HasError() {
		return alertConfigModel{}, fmt.Errorf("mapping receiver: %w", core.DiagsToError(diags))
	}

	mockReceivers, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
		mockReceiver,
	})
	if diags.HasError() {
		return alertConfigModel{}, fmt.Errorf("mapping receivers: %w", core.DiagsToError(diags))
	}

	mockGroupByList, diags := types.ListValueFrom(ctx, types.StringType, []attr.Value{
		types.StringValue("job"),
	})
	if diags.HasError() {
		return alertConfigModel{}, fmt.Errorf("mapping group by list: %w", core.DiagsToError(diags))
	}

	mockRoute, diags := types.ObjectValue(routeTypes, map[string]attr.Value{
		"receiver":        types.StringValue("email-me"),
		"group_by":        mockGroupByList,
		"group_wait":      types.StringValue("30s"),
		"group_interval":  types.StringValue("5m"),
		"repeat_interval": types.StringValue("4h"),
		"match":           types.MapNull(types.StringType),
		"match_regex":     types.MapNull(types.StringType),
		"routes":          types.ListNull(getRouteListType()),
	})
	if diags.HasError() {
		return alertConfigModel{}, fmt.Errorf("mapping route: %w", core.DiagsToError(diags))
	}

	mockGlobalConfig, diags := types.ObjectValue(globalConfigurationTypes, map[string]attr.Value{
		"opsgenie_api_key":   types.StringNull(),
		"opsgenie_api_url":   types.StringNull(),
		"resolve_timeout":    types.StringValue("5m"),
		"smtp_auth_identity": types.StringNull(),
		"smtp_auth_password": types.StringNull(),
		"smtp_auth_username": types.StringNull(),
		"smtp_from":          types.StringValue("argus@argus.stackit.cloud"),
		"smtp_smart_host":    types.StringNull(),
	})
	if diags.HasError() {
		return alertConfigModel{}, fmt.Errorf("mapping global config: %w", core.DiagsToError(diags))
	}

	return alertConfigModel{
		Receivers:           mockReceivers,
		Route:               mockRoute,
		GlobalConfiguration: mockGlobalConfig,
	}, nil
}

func mapGlobalConfigToAttributes(respGlobalConfigs *argus.Global, globalConfigsTF *globalConfigurationModel) (basetypes.ObjectValue, error) {
	if respGlobalConfigs == nil {
		return types.ObjectNull(globalConfigurationTypes), nil
	}

	// This bypass is needed because these values are not returned in the API GET response
	smtpSmartHost := respGlobalConfigs.SmtpSmarthost
	smtpAuthIdentity := respGlobalConfigs.SmtpAuthIdentity
	smtpAuthPassword := respGlobalConfigs.SmtpAuthPassword
	smtpAuthUsername := respGlobalConfigs.SmtpAuthUsername
	if globalConfigsTF != nil {
		if respGlobalConfigs.SmtpSmarthost == nil &&
			!globalConfigsTF.SmtpSmartHost.IsNull() && !globalConfigsTF.SmtpSmartHost.IsUnknown() {
			smtpSmartHost = utils.Ptr(globalConfigsTF.SmtpSmartHost.ValueString())
		}
		if respGlobalConfigs.SmtpAuthIdentity == nil &&
			!globalConfigsTF.SmtpAuthIdentity.IsNull() && !globalConfigsTF.SmtpAuthIdentity.IsUnknown() {
			smtpAuthIdentity = utils.Ptr(globalConfigsTF.SmtpAuthIdentity.ValueString())
		}
		if respGlobalConfigs.SmtpAuthPassword == nil &&
			!globalConfigsTF.SmtpAuthPassword.IsNull() && !globalConfigsTF.SmtpAuthPassword.IsUnknown() {
			smtpAuthPassword = utils.Ptr(globalConfigsTF.SmtpAuthPassword.ValueString())
		}
		if respGlobalConfigs.SmtpAuthUsername == nil &&
			!globalConfigsTF.SmtpAuthUsername.IsNull() && !globalConfigsTF.SmtpAuthUsername.IsUnknown() {
			smtpAuthUsername = utils.Ptr(globalConfigsTF.SmtpAuthUsername.ValueString())
		}
	}

	globalConfigObject, diags := types.ObjectValue(globalConfigurationTypes, map[string]attr.Value{
		"opsgenie_api_key":   types.StringPointerValue(respGlobalConfigs.OpsgenieApiKey),
		"opsgenie_api_url":   types.StringPointerValue(respGlobalConfigs.OpsgenieApiUrl),
		"resolve_timeout":    types.StringPointerValue(respGlobalConfigs.ResolveTimeout),
		"smtp_from":          types.StringPointerValue(respGlobalConfigs.SmtpFrom),
		"smtp_auth_identity": types.StringPointerValue(smtpAuthIdentity),
		"smtp_auth_password": types.StringPointerValue(smtpAuthPassword),
		"smtp_auth_username": types.StringPointerValue(smtpAuthUsername),
		"smtp_smart_host":    types.StringPointerValue(smtpSmartHost),
	})
	if diags.HasError() {
		return types.ObjectNull(globalConfigurationTypes), fmt.Errorf("mapping global config: %w", core.DiagsToError(diags))
	}

	return globalConfigObject, nil
}

func mapReceiversToAttributes(ctx context.Context, respReceivers *[]argus.Receivers) (basetypes.ListValue, error) {
	if respReceivers == nil {
		return types.ListNull(types.ObjectType{AttrTypes: receiversTypes}), nil
	}
	receiversList := []attr.Value{}
	emptyList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{})
	if diags.HasError() {
		// Should not happen
		return emptyList, fmt.Errorf("mapping empty list: %w", core.DiagsToError(diags))
	}

	if len(*respReceivers) == 0 {
		return emptyList, nil
	}

	for i := range *respReceivers {
		receiver := (*respReceivers)[i]

		emailConfigList := []attr.Value{}
		if receiver.EmailConfigs != nil {
			for _, emailConfig := range *receiver.EmailConfigs {
				emailConfigMap := map[string]attr.Value{
					"auth_identity": types.StringPointerValue(emailConfig.AuthIdentity),
					"auth_password": types.StringPointerValue(emailConfig.AuthPassword),
					"auth_username": types.StringPointerValue(emailConfig.AuthUsername),
					"from":          types.StringPointerValue(emailConfig.From),
					"smart_host":    types.StringPointerValue(emailConfig.Smarthost),
					"to":            types.StringPointerValue(emailConfig.To),
				}
				emailConfigModel, diags := types.ObjectValue(emailConfigsTypes, emailConfigMap)
				if diags.HasError() {
					return emptyList, fmt.Errorf("mapping email config: %w", core.DiagsToError(diags))
				}
				emailConfigList = append(emailConfigList, emailConfigModel)
			}
		}

		opsgenieConfigList := []attr.Value{}
		if receiver.OpsgenieConfigs != nil {
			for _, opsgenieConfig := range *receiver.OpsgenieConfigs {
				opsGenieConfigMap := map[string]attr.Value{
					"api_key": types.StringPointerValue(opsgenieConfig.ApiKey),
					"api_url": types.StringPointerValue(opsgenieConfig.ApiUrl),
					"tags":    types.StringPointerValue(opsgenieConfig.Tags),
				}
				opsGenieConfigModel, diags := types.ObjectValue(opsgenieConfigsTypes, opsGenieConfigMap)
				if diags.HasError() {
					return emptyList, fmt.Errorf("mapping opsgenie config: %w", core.DiagsToError(diags))
				}
				opsgenieConfigList = append(opsgenieConfigList, opsGenieConfigModel)
			}
		}

		webhooksConfigList := []attr.Value{}
		if receiver.WebHookConfigs != nil {
			for _, webhookConfig := range *receiver.WebHookConfigs {
				webHookConfigsMap := map[string]attr.Value{
					"url":      types.StringPointerValue(webhookConfig.Url),
					"ms_teams": types.BoolPointerValue(webhookConfig.MsTeams),
				}
				webHookConfigsModel, diags := types.ObjectValue(webHooksConfigsTypes, webHookConfigsMap)
				if diags.HasError() {
					return emptyList, fmt.Errorf("mapping webhooks config: %w", core.DiagsToError(diags))
				}
				webhooksConfigList = append(webhooksConfigList, webHookConfigsModel)
			}
		}

		if receiver.Name == nil {
			return emptyList, fmt.Errorf("receiver name is nil")
		}

		var emailConfigs basetypes.ListValue
		if len(emailConfigList) == 0 {
			emailConfigs = types.ListNull(types.ObjectType{AttrTypes: emailConfigsTypes})
		} else {
			emailConfigs, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: emailConfigsTypes}, emailConfigList)
			if diags.HasError() {
				return emptyList, fmt.Errorf("mapping email configs: %w", core.DiagsToError(diags))
			}
		}

		var opsGenieConfigs basetypes.ListValue
		if len(opsgenieConfigList) == 0 {
			opsGenieConfigs = types.ListNull(types.ObjectType{AttrTypes: opsgenieConfigsTypes})
		} else {
			opsGenieConfigs, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: opsgenieConfigsTypes}, opsgenieConfigList)
			if diags.HasError() {
				return emptyList, fmt.Errorf("mapping opsgenie configs: %w", core.DiagsToError(diags))
			}
		}

		var webHooksConfigs basetypes.ListValue
		if len(webhooksConfigList) == 0 {
			webHooksConfigs = types.ListNull(types.ObjectType{AttrTypes: webHooksConfigsTypes})
		} else {
			webHooksConfigs, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: webHooksConfigsTypes}, webhooksConfigList)
			if diags.HasError() {
				return emptyList, fmt.Errorf("mapping webhooks configs: %w", core.DiagsToError(diags))
			}
		}

		receiverMap := map[string]attr.Value{
			"name":             types.StringPointerValue(receiver.Name),
			"email_configs":    emailConfigs,
			"opsgenie_configs": opsGenieConfigs,
			"webhooks_configs": webHooksConfigs,
		}

		receiversModel, diags := types.ObjectValue(receiversTypes, receiverMap)
		if diags.HasError() {
			return emptyList, fmt.Errorf("mapping receiver: %w", core.DiagsToError(diags))
		}

		receiversList = append(receiversList, receiversModel)
	}

	returnReceiversList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: receiversTypes}, receiversList)
	if diags.HasError() {
		return emptyList, fmt.Errorf("mapping receivers list: %w", core.DiagsToError(diags))
	}
	return returnReceiversList, nil
}

func mapRouteToAttributes(ctx context.Context, route *argus.Route) (attr.Value, error) {
	if route == nil {
		return types.ObjectNull(routeTypes), nil
	}

	groupByModel, diags := types.ListValueFrom(ctx, types.StringType, route.GroupBy)
	if diags.HasError() {
		return types.ObjectNull(routeTypes), fmt.Errorf("mapping group by: %w", core.DiagsToError(diags))
	}

	matchModel, diags := types.MapValueFrom(ctx, types.StringType, route.Match)
	if diags.HasError() {
		return types.ObjectNull(routeTypes), fmt.Errorf("mapping match: %w", core.DiagsToError(diags))
	}

	matchRegexModel, diags := types.MapValueFrom(ctx, types.StringType, route.MatchRe)
	if diags.HasError() {
		return types.ObjectNull(routeTypes), fmt.Errorf("mapping match regex: %w", core.DiagsToError(diags))
	}

	childRoutes, err := mapChildRoutesToAttributes(ctx, route.Routes)
	if err != nil {
		return types.ObjectNull(routeTypes), fmt.Errorf("mapping child routes: %w", err)
	}

	routeMap := map[string]attr.Value{
		"group_by":        groupByModel,
		"group_interval":  types.StringPointerValue(route.GroupInterval),
		"group_wait":      types.StringPointerValue(route.GroupWait),
		"match":           matchModel,
		"match_regex":     matchRegexModel,
		"receiver":        types.StringPointerValue(route.Receiver),
		"repeat_interval": types.StringPointerValue(route.RepeatInterval),
		"routes":          childRoutes,
	}

	routeModel, diags := types.ObjectValue(routeTypes, routeMap)
	if diags.HasError() {
		return types.ObjectNull(routeTypes), fmt.Errorf("converting route to TF types: %w", core.DiagsToError(diags))
	}

	return routeModel, nil
}

// mapChildRoutesToAttributes maps the child routes to the Terraform attributes
// This should be a recursive function to handle nested child routes
// However, the API does not currently have the correct type for the child routes
// In the future, the current implementation should be the final case of the recursive function
func mapChildRoutesToAttributes(ctx context.Context, routes *[]argus.RouteSerializer) (basetypes.ListValue, error) {
	nullList := types.ListNull(getRouteListType())
	if routes == nil {
		return nullList, nil
	}

	routesList := []attr.Value{}
	for _, route := range *routes {
		groupByModel, diags := types.ListValueFrom(ctx, types.StringType, route.GroupBy)
		if diags.HasError() {
			return nullList, fmt.Errorf("mapping group by: %w", core.DiagsToError(diags))
		}

		matchModel, diags := types.MapValueFrom(ctx, types.StringType, route.Match)
		if diags.HasError() {
			return nullList, fmt.Errorf("mapping match: %w", core.DiagsToError(diags))
		}

		matchRegexModel, diags := types.MapValueFrom(ctx, types.StringType, route.MatchRe)
		if diags.HasError() {
			return nullList, fmt.Errorf("mapping match regex: %w", core.DiagsToError(diags))
		}

		routeMap := map[string]attr.Value{
			"group_by":        groupByModel,
			"group_interval":  types.StringPointerValue(route.GroupInterval),
			"group_wait":      types.StringPointerValue(route.GroupWait),
			"match":           matchModel,
			"match_regex":     matchRegexModel,
			"receiver":        types.StringPointerValue(route.Receiver),
			"repeat_interval": types.StringPointerValue(route.RepeatInterval),
		}

		routeModel, diags := types.ObjectValue(getRouteListType().AttrTypes, routeMap)
		if diags.HasError() {
			return types.ListNull(getRouteListType()), fmt.Errorf("converting child route to TF types: %w", core.DiagsToError(diags))
		}

		routesList = append(routesList, routeModel)
	}

	returnRoutesList, diags := types.ListValueFrom(ctx, getRouteListType(), routesList)
	if diags.HasError() {
		return nullList, fmt.Errorf("mapping child routes list: %w", core.DiagsToError(diags))
	}
	return returnRoutesList, nil
}

func toCreatePayload(model *Model) (*argus.CreateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	elements := model.Parameters.Elements()
	pa := make(map[string]interface{}, len(elements))
	for k := range elements {
		pa[k] = elements[k].String()
	}
	return &argus.CreateInstancePayload{
		Name:      conversion.StringValueToPointer(model.Name),
		PlanId:    conversion.StringValueToPointer(model.PlanId),
		Parameter: &pa,
	}, nil
}

func toUpdateMetricsStorageRetentionPayload(retentionDaysRaw, retentionDays5m, retentionDays1h *int64, resp *argus.GetMetricsStorageRetentionResponse) (*argus.UpdateMetricsStorageRetentionPayload, error) {
	var retentionTimeRaw string
	var retentionTime5m string
	var retentionTime1h string

	if resp == nil || resp.MetricsRetentionTimeRaw == nil || resp.MetricsRetentionTime5m == nil || resp.MetricsRetentionTime1h == nil {
		return nil, fmt.Errorf("nil response")
	}

	if retentionDaysRaw == nil {
		retentionTimeRaw = *resp.MetricsRetentionTimeRaw
	} else {
		retentionTimeRaw = fmt.Sprintf("%dd", *retentionDaysRaw)
	}

	if retentionDays5m == nil {
		retentionTime5m = *resp.MetricsRetentionTime5m
	} else {
		retentionTime5m = fmt.Sprintf("%dd", *retentionDays5m)
	}

	if retentionDays1h == nil {
		retentionTime1h = *resp.MetricsRetentionTime1h
	} else {
		retentionTime1h = fmt.Sprintf("%dd", *retentionDays1h)
	}

	return &argus.UpdateMetricsStorageRetentionPayload{
		MetricsRetentionTimeRaw: &retentionTimeRaw,
		MetricsRetentionTime5m:  &retentionTime5m,
		MetricsRetentionTime1h:  &retentionTime1h,
	}, nil
}

func updateACL(ctx context.Context, projectId, instanceId string, acl []string, client *argus.APIClient) error {
	payload := argus.UpdateACLPayload{
		Acl: utils.Ptr(acl),
	}

	_, err := client.UpdateACL(ctx, instanceId, projectId).UpdateACLPayload(payload).Execute()
	if err != nil {
		return fmt.Errorf("updating ACL: %w", err)
	}

	return nil
}

func toUpdatePayload(model *Model) (*argus.UpdateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	elements := model.Parameters.Elements()
	pa := make(map[string]interface{}, len(elements))
	for k, v := range elements {
		pa[k] = v.String()
	}
	return &argus.UpdateInstancePayload{
		Name:      conversion.StringValueToPointer(model.Name),
		PlanId:    conversion.StringValueToPointer(model.PlanId),
		Parameter: &pa,
	}, nil
}

func toUpdateAlertConfigPayload(ctx context.Context, model *alertConfigModel) (*argus.UpdateAlertConfigsPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if model.Receivers.IsNull() || model.Receivers.IsUnknown() {
		return nil, fmt.Errorf("receivers in the model are null or unknown")
	}

	if model.Route.IsNull() || model.Route.IsUnknown() {
		return nil, fmt.Errorf("route in the model is null or unknown")
	}

	var err error

	payload := argus.UpdateAlertConfigsPayload{}

	payload.Receivers, err = toReceiverPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("mapping receivers: %w", err)
	}

	routeTF := routeModel{}
	diags := model.Route.As(ctx, &routeTF, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("mapping route: %w", core.DiagsToError(diags))
	}

	payload.Route, err = toRoutePayload(ctx, &routeTF)
	if err != nil {
		return nil, fmt.Errorf("mapping route: %w", err)
	}

	if !model.GlobalConfiguration.IsNull() && !model.GlobalConfiguration.IsUnknown() {
		payload.Global, err = toGlobalConfigPayload(ctx, model)
		if err != nil {
			return nil, fmt.Errorf("mapping global: %w", err)
		}
	}

	return &payload, nil
}

func toReceiverPayload(ctx context.Context, model *alertConfigModel) (*[]argus.UpdateAlertConfigsPayloadReceiversInner, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	receiversModel := []receiversModel{}
	diags := model.Receivers.ElementsAs(ctx, &receiversModel, false)
	if diags.HasError() {
		return nil, fmt.Errorf("mapping receivers: %w", core.DiagsToError(diags))
	}

	receivers := []argus.UpdateAlertConfigsPayloadReceiversInner{}

	for i := range receiversModel {
		receiver := receiversModel[i]
		receiverPayload := argus.UpdateAlertConfigsPayloadReceiversInner{
			Name: conversion.StringValueToPointer(receiver.Name),
		}

		if !receiver.EmailConfigs.IsNull() && !receiver.EmailConfigs.IsUnknown() {
			emailConfigs := []emailConfigsModel{}
			diags := receiver.EmailConfigs.ElementsAs(ctx, &emailConfigs, false)
			if diags.HasError() {
				return nil, fmt.Errorf("mapping email configs: %w", core.DiagsToError(diags))
			}
			payloadEmailConfigs := []argus.CreateAlertConfigReceiverPayloadEmailConfigsInner{}
			for i := range emailConfigs {
				emailConfig := emailConfigs[i]
				payloadEmailConfigs = append(payloadEmailConfigs, argus.CreateAlertConfigReceiverPayloadEmailConfigsInner{
					AuthIdentity: conversion.StringValueToPointer(emailConfig.AuthIdentity),
					AuthPassword: conversion.StringValueToPointer(emailConfig.AuthPassword),
					AuthUsername: conversion.StringValueToPointer(emailConfig.AuthUsername),
					From:         conversion.StringValueToPointer(emailConfig.From),
					Smarthost:    conversion.StringValueToPointer(emailConfig.Smarthost),
					To:           conversion.StringValueToPointer(emailConfig.To),
				})
			}
			receiverPayload.EmailConfigs = &payloadEmailConfigs
		}

		if !receiver.OpsGenieConfigs.IsNull() && !receiver.OpsGenieConfigs.IsUnknown() {
			opsgenieConfigs := []opsgenieConfigsModel{}
			diags := receiver.OpsGenieConfigs.ElementsAs(ctx, &opsgenieConfigs, false)
			if diags.HasError() {
				return nil, fmt.Errorf("mapping opsgenie configs: %w", core.DiagsToError(diags))
			}
			payloadOpsGenieConfigs := []argus.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner{}
			for i := range opsgenieConfigs {
				opsgenieConfig := opsgenieConfigs[i]
				payloadOpsGenieConfigs = append(payloadOpsGenieConfigs, argus.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner{
					ApiKey: conversion.StringValueToPointer(opsgenieConfig.ApiKey),
					ApiUrl: conversion.StringValueToPointer(opsgenieConfig.ApiUrl),
					Tags:   conversion.StringValueToPointer(opsgenieConfig.Tags),
				})
			}
			receiverPayload.OpsgenieConfigs = &payloadOpsGenieConfigs
		}

		if !receiver.WebHooksConfigs.IsNull() && !receiver.WebHooksConfigs.IsUnknown() {
			receiverWebHooksConfigs := []webHooksConfigsModel{}
			diags := receiver.WebHooksConfigs.ElementsAs(ctx, &receiverWebHooksConfigs, false)
			if diags.HasError() {
				return nil, fmt.Errorf("mapping webhooks configs: %w", core.DiagsToError(diags))
			}
			payloadWebHooksConfigs := []argus.CreateAlertConfigReceiverPayloadWebHookConfigsInner{}
			for i := range receiverWebHooksConfigs {
				webHooksConfig := receiverWebHooksConfigs[i]
				payloadWebHooksConfigs = append(payloadWebHooksConfigs, argus.CreateAlertConfigReceiverPayloadWebHookConfigsInner{
					Url:     conversion.StringValueToPointer(webHooksConfig.Url),
					MsTeams: conversion.BoolValueToPointer(webHooksConfig.MsTeams),
				})
			}
			receiverPayload.WebHookConfigs = &payloadWebHooksConfigs
		}

		receivers = append(receivers, receiverPayload)
	}
	return &receivers, nil
}

func toRoutePayload(ctx context.Context, routeTF *routeModel) (*argus.UpdateAlertConfigsPayloadRoute, error) {
	if routeTF == nil {
		return nil, fmt.Errorf("nil route model")
	}

	var groupByPayload *[]string
	var matchPayload *map[string]interface{}
	var matchRegexPayload *map[string]interface{}
	var childRoutesPayload *[]argus.CreateAlertConfigRoutePayloadRoutesInner

	if !routeTF.GroupBy.IsNull() && !routeTF.GroupBy.IsUnknown() {
		groupByPayload = &[]string{}
		diags := routeTF.GroupBy.ElementsAs(ctx, groupByPayload, false)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping group by: %w", core.DiagsToError(diags))
		}
	}

	if !routeTF.Match.IsNull() && !routeTF.Match.IsUnknown() {
		matchMap, err := conversion.ToStringInterfaceMap(ctx, routeTF.Match)
		if err != nil {
			return nil, fmt.Errorf("mapping match: %w", err)
		}
		matchPayload = &matchMap
	}

	if !routeTF.MatchRegex.IsNull() && !routeTF.MatchRegex.IsUnknown() {
		matchRegexMap, err := conversion.ToStringInterfaceMap(ctx, routeTF.MatchRegex)
		if err != nil {
			return nil, fmt.Errorf("mapping match regex: %w", err)
		}
		matchRegexPayload = &matchRegexMap
	}

	if !routeTF.Routes.IsNull() && !routeTF.Routes.IsUnknown() {
		childRoutes := []routeModel{}
		diags := routeTF.Routes.ElementsAs(ctx, &childRoutes, false)
		if diags.HasError() {
			// If there is an error, we will try to map the child routes as if they are the last child routes
			// This is done because the last child routes in the recursion have a different structure (don't have the `routes` fields)
			// and need to be unpacked to a different struct (routeModelNoRoutes)
			lastChildRoutes := []routeModelNoRoutes{}
			diags = routeTF.Routes.ElementsAs(ctx, &lastChildRoutes, true)
			if diags.HasError() {
				return nil, fmt.Errorf("mapping child routes: %w", core.DiagsToError(diags))
			}
			for i := range lastChildRoutes {
				childRoute := routeModel{
					GroupBy:        lastChildRoutes[i].GroupBy,
					GroupInterval:  lastChildRoutes[i].GroupInterval,
					GroupWait:      lastChildRoutes[i].GroupWait,
					Match:          lastChildRoutes[i].Match,
					MatchRegex:     lastChildRoutes[i].MatchRegex,
					Receiver:       lastChildRoutes[i].Receiver,
					RepeatInterval: lastChildRoutes[i].RepeatInterval,
					Routes:         types.ListNull(getRouteListType()),
				}
				childRoutes = append(childRoutes, childRoute)
			}
		}

		childRoutesList := []argus.CreateAlertConfigRoutePayloadRoutesInner{}
		for i := range childRoutes {
			childRoute := childRoutes[i]
			childRoutePayload, err := toRoutePayload(ctx, &childRoute)
			if err != nil {
				return nil, fmt.Errorf("mapping child route: %w", err)
			}
			childRoutesList = append(childRoutesList, *toChildRoutePayload(childRoutePayload))
		}

		childRoutesPayload = &childRoutesList
	}

	return &argus.UpdateAlertConfigsPayloadRoute{
		GroupBy:        groupByPayload,
		GroupInterval:  conversion.StringValueToPointer(routeTF.GroupInterval),
		GroupWait:      conversion.StringValueToPointer(routeTF.GroupWait),
		Match:          matchPayload,
		MatchRe:        matchRegexPayload,
		Receiver:       conversion.StringValueToPointer(routeTF.Receiver),
		RepeatInterval: conversion.StringValueToPointer(routeTF.RepeatInterval),
		Routes:         childRoutesPayload,
	}, nil
}

func toChildRoutePayload(in *argus.UpdateAlertConfigsPayloadRoute) *argus.CreateAlertConfigRoutePayloadRoutesInner {
	if in == nil {
		return nil
	}
	return &argus.CreateAlertConfigRoutePayloadRoutesInner{
		GroupBy:        in.GroupBy,
		GroupInterval:  in.GroupInterval,
		GroupWait:      in.GroupWait,
		Match:          in.Match,
		MatchRe:        in.MatchRe,
		Receiver:       in.Receiver,
		RepeatInterval: in.RepeatInterval,
		// Routes not currently supported
	}
}

func toGlobalConfigPayload(ctx context.Context, model *alertConfigModel) (*argus.UpdateAlertConfigsPayloadGlobal, error) {
	globalConfigModel := globalConfigurationModel{}
	diags := model.GlobalConfiguration.As(ctx, &globalConfigModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("mapping global configuration: %w", core.DiagsToError(diags))
	}

	return &argus.UpdateAlertConfigsPayloadGlobal{
		OpsgenieApiKey:   conversion.StringValueToPointer(globalConfigModel.OpsgenieApiKey),
		OpsgenieApiUrl:   conversion.StringValueToPointer(globalConfigModel.OpsgenieApiUrl),
		ResolveTimeout:   conversion.StringValueToPointer(globalConfigModel.ResolveTimeout),
		SmtpAuthIdentity: conversion.StringValueToPointer(globalConfigModel.SmtpAuthIdentity),
		SmtpAuthPassword: conversion.StringValueToPointer(globalConfigModel.SmtpAuthPassword),
		SmtpAuthUsername: conversion.StringValueToPointer(globalConfigModel.SmtpAuthUsername),
		SmtpFrom:         conversion.StringValueToPointer(globalConfigModel.SmtpFrom),
		SmtpSmarthost:    conversion.StringValueToPointer(globalConfigModel.SmtpSmartHost),
	}, nil
}

func (r *instanceResource) loadPlanId(ctx context.Context, model *Model) error {
	projectId := model.ProjectId.ValueString()
	res, err := r.client.ListPlans(ctx, projectId).Execute()
	if err != nil {
		return err
	}

	planName := model.PlanName.ValueString()
	avl := ""
	plans := *res.Plans
	for i := range plans {
		p := plans[i]
		if p.Name == nil {
			continue
		}
		if strings.EqualFold(*p.Name, planName) && p.PlanId != nil {
			model.PlanId = types.StringPointerValue(p.PlanId)
			break
		}
		avl = fmt.Sprintf("%s\n- %s", avl, *p.Name)
	}
	if model.PlanId.ValueString() == "" {
		return fmt.Errorf("couldn't find plan_name '%s', available names are: %s", planName, avl)
	}
	return nil
}
