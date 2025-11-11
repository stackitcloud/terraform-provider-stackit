package observability

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	observabilityUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/stackit-sdk-go/services/observability/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

const (
	DefaultScheme                   = observability.CREATESCRAPECONFIGPAYLOADSCHEME_HTTP // API default is "http"
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
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	InstanceId     types.String `tfsdk:"instance_id"`
	Name           types.String `tfsdk:"name"`
	MetricsPath    types.String `tfsdk:"metrics_path"`
	Scheme         types.String `tfsdk:"scheme"`
	ScrapeInterval types.String `tfsdk:"scrape_interval"`
	ScrapeTimeout  types.String `tfsdk:"scrape_timeout"`
	SampleLimit    types.Int64  `tfsdk:"sample_limit"`
	SAML2          types.Object `tfsdk:"saml2"`
	BasicAuth      types.Object `tfsdk:"basic_auth"`
	Targets        types.List   `tfsdk:"targets"`
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

// NewScrapeConfigResource is a helper function to simplify the provider implementation.
func NewScrapeConfigResource() resource.Resource {
	return &scrapeConfigResource{}
}

// scrapeConfigResource is the resource implementation.
type scrapeConfigResource struct {
	client *observability.APIClient
}

// Metadata returns the resource type name.
func (r *scrapeConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_scrapeconfig"
}

// Configure adds the provider configured client to the resource.
func (r *scrapeConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := observabilityUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Observability scrape config client configured")
}

// Schema defines the schema for the resource.
func (r *scrapeConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Observability scrape config resource schema. Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on resource level.",
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
				Description: "Observability instance ID to which the scraping job is associated.",
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
				Default:     stringdefault.StaticString(string(DefaultScheme)),
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
				Description: "A SAML2 configuration block.",
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

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model, &saml2Model, &basicAuthModel, targetsModel)
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
	tflog.Info(ctx, "Observability scrape config created")
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
	tflog.Info(ctx, "Observability scrape config read")
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

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, &saml2Model, &basicAuthModel, targetsModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating scrape config", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	_, err = r.client.UpdateScrapeConfig(ctx, instanceId, scName, projectId).UpdateScrapeConfigPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating scrape config", fmt.Sprintf("Calling API: %v", err))
		return
	}
	// We do not have an update status provided by the observability scrape config api, so we cannot use a waiter here, hence a simple sleep is used.
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
	tflog.Info(ctx, "Observability scrape config updated")
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

	tflog.Info(ctx, "Observability scrape config deleted")
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
	tflog.Info(ctx, "Observability scrape config state imported")
}

func mapFields(ctx context.Context, sc *observability.Job, model *Model) error {
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.InstanceId.ValueString(), scName)
	model.Name = types.StringValue(scName)
	model.MetricsPath = types.StringPointerValue(sc.MetricsPath)
	model.Scheme = types.StringValue(string(sc.GetScheme()))
	model.ScrapeInterval = types.StringPointerValue(sc.ScrapeInterval)
	model.ScrapeTimeout = types.StringPointerValue(sc.ScrapeTimeout)
	model.SampleLimit = types.Int64PointerValue(sc.SampleLimit)
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
	return nil
}

