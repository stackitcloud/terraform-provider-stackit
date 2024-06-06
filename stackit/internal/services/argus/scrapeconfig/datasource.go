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
	resp.Schema = schema.Schema{
		Description: "Argus scrape config data source schema. Must have a `region` specified in the provider configuration.",
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
			"bearer_token": schema.StringAttribute{
				Description: "Sets the 'Authorization' header on every scrape request with the configured bearer token.",
				Computed:    true,
				Sensitive:   true,
			},
			"honor_labels": schema.BoolAttribute{
				Description: "It controls whether Prometheus respects the labels in scraped data. Note that any globally configured 'external_labels' are unaffected by this setting. Defaults to `false`",
				Computed:    true,
			},
			"honor_timestamps": schema.BoolAttribute{
				Description: "It controls whether Prometheus respects the timestamps present in scraped data. Defaults to `false`",
				Computed:    true,
			},
			"http_sd_configs": schema.ListNestedAttribute{
				Description: "HTTP-based service discovery provides a more generic way to configure static targets and serves as an interface to plug in custom service discovery mechanisms.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"basic_auth": schema.SingleNestedAttribute{
							Description: "Sets the 'Authorization' header on every scrape request with the configured username and password.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"username": schema.StringAttribute{
									Description: "Specifies basic auth username.",
									Required:    true,
									Validators: []validator.String{
										stringvalidator.LengthBetween(1, 200),
									},
								},
								"password": schema.StringAttribute{
									Description: "Specifies basic auth password.",
									Required:    true,
									Sensitive:   true,
									Validators: []validator.String{
										stringvalidator.LengthBetween(1, 200),
									},
								},
							},
						},
						"oauth2": schema.SingleNestedAttribute{
							Description: "OAuth 2.0 authentication using the client credentials grant type.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"client_id": schema.StringAttribute{
									Description: "",
									Required:    true,
									Validators: []validator.String{
										stringvalidator.LengthBetween(1, 200),
									},
								},
								"client_secret": schema.StringAttribute{
									Description: "",
									Required:    true,
									Sensitive:   true,
									Validators: []validator.String{
										stringvalidator.LengthBetween(1, 200),
									},
								},
								"token_url": schema.StringAttribute{
									Description: "The URL to fetch the token from.",
									Required:    true,
									Validators: []validator.String{
										stringvalidator.LengthBetween(1, 200),
									},
								},
								"scopes": schema.ListAttribute{
									Description: `The URL to fetch the token from.`,
									Computed:    true,
									ElementType: types.StringType,
								},
								"tls_config": schema.SingleNestedAttribute{
									Description: "Configures the scrape request's TLS settings.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"insecure_skip_verify": schema.BoolAttribute{
											Description: "Disable validation of the server certificate. Defaults to `false`",
											Computed:    true,
										},
									},
								},
							},
						},
						"refresh_interval": schema.StringAttribute{
							Description: "Refresh interval to re-query the endpoint. Defaults to `60s`",
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(2, 8),
							},
						},
						"tls_config": schema.SingleNestedAttribute{
							Description: "Configures the scrape request's TLS settings.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"insecure_skip_verify": schema.BoolAttribute{
									Description: "Disable validation of the server certificate. Defaults to `false`",
									Computed:    true,
								},
							},
						},
						"url": schema.StringAttribute{
							Description: "URL from which the targets are fetched.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtMost(400),
							},
						},
					},
				},
			},
			"metrics_relabel_configs": schema.ListNestedAttribute{
				Description: "List of metric relabel configurations.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Description: "Action to perform based on regex matching. Defaults to `replace`",
							Computed:    true,
						},
						"modulus": schema.Float64Attribute{
							Description: "Modulus to take of the hash of the source label values.",
							Computed:    true,
						},
						"regex": schema.StringAttribute{
							Description: "Regular expression against which the extracted value is matched. Defaults to `.*`",
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 400),
							},
						},
						"replacement": schema.StringAttribute{
							Description: "Replacement value against which a regex replace is performed if the regular expression matches.",
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 200),
							},
						},
						"separator": schema.StringAttribute{
							Description: "Separator placed between concatenated source label values.",
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 20),
							},
						},
						"target_label": schema.StringAttribute{
							Description: "Label to which the resulting value is written in a replace action.",
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 200),
							},
						},
						"source_labels": schema.ListAttribute{
							Description: `The source labels select values from existing labels.`,
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"oauth2": schema.SingleNestedAttribute{
				Description: "OAuth 2.0 authentication using the client credentials grant type.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						Description: "",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 200),
						},
					},
					"client_secret": schema.StringAttribute{
						Description: "",
						Required:    true,
						Sensitive:   true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 200),
						},
					},
					"token_url": schema.StringAttribute{
						Description: "The URL to fetch the token from.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 200),
						},
					},
					"scopes": schema.ListAttribute{
						Description: `The URL to fetch the token from.`,
						Computed:    true,
						ElementType: types.StringType,
					},
					"tls_config": schema.SingleNestedAttribute{
						Description: "Configures the scrape request's TLS settings.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"insecure_skip_verify": schema.BoolAttribute{
								Description: "Disable validation of the server certificate. Defaults to `false`",
								Computed:    true,
							},
						},
					},
				},
			},
			"tls_config": schema.SingleNestedAttribute{
				Description: "",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"insecure_skip_verify": schema.BoolAttribute{
						Description: "Disable validation of the server certificate.",
						Computed:    true,
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
