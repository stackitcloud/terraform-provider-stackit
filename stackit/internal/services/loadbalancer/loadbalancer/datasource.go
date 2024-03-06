package loadbalancer

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &loadBalancerDataSource{}
)

// NewLoadBalancerDataSource is a helper function to simplify the provider implementation.
func NewLoadBalancerDataSource() datasource.DataSource {
	return &loadBalancerDataSource{}
}

// loadBalancerDataSource is the data source implementation.
type loadBalancerDataSource struct {
	client *loadbalancer.APIClient
}

// Metadata returns the data source type name.
func (r *loadBalancerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancer"
}

// Configure adds the provider configured client to the data source.
func (r *loadBalancerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *loadbalancer.APIClient
	var err error
	if providerData.LoadBalancerCustomEndpoint != "" {
		apiClient, err = loadbalancer.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.LoadBalancerCustomEndpoint),
		)
	} else {
		apiClient, err = loadbalancer.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Load balancer client configured")
}

// Schema defines the schema for the data source.
func (r *loadBalancerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                        "Load Balancer data source schema. Must have a `region` specified in the provider configuration.",
		"id":                          "Terraform's internal resource ID. It is structured as \"`project_id`\",\"`name`\".",
		"project_id":                  "STACKIT project ID to which the Load Balancer is associated.",
		"external_address":            "External Load Balancer IP address where this Load Balancer is exposed.",
		"listeners":                   "List of all listeners which will accept traffic. Limited to 20.",
		"port":                        "Port number where we listen for traffic.",
		"protocol":                    "Protocol is the highest network protocol we understand to load balance.",
		"target_pool":                 "Reference target pool by target pool name.",
		"name":                        "Load balancer name.",
		"networks":                    "List of networks that listeners and targets reside in.",
		"network_id":                  "Openstack network ID.",
		"role":                        "The role defines how the load balancer is using the network.",
		"options":                     "Defines any optional functionality you want to have enabled on your load balancer.",
		"acl":                         "Load Balancer is accessible only from an IP address in this range.",
		"private_network_only":        "If true, Load Balancer is accessible only via a private network IP address.",
		"session_persistence":         "Here you can setup various session persistence options, so far only \"`use_source_ip_address`\" is supported.",
		"use_source_ip_address":       "If true then all connections from one source IP address are redirected to the same target. This setting changes the load balancing algorithm to Maglev.",
		"server_name_indicators":      "A list of domain names to match in order to pass TLS traffic to the target pool in the current listener",
		"server_name_indicators.name": "A domain name to match in order to pass TLS traffic to the target pool in the current listener",
		"private_address":             "Transient private Load Balancer IP address. It can change any time.",
		"target_pools":                "List of all target pools which will be used in the Load Balancer. Limited to 20.",
		"healthy_threshold":           "Healthy threshold of the health checking.",
		"interval":                    "Interval duration of health checking in seconds.",
		"interval_jitter":             "Interval duration threshold of the health checking in seconds.",
		"timeout":                     "Active health checking timeout duration in seconds.",
		"unhealthy_threshold":         "Unhealthy threshold of the health checking.",
		"target_pools.name":           "Target pool name.",
		"target_port":                 "Identical port number where each target listens for traffic.",
		"targets":                     "List of all targets which will be used in the pool. Limited to 250.",
		"targets.display_name":        "Target display name",
		"ip":                          "Target IP",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"external_address": schema.StringAttribute{
				Description: descriptions["external_address"],
				Computed:    true,
			},
			"listeners": schema.ListNestedAttribute{
				Description: descriptions["listeners"],
				Computed:    true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 20),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"display_name": schema.StringAttribute{
							Description: descriptions["listeners.display_name"],
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
							Computed:    true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					validate.NoSeparator(),
				},
			},
			"networks": schema.ListNestedAttribute{
				Description: descriptions["networks"],
				Computed:    true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{
							Description: descriptions["network_id"],
							Computed:    true,
							Validators: []validator.String{
								validate.UUID(),
								validate.NoSeparator(),
							},
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
					"acl": schema.SetAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
						Computed:    true,
						Validators: []validator.Set{
							setvalidator.ValueStringsAre(
								validate.CIDR(),
							),
						},
					},
					"private_network_only": schema.BoolAttribute{
						Description: descriptions["private_network_only"],
						Computed:    true,
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
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 20),
				},
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
							Computed:    true,
							Validators: []validator.List{
								listvalidator.SizeBetween(1, 250),
							},
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
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *loadBalancerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)

	lbResp, err := r.client.GetLoadBalancer(ctx, projectId, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading load balancer", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(lbResp, &model)
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
