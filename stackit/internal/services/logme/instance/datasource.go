package logme

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	logmeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logme/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/services/logme"
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
	client *logme.APIClient
}

// Metadata returns the data source type name.
func (r *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logme_instance"
}

// Configure adds the provider configured client to the data source.
func (r *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := logmeUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "LogMe instance client configured")
}

// Schema defines the schema for the data source.
func (r *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "LogMe instance data source schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal data source. identifier. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the LogMe instance.",
		"project_id":  "STACKIT Project ID to which the instance is associated.",
		"name":        "Instance name.",
		"version":     "The service version.",
		"plan_name":   "The selected plan name.",
		"plan_id":     "The selected plan ID.",
	}

	parametersDescriptions := map[string]string{
		"sgw_acl":                "Comma separated list of IP networks in CIDR notation which are allowed to access this instance.",
		"enable_monitoring":      "Enable monitoring.",
		"graphite":               "If set, monitoring with Graphite will be enabled. Expects the host and port where the Graphite metrics should be sent to (host:port).",
		"max_disk_threshold":     "The maximum disk threshold in MB. If the disk usage exceeds this threshold, the instance will be stopped.",
		"metrics_frequency":      "The frequency in seconds at which metrics are emitted (in seconds).",
		"metrics_prefix":         "The prefix for the metrics. Could be useful when using Graphite monitoring to prefix the metrics with a certain value, like an API key.",
		"monitoring_instance_id": "The ID of the STACKIT monitoring instance.",
		"java_heapspace":         "The amount of memory (in MB) allocated as heap by the JVM for OpenSearch.",
		"java_maxmetaspace":      "The amount of memory (in MB) used by the JVM to store metadata for OpenSearch.",
		"ism_deletion_after":     "Combination of an integer and a timerange when an index will be considered \"old\" and can be deleted. Possible values for the timerange are `s`, `m`, `h` and `d`.",
		"ism_job_interval":       "Jitter of the execution time.",
		"syslog":                 "List of syslog servers to send logs to.",
		"opensearch-tls-ciphers": "List of ciphers to use for TLS.",
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
					"enable_monitoring": schema.BoolAttribute{
						Description: parametersDescriptions["enable_monitoring"],
						Computed:    true,
					},
					"fluentd_tcp": schema.Int64Attribute{
						Description: parametersDescriptions["fluentd_tcp"],
						Computed:    true,
					},
					"fluentd_tls": schema.Int64Attribute{
						Description: parametersDescriptions["fluentd_tls"],
						Computed:    true,
					},
					"fluentd_tls_ciphers": schema.StringAttribute{
						Description: parametersDescriptions["fluentd_tls_ciphers"],
						Computed:    true,
					},
					"fluentd_tls_max_version": schema.StringAttribute{
						Description: parametersDescriptions["fluentd_tls_max_version"],
						Computed:    true,
					},
					"fluentd_tls_min_version": schema.StringAttribute{
						Description: parametersDescriptions["fluentd_tls_min_version"],
						Computed:    true,
					},
					"fluentd_tls_version": schema.StringAttribute{
						Description: parametersDescriptions["fluentd_tls_version"],
						Computed:    true,
					},
					"fluentd_udp": schema.Int64Attribute{
						Description: parametersDescriptions["fluentd_udp"],
						Computed:    true,
					},
					"graphite": schema.StringAttribute{
						Description: parametersDescriptions["graphite"],
						Computed:    true,
					},
					"ism_deletion_after": schema.StringAttribute{
						Description: parametersDescriptions["ism_deletion_after"],
						Computed:    true,
					},
					"ism_jitter": schema.Float64Attribute{
						Description: parametersDescriptions["ism_jitter"],
						Computed:    true,
					},
					"ism_job_interval": schema.Int64Attribute{
						Description: parametersDescriptions["ism_job_interval"],
						Computed:    true,
					},
					"java_heapspace": schema.Int64Attribute{
						Description: parametersDescriptions["java_heapspace"],
						Computed:    true,
					},
					"java_maxmetaspace": schema.Int64Attribute{
						Description: parametersDescriptions["java_maxmetaspace"],
						Computed:    true,
					},
					"max_disk_threshold": schema.Int64Attribute{
						Description: parametersDescriptions["max_disk_threshold"],
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
					"monitoring_instance_id": schema.StringAttribute{
						Description: parametersDescriptions["monitoring_instance_id"],
						Computed:    true,
					},
					"opensearch_tls_ciphers": schema.ListAttribute{
						Description: parametersDescriptions["opensearch_tls_ciphers"],
						ElementType: types.StringType,
						Computed:    true,
					},
					"opensearch_tls_protocols": schema.ListAttribute{
						Description: parametersDescriptions["opensearch_tls_protocols"],
						ElementType: types.StringType,
						Computed:    true,
					},
					"syslog": schema.ListAttribute{
						Description: parametersDescriptions["syslog"],
						ElementType: types.StringType,
						Computed:    true,
					},
				},
				Computed: true,
			},
			"cf_guid": schema.StringAttribute{
				Computed: true,
			},
			"cf_space_guid": schema.StringAttribute{
				Computed: true,
			},
			"dashboard_url": schema.StringAttribute{
				Computed: true,
			},
			"image_url": schema.StringAttribute{
				Computed: true,
			},
			"cf_organization_guid": schema.StringAttribute{
				Computed: true,
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

	ctx = core.InitProviderContext(ctx)

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
				http.StatusGone:      fmt.Sprintf("Instance %q is gone.", instanceId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

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
	tflog.Info(ctx, "LogMe instance read")
}
