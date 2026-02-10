package alb

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	albUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/alb/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	albSdk "github.com/stackitcloud/stackit-sdk-go/services/alb"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &albDataSource{}
)

// NewApplicationLoadBalancerDataSource is a helper function to simplify the provider implementation.
func NewApplicationLoadBalancerDataSource() datasource.DataSource {
	return &albDataSource{}
}

// albDataSource is the data source implementation.
type albDataSource struct {
	client       *albSdk.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *albDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_load_balancer"
}

// Configure adds the provider configured client to the data source.
func (r *albDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (r *albDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	protocolOptions := albUtils.ToStringList(albSdk.AllowedListenerProtocolEnumValues)
	roleOptions := albUtils.ToStringList(albSdk.AllowedNetworkRoleEnumValues)

	descriptions := map[string]string{
		"main":       "Application Load Balancer resource schema.",
		"id":         "Terraform's internal resource ID. It is structured as `project_id`,`region`,`name`.",
		"project_id": "STACKIT project ID to which the Application Load Balancer is associated.",
		"region":     "The resource region. If not defined, the provider region is used.",
		"disable_target_security_group_assignment": "Disable target security group assignemt to allow targets outside of the given network. Connectivity to targets need to be ensured by the customer, including routing and Security Groups (targetSecurityGroup can be assigned). Not changeable after creation.",
		"errors":                            "Reports all errors a Application Load Balancer has.",
		"errors.type":                       "Enum: \"TYPE_UNSPECIFIED\" \"TYPE_INTERNAL\" \"TYPE_QUOTA_SECGROUP_EXCEEDED\" \"TYPE_QUOTA_SECGROUPRULE_EXCEEDED\" \"TYPE_PORT_NOT_CONFIGURED\" \"TYPE_FIP_NOT_CONFIGURED\" \"TYPE_TARGET_NOT_ACTIVE\" \"TYPE_METRICS_MISCONFIGURED\" \"TYPE_LOGS_MISCONFIGURED\"\nThe error type specifies which part of the Application Load Balancer encountered the error. I.e. the API will not check if a provided public IP is actually available in the project. Instead the Application Load Balancer with try to use the provided IP and if not available reports TYPE_FIP_NOT_CONFIGURED error.",
		"errors.description":                "The error description contains additional helpful user information to fix the error state of the Application Load Balancer. For example the IP 45.135.247.139 does not exist in the project, then the description will report: Floating IP \"45.135.247.139\" could not be found.",
		"external_address":                  "The external IP address where this Application Load Balancer is exposed. Not changeable after creation.",
		"labels":                            "Labels represent user-defined metadata as key-value pairs. Label count cannot exceed 64 per ALB.",
		"listeners":                         "List of all listeners which will accept traffic. Limited to 20.",
		"listeners.name":                    "Unique name for the listener",
		"http":                              "Configuration for handling HTTP traffic on this listener.",
		"hosts":                             "Defines routing rules grouped by hostname.",
		"host":                              "Hostname to match. Supports wildcards (e.g. *.example.com).",
		"rules":                             "Routing rules under the specified host, matched by path prefix.",
		"cookie_persistence":                "Routing persistence via cookies.",
		"cookie_persistence.name":           "The name of the cookie to use.",
		"ttl":                               "TTL specifies the time-to-live for the cookie. The default value is 0s, and it acts as a session cookie, expiring when the client session ends.",
		"headers":                           "Headers for the rule.",
		"headers.exact_match":               "Exact match for the header value.",
		"headers.name":                      "Header name.",
		"path":                              "Routing via path.",
		"path.exact_match":                  "Exact path match. Only a request path exactly equal to the value will match, e.g. '/foo' matches only '/foo', not '/foo/bar' or '/foobar'.",
		"path.prefix":                       "Prefix path match. Only matches on full segment boundaries, e.g. '/foo' matches '/foo' and '/foo/bar' but NOT '/foobar'.",
		"query_parameters":                  "Query parameters for the rule.",
		"query_parameters.exact_match":      "Exact match for the query parameters value.",
		"query_parameters.name":             "Query parameter name.",
		"target_pool":                       "Reference target pool by target pool name.",
		"web_socket":                        "If enabled, when client sends an HTTP request with and Upgrade header, indicating the desire to establish a Websocket connection, if backend server supports WebSocket, it responds with HTTP 101 status code, switching protocols from HTTP to WebSocket. Hence the client and the server can exchange data in real-time using one long-lived TCP connection.",
		"https":                             "Configuration for handling HTTPS traffic on this listener.",
		"certificate_config":                "TLS termination certificate configuration.",
		"certificate_ids":                   "Certificate IDs for TLS termination.",
		"port":                              "Port number on which the listener receives incoming traffic.",
		"protocol":                          "Protocol is the highest network protocol we understand to load balance. " + utils.FormatPossibleValues(protocolOptions...),
		"waf_config_name":                   "Enable Web Application Firewall (WAF), referenced by name. See \"Application Load Balancer - Web Application Firewall API\" for more information.",
		"load_balancer_security_group":      "Security Group permitting network traffic from the LoadBalancer to the targets. Useful when disableTargetSecurityGroupAssignment=true to manually assign target security groups to targets.",
		"load_balancer_security_group.id":   "ID of the security Group",
		"load_balancer_security_group.name": "Name of the security Group",
		"name":                              "Application Load balancer name.",
		"networks":                          "List of networks that listeners and targets reside in.",
		"network_id":                        "STACKIT network ID the Application Load Balancer and/or targets are in.",
		"role":                              "The role defines how the Application Load Balancer is using the network. " + utils.FormatPossibleValues(roleOptions...),
		"options":                           "Defines any optional functionality you want to have enabled on your Application Load Balancer.",
		"access_control":                    "Use this option to limit the IP ranges that can use the Application Load Balancer.",
		"allowed_source_ranges":             "Application Load Balancer is accessible only from an IP address in this range.", "ephemeral_address": "This option automates the handling of the external IP address for an Application Load Balancer. If set to true a new IP address will be automatically created. It will also be automatically deleted when the Load Balancer is deleted.",
		"observability":                          "We offer Load Balancer observability via STACKIT Observability or external solutions.",
		"observability_logs":                     "Observability logs configuration.",
		"observability_logs_credentials_ref":     "Credentials reference for logging. This reference is created via the observability create endpoint and the credential needs to contain the basic auth username and password for the logging solution the push URL points to. Then this enables monitoring via remote write for the Application Load Balancer.",
		"observability_logs_push_url":            "The Observability(Logs)/Loki remote write Push URL you want the logs to be shipped to.",
		"observability_metrics":                  "Observability metrics configuration.",
		"observability_metrics_credentials_ref":  "Credentials reference for metrics. This reference is created via the observability create endpoint and the credential needs to contain the basic auth username and password for the metrics solution the push URL points to. Then this enables monitoring via remote write for the Application Load Balancer.",
		"observability_metrics_push_url":         "The Observability(Metrics)/Prometheus remote write push URL you want the metrics to be shipped to.",
		"plan_id":                                "Service Plan configures the size of the Application Load Balancer.",
		"private_network_only":                   "Application Load Balancer is accessible only via a private network ip address. Not changeable after creation.",
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
		MarkdownDescription: `Application Load Balancer data source schema. Must have a region specified in the provider configuration.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Computed:    true,
			},
			"disable_target_security_group_assignment": schema.BoolAttribute{
				Description: descriptions["disable_target_security_group_assignment"],
				Computed:    true,
			},
			"errors": schema.SetNestedAttribute{
				Description: descriptions["errors"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: descriptions["errors.type"],
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: descriptions["errors.description"],
							Computed:    true,
						},
					},
				},
			},
			"external_address": schema.StringAttribute{
				Description: descriptions["external_address"],
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				Computed:    true,
				ElementType: types.StringType,
			},
			"plan_id": schema.StringAttribute{
				Description: descriptions["plan_id"],
				Computed:    true,
			},
			"listeners": schema.ListNestedAttribute{
				Description: descriptions["listeners"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: descriptions["listeners.name"],
							Computed:    true,
						},
						"port": schema.Int64Attribute{
							Description: descriptions["port"],
							Computed:    true,
						},
						"protocol": schema.StringAttribute{
							Description: descriptions["protocol"],
							Computed:    true,
						},
						"waf_config_name": schema.StringAttribute{
							Description: descriptions["waf_config_name"],
							Computed:    true,
						},
						"http": schema.SingleNestedAttribute{
							Description: "Configuration for HTTP traffic.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"hosts": schema.ListNestedAttribute{
									Description: descriptions["hosts"],
									Computed:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"host": schema.StringAttribute{
												Description: descriptions["host"],
												Computed:    true,
											},
											"rules": schema.ListNestedAttribute{
												Description: descriptions["rules"],
												Computed:    true,
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"target_pool": schema.StringAttribute{
															Description: descriptions["target_pool"],
															Computed:    true,
														},
														"web_socket": schema.BoolAttribute{
															Description: descriptions["web_socket"],
															Computed:    true,
														},
														"path": schema.SingleNestedAttribute{
															Description: descriptions["path"],
															Computed:    true,
															Attributes: map[string]schema.Attribute{
																"exact_match": schema.StringAttribute{
																	Description: descriptions["path.exact_match"],
																	Computed:    true,
																},
																"prefix": schema.StringAttribute{
																	Description: descriptions["path.prefix"],
																	Computed:    true,
																},
															},
														},
														"headers": schema.SetNestedAttribute{
															Description: descriptions["headers"],
															Computed:    true,
															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: descriptions["headers.name"],
																		Computed:    true,
																	},
																	"exact_match": schema.StringAttribute{
																		Description: descriptions["headers.exact_match"],
																		Computed:    true,
																	},
																},
															},
														},
														"query_parameters": schema.SetNestedAttribute{
															Description: descriptions["query_parameters"],
															Computed:    true,
															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: descriptions["query_parameters.name"],
																		Computed:    true,
																	},
																	"exact_match": schema.StringAttribute{
																		Description: descriptions["query_parameters.exact_match"],
																		Computed:    true,
																	},
																},
															},
														},
														"cookie_persistence": schema.SingleNestedAttribute{
															Description: descriptions["cookie_persistence"],
															Computed:    true,
															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	Description: descriptions["cookie_persistence.name"],
																	Computed:    true,
																},
																"ttl": schema.StringAttribute{
																	Description: descriptions["ttl"],
																	Computed:    true,
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
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"certificate_config": schema.SingleNestedAttribute{
									Description: descriptions["certificate_config"],
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"certificate_ids": schema.SetAttribute{
											Description: descriptions["certificate_ids"],
											Computed:    true,
											ElementType: types.StringType,
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
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: descriptions["load_balancer_security_group.name"],
						Computed:    true,
					},
					"id": schema.StringAttribute{
						Description: descriptions["load_balancer_security_group.id"],
						Computed:    true,
					},
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
			},
			"networks": schema.SetNestedAttribute{
				Description: descriptions["networks"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{
							Description: descriptions["network_id"],
							Computed:    true,
						},
						"role": schema.StringAttribute{
							Description: descriptions["role"],
							Computed:    true,
						},
					},
				},
			},
			"options": schema.SingleNestedAttribute{
				Description: descriptions["options"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"access_control": schema.SingleNestedAttribute{
						Description: descriptions["access_control"],
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"allowed_source_ranges": schema.SetAttribute{
								Description: descriptions["allowed_source_ranges"],
								ElementType: types.StringType,
								Computed:    true,
							},
						},
					},
					"ephemeral_address": schema.BoolAttribute{
						Description: descriptions["ephemeral_address"],
						Computed:    true,
					},
					"private_network_only": schema.BoolAttribute{
						Description: descriptions["private_network_only"],
						Computed:    true,
					},
					"observability": schema.SingleNestedAttribute{
						Description: descriptions["observability"],
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"logs": schema.SingleNestedAttribute{
								Description: descriptions["observability_logs"],
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"credentials_ref": schema.StringAttribute{
										Description: descriptions["observability_logs_credentials_ref"],
										Computed:    true,
									},
									"push_url": schema.StringAttribute{
										Description: descriptions["observability_logs_credentials_ref"],
										Computed:    true,
									},
								},
							},
							"metrics": schema.SingleNestedAttribute{
								Description: descriptions["observability_metrics"],
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"credentials_ref": schema.StringAttribute{
										Description: descriptions["observability_metrics_credentials_ref"],
										Computed:    true,
									},
									"push_url": schema.StringAttribute{
										Description: descriptions["observability_metrics_credentials_ref"],
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
			},
			"target_pools": schema.ListNestedAttribute{
				Description: descriptions["target_pools"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"active_health_check": schema.SingleNestedAttribute{
							Description: descriptions["active_health_check"],
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"healthy_threshold": schema.Int64Attribute{
									Description: descriptions["healthy_threshold"],
									Computed:    true,
								},
								"interval": schema.StringAttribute{
									Description: descriptions["interval"],
									Computed:    true,
								},
								"interval_jitter": schema.StringAttribute{
									Description: descriptions["interval_jitter"],
									Computed:    true,
								},
								"timeout": schema.StringAttribute{
									Description: descriptions["timeout"],
									Computed:    true,
								},
								"unhealthy_threshold": schema.Int64Attribute{
									Description: descriptions["unhealthy_threshold"],
									Computed:    true,
								},
								"http_health_checks": schema.SingleNestedAttribute{
									Description: descriptions["http_health_checks"],
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"path": schema.StringAttribute{
											Description: descriptions["http_health_checks.path"],
											Computed:    true,
										},
										"ok_status": schema.SetAttribute{
											Description: descriptions["http_health_checks.ok_status"],
											Computed:    true,
											ElementType: types.StringType,
										},
									},
								},
							},
						},
						"name": schema.StringAttribute{
							Description: descriptions["target_pools.name"],
							Computed:    true,
						},
						"target_port": schema.Int64Attribute{
							Description: descriptions["target_port"],
							Computed:    true,
						},
						"targets": schema.SetNestedAttribute{
							Description: descriptions["targets"],
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"display_name": schema.StringAttribute{
										Description: descriptions["targets.display_name"],
										Computed:    true,
									},
									"ip": schema.StringAttribute{
										Description: descriptions["ip"],
										Computed:    true,
									},
								},
							},
						},
						"tls_config": schema.SingleNestedAttribute{
							Description: descriptions["tls_config"],
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"enabled": schema.BoolAttribute{
									Description: descriptions["tls_config.enabled"],
									Computed:    true,
								},
								"skip_certificate_validation": schema.BoolAttribute{
									Description: descriptions["tls_config.skip_certificate_validation"],
									Computed:    true,
								},
								"custom_ca": schema.StringAttribute{
									Description: descriptions["tls_config.custom_ca"],
									Computed:    true,
								},
							},
						},
					},
				},
			},
			"target_security_group": schema.SingleNestedAttribute{
				Description: descriptions["target_security_group"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: descriptions["target_security_group.name"],
						Computed:    true,
					},
					"id": schema.StringAttribute{
						Description: descriptions["target_security_group.id"],
						Computed:    true,
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

// Read refreshes the Terraform state with the latest data.
func (r *albDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)
	ctx = tflog.SetField(ctx, "region", region)

	albResp, err := r.client.GetLoadBalancer(ctx, projectId, region, name).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading application load balancer",
			fmt.Sprintf("Application Load Balancer with name %q does not exist in project %q.", name, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, albResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading application load balancer", fmt.Sprintf("Processing API payload: %v", err))
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
