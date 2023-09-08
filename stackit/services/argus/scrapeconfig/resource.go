package argus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/validate"
)

const (
	DefaultScheme                   = "https" // API default is "http"
	DefaultScrapeInterval           = "5m"
	DefaultScrapeTimeout            = "2m"
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
	SAML2          *SAML2       `tfsdk:"saml2"`
	BasicAuth      *BasicAuth   `tfsdk:"basic_auth"`
	Targets        []Target     `tfsdk:"targets"`
}

type SAML2 struct {
	EnableURLParameters types.Bool `tfsdk:"enable_url_parameters"`
}

type Target struct {
	URLs   []types.String `tfsdk:"urls"`
	Labels types.Map      `tfsdk:"labels"`
}

type BasicAuth struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
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
func (r *scrapeConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected stackit.ProviderData, got %T. Please report this issue to the provider developers.", req.ProviderData))
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
		resp.Diagnostics.AddError("Could not Configure API Client", err.Error())
		return
	}
	r.client = apiClient
}

// Schema defines the schema for the resource.
func (r *scrapeConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
				Description: "Specifies the http scheme. E.g. `https`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(DefaultScheme),
			},
			"scrape_interval": schema.StringAttribute{
				Description: "Specifies the scrape interval as duration string. E.g. `5m`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 8),
				},
				Default: stringdefault.StaticString(DefaultScrapeInterval),
			},
			"scrape_timeout": schema.StringAttribute{
				Description: "Specifies the scrape timeout as duration string. E.g.`2m`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 8),
				},
				Default: stringdefault.StaticString(DefaultScrapeTimeout),
			},
			"saml2": schema.SingleNestedAttribute{
				Description: "A SAML2 configuration block.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enable_url_parameters": schema.BoolAttribute{
						Description: "Are URL parameters be enabled?",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(DefaultSAML2EnableURLParameters),
					},
				},
			},
			"basic_auth": schema.SingleNestedAttribute{
				Description: "A basic authentication block.",
				Optional:    true,
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

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		resp.Diagnostics.AddError("Error creating scrape config", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	_, err = r.client.CreateScrapeConfig(ctx, instanceId, projectId).CreateScrapeConfigPayload(*payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating scrape config", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = argus.CreateScrapeConfigWaitHandler(ctx, r.client, instanceId, scName, projectId).SetTimeout(3 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error creating scrape config", fmt.Sprintf("ScrapeConfig creation waiting: %v", err))
		return
	}
	got, err := r.client.GetScrapeConfig(ctx, instanceId, scName, projectId).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating scrape config", fmt.Sprintf("ScrapeConfig creation waiting: %v", err))
		return
	}
	err = mapFields(got.Data, &model)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping fields", fmt.Sprintf("Project id %s, ScrapeConfig id %s: %v", projectId, scName, err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "ARGUS scrape config created")
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
		resp.Diagnostics.AddError("Error reading scrape config", fmt.Sprintf("Project id = %s, instance id = %s, scrape config name = %s: %v", projectId, instanceId, scName, err))
		return
	}

	// Map response body to schema and populate Computed attribute values
	err = mapFields(scResp.Data, &model)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping fields", fmt.Sprintf("Project id = %s, instance id = %s, sc name = %s: %v", projectId, instanceId, scName, err))
		return
	}
	// Set refreshed model
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "ARGUS scrape config read")
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

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		resp.Diagnostics.AddError("Error updating scrape config", fmt.Sprintf("Could not create API payload: %v", err))
		return
	}
	_, err = r.client.UpdateScrapeConfig(ctx, instanceId, scName, projectId).UpdateScrapeConfigPayload(*payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating scrape config", fmt.Sprintf("Project id = %s, instance id = %s, scrape config name = %s: %v", projectId, instanceId, scName, err))
		return
	}
	// We do not have an update status provided by the argus scrape config api, so we cannot use a waiter here, hence a simple sleep is used.
	time.Sleep(15 * time.Second)

	// Fetch updated ScrapeConfig
	scResp, err := r.client.GetScrapeConfig(ctx, instanceId, scName, projectId).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated data", fmt.Sprintf("Project id %s, instance id %s, jo name %s: %v", projectId, instanceId, scName, err))
		return
	}
	err = mapFields(scResp.Data, &model)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping fields in update", "project id = "+projectId+", instance id = "+instanceId+", scrape config name = "+scName+", "+err.Error())
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "ARGUS scrape config updated")
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
		resp.Diagnostics.AddError("Error deleting scrape config", "project id = "+projectId+", instance id = "+instanceId+", scrape config name = "+scName+", "+err.Error())
		return
	}
	_, err = argus.DeleteScrapeConfigWaitHandler(ctx, r.client, instanceId, scName, projectId).SetTimeout(1 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting scrape config", fmt.Sprintf("ScrapeConfig deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "ARGUS scrape config deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,name
func (r *scrapeConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id],[name]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[2])...)
	tflog.Info(ctx, "ARGUS scrape config state imported")
}

func mapFields(sc *argus.Job, model *Model) error {
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
	handleSAML2(sc, model)
	handleBasicAuth(sc, model)
	handleTargets(sc, model)
	return nil
}

