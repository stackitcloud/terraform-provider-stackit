package postgresql

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresql"
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
	client *postgresql.APIClient
}

// Metadata returns the data source type name.
func (r *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_instance"
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

	var apiClient *postgresql.APIClient
	var err error
	if providerData.PostgreSQLCustomEndpoint != "" {
		apiClient, err = postgresql.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.PostgreSQLCustomEndpoint),
		)
	} else {
		apiClient, err = postgresql.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "PostgreSQL instance client configured")
}

// Schema defines the schema for the data source.
func (r *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main": "PostgreSQL instance data source schema. Must have a `region` specified in the provider configuration.",
		"deprecation_message": strings.Join(
			[]string{
				"The STACKIT PostgreSQL service will reach its end of support on June 30th 2024.",
				"Data sources of this type will stop working after that.",
				"Use stackit_postgresflex_instance instead.",
				"For more details, check https://docs.stackit.cloud/stackit/en/bring-your-data-to-stackit-postgresql-flex-138347648.html",
			},
			" ",
		),
		"id":          "Terraform's internal data source. identifier. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the PostgreSQL instance.",
		"project_id":  "STACKIT Project ID to which the instance is associated.",
		"name":        "Instance name.",
		"version":     "The service version.",
		"plan_name":   "The selected plan name.",
		"plan_id":     "The selected plan ID.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		// Callout block: https://developer.hashicorp.com/terraform/registry/providers/docs#callouts
		MarkdownDescription: fmt.Sprintf("%s\n\n!> %s", descriptions["main"], descriptions["deprecation_message"]),
		DeprecationMessage:  descriptions["deprecation_message"],
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
					"enable_monitoring": schema.BoolAttribute{
						Computed: true,
					},
					"metrics_frequency": schema.Int64Attribute{
						Computed: true,
					},
					"metrics_prefix": schema.StringAttribute{
						Computed: true,
					},
					"monitoring_instance_id": schema.StringAttribute{
						Computed: true,
					},
					"plugins": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
					},
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
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance data", "The STACKIT PostgreSQL service will reach its end of support on June 30th 2024. Resources of this type will stop working after that. Use stackit_postgresflex_instance instead.")
}
