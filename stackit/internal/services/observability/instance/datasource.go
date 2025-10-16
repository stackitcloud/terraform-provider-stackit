package observability

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	observabilityUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &instanceDataSource{}
)

// NewInstanceDataSource is a helper function to simplify the provider implementation.
func NewInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

// instanceDataSource is the data source implementation.
type instanceDataSource struct {
	client *observability.APIClient
}

// Metadata returns the data source type name.
func (d *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_instance"
}

func (d *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := observabilityUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Observability instance client configured")
}

// Schema defines the schema for the data source.
func (d *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Observability instance data source schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`instance_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the instance is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "The Observability instance ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Observability instance.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(300),
				},
			},
			"plan_name": schema.StringAttribute{
				Description: "Specifies the Observability plan. E.g. `Observability-Monitoring-Medium-EU01`.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(200),
				},
			},
			"plan_id": schema.StringAttribute{
				Description: "The Observability plan ID.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"parameters": schema.MapAttribute{
				Description: "Additional parameters.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"dashboard_url": schema.StringAttribute{
				Description: "Specifies Observability instance dashboard URL.",
				Computed:    true,
			},
			"is_updatable": schema.BoolAttribute{
				Description: "Specifies if the instance can be updated.",
				Computed:    true,
			},
			"grafana_public_read_access": schema.BoolAttribute{
				Description: "If true, anyone can access Grafana dashboards without logging in.",
				Computed:    true,
			},
			"grafana_url": schema.StringAttribute{
				Description: "Specifies Grafana URL.",
				Computed:    true,
			},
			"grafana_initial_admin_user": schema.StringAttribute{
				Description: "Specifies an initial Grafana admin username.",
				Computed:    true,
			},
			"grafana_initial_admin_password": schema.StringAttribute{
				Description: "Specifies an initial Grafana admin password.",
				Computed:    true,
				Sensitive:   true,
			},
			"traces_retention_days": schema.Int64Attribute{
				Description: "Specifies for how many days the traces are kept. Default is set to `7`.",
				Computed:    true,
			},
			"logs_retention_days": schema.Int64Attribute{
				Description: "Specifies for how many days the logs are kept. Default is set to `7`.",
				Computed:    true,
			},
			"metrics_retention_days": schema.Int64Attribute{
				Description: "Specifies for how many days the raw metrics are kept. Default is set to `90`.",
				Computed:    true,
			},
			"metrics_retention_days_5m_downsampling": schema.Int64Attribute{
				Description: "Specifies for how many days the 5m downsampled metrics are kept. must be less than the value of the general retention. Default is set to `90`.",
				Computed:    true,
			},
			"metrics_retention_days_1h_downsampling": schema.Int64Attribute{
				Description: "Specifies for how many days the 1h downsampled metrics are kept. must be less than the value of the 5m downsampling retention. Default is set to `90`.",
				Computed:    true,
			},
			"metrics_url": schema.StringAttribute{
				Description: "Specifies metrics URL.",
				Computed:    true,
			},
			"metrics_push_url": schema.StringAttribute{
				Description: "Specifies URL for pushing metrics.",
				Computed:    true,
			},
			"targets_url": schema.StringAttribute{
				Description: "Specifies Targets URL.",
				Computed:    true,
			},
			"alerting_url": schema.StringAttribute{
				Description: "Specifies Alerting URL.",
				Computed:    true,
			},
			"logs_url": schema.StringAttribute{
				Description: "Specifies Logs URL.",
				Computed:    true,
			},
			"logs_push_url": schema.StringAttribute{
				Description: "Specifies URL for pushing logs.",
				Computed:    true,
			},
			"jaeger_traces_url": schema.StringAttribute{
				Computed: true,
			},
			"jaeger_ui_url": schema.StringAttribute{
				Computed: true,
			},
			"otlp_traces_url": schema.StringAttribute{
				Computed: true,
			},
			"zipkin_spans_url": schema.StringAttribute{
				Computed: true,
			},
			"acl": schema.SetAttribute{
				Description: "The access control list for this instance. Each entry is an IP address range that is permitted to access, in CIDR notation.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"alert_config": schema.SingleNestedAttribute{
				Description: "Alert configuration for the instance.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"receivers": schema.ListNestedAttribute{
						Description: "List of alert receivers.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Description: "Name of the receiver.",
									Computed:    true,
								},
								"email_configs": schema.ListNestedAttribute{
									Description: "List of email configurations.",
									Computed:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"auth_identity": schema.StringAttribute{
												Description: "SMTP authentication information. Must be a valid email address",
												Computed:    true,
											},
											"auth_password": schema.StringAttribute{
												Description: "SMTP authentication password.",
												Computed:    true,
												Sensitive:   true,
											},
											"auth_username": schema.StringAttribute{
												Description: "SMTP authentication username.",
												Computed:    true,
											},
											"from": schema.StringAttribute{
												Description: "The sender email address. Must be a valid email address",
												Computed:    true,
											},
											"send_resolved": schema.BoolAttribute{
												Description: "Whether to notify about resolved alerts.",
												Computed:    true,
											},
											"smart_host": schema.StringAttribute{
												Description: "The SMTP host through which emails are sent.",
												Computed:    true,
											},
											"to": schema.StringAttribute{
												Description: "The email address to send notifications to. Must be a valid email address",
												Computed:    true,
											},
										},
									},
								},
								"opsgenie_configs": schema.ListNestedAttribute{
									Description: "List of OpsGenie configurations.",
									Computed:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"api_key": schema.StringAttribute{
												Description: "The API key for OpsGenie.",
												Computed:    true,
											},
											"api_url": schema.StringAttribute{
												Description: "The host to send OpsGenie API requests to. Must be a valid URL",
												Computed:    true,
											},
											"tags": schema.StringAttribute{
												Description: "Comma separated list of tags attached to the notifications.",
												Computed:    true,
											},
											"priority": schema.StringAttribute{
												Description: "Priority of the alert. " + utils.FormatPossibleValues([]string{"P1", "P2", "P3", "P4", "P5"}...),
												Computed:    true,
											},
											"send_resolved": schema.BoolAttribute{
												Description: "Whether to notify about resolved alerts.",
												Computed:    true,
											},
										},
									},
								},
								"webhooks_configs": schema.ListNestedAttribute{
									Description: "List of Webhooks configurations.",
									Computed:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"url": schema.StringAttribute{
												Description: "The endpoint to send HTTP POST requests to. Must be a valid URL",
												Computed:    true,
												Sensitive:   true,
											},
											"ms_teams": schema.BoolAttribute{
												Description: "Microsoft Teams webhooks require special handling, set this to true if the webhook is for Microsoft Teams.",
												Computed:    true,
											},
											"google_chat": schema.BoolAttribute{
												Description: "Google Chat webhooks require special handling, set this to true if the webhook is for Google Chat.",
												Computed:    true,
											},
											"send_resolved": schema.BoolAttribute{
												Description: "Whether to notify about resolved alerts.",
												Computed:    true,
											},
										},
									},
								},
							},
						},
					},
					"route": schema.SingleNestedAttribute{
						Description: "The route for the alert.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"group_by": schema.ListAttribute{
								Description: "The labels by which incoming alerts are grouped together. For example, multiple alerts coming in for cluster=A and alertname=LatencyHigh would be batched into a single group. To aggregate by all possible labels use the special value '...' as the sole label name, for example: group_by: ['...']. This effectively disables aggregation entirely, passing through all alerts as-is. This is unlikely to be what you want, unless you have a very low alert volume or your upstream notification system performs its own grouping.",
								Computed:    true,
								ElementType: types.StringType,
							},
							"group_interval": schema.StringAttribute{
								Description: "How long to wait before sending a notification about new alerts that are added to a group of alerts for which an initial notification has already been sent. (Usually ~5m or more.)",
								Computed:    true,
							},
							"group_wait": schema.StringAttribute{
								Description: "How long to initially wait to send a notification for a group of alerts. Allows to wait for an inhibiting alert to arrive or collect more initial alerts for the same group. (Usually ~0s to few minutes.) .",
								Computed:    true,
							},
							"receiver": schema.StringAttribute{
								Description: "The name of the receiver to route the alerts to.",
								Computed:    true,
							},
							"repeat_interval": schema.StringAttribute{
								Description: "How long to wait before sending a notification again if it has already been sent successfully for an alert. (Usually ~3h or more).",
								Computed:    true,
							},
							"routes": getDatasourceRouteNestedObject(),
						},
					},
					"global": schema.SingleNestedAttribute{
						Description: "Global configuration for the alerts.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"opsgenie_api_key": schema.StringAttribute{
								Description: "The API key for OpsGenie.",
								Computed:    true,
								Sensitive:   true,
							},
							"opsgenie_api_url": schema.StringAttribute{
								Description: "The host to send OpsGenie API requests to. Must be a valid URL",
								Computed:    true,
							},
							"resolve_timeout": schema.StringAttribute{
								Description: "The default value used by alertmanager if the alert does not include EndsAt. After this time passes, it can declare the alert as resolved if it has not been updated. This has no impact on alerts from Prometheus, as they always include EndsAt.",
								Computed:    true,
							},
							"smtp_auth_identity": schema.StringAttribute{
								Description: "SMTP authentication information. Must be a valid email address",
								Computed:    true,
							},
							"smtp_auth_password": schema.StringAttribute{
								Description: "SMTP Auth using LOGIN and PLAIN.",
								Computed:    true,
								Sensitive:   true,
							},
							"smtp_auth_username": schema.StringAttribute{
								Description: "SMTP Auth using CRAM-MD5, LOGIN and PLAIN. If empty, Alertmanager doesn't authenticate to the SMTP server.",
								Computed:    true,
							},
							"smtp_from": schema.StringAttribute{
								Description: "The default SMTP From header field. Must be a valid email address",
								Computed:    true,
							},
							"smtp_smart_host": schema.StringAttribute{
								Description: "The default SMTP smarthost used for sending emails, including port number. Port number usually is 25, or 587 for SMTP over TLS (sometimes referred to as STARTTLS).",
								Computed:    true,
							},
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	instanceResp, err := d.client.GetInstance(ctx, instanceId, projectId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading instance",
			fmt.Sprintf("Instance with ID %q does not exist in project %q.", instanceId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	if instanceResp != nil && instanceResp.Status != nil && *instanceResp.Status == observability.GETINSTANCERESPONSESTATUS_DELETE_SUCCEEDED {
		resp.State.RemoveResource(ctx)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", "Instance was deleted successfully")
		return
	}

	aclListResp, err := d.client.ListACL(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API to list ACL data: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, instanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to instance populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan, err := loadPlanId(ctx, *d.client, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Loading service plan: %v", err))
		return
	}

	err = mapACLField(aclListResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API response for the ACL: %v", err))
		return
	}

	// Set state to fully populated data
	diags = setACL(ctx, &resp.State, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// There are some plans which does not offer storage e.g. like Observability-Metrics-Endpoint-100k-EU01
	if plan.GetLogsStorage() != 0 && plan.GetTracesStorage() != 0 {
		metricsRetentionResp, err := d.client.GetMetricsStorageRetention(ctx, instanceId, projectId).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API to get metrics retention: %v", err))
			return
		}
		// Map response body to schema
		err = mapMetricsRetentionField(metricsRetentionResp, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API response for the metrics retention: %v", err))
			return
		}
		// Set state to fully populated data
		diags := setMetricsRetentions(ctx, &resp.State, &model)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Handle Logs Retentions
		logsRetentionResp, err := d.client.GetLogsConfigs(ctx, instanceId, projectId).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API to get logs retention: %v", err))
			return
		}

		err = mapLogsRetentionField(logsRetentionResp, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API response for the logs retention: %v", err))
			return
		}

		diags = setLogsRetentions(ctx, &resp.State, &model)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Handle Traces Retentions
		tracesRetentionResp, err := d.client.GetTracesConfigs(ctx, instanceId, projectId).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API to get traces retention: %v", err))
			return
		}

		err = mapTracesRetentionField(tracesRetentionResp, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API response for the traces retention: %v", err))
			return
		}

		diags = setTracesRetentions(ctx, &resp.State, &model)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// There are plans where no alert matchers and receivers are present e.g. like Observability-Metrics-Endpoint-100k-EU01
	if plan.GetAlertMatchers() != 0 && plan.GetAlertReceivers() != 0 {
		alertConfigResp, err := d.client.GetAlertConfigs(ctx, instanceId, projectId).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API to get alert config: %v", err))
			return
		}
		// Map response body to schema
		err = mapAlertConfigField(ctx, alertConfigResp, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response for the alert config: %v", err))
			return
		}

		// Set state to fully populated data
		diags = setAlertConfig(ctx, &resp.State, &model)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Info(ctx, "Observability instance read")
}
