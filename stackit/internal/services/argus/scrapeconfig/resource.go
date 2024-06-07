package argus

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
	"github.com/stackitcloud/stackit-sdk-go/services/argus/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

const (
	DefaultScheme                   = "https" // API default is "http"
	DefaultScrapeInterval           = "5m"
	DefaultScrapeTimeout            = "2m"
	DefaultSampleLimit              = int64(5000)
	DefaultSAML2EnableURLParameters = true
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &scrapeConfigResource{}
	_ resource.ResourceWithConfigure   = &scrapeConfigResource{}
	_ resource.ResourceWithImportState = &scrapeConfigResource{}
)

type Model struct {
	Id                    types.String `tfsdk:"id"` // needed by TF
	ProjectId             types.String `tfsdk:"project_id"`
	InstanceId            types.String `tfsdk:"instance_id"`
	Name                  types.String `tfsdk:"name"`
	MetricsPath           types.String `tfsdk:"metrics_path"`
	Scheme                types.String `tfsdk:"scheme"`
	ScrapeInterval        types.String `tfsdk:"scrape_interval"`
	ScrapeTimeout         types.String `tfsdk:"scrape_timeout"`
	SampleLimit           types.Int64  `tfsdk:"sample_limit"`
	SAML2                 types.Object `tfsdk:"saml2"`
	BasicAuth             types.Object `tfsdk:"basic_auth"`
	Targets               types.List   `tfsdk:"targets"`
	BearerToken           types.String `tfsdk:"bearer_token"`
	HonorLabels           types.Bool   `tfsdk:"honor_labels"`
	HonorTimeStamps       types.Bool   `tfsdk:"honor_timestamps"`
	HttpSdConfigs         types.List   `tfsdk:"http_sd_configs"`
	MetricsRelabelConfigs types.List   `tfsdk:"metrics_relabel_configs"`
	Oauth2                types.Object `tfsdk:"oauth2"`
	TlsConfig             types.Object `tfsdk:"tls_config"`
}

// Struct corresponding to Model.SAML2
type saml2Model struct {
	EnableURLParameters types.Bool `tfsdk:"enable_url_parameters"`
}

// Types corresponding to saml2Model
var saml2Types = map[string]attr.Type{
	"enable_url_parameters": types.BoolType,
}

// Struct corresponding to Model.BasicAuth
type basicAuthModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

// Types corresponding to basicAuthModel
var basicAuthTypes = map[string]attr.Type{
	"username": types.StringType,
	"password": types.StringType,
}

// Struct corresponding to Model.Targets[i]
type targetModel struct {
	URLs   types.List `tfsdk:"urls"`
	Labels types.Map  `tfsdk:"labels"`
}

// Types corresponding to targetModel
var targetTypes = map[string]attr.Type{
	"urls":   types.ListType{ElemType: types.StringType},
	"labels": types.MapType{ElemType: types.StringType},
}

// Struct corresponding to Model.HttpSdConfigs
type httpSdConfigModel struct {
	BasicAuth       types.Object `tfsdk:"basic_auth"`
	Oauth2          types.Object `tfsdk:"oauth2"`
	RefreshInterval types.String `tfsdk:"refresh_interval"`
	TlsConfig       types.Object `tfsdk:"tls_config"`
	Url             types.String `tfsdk:"url"`
}

// Types corresponding to httpSdConfigModel
var httpSdConfigsTypes = map[string]attr.Type{
	"basic_auth":       types.ObjectType{AttrTypes: basicAuthTypes},
	"oauth2":           types.ObjectType{AttrTypes: oauth2Types},
	"refresh_interval": types.StringType,
	"tls_config":       types.ObjectType{AttrTypes: tlsConfigTypes},
	"url":              types.StringType,
}

// Struct corresponding to Model.MetricsRelabelConfigs
type metricsRelabelConfigModel struct {
	Action       types.String `tfsdk:"action"`
	Modulus      types.Int64  `tfsdk:"modulus"`
	Regex        types.String `tfsdk:"regex"`
	Replacement  types.String `tfsdk:"replacement"`
	Separator    types.String `tfsdk:"separator"`
	SourceLabels types.List   `tfsdk:"source_labels"`
	TargetLabel  types.String `tfsdk:"target_label"`
}

// Types corresponding to metricsRelabelConfigModel
var metricsRelabelConfigsTypes = map[string]attr.Type{
	"action":        types.StringType,
	"modulus":       types.Int64Type,
	"regex":         types.StringType,
	"replacement":   types.StringType,
	"separator":     types.StringType,
	"source_labels": types.ListType{ElemType: types.StringType},
	"target_label":  types.StringType,
}

// Struct corresponding to Model.Oauth2
type oauth2Model struct {
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Scopes       types.List   `tfsdk:"scopes"`
	TlsConfig    types.Object `tfsdk:"tls_config"`
	TokenUrl     types.String `tfsdk:"token_url"`
}

// Types corresponding to oauth2Model
var oauth2Types = map[string]attr.Type{
	"client_id":     types.StringType,
	"client_secret": types.StringType,
	"scopes":        types.ListType{ElemType: types.StringType},
	"tls_config":    types.ObjectType{AttrTypes: tlsConfigTypes},
	"token_url":     types.StringType,
}

// Struct corresponding to Model.TlsConfig
type tlsConfigModel struct {
	InsecureSkipVerify types.Bool `tfsdk:"insecure_skip_verify"`
}

// Types corresponding to tlsConfigModel
var tlsConfigTypes = map[string]attr.Type{
	"insecure_skip_verify": types.BoolType,
}

// NewScrapeConfigResource is a helper function to simplify the provider implementation.
func NewScrapeConfigResource() resource.Resource {
	return &scrapeConfigResource{}
}

// scrapeConfigResource is the resource implementation.
type scrapeConfigResource struct {
	client *argus.APIClient
}

// Metadata returns the resource type name.
func (r *scrapeConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_argus_scrapeconfig"
}

// Configure adds the provider configured client to the resource.
func (r *scrapeConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *argus.APIClient
	var err error
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Argus scrape config client configured")
}

