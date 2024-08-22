package argus

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &scrapeConfigDataSource{}
)

// NewScrapeConfigDataSource is a helper function to simplify the provider implementation.
func NewScrapeConfigDataSource() datasource.DataSource {
	return &scrapeConfigDataSource{}
}

// scrapeConfigDataSource is the data source implementation.
type scrapeConfigDataSource struct {
	client *argus.APIClient
}

// Metadata returns the data source type name.
func (d *scrapeConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_argus_scrapeconfig"
}

func (d *scrapeConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var apiClient *argus.APIClient
	var err error

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}
	d.client = apiClient
}

// Schema defines the schema for the data source.
func (d *scrapeConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main": "Argus scrape config data source schema. Must have a `region` specified in the provider configuration.",
		"deprecation_message": "The `stackit_argus_scrapeconfig` data source has been deprecated and will be removed after February 26th 2025. " +
			"Please use `stackit_observability_scrapeconfig` instead, which offers the exact same functionality.",
	}
	resp.Schema = schema.Schema{
		Description:         fmt.Sprintf("%s\n%s", descriptions["main"], descriptions["deprecation_message"]),
		MarkdownDescription: fmt.Sprintf("%s\n\n!> %s", descriptions["main"], descriptions["deprecation_message"]),
		DeprecationMessage:  descriptions["deprecation_message"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`instance_id`,`name`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the scraping job is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "Argus instance ID to which the scraping job is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Specifies the name of the scraping job",
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"metrics_path": schema.StringAttribute{
				Description: "Specifies the job scraping url path.",
				Computed:    true,
			},

			"scheme": schema.StringAttribute{
				Description: "Specifies the http scheme.",
				Computed:    true,
			},

			"scrape_interval": schema.StringAttribute{
				Description: "Specifies the scrape interval as duration string.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 8),
				},
				Computed: true,
			},

			"sample_limit": schema.Int64Attribute{
				Description: "Specifies the scrape sample limit.",
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 3000000),
				},
			},

			"scrape_timeout": schema.StringAttribute{
				Description: "Specifies the scrape timeout as duration string.",
				Computed:    true,
			},
			"saml2": schema.SingleNestedAttribute{
				Description: "A SAML2 configuration block.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"enable_url_parameters": schema.BoolAttribute{
						Description: "Specifies if URL parameters are enabled",
						Computed:    true,
					},
				},
			},
			"basic_auth": schema.SingleNestedAttribute{
				Description: "A basic authentication block.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Description: "Specifies basic auth username.",
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 200),
						},
					},
					"password": schema.StringAttribute{
						Description: "Specifies basic auth password.",
						Computed:    true,
						Sensitive:   true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 200),
						},
					},
				},
			},
			"targets": schema.ListNestedAttribute{
				Description: "The targets list (specified by the static config).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"urls": schema.ListAttribute{
							Description: "Specifies target URLs.",
							Computed:    true,
							ElementType: types.StringType,
							Validators: []validator.List{
								listvalidator.ValueStringsAre(
									stringvalidator.LengthBetween(1, 500),
								),
							},
						},
						"labels": schema.MapAttribute{
							Description: "Specifies labels.",
							Computed:    true,
							ElementType: types.StringType,
							Validators: []validator.Map{
								mapvalidator.SizeAtMost(10),
								mapvalidator.ValueStringsAre(stringvalidator.LengthBetween(0, 200)),
								mapvalidator.KeysAre(stringvalidator.LengthBetween(0, 200)),
							},
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *scrapeConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	scName := model.Name.ValueString()

	scResp, err := d.client.GetScrapeConfig(ctx, instanceId, scName, projectId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Unable to read scrape config", err.Error())
		return
	}

	err = mapFields(ctx, scResp.Data, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Mapping fields", err.Error())
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Argus scrape config read")
}