func mapBasicAuth(sc *observability.Job, model *Model) error {
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

func mapSAML2(sc *observability.Job, model *Model) error {
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

func mapTargets(ctx context.Context, sc *observability.Job, model *Model) error {
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

func toCreatePayload(ctx context.Context, model *Model, saml2Model *saml2Model, basicAuthModel *basicAuthModel, targetsModel []targetModel) (*observability.CreateScrapeConfigPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	sc := observability.CreateScrapeConfigPayload{
		JobName:        conversion.StringValueToPointer(model.Name),
		MetricsPath:    conversion.StringValueToPointer(model.MetricsPath),
		ScrapeInterval: conversion.StringValueToPointer(model.ScrapeInterval),
		ScrapeTimeout:  conversion.StringValueToPointer(model.ScrapeTimeout),
		// potentially lossy conversion, depending on the allowed range for sample_limit
		SampleLimit: sdkUtils.Ptr(float64(model.SampleLimit.ValueInt64())),
		Scheme:      observability.CreateScrapeConfigPayloadGetSchemeAttributeType(conversion.StringValueToPointer(model.Scheme)),
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

	if sc.BasicAuth == nil && !basicAuthModel.Username.IsNull() && !basicAuthModel.Password.IsNull() {
		sc.BasicAuth = &observability.CreateScrapeConfigPayloadBasicAuth{
			Username: conversion.StringValueToPointer(basicAuthModel.Username),
			Password: conversion.StringValueToPointer(basicAuthModel.Password),
		}
	}

	t := make([]observability.CreateScrapeConfigPayloadStaticConfigsInner, len(targetsModel))
	for i, target := range targetsModel {
		ti := observability.CreateScrapeConfigPayloadStaticConfigsInner{}

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

	return &sc, nil
}

func setDefaultsCreateScrapeConfig(sc *observability.CreateScrapeConfigPayload, model *Model, saml2Model *saml2Model) {
	if sc == nil {
		return
	}
	if model.Scheme.IsNull() || model.Scheme.IsUnknown() {
		sc.Scheme = DefaultScheme.Ptr()
	}
	if model.ScrapeInterval.IsNull() || model.ScrapeInterval.IsUnknown() {
		sc.ScrapeInterval = sdkUtils.Ptr(DefaultScrapeInterval)
	}
	if model.ScrapeTimeout.IsNull() || model.ScrapeTimeout.IsUnknown() {
		sc.ScrapeTimeout = sdkUtils.Ptr(DefaultScrapeTimeout)
	}
	if model.SampleLimit.IsNull() || model.SampleLimit.IsUnknown() {
		sc.SampleLimit = sdkUtils.Ptr(float64(DefaultSampleLimit))
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

func toUpdatePayload(ctx context.Context, model *Model, saml2Model *saml2Model, basicAuthModel *basicAuthModel, targetsModel []targetModel) (*observability.UpdateScrapeConfigPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	sc := observability.UpdateScrapeConfigPayload{
		MetricsPath:    conversion.StringValueToPointer(model.MetricsPath),
		ScrapeInterval: conversion.StringValueToPointer(model.ScrapeInterval),
		ScrapeTimeout:  conversion.StringValueToPointer(model.ScrapeTimeout),
		// potentially lossy conversion, depending on the allowed range for sample_limit
		SampleLimit: sdkUtils.Ptr(float64(model.SampleLimit.ValueInt64())),
		Scheme:      observability.UpdateScrapeConfigPayloadGetSchemeAttributeType(conversion.StringValueToPointer(model.Scheme)),
	}
	setDefaultsUpdateScrapeConfig(&sc, model)

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
		sc.BasicAuth = &observability.CreateScrapeConfigPayloadBasicAuth{
			Username: conversion.StringValueToPointer(basicAuthModel.Username),
			Password: conversion.StringValueToPointer(basicAuthModel.Password),
		}
	}

	t := make([]observability.UpdateScrapeConfigPayloadStaticConfigsInner, len(targetsModel))
	for i, target := range targetsModel {
		ti := observability.UpdateScrapeConfigPayloadStaticConfigsInner{}

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

	return &sc, nil
}

func setDefaultsUpdateScrapeConfig(sc *observability.UpdateScrapeConfigPayload, model *Model) {
	if sc == nil {
		return
	}
	if model.Scheme.IsNull() || model.Scheme.IsUnknown() {
		sc.Scheme = observability.UpdateScrapeConfigPayloadGetSchemeAttributeType(DefaultScheme.Ptr())
	}
	if model.ScrapeInterval.IsNull() || model.ScrapeInterval.IsUnknown() {
		sc.ScrapeInterval = sdkUtils.Ptr(DefaultScrapeInterval)
	}
	if model.ScrapeTimeout.IsNull() || model.ScrapeTimeout.IsUnknown() {
		sc.ScrapeTimeout = sdkUtils.Ptr(DefaultScrapeTimeout)
	}
	if model.SampleLimit.IsNull() || model.SampleLimit.IsUnknown() {
		sc.SampleLimit = sdkUtils.Ptr(float64(DefaultSampleLimit))
	}
}
