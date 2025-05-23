package rabbitmq

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	rabbitmqUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/rabbitmq/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq"
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
	client *rabbitmq.APIClient
}

// Metadata returns the data source type name.
func (r *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rabbitmq_instance"
}

// Configure adds the provider configured client to the data source.
func (r *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := rabbitmqUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "RabbitMQ instance client configured")
}

// Schema defines the schema for the data source.
func (r *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "RabbitMQ instance data source schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal data source. identifier. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the RabbitMQ instance.",
		"project_id":  "STACKIT Project ID to which the instance is associated.",
		"name":        "Instance name.",
		"version":     "The service version.",
		"plan_name":   "The selected plan name.",
		"plan_id":     "The selected plan ID.",
	}

	parametersDescriptions := map[string]string{
		"sgw_acl":                "Comma separated list of IP networks in CIDR notation which are allowed to access this instance.",
		"consumer_timeout":       "The timeout in milliseconds for the consumer.",
		"enable_monitoring":      "Enable monitoring.",
		"graphite":               "Graphite server URL (host and port). If set, monitoring with Graphite will be enabled.",
		"max_disk_threshold":     "The maximum disk threshold in MB. If the disk usage exceeds this threshold, the instance will be stopped.",
		"metrics_frequency":      "The frequency in seconds at which metrics are emitted.",
		"metrics_prefix":         "The prefix for the metrics. Could be useful when using Graphite monitoring to prefix the metrics with a certain value, like an API key",
		"monitoring_instance_id": "The ID of the STACKIT monitoring instance.",
		"plugins":                "List of plugins to install. Must be a supported plugin name.",
		"roles":                  "List of roles to assign to the instance.",
		"syslog":                 "List of syslog servers to send logs to.",
		"tls_ciphers":            "List of TLS ciphers to use.",
		"tls_protocols":          "TLS protocol to use.",
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
					"consumer_timeout": schema.Int64Attribute{
						Description: parametersDescriptions["consumer_timeout"],
						Computed:    true,
					},
					"enable_monitoring": schema.BoolAttribute{
						Description: parametersDescriptions["enable_monitoring"],
						Computed:    true,
					},
					"graphite": schema.StringAttribute{
						Description: parametersDescriptions["graphite"],
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
					"plugins": schema.ListAttribute{
						Description: parametersDescriptions["plugins"],
						ElementType: types.StringType,
						Computed:    true,
					},
					"roles": schema.ListAttribute{
						Description: parametersDescriptions["roles"],
						ElementType: types.StringType,
						Computed:    true,
					},
					"syslog": schema.ListAttribute{
						Description: parametersDescriptions["syslog"],
						ElementType: types.StringType,
						Computed:    true,
					},
					"tls_ciphers": schema.ListAttribute{
						Description: parametersDescriptions["tls_ciphers"],
						ElementType: types.StringType,
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
	tflog.Info(ctx, "RabbitMQ instance read")
}