func handleBasicAuth(sc *argus.Job, model *Model) {
	if sc.BasicAuth == nil {
		model.BasicAuth = nil
		return
	}
	model.BasicAuth = &BasicAuth{
		Username: types.StringPointerValue(sc.BasicAuth.Username),
		Password: types.StringPointerValue(sc.BasicAuth.Password),
	}
}

func handleSAML2(sc *argus.Job, model *Model) {
	if (sc.Params == nil || *sc.Params == nil) && model.SAML2 == nil {
		return
	}

	if model.SAML2 == nil {
		model.SAML2 = &SAML2{}
	}

	flag := true
	if sc.Params == nil || *sc.Params == nil {
		return
	}
	p := *sc.Params
	if v, ok := p["saml2"]; ok {
		if len(v) == 1 && v[0] == "disabled" {
			flag = false
		}
	}

	model.SAML2 = &SAML2{
		EnableURLParameters: types.BoolValue(flag),
	}
}

func handleTargets(sc *argus.Job, model *Model) {
	if sc == nil || sc.StaticConfigs == nil {
		model.Targets = []Target{}
		return
	}
	newTargets := []Target{}
	for i, sc := range *sc.StaticConfigs {
		nt := Target{
			URLs: []types.String{},
		}
		if sc.Targets != nil {
			for _, v := range *sc.Targets {
				nt.URLs = append(nt.URLs, types.StringValue(v))
			}
		}

		if len(model.Targets) > i && model.Targets[i].Labels.IsNull() || sc.Labels == nil {
			nt.Labels = types.MapNull(types.StringType)
		} else {
			newl := map[string]attr.Value{}
			for k, v := range *sc.Labels {
				newl[k] = types.StringValue(v)
			}
			nt.Labels = types.MapValueMust(types.StringType, newl)
		}
		newTargets = append(newTargets, nt)
	}
	model.Targets = newTargets
}

func toCreatePayload(ctx context.Context, model *Model) (*argus.CreateScrapeConfigPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	sc := argus.CreateScrapeConfigPayload{
		JobName:        model.Name.ValueStringPointer(),
		MetricsPath:    model.MetricsPath.ValueStringPointer(),
		ScrapeInterval: model.ScrapeInterval.ValueStringPointer(),
		ScrapeTimeout:  model.ScrapeTimeout.ValueStringPointer(),
		Scheme:         model.Scheme.ValueStringPointer(),
	}
	setDefaultsCreateScrapeConfig(&sc, model)

	if model.SAML2 != nil && !model.SAML2.EnableURLParameters.ValueBool() {
		m := make(map[string]interface{})
		if sc.Params != nil {
			m = *sc.Params
		}
		m["saml2"] = []string{"disabled"}
		sc.Params = &m
	}

	if model.BasicAuth != nil {
		if sc.BasicAuth == nil {
			sc.BasicAuth = &argus.UpdateScrapeConfigPayloadBasicAuth{
				Username: model.BasicAuth.Username.ValueStringPointer(),
				Password: model.BasicAuth.Password.ValueStringPointer(),
			}
		}
	}

	t := make([]argus.CreateScrapeConfigPayloadStaticConfigsInner, len(model.Targets))
	for i, target := range model.Targets {
		ti := argus.CreateScrapeConfigPayloadStaticConfigsInner{}
		tgts := []string{}
		for _, v := range target.URLs {
			tgts = append(tgts, v.ValueString())
		}
		ti.Targets = &tgts

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

func setDefaultsCreateScrapeConfig(sc *argus.CreateScrapeConfigPayload, model *Model) {
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
	// Make the API default more explicit by setting the field.
	if model.SAML2 == nil || model.SAML2.EnableURLParameters.IsNull() || model.SAML2.EnableURLParameters.IsUnknown() {
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

func toUpdatePayload(ctx context.Context, model *Model) (*argus.UpdateScrapeConfigPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	sc := argus.UpdateScrapeConfigPayload{
		MetricsPath:    model.MetricsPath.ValueStringPointer(),
		ScrapeInterval: model.ScrapeInterval.ValueStringPointer(),
		ScrapeTimeout:  model.ScrapeTimeout.ValueStringPointer(),
		Scheme:         model.Scheme.ValueStringPointer(),
	}
	setDefaultsUpdateScrapeConfig(&sc, model)

	if model.SAML2 != nil && !model.SAML2.EnableURLParameters.ValueBool() {
		m := make(map[string]interface{})
		if sc.Params != nil {
			m = *sc.Params
		}
		m["saml2"] = []string{"disabled"}
		sc.Params = &m
	}

	if model.BasicAuth != nil {
		if sc.BasicAuth == nil {
			sc.BasicAuth = &argus.UpdateScrapeConfigPayloadBasicAuth{
				Username: model.BasicAuth.Username.ValueStringPointer(),
				Password: model.BasicAuth.Password.ValueStringPointer(),
			}
		}
	}

	t := make([]argus.UpdateScrapeConfigPayloadStaticConfigsInner, len(model.Targets))
	for i, target := range model.Targets {
		ti := argus.UpdateScrapeConfigPayloadStaticConfigsInner{}
		tgts := []string{}
		for _, v := range target.URLs {
			tgts = append(tgts, v.ValueString())
		}
		ti.Targets = &tgts

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
}
