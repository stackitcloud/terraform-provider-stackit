package redis

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/redis"
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
	client *redis.APIClient
}

// Metadata returns the data source type name.
func (r *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_instance"
}

// Configure adds the provider configured client to the data source.
func (r *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *redis.APIClient
	var err error
	if providerData.RedisCustomEndpoint != "" {
		apiClient, err = redis.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.RedisCustomEndpoint),
		)
	} else {
		apiClient, err = redis.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.GetRegion()),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Redis instance client configured")
}

// Schema defines the schema for the data source.
func (r *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "Redis instance data source schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal data source. identifier. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the Redis instance.",
		"project_id":  "STACKIT Project ID to which the instance is associated.",
		"name":        "Instance name.",
		"version":     "The service version.",
		"plan_name":   "The selected plan name.",
		"plan_id":     "The selected plan ID.",
	}

	parametersDescriptions := map[string]string{
		"sgw_acl":                 "Comma separated list of IP networks in CIDR notation which are allowed to access this instance.",
		"down_after_milliseconds": "The number of milliseconds after which the instance is considered down.",
		"enable_monitoring":       "Enable monitoring.",
		"failover_timeout":        "The failover timeout in milliseconds.",
		"graphite":                "Graphite server URL (host and port). If set, monitoring with Graphite will be enabled.",
		"lazyfree_lazy_eviction":  "The lazy eviction enablement (yes or no).",
		"lazyfree_lazy_expire":    "The lazy expire enablement (yes or no).",
		"lua_time_limit":          "The Lua time limit.",
		"max_disk_threshold":      "The maximum disk threshold in MB. If the disk usage exceeds this threshold, the instance will be stopped.",
		"maxclients":              "The maximum number of clients.",
		"maxmemory_policy":        "The policy to handle the maximum memory (volatile-lru, noeviction, etc).",
		"maxmemory_samples":       "The maximum memory samples.",
		"metrics_frequency":       "The frequency in seconds at which metrics are emitted.",
		"metrics_prefix":          "The prefix for the metrics. Could be useful when using Graphite monitoring to prefix the metrics with a certain value, like an API key",
		"min_replicas_max_lag":    "The minimum replicas maximum lag.",
		"monitoring_instance_id":  "The ID of the STACKIT monitoring instance.",
		"notify_keyspace_events":  "The notify keyspace events.",
		"snapshot":                "The snapshot configuration.",
		"syslog":                  "List of syslog servers to send logs to.",
		"tls_ciphers":             "List of TLS ciphers to use.",
		"tls_ciphersuites":        "TLS cipher suites to use.",
		"tls_protocols":           "TLS protocol to use.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Computed:    true,
			},
			"plan_name": schema.StringAttribute{
				Description: descriptions["plan_name"],
				Computed:    true,
			},
			"plan_id": schema.StringAttribute{
				Description: descriptions["plan_id"],
				Computed:    true,
			},
			"parameters": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"sgw_acl": schema.StringAttribute{
						Description: parametersDescriptions["sgw_acl"],
						Computed:    true,
					},
					"down_after_milliseconds": schema.Int64Attribute{
						Description: parametersDescriptions["down_after_milliseconds"],
						Computed:    true,
					},
					"enable_monitoring": schema.BoolAttribute{
						Description: parametersDescriptions["enable_monitoring"],
						Computed:    true,
					},
					"failover_timeout": schema.Int64Attribute{
						Description: parametersDescriptions["failover_timeout"],
						Computed:    true,
					},
					"graphite": schema.StringAttribute{
						Description: parametersDescriptions["graphite"],
						Computed:    true,
					},
					"lazyfree_lazy_eviction": schema.StringAttribute{
						Description: parametersDescriptions["lazyfree_lazy_eviction"],
						Computed:    true,
					},
					"lazyfree_lazy_expire": schema.StringAttribute{
						Description: parametersDescriptions["lazyfree_lazy_expire"],
						Computed:    true,
					},
					"lua_time_limit": schema.Int64Attribute{
						Description: parametersDescriptions["lua_time_limit"],
						Computed:    true,
					},
					"max_disk_threshold": schema.Int64Attribute{
						Description: parametersDescriptions["max_disk_threshold"],
						Computed:    true,
					},
					"maxclients": schema.Int64Attribute{
						Description: parametersDescriptions["maxclients"],
						Computed:    true,
					},
					"maxmemory_policy": schema.StringAttribute{
						Description: parametersDescriptions["maxmemory_policy"],
						Computed:    true,
					},
					"maxmemory_samples": schema.Int64Attribute{
						Description: parametersDescriptions["maxmemory_samples"],
						Computed:    true,
					},
					"metrics_frequency": schema.Int64Attribute{
						Description: parametersDescriptions["metrics_frequency"],
						Computed:    true,
					},
					"metrics_prefix": schema.StringAttribute{
						Description: parametersDescriptions["metrics_prefix"],
						Computed:    true,
					},
					"min_replicas_max_lag": schema.Int64Attribute{
						Description: parametersDescriptions["min_replicas_max_lag"],
						Computed:    true,
					},
					"monitoring_instance_id": schema.StringAttribute{
						Description: parametersDescriptions["monitoring_instance_id"],
						Computed:    true,
					},
					"notify_keyspace_events": schema.StringAttribute{
						Description: parametersDescriptions["notify_keyspace_events"],
						Computed:    true,
					},
					"snapshot": schema.StringAttribute{
						Description: parametersDescriptions["snapshot"],
						Computed:    true,
					},
					"syslog": schema.ListAttribute{
						ElementType: types.StringType,
						Description: parametersDescriptions["syslog"],
						Computed:    true,
					},
					"tls_ciphers": schema.ListAttribute{
						ElementType: types.StringType,
						Description: parametersDescriptions["tls_ciphers"],
						Computed:    true,
					},
					"tls_ciphersuites": schema.StringAttribute{
						Description: parametersDescriptions["tls_ciphersuites"],
						Computed:    true,
					},
					"tls_protocols": schema.StringAttribute{
						Description: parametersDescriptions["tls_protocols"],
						Computed:    true,
					},
				},
				Computed: true,
			},
			"cf_guid": schema.StringAttribute{
				Description: descriptions["cf_guid"],
				Computed:    true,
			},
			"cf_space_guid": schema.StringAttribute{
				Description: descriptions["cf_space_guid"],
				Computed:    true,
			},
			"dashboard_url": schema.StringAttribute{
				Description: descriptions["dashboard_url"],
				Computed:    true,
			},
			"image_url": schema.StringAttribute{
				Description: descriptions["image_url"],
				Computed:    true,
			},
			"cf_organization_guid": schema.StringAttribute{
				Description: descriptions["cf_organization_guid"],
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	instanceResp, err := r.client.GetInstance(ctx, projectId, instanceId).Execute()
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

	err = mapFields(instanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Compute and store values not present in the API response
	err = loadPlanNameAndVersion(ctx, r.client, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Loading service plan details: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Redis instance read")
}
