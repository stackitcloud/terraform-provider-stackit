package rabbitmq

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
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
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *rabbitmq.APIClient
	var err error
	if providerData.RabbitMQCustomEndpoint != "" {
		apiClient, err = rabbitmq.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.RabbitMQCustomEndpoint),
		)
	} else {
		apiClient, err = rabbitmq.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "RabbitMQ instance client configured")
}

// Schema defines the schema for the data source.
func (r *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "RabbitMQ instance data source schema.",
		"id":          "Terraform's internal data source. identifier. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the RabbitMQ instance.",
		"project_id":  "STACKIT Project ID to which the instance is associated.",
		"name":        "Instance name.",
		"version":     "The service version.",
		"plan_name":   "The selected plan name.",
		"plan_id":     "The selected plan ID.",
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
						Computed: true,
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API: %v", err))
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