// Schema defines the schema for the resource.
func (r *scrapeConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Argus scrape config resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`,`name`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the scraping job is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "Argus instance ID to which the scraping job is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Specifies the name of the scraping job.",
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
					stringvalidator.LengthBetween(1, 200),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"metrics_path": schema.StringAttribute{
				Description: "Specifies the job scraping url path. E.g. `/metrics`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"scheme": schema.StringAttribute{
				Description: "Specifies the http scheme. Defaults to `https`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(DefaultScheme),
			},
			"scrape_interval": schema.StringAttribute{
				Description: "Specifies the scrape interval as duration string. Defaults to `5m`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 8),
				},
				Default: stringdefault.StaticString(DefaultScrapeInterval),
			},
			"scrape_timeout": schema.StringAttribute{
				Description: "Specifies the scrape timeout as duration string. Defaults to `2m`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 8),
				},
				Default: stringdefault.StaticString(DefaultScrapeTimeout),
			},
			"sample_limit": schema.Int64Attribute{
				Description: "Specifies the scrape sample limit. Upper limit depends on the service plan. Defaults to `5000`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 3000000),
				},
				Default: int64default.StaticInt64(DefaultSampleLimit),
			},
			"saml2": schema.SingleNestedAttribute{
				Description: "A SAML2 configuration block. Defaults to `true`",
				Optional:    true,
				Computed:    true,
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"enable_url_parameters": types.BoolType,
						},
						map[string]attr.Value{
							"enable_url_parameters": types.BoolValue(DefaultSAML2EnableURLParameters),
						},
					),
				),
				Attributes: map[string]schema.Attribute{
					"enable_url_parameters": schema.BoolAttribute{
						Description: "Specifies if URL parameters are enabled. Defaults to `true`",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(DefaultSAML2EnableURLParameters),
					},
				},
			},
			"basic_auth": schema.SingleNestedAttribute{
				Description: "A basic authentication block.",
				Optional:    true,
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
			"targets": schema.ListNestedAttribute{
				Description: "The targets list (specified by the static config).",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"urls": schema.ListAttribute{
							Description: "Specifies target URLs.",
							Required:    true,
							ElementType: types.StringType,
							Validators: []validator.List{
								listvalidator.ValueStringsAre(
									stringvalidator.LengthBetween(1, 500),
								),
							},
						},
						"labels": schema.MapAttribute{
							Description: "Specifies labels.",
							Optional:    true,
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
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
			},
			"honor_labels": schema.BoolAttribute{
				Description: "It controls whether Prometheus respects the labels in scraped data. Note that any globally configured 'external_labels' are unaffected by this setting. Defaults to `false`",
				Optional:    true,
				Computed:    true,
			},
			"honor_timestamps": schema.BoolAttribute{
				Description: "It controls whether Prometheus respects the timestamps present in scraped data. Defaults to `false`",
				Optional:    true,
				Computed:    true,
			},
			"http_sd_configs": schema.ListNestedAttribute{
				Description: "HTTP-based service discovery provides a more generic way to configure static targets and serves as an interface to plug in custom service discovery mechanisms.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"basic_auth": schema.SingleNestedAttribute{
							Description: "Sets the 'Authorization' header on every scrape request with the configured username and password.",
							Optional:    true,
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
							Optional:    true,
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
									Optional:    true,
									Computed:    true,
									ElementType: types.StringType,
								},
								"tls_config": schema.SingleNestedAttribute{
									Description: "Configures the scrape request's TLS settings.",
									Optional:    true,
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"insecure_skip_verify": schema.BoolAttribute{
											Description: "Disable validation of the server certificate. Defaults to `false`",
											Optional:    true,
											Computed:    true,
										},
									},
								},
							},
						},
						"refresh_interval": schema.StringAttribute{
							Description: "Refresh interval to re-query the endpoint. Defaults to `60s`",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(2, 8),
							},
						},
						"tls_config": schema.SingleNestedAttribute{
							Description: "Configures the scrape request's TLS settings.",
							Optional:    true,
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"insecure_skip_verify": schema.BoolAttribute{
									Description: "Disable validation of the server certificate. Defaults to `false`",
									Optional:    true,
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
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Description: "Action to perform based on regex matching. Defaults to `replace`",
							Optional:    true,
							Computed:    true,
						},
						"modulus": schema.Int64Attribute{
							Description: "Modulus to take of the hash of the source label values.",
							Optional:    true,
							Computed:    true,
						},
						"regex": schema.StringAttribute{
							Description: "Regular expression against which the extracted value is matched. Defaults to `.*`",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 400),
							},
						},
						"replacement": schema.StringAttribute{
							Description: "Replacement value against which a regex replace is performed if the regular expression matches.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 200),
							},
						},
						"separator": schema.StringAttribute{
							Description: "Separator placed between concatenated source label values.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 20),
							},
						},
						"target_label": schema.StringAttribute{
							Description: "Label to which the resulting value is written in a replace action.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 200),
							},
						},
						"source_labels": schema.ListAttribute{
							Description: `The source labels select values from existing labels.`,
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"oauth2": schema.SingleNestedAttribute{
				Description: "OAuth 2.0 authentication using the client credentials grant type.",
				Optional:    true,
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
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
					},
					"tls_config": schema.SingleNestedAttribute{
						Description: "Configures the scrape request's TLS settings.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"insecure_skip_verify": schema.BoolAttribute{
								Description: "Disable validation of the server certificate. Defaults to `false`",
								Optional:    true,
								Computed:    true,
							},
						},
					},
				},
			},
			"tls_config": schema.SingleNestedAttribute{
				Description: "",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"insecure_skip_verify": schema.BoolAttribute{
						Description: "Disable validation of the server certificate.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *scrapeConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	scName := model.Name.ValueString()

	saml2Model := saml2Model{}
	if !model.SAML2.IsNull() && !model.SAML2.IsUnknown() {
		diags = model.SAML2.As(ctx, &saml2Model, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	basicAuthModel := basicAuthModel{}
	if !model.BasicAuth.IsNull() && !model.BasicAuth.IsUnknown() {
		diags = model.BasicAuth.As(ctx, &basicAuthModel, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	targetsModel := []targetModel{}
	if !model.Targets.IsNull() && !model.Targets.IsUnknown() {
		diags = model.Targets.ElementsAs(ctx, &targetsModel, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	httpSdConfigsModel := []httpSdConfigModel{}
	if !model.HttpSdConfigs.IsNull() && !model.HttpSdConfigs.IsUnknown() {
		diags = model.HttpSdConfigs.ElementsAs(ctx, &httpSdConfigsModel, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	metricsRelabelConfigsModel := []metricsRelabelConfigModel{}
	if !model.MetricsRelabelConfigs.IsNull() && !model.MetricsRelabelConfigs.IsUnknown() {
		diags = model.MetricsRelabelConfigs.ElementsAs(ctx, &metricsRelabelConfigsModel, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	oauth2Model := oauth2Model{}
	if !model.Oauth2.IsNull() && !model.Oauth2.IsUnknown() {
		diags = model.Oauth2.As(ctx, &oauth2Model, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tlsConfigModel := tlsConfigModel{}
	if !model.TlsConfig.IsNull() && !model.TlsConfig.IsUnknown() {
		diags = model.TlsConfig.As(ctx, &tlsConfigModel, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model, &saml2Model, &basicAuthModel, &targetsModel, &httpSdConfigsModel, &metricsRelabelConfigsModel, &oauth2Model, &tlsConfigModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating scrape config", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	_, err = r.client.CreateScrapeConfig(ctx, instanceId, projectId).CreateScrapeConfigPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating scrape config", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.CreateScrapeConfigWaitHandler(ctx, r.client, instanceId, scName, projectId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating scrape config", fmt.Sprintf("Scrape config creation waiting: %v", err))
		return
	}
	got, err := r.client.GetScrapeConfig(ctx, instanceId, scName, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating scrape config", fmt.Sprintf("Calling API for updated data: %v", err))
		return
	}
	err = mapFields(ctx, got.Data, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating scrape config", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Argus scrape config created")
}

// Read refreshes the Terraform state with the latest data.
func (r *scrapeConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	scName := model.Name.ValueString()

	scResp, err := r.client.GetScrapeConfig(ctx, instanceId, scName, projectId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading scrape config", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, scResp.Data, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading scrape config", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed model
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Argus scrape config read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *scrapeConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	scName := model.Name.ValueString()

	saml2Model := saml2Model{}
	if !model.SAML2.IsNull() && !model.SAML2.IsUnknown() {
		diags = model.SAML2.As(ctx, &saml2Model, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	basicAuthModel := basicAuthModel{}
	if !model.BasicAuth.IsNull() && !model.BasicAuth.IsUnknown() {
		diags = model.BasicAuth.As(ctx, &basicAuthModel, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	targetsModel := []targetModel{}
	if !model.Targets.IsNull() && !model.Targets.IsUnknown() {
		diags = model.Targets.ElementsAs(ctx, &targetsModel, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	httpSdConfigsModel := []httpSdConfigModel{}
	if !model.HttpSdConfigs.IsNull() && !model.HttpSdConfigs.IsUnknown() {
		diags = model.HttpSdConfigs.ElementsAs(ctx, &httpSdConfigsModel, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	metricsRelabelConfigsModel := []metricsRelabelConfigModel{}
	if !model.MetricsRelabelConfigs.IsNull() && !model.MetricsRelabelConfigs.IsUnknown() {
		diags = model.MetricsRelabelConfigs.ElementsAs(ctx, &metricsRelabelConfigsModel, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	oauth2Model := oauth2Model{}
	if !model.Oauth2.IsNull() && !model.Oauth2.IsUnknown() {
		diags = model.Oauth2.As(ctx, &oauth2Model, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tlsConfigModel := tlsConfigModel{}
	if !model.TlsConfig.IsNull() && !model.TlsConfig.IsUnknown() {
		diags = model.TlsConfig.As(ctx, &tlsConfigModel, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, &saml2Model, &basicAuthModel, &targetsModel, &metricsRelabelConfigsModel, &tlsConfigModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating scrape config", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	_, err = r.client.UpdateScrapeConfig(ctx, instanceId, scName, projectId).UpdateScrapeConfigPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating scrape config", fmt.Sprintf("Calling API: %v", err))
		return
	}
	// We do not have an update status provided by the argus scrape config api, so we cannot use a waiter here, hence a simple sleep is used.
	time.Sleep(15 * time.Second)

	// Fetch updated ScrapeConfig
	scResp, err := r.client.GetScrapeConfig(ctx, instanceId, scName, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating scrape config", fmt.Sprintf("Calling API for updated data: %v", err))
		return
	}
	err = mapFields(ctx, scResp.Data, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating scrape config", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Argus scrape config updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *scrapeConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	scName := model.Name.ValueString()

	// Delete existing ScrapeConfig
	_, err := r.client.DeleteScrapeConfig(ctx, instanceId, scName, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting scrape config", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteScrapeConfigWaitHandler(ctx, r.client, instanceId, scName, projectId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting scrape config", fmt.Sprintf("Scrape config deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Argus scrape config deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,name
func (r *scrapeConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing scrape config",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id],[name]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[2])...)
	tflog.Info(ctx, "Argus scrape config state imported")
}

func mapFields(ctx context.Context, sc *argus.Job, model *Model) error {
	if sc == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var scName string
	if model.Name.ValueString() != "" {
		scName = model.Name.ValueString()
	} else if sc.JobName != nil {
		scName = *sc.JobName
	} else {
		return fmt.Errorf("scrape config name not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		model.InstanceId.ValueString(),
		scName,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.Name = types.StringValue(scName)

	model.MetricsPath = types.StringPointerValue(sc.MetricsPath)
	model.Scheme = types.StringPointerValue(sc.Scheme)
	model.ScrapeInterval = types.StringPointerValue(sc.ScrapeInterval)
	model.ScrapeTimeout = types.StringPointerValue(sc.ScrapeTimeout)
	model.SampleLimit = types.Int64PointerValue(sc.SampleLimit)
	model.BearerToken = types.StringPointerValue(sc.BearerToken)
	model.HonorLabels = types.BoolPointerValue(sc.HonorLabels)
	model.HonorTimeStamps = types.BoolPointerValue(sc.HonorTimeStamps)
	err := mapSAML2(sc, model)
	if err != nil {
		return fmt.Errorf("map saml2: %w", err)
	}
	err = mapBasicAuth(sc, model)
	if err != nil {
		return fmt.Errorf("map basic auth: %w", err)
	}
	err = mapTargets(ctx, sc, model)
	if err != nil {
		return fmt.Errorf("map targets: %w", err)
	}
	err = mapHttpSdConfigs(ctx, sc, model)
	if err != nil {
		return fmt.Errorf("map http sd configs: %w", err)
	}
	err = mapMetricsRelabelConfigs(ctx, sc, model)
	if err != nil {
		return fmt.Errorf("map metrics relabel configs: %w", err)
	}
	err = mapOauth2(ctx, sc, model)
	if err != nil {
		return fmt.Errorf("map oauth2: %w", err)
	}
	err = mapTlsConfig(sc, model)
	if err != nil {
		return fmt.Errorf("map tls config: %w", err)
	}
	return nil
}

func mapBasicAuth(sc *argus.Job, model *Model) error {
	if sc.BasicAuth == nil {
		model.BasicAuth = types.ObjectNull(basicAuthTypes)
		return nil
	}
	basicAuthMap := map[string]attr.Value{
		"username": types.StringValue(*sc.BasicAuth.Username),
		"password": types.StringValue(*sc.BasicAuth.Password),
	}
	basicAuthTF, diags := types.ObjectValue(basicAuthTypes, basicAuthMap)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.BasicAuth = basicAuthTF
	return nil
}

func mapSAML2(sc *argus.Job, model *Model) error {
	if (sc.Params == nil || *sc.Params == nil) && model.SAML2.IsNull() {
		return nil
	}

	if model.SAML2.IsNull() || model.SAML2.IsUnknown() {
		model.SAML2 = types.ObjectNull(saml2Types)
	}

	flag := true
	if sc.Params == nil || *sc.Params == nil {
		return nil
	}
	p := *sc.Params
	if v, ok := p["saml2"]; ok {
		if len(v) == 1 && v[0] == "disabled" {
			flag = false
		}
	}

	saml2Map := map[string]attr.Value{
		"enable_url_parameters": types.BoolValue(flag),
	}
	saml2TF, diags := types.ObjectValue(saml2Types, saml2Map)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.SAML2 = saml2TF
	return nil
}

func mapTargets(ctx context.Context, sc *argus.Job, model *Model) error {
	if sc == nil || sc.StaticConfigs == nil {
		model.Targets = types.ListNull(types.ObjectType{AttrTypes: targetTypes})
		return nil
	}

	targetsModel := []targetModel{}
	if !model.Targets.IsNull() && !model.Targets.IsUnknown() {
		diags := model.Targets.ElementsAs(ctx, &targetsModel, false)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
	}

	newTargets := []attr.Value{}
	for i, sc := range *sc.StaticConfigs {
		nt := targetModel{}

		// Map URLs
		urls := []attr.Value{}
		if sc.Targets != nil {
			for _, v := range *sc.Targets {
				urls = append(urls, types.StringValue(v))
			}
		}
		nt.URLs = types.ListValueMust(types.StringType, urls)

		// Map Labels
		if len(model.Targets.Elements()) > i && targetsModel[i].Labels.IsNull() || sc.Labels == nil {
			nt.Labels = types.MapNull(types.StringType)
		} else {
			newl := map[string]attr.Value{}
			for k, v := range *sc.Labels {
				newl[k] = types.StringValue(v)
			}
			nt.Labels = types.MapValueMust(types.StringType, newl)
		}

		// Build target
		targetMap := map[string]attr.Value{
			"urls":   nt.URLs,
			"labels": nt.Labels,
		}
		targetTF, diags := types.ObjectValue(targetTypes, targetMap)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}

		newTargets = append(newTargets, targetTF)
	}

	targetsTF, diags := types.ListValue(types.ObjectType{AttrTypes: targetTypes}, newTargets)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	model.Targets = targetsTF
	return nil
}

func mapTlsConfig(sc *argus.Job, model *Model) error {
	if sc.TlsConfig == nil {
		model.TlsConfig = types.ObjectNull(tlsConfigTypes)
		return nil
	}
	tlsConfigMap := map[string]attr.Value{
		"insecure_skip_verify": types.BoolValue(*sc.TlsConfig.InsecureSkipVerify),
	}
	tlsConfigTF, diags := types.ObjectValue(tlsConfigTypes, tlsConfigMap)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.TlsConfig = tlsConfigTF
	return nil
}

func mapOauth2(ctx context.Context, sc *argus.Job, model *Model) error {
	if sc.Oauth2 == nil {
		model.Oauth2 = types.ObjectNull(oauth2Types)
		return nil
	}

	var diags diag.Diagnostics
	oauth2Model := oauth2Model{}
	if !model.Oauth2.IsNull() && !model.Oauth2.IsUnknown() {
		diags := model.Oauth2.As(ctx, &oauth2Model, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("converting oauth2 object: %v", diags.Errors())
		}
	}

	tlsConfigTF := types.ObjectNull(tlsConfigTypes)
	if sc.Oauth2.TlsConfig != nil {
		insecureSkipVerify := types.BoolNull()
		if sc.Oauth2.TlsConfig.InsecureSkipVerify != nil {
			insecureSkipVerify = types.BoolValue(*sc.Oauth2.TlsConfig.InsecureSkipVerify)
		}

		tlsConfigMap := map[string]attr.Value{
			"insecure_skip_verify": insecureSkipVerify,
		}

		tlsConfigTF, diags = types.ObjectValue(tlsConfigTypes, tlsConfigMap)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
	}

	var scopesTF basetypes.ListValue

	scopes := []attr.Value{}
	if sc.Oauth2.Scopes != nil {
		for _, scope := range *sc.Oauth2.Scopes {
			scopes = append(scopes, types.StringValue(scope))
		}

		scopesTF, diags = types.ListValue(types.StringType, scopes)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
	} else {
		scopesTF = types.ListNull(types.StringType)
	}

	oauth2Map := map[string]attr.Value{
		"client_id":     types.StringValue(*sc.Oauth2.ClientId),
		"client_secret": types.StringValue(*sc.Oauth2.ClientSecret),
		"token_url":     types.StringValue(*sc.Oauth2.TokenUrl),
		"tls_config":    tlsConfigTF,
		"scopes":        scopesTF,
	}

	oauth2TF, diags := types.ObjectValue(oauth2Types, oauth2Map)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Oauth2 = oauth2TF
	return nil
}

func mapHttpSdConfigs(ctx context.Context, sc *argus.Job, model *Model) error {
	if sc == nil || sc.HttpSdConfigs == nil {
		model.HttpSdConfigs = types.ListNull(types.ObjectType{AttrTypes: httpSdConfigsTypes})
		return nil
	}

	var diags diag.Diagnostics
	httpSdConfigsModel := []httpSdConfigModel{}
	if !model.HttpSdConfigs.IsNull() && !model.HttpSdConfigs.IsUnknown() {
		diags := model.HttpSdConfigs.ElementsAs(ctx, &httpSdConfigsModel, false)
		if diags.HasError() {
			return fmt.Errorf("converting http sd configs object: %v", diags.Errors())
		}
	}

	tlsConfigTF := types.ObjectNull(tlsConfigTypes)

	newHttpSdConfigs := []attr.Value{}
	for _, httpSdConfig := range *sc.HttpSdConfigs {
		if httpSdConfig.TlsConfig != nil {
			insecureSkipVerify := types.BoolNull()
			if httpSdConfig.TlsConfig.InsecureSkipVerify != nil {
				insecureSkipVerify = types.BoolValue(*httpSdConfig.TlsConfig.InsecureSkipVerify)
			}

			tlsConfigMap := map[string]attr.Value{
				"insecure_skip_verify": insecureSkipVerify,
			}

			tlsConfigTF, diags = types.ObjectValue(tlsConfigTypes, tlsConfigMap)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}

		basicAuthTF := types.ObjectNull(basicAuthTypes)

		if httpSdConfig.BasicAuth != nil {
			basicAuthMap := map[string]attr.Value{
				"username": types.StringValue(*httpSdConfig.BasicAuth.Username),
				"password": types.StringValue(*httpSdConfig.BasicAuth.Password),
			}
			basicAuthTF, diags = types.ObjectValue(basicAuthTypes, basicAuthMap)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}

		oauth2TF := types.ObjectNull(oauth2Types)
		if httpSdConfig.Oauth2 != nil {
			oauth2TlsConfigTF := types.ObjectNull(tlsConfigTypes)
			if httpSdConfig.Oauth2.TlsConfig != nil {
				oauth2InsecureSkipVerify := types.BoolNull()
				if httpSdConfig.Oauth2.TlsConfig.InsecureSkipVerify != nil {
					oauth2InsecureSkipVerify = types.BoolValue(*httpSdConfig.Oauth2.TlsConfig.InsecureSkipVerify)
				}

				oauth2TlsConfigMap := map[string]attr.Value{
					"insecure_skip_verify": oauth2InsecureSkipVerify,
				}

				oauth2TlsConfigTF, diags = types.ObjectValue(tlsConfigTypes, oauth2TlsConfigMap)
				if diags.HasError() {
					return core.DiagsToError(diags)
				}
			}

			var oauth2ScopesTF basetypes.ListValue
			scopes := []attr.Value{}
			if httpSdConfig.Oauth2.Scopes != nil {
				for _, scope := range *httpSdConfig.Oauth2.Scopes {
					scopes = append(scopes, types.StringValue(scope))
				}
				oauth2ScopesTF, diags = types.ListValue(types.StringType, scopes)
				if diags.HasError() {
					return core.DiagsToError(diags)
				}
			} else {
				oauth2ScopesTF = types.ListNull(types.StringType)
			}

			oauth2Map := map[string]attr.Value{
				"client_id":     types.StringValue(*httpSdConfig.Oauth2.ClientId),
				"client_secret": types.StringValue(*httpSdConfig.Oauth2.ClientSecret),
				"token_url":     types.StringValue(*httpSdConfig.Oauth2.TokenUrl),
				"tls_config":    oauth2TlsConfigTF,
				"scopes":        oauth2ScopesTF,
			}

			oauth2TF, diags = types.ObjectValue(oauth2Types, oauth2Map)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}

		// Build target
		httpSdConfigMap := map[string]attr.Value{
			"refresh_interval": types.StringValue(*httpSdConfig.RefreshInterval),
			"url":              types.StringValue(*httpSdConfig.Url),
			"tls_config":       tlsConfigTF,
			"oauth2":           oauth2TF,
			"basic_auth":       basicAuthTF,
		}

		httpSdConfigTF, diags := types.ObjectValue(httpSdConfigsTypes, httpSdConfigMap)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}

		newHttpSdConfigs = append(newHttpSdConfigs, httpSdConfigTF)
	}

	httpSdConfigsTF, diags := types.ListValue(types.ObjectType{AttrTypes: httpSdConfigsTypes}, newHttpSdConfigs)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.HttpSdConfigs = httpSdConfigsTF
	return nil
}

func mapMetricsRelabelConfigs(ctx context.Context, sc *argus.Job, model *Model) error {
	if sc == nil || sc.MetricsRelabelConfigs == nil {
		model.MetricsRelabelConfigs = types.ListNull(types.ObjectType{AttrTypes: metricsRelabelConfigsTypes})
		return nil
	}
	var diags diag.Diagnostics
	metricsRelabelConfigsModel := []metricsRelabelConfigModel{}
	if !model.MetricsRelabelConfigs.IsNull() && !model.MetricsRelabelConfigs.IsUnknown() {
		diags := model.MetricsRelabelConfigs.ElementsAs(ctx, &metricsRelabelConfigsModel, false)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
	}

	metricsRelabelConfigs := []attr.Value{}
	for _, metricsRelabelConfigsResp := range *sc.MetricsRelabelConfigs {
		var sourceLabelsTF basetypes.ListValue

		if metricsRelabelConfigsResp.SourceLabels != nil {
			sourceLabelsTF, diags = types.ListValueFrom(ctx, types.StringType, *metricsRelabelConfigsResp.SourceLabels)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		} else {
			sourceLabelsTF = types.ListNull(types.StringType)
		}

		metricsRelabelConfig := map[string]attr.Value{
			"action":        types.StringPointerValue(metricsRelabelConfigsResp.Action),
			"modulus":       types.Int64PointerValue(metricsRelabelConfigsResp.Modulus),
			"regex":         types.StringPointerValue(metricsRelabelConfigsResp.Regex),
			"replacement":   types.StringPointerValue(metricsRelabelConfigsResp.Replacement),
			"separator":     types.StringPointerValue(metricsRelabelConfigsResp.Separator),
			"target_label":  types.StringPointerValue(metricsRelabelConfigsResp.TargetLabel),
			"source_labels": sourceLabelsTF,
		}

		metricsRelabelConfigTF, diags := basetypes.NewObjectValue(metricsRelabelConfigsTypes, metricsRelabelConfig)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
		metricsRelabelConfigs = append(metricsRelabelConfigs, metricsRelabelConfigTF)
	}
	metricsRelabelConfigsTF, diags := basetypes.NewListValue(types.ObjectType{AttrTypes: metricsRelabelConfigsTypes}, metricsRelabelConfigs)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.MetricsRelabelConfigs = metricsRelabelConfigsTF
	return nil
}

func toCreatePayload(ctx context.Context, model *Model, saml2Model *saml2Model, basicAuthObj *basicAuthModel, targetsModel *[]targetModel, httpSdConfigsModel *[]httpSdConfigModel, metricsRelabelConfigsModel *[]metricsRelabelConfigModel, oauth2Obj *oauth2Model, tlsConfigObj *tlsConfigModel) (*argus.CreateScrapeConfigPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	sc := argus.CreateScrapeConfigPayload{
		JobName:        conversion.StringValueToPointer(model.Name),
		MetricsPath:    conversion.StringValueToPointer(model.MetricsPath),
		ScrapeInterval: conversion.StringValueToPointer(model.ScrapeInterval),
		ScrapeTimeout:  conversion.StringValueToPointer(model.ScrapeTimeout),
		// potentially lossy conversion, depending on the allowed range for sample_limit
		SampleLimit:     utils.Ptr(float64(model.SampleLimit.ValueInt64())),
		Scheme:          conversion.StringValueToPointer(model.Scheme),
		BearerToken:     conversion.StringValueToPointer(model.BearerToken),
		HonorLabels:     conversion.BoolValueToPointer(model.HonorLabels),
		HonorTimeStamps: conversion.BoolValueToPointer(model.HonorTimeStamps),
	}
	setDefaultsCreateScrapeConfig(&sc, model, saml2Model)

	if !saml2Model.EnableURLParameters.IsNull() && !saml2Model.EnableURLParameters.IsUnknown() {
		m := make(map[string]interface{})
		if sc.Params != nil {
			m = *sc.Params
		}
		if saml2Model.EnableURLParameters.ValueBool() {
			m["saml2"] = []string{"enabled"}
		} else {
			m["saml2"] = []string{"disabled"}
		}
		sc.Params = &m
	}

	if sc.BasicAuth == nil && !basicAuthObj.Username.IsNull() && !basicAuthObj.Password.IsNull() {
		sc.BasicAuth = &argus.CreateScrapeConfigPayloadBasicAuth{
			Username: conversion.StringValueToPointer(basicAuthObj.Username),
			Password: conversion.StringValueToPointer(basicAuthObj.Password),
		}
	}

	t := make([]argus.CreateScrapeConfigPayloadStaticConfigsInner, len(*targetsModel))
	for i, target := range *targetsModel {
		ti := argus.CreateScrapeConfigPayloadStaticConfigsInner{}

		urls := []string{}
		diags := target.URLs.ElementsAs(ctx, &urls, false)
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}
		ti.Targets = &urls

		labels := map[string]interface{}{}
		for k, v := range target.Labels.Elements() {
			labels[k], _ = conversion.ToString(ctx, v)
		}
		ti.Labels = &labels
		t[i] = ti
	}
	sc.StaticConfigs = &t

	if sc.TlsConfig == nil && !tlsConfigObj.InsecureSkipVerify.IsNull() && !tlsConfigObj.InsecureSkipVerify.IsNull() {
		sc.TlsConfig = &argus.CreateScrapeConfigPayloadHttpSdConfigsInnerOauth2TlsConfig{
			InsecureSkipVerify: conversion.BoolValueToPointer(tlsConfigObj.InsecureSkipVerify),
		}
	}

	metricsRelabelConfigs := make([]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner, len(*metricsRelabelConfigsModel))

	for i, metricsRelabelConfig := range *metricsRelabelConfigsModel { //nolint:gocritic // disable linter temporarily
		metricsRelabelConfigsInner := argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{}

		metricsRelabelConfigsInner.Action = conversion.StringValueToPointer(metricsRelabelConfig.Action)
		metricsRelabelConfigsInner.Modulus = utils.Ptr(float64(metricsRelabelConfig.Modulus.ValueInt64()))
		metricsRelabelConfigsInner.Regex = conversion.StringValueToPointer(metricsRelabelConfig.Regex)
		metricsRelabelConfigsInner.Replacement = conversion.StringValueToPointer(metricsRelabelConfig.Replacement)
		metricsRelabelConfigsInner.Separator = conversion.StringValueToPointer(metricsRelabelConfig.Separator)
		metricsRelabelConfigsInner.TargetLabel = conversion.StringValueToPointer(metricsRelabelConfig.TargetLabel)

		sourceLabels := []string{}
		diags := metricsRelabelConfig.SourceLabels.ElementsAs(ctx, &sourceLabels, true)
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}
		metricsRelabelConfigsInner.SourceLabels = &sourceLabels
		metricsRelabelConfigs[i] = metricsRelabelConfigsInner
	}

	sc.MetricsRelabelConfigs = &metricsRelabelConfigs

	if sc.Oauth2 == nil && !oauth2Obj.ClientId.IsNull() && !oauth2Obj.ClientSecret.IsNull() {
		scopes := []string{}
		diags := oauth2Obj.Scopes.ElementsAs(ctx, &scopes, true)
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}

		sc.Oauth2 = &argus.CreateScrapeConfigPayloadHttpSdConfigsInnerOauth2{
			ClientId:     conversion.StringValueToPointer(oauth2Obj.ClientId),
			ClientSecret: conversion.StringValueToPointer(oauth2Obj.ClientSecret),
			TokenUrl:     conversion.StringValueToPointer(oauth2Obj.TokenUrl),
			Scopes:       &scopes,
		}

		oauth2TlsConfig := tlsConfigModel{}
		diags = oauth2Obj.TlsConfig.As(ctx, &oauth2TlsConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}

		if !oauth2Obj.TlsConfig.IsNull() {
			sc.Oauth2.TlsConfig = &argus.CreateScrapeConfigPayloadHttpSdConfigsInnerOauth2TlsConfig{
				InsecureSkipVerify: conversion.BoolValueToPointer(oauth2TlsConfig.InsecureSkipVerify),
			}
		}
	}

	httpSdConfigs := make([]argus.CreateScrapeConfigPayloadHttpSdConfigsInner, len(*httpSdConfigsModel))

	for i, httpSdConfig := range *httpSdConfigsModel {
		httpSdConfigsInner := argus.CreateScrapeConfigPayloadHttpSdConfigsInner{}

		httpSdConfigsInner.Url = conversion.StringValueToPointer(httpSdConfig.Url)
		httpSdConfigsInner.RefreshInterval = conversion.StringValueToPointer(httpSdConfig.RefreshInterval)

		basicAuth := basicAuthModel{}
		diags := httpSdConfig.BasicAuth.As(ctx, &basicAuth, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}

		if httpSdConfigsInner.BasicAuth == nil && !basicAuth.Username.IsNull() && !basicAuth.Password.IsNull() {
			httpSdConfigsInner.BasicAuth = &argus.CreateScrapeConfigPayloadBasicAuth{
				Username: conversion.StringValueToPointer(basicAuth.Username),
				Password: conversion.StringValueToPointer(basicAuth.Password),
			}
		}

		httpSdConfigTls := tlsConfigModel{}
		diags = httpSdConfig.TlsConfig.As(ctx, &httpSdConfigTls, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}

		if httpSdConfigsInner.TlsConfig == nil && !httpSdConfigTls.InsecureSkipVerify.IsNull() {
			httpSdConfigsInner.TlsConfig = &argus.CreateScrapeConfigPayloadHttpSdConfigsInnerOauth2TlsConfig{
				InsecureSkipVerify: conversion.BoolValueToPointer(httpSdConfigTls.InsecureSkipVerify),
			}
		}

		hsciOauth2 := oauth2Model{}
		if !httpSdConfig.Oauth2.IsNull() {
			diags = httpSdConfig.Oauth2.As(ctx, &hsciOauth2, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, core.DiagsToError(diags)
			}

			if httpSdConfigsInner.Oauth2 == nil && !hsciOauth2.ClientId.IsNull() && !hsciOauth2.ClientSecret.IsNull() && !hsciOauth2.TokenUrl.IsNull() && !hsciOauth2.Scopes.IsNull() && !hsciOauth2.TlsConfig.IsNull() {
				scopes := []string{}
				diags = hsciOauth2.Scopes.ElementsAs(ctx, &scopes, true)
				if diags.HasError() {
					return nil, core.DiagsToError(diags)
				}

				httpSdConfigsInner.Oauth2 = &argus.CreateScrapeConfigPayloadHttpSdConfigsInnerOauth2{
					ClientId:     conversion.StringValueToPointer(hsciOauth2.ClientId),
					ClientSecret: conversion.StringValueToPointer(hsciOauth2.ClientSecret),
					Scopes:       &scopes,
					TokenUrl:     conversion.StringValueToPointer(hsciOauth2.TokenUrl),
				}

				oauth2HttpSdConfigTls := tlsConfigModel{}
				diags = hsciOauth2.TlsConfig.As(ctx, &oauth2HttpSdConfigTls, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, core.DiagsToError(diags)
				}

				if !hsciOauth2.TlsConfig.IsNull() {
					httpSdConfigsInner.Oauth2.TlsConfig = &argus.CreateScrapeConfigPayloadHttpSdConfigsInnerOauth2TlsConfig{
						InsecureSkipVerify: conversion.BoolValueToPointer(oauth2HttpSdConfigTls.InsecureSkipVerify),
					}
				}
			}
		}
		httpSdConfigs[i] = httpSdConfigsInner
	}

	sc.HttpSdConfigs = &httpSdConfigs

	return &sc, nil
}

func setDefaultsCreateScrapeConfig(sc *argus.CreateScrapeConfigPayload, model *Model, saml2Model *saml2Model) {
	if sc == nil {
		return
	}
	if model.Scheme.IsNull() || model.Scheme.IsUnknown() {
		sc.Scheme = utils.Ptr(DefaultScheme)
	}
	if model.ScrapeInterval.IsNull() || model.ScrapeInterval.IsUnknown() {
		sc.ScrapeInterval = utils.Ptr(DefaultScrapeInterval)
	}
	if model.ScrapeTimeout.IsNull() || model.ScrapeTimeout.IsUnknown() {
		sc.ScrapeTimeout = utils.Ptr(DefaultScrapeTimeout)
	}
	if model.SampleLimit.IsNull() || model.SampleLimit.IsUnknown() {
		sc.SampleLimit = utils.Ptr(float64(DefaultSampleLimit))
	}
	// Make the API default more explicit by setting the field.
	if saml2Model.EnableURLParameters.IsNull() || saml2Model.EnableURLParameters.IsUnknown() {
		m := map[string]interface{}{}
		if sc.Params != nil {
			m = *sc.Params
		}
		if DefaultSAML2EnableURLParameters {
			m["saml2"] = []string{"enabled"}
		} else {
			m["saml2"] = []string{"disabled"}
		}
		sc.Params = &m
	}
}

func toUpdatePayload(ctx context.Context, model *Model, saml2Model *saml2Model, basicAuthModel *basicAuthModel, targetsModel *[]targetModel, metricsRelabelConfigsModel *[]metricsRelabelConfigModel, tlsConfigModel *tlsConfigModel) (*argus.UpdateScrapeConfigPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	sc := argus.UpdateScrapeConfigPayload{
		MetricsPath:    conversion.StringValueToPointer(model.MetricsPath),
		ScrapeInterval: conversion.StringValueToPointer(model.ScrapeInterval),
		ScrapeTimeout:  conversion.StringValueToPointer(model.ScrapeTimeout),
		// potentially lossy conversion, depending on the allowed range for sample_limit
		SampleLimit:     utils.Ptr(float64(model.SampleLimit.ValueInt64())),
		Scheme:          conversion.StringValueToPointer(model.Scheme),
		BearerToken:     conversion.StringValueToPointer(model.BearerToken),
		HonorLabels:     conversion.BoolValueToPointer(model.HonorLabels),
		HonorTimeStamps: conversion.BoolValueToPointer(model.HonorTimeStamps),
	}
	setDefaultsUpdateScrapeConfig(&sc, model)

	var diags diag.Diagnostics

	if !saml2Model.EnableURLParameters.IsNull() && !saml2Model.EnableURLParameters.IsUnknown() {
		m := make(map[string]interface{})
		if sc.Params != nil {
			m = *sc.Params
		}
		if saml2Model.EnableURLParameters.ValueBool() {
			m["saml2"] = []string{"enabled"}
		} else {
			m["saml2"] = []string{"disabled"}
		}
		sc.Params = &m
	}

	if sc.BasicAuth == nil && !basicAuthModel.Username.IsNull() && !basicAuthModel.Password.IsNull() {
		sc.BasicAuth = &argus.CreateScrapeConfigPayloadBasicAuth{
			Username: conversion.StringValueToPointer(basicAuthModel.Username),
			Password: conversion.StringValueToPointer(basicAuthModel.Password),
		}
	}

	t := make([]argus.UpdateScrapeConfigPayloadStaticConfigsInner, len(*targetsModel))
	for i, target := range *targetsModel {
		ti := argus.UpdateScrapeConfigPayloadStaticConfigsInner{}

		urls := []string{}
		diags := target.URLs.ElementsAs(ctx, &urls, false)
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}
		ti.Targets = &urls

		ls := map[string]interface{}{}
		for k, v := range target.Labels.Elements() {
			ls[k], _ = conversion.ToString(ctx, v)
		}
		ti.Labels = &ls
		t[i] = ti
	}
	sc.StaticConfigs = &t

	if sc.TlsConfig == nil && !tlsConfigModel.InsecureSkipVerify.IsNull() && !tlsConfigModel.InsecureSkipVerify.IsNull() {
		sc.TlsConfig = &argus.CreateScrapeConfigPayloadHttpSdConfigsInnerOauth2TlsConfig{
			InsecureSkipVerify: conversion.BoolValueToPointer(tlsConfigModel.InsecureSkipVerify),
		}
	}

	mrcs := make([]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner, len(*metricsRelabelConfigsModel))

	for i, metricsRelabelConfig := range *metricsRelabelConfigsModel { //nolint:gocritic // disable linter temporarily
		mrcsi := argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{}

		mrcsi.Action = conversion.StringValueToPointer(metricsRelabelConfig.Action)
		mrcsi.Modulus = utils.Ptr(float64(metricsRelabelConfig.Modulus.ValueInt64()))
		mrcsi.Regex = conversion.StringValueToPointer(metricsRelabelConfig.Regex)
		mrcsi.Replacement = conversion.StringValueToPointer(metricsRelabelConfig.Replacement)
		mrcsi.Separator = conversion.StringValueToPointer(metricsRelabelConfig.Separator)
		mrcsi.TargetLabel = conversion.StringValueToPointer(metricsRelabelConfig.TargetLabel)

		sourceLabels := []string{}
		diags = metricsRelabelConfig.SourceLabels.ElementsAs(ctx, &sourceLabels, true)
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}
		mrcsi.SourceLabels = &sourceLabels
		mrcs[i] = mrcsi
	}

	sc.MetricsRelabelConfigs = &mrcs

	return &sc, nil
}

func setDefaultsUpdateScrapeConfig(sc *argus.UpdateScrapeConfigPayload, model *Model) {
	if sc == nil {
		return
	}
	if model.Scheme.IsNull() || model.Scheme.IsUnknown() {
		sc.Scheme = utils.Ptr(DefaultScheme)
	}
	if model.ScrapeInterval.IsNull() || model.ScrapeInterval.IsUnknown() {
		sc.ScrapeInterval = utils.Ptr(DefaultScrapeInterval)
	}
	if model.ScrapeTimeout.IsNull() || model.ScrapeTimeout.IsUnknown() {
		sc.ScrapeTimeout = utils.Ptr(DefaultScrapeTimeout)
	}
	if model.SampleLimit.IsNull() || model.SampleLimit.IsUnknown() {
		sc.SampleLimit = utils.Ptr(float64(DefaultSampleLimit))
	}
}
