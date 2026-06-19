package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	workflowsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/workflows/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

const identityProviderTypeOAuth2 = "oauth2"

var (
	_ resource.Resource                   = &instanceResource{}
	_ resource.ResourceWithConfigure      = &instanceResource{}
	_ resource.ResourceWithImportState    = &instanceResource{}
	_ resource.ResourceWithModifyPlan     = &instanceResource{}
	_ resource.ResourceWithValidateConfig = &instanceResource{}
)

var schemaDescriptions = map[string]string{
	"id":                                   "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
	"instance_id":                          "Workflows instance ID.",
	"region":                               "STACKIT region. If not set, the provider region is used.",
	"project_id":                           "STACKIT project ID associated with the Workflows instance.",
	"display_name":                         "Instance display name. Max 25 characters.",
	"description":                          "Instance description. Max 256 characters.",
	"version":                              "Workflows version (e.g. `workflows-3.0-airflow-3.1`). Discover valid values via the `stackit_workflows_provider_options` data source.",
	"enable_stackit_example_dags":          "Include the STACKIT sample DAGs. Honored on Airflow 3 instances; older versions may reject.",
	"enable_airflow_example_dags":          "Enable the Airflow built-in example DAGs. Honored on Airflow 3 instances; older versions may reject.",
	"observability_id":                     "STACKIT Observability instance to receive metrics and logs.",
	"network":                              "Attach the instance to a STACKIT network. Changes force replacement.",
	"network.id":                           "STACKIT network ID.",
	"identity_provider":                    "Identity provider configuration. Only `oauth2` is currently supported.",
	"identity_provider.type":               "Identity provider type (`oauth2`).",
	"identity_provider.name":               "Display name for the IdP. `azure`, `okta`, `aws_cognito`, `keycloak` enable provider-specific token parsing.",
	"identity_provider.client_id":          "OAuth2 client ID.",
	"identity_provider.client_secret":      "OAuth2 client secret. Sensitive; must be re-sent on every IdP update.",
	"identity_provider.scope":              "OAuth2 scopes (space-separated, e.g. `openid email`).",
	"identity_provider.discovery_endpoint": "OAuth2 discovery endpoint (`.well-known/openid-configuration`).",
	"identity_provider.api_audience":       "Allowed audiences for the ID token.",
	"identity_provider.resource":           "OAuth2 resource indicator.",
	"identity_provider.roles_claim":        "Name of the claim that carries the user's roles.",
	"endpoints":                            "Instance endpoints. Populated by the server.",
	"endpoints.url":                        "Primary endpoint URL (Airflow UI).",
	"endpoints.redirect_url":               "OAuth2 redirect URL configured on the instance.",
	"status": fmt.Sprintf(
		"Lifecycle status of the Workflows instance. %s",
		tfutils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(workflows.AllowedInstanceStatusEnumValues)...),
	),
	"status_message": "Human-readable status detail. Populated by the server when status is `failed` or during convergence; empty otherwise.",
	"created_at":     "Creation timestamp (RFC 3339).",
}

type Model struct {
	ID                       types.String `tfsdk:"id"`
	InstanceID               types.String `tfsdk:"instance_id"`
	Region                   types.String `tfsdk:"region"`
	ProjectID                types.String `tfsdk:"project_id"`
	DisplayName              types.String `tfsdk:"display_name"`
	Description              types.String `tfsdk:"description"`
	Version                  types.String `tfsdk:"version"`
	EnableStackitExampleDags types.Bool   `tfsdk:"enable_stackit_example_dags"`
	EnableAirflowExampleDags types.Bool   `tfsdk:"enable_airflow_example_dags"`
	ObservabilityID          types.String `tfsdk:"observability_id"`
	Network                  types.Object `tfsdk:"network"`
	IdentityProvider         types.Object `tfsdk:"identity_provider"`
	Endpoints                types.Object `tfsdk:"endpoints"`
	Status                   types.String `tfsdk:"status"`
	StatusMessage            types.String `tfsdk:"status_message"`
	CreatedAt                types.String `tfsdk:"created_at"`
}

type networkModel struct {
	ID types.String `tfsdk:"id"`
}

var networkTypes = map[string]attr.Type{
	"id": basetypes.StringType{},
}

type identityProviderModel struct {
	Type              types.String `tfsdk:"type"`
	Name              types.String `tfsdk:"name"`
	ClientID          types.String `tfsdk:"client_id"`
	ClientSecret      types.String `tfsdk:"client_secret"`
	Scope             types.String `tfsdk:"scope"`
	DiscoveryEndpoint types.String `tfsdk:"discovery_endpoint"`
	APIAudience       types.Set    `tfsdk:"api_audience"`
	Resource          types.String `tfsdk:"resource"`
	RolesClaim        types.String `tfsdk:"roles_claim"`
}

var identityProviderTypes = map[string]attr.Type{
	"type":               basetypes.StringType{},
	"name":               basetypes.StringType{},
	"client_id":          basetypes.StringType{},
	"client_secret":      basetypes.StringType{},
	"scope":              basetypes.StringType{},
	"discovery_endpoint": basetypes.StringType{},
	"api_audience":       basetypes.SetType{ElemType: types.StringType},
	"resource":           basetypes.StringType{},
	"roles_claim":        basetypes.StringType{},
}

type endpointsModel struct {
	URL         types.String `tfsdk:"url"`
	RedirectURL types.String `tfsdk:"redirect_url"`
}

var endpointsTypes = map[string]attr.Type{
	"url":          basetypes.StringType{},
	"redirect_url": basetypes.StringType{},
}

type instanceResource struct {
	client       *workflows.APIClient
	providerData core.ProviderData
}

func NewWorkflowsInstanceResource() resource.Resource {
	return &instanceResource{}
}

func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.providerData = providerData

	features.CheckExperimentEnabled(ctx, &r.providerData, features.WorkflowsExperiment, "stackit_workflows_instance", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := workflowsUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
}

// ModifyPlan normalizes the planned state before apply.
func (r *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { //nolint:gocritic // function signature required by Terraform
	var configModel Model
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tfutils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize description so plan == state regardless of how the user spells
	// "no description":
	//   - empty literal (description = "")  → null
	//   - attribute omitted from HCL (config null) → null, overriding any
	//     Unknown the framework would otherwise compute for an Optional+Computed
	//     attribute. This lets ClearableString send "" to the server on Update
	//     and treats "remove the line" as an intentional clear.
	if configModel.Description.IsNull() {
		planModel.Description = types.StringNull()
	} else if !planModel.Description.IsUnknown() && planModel.Description.ValueString() == "" {
		planModel.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
}

// ValidateConfig catches misconfigured identity providers at plan time so the
// user sees a precise error instead of a generic server rejection on Create.
// Unknown values (e.g. unresolved variables) are deferred.
func (r *instanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateInstanceConfig(ctx, &model, &resp.Diagnostics)
}

func validateInstanceConfig(ctx context.Context, model *Model, diags *diag.Diagnostics) {
	if model.IdentityProvider.IsNull() || model.IdentityProvider.IsUnknown() {
		return
	}
	var ipm identityProviderModel
	if d := model.IdentityProvider.As(ctx, &ipm, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); d.HasError() {
		return
	}
	if ipm.Type.IsNull() || ipm.Type.IsUnknown() || ipm.Type.ValueString() != identityProviderTypeOAuth2 {
		return
	}
	required := []struct {
		name string
		val  types.String
	}{
		{"client_id", ipm.ClientID},
		{"client_secret", ipm.ClientSecret},
		{"scope", ipm.Scope},
		{"discovery_endpoint", ipm.DiscoveryEndpoint},
	}
	for _, f := range required {
		if f.val.IsUnknown() {
			continue
		}
		if f.val.IsNull() || f.val.ValueString() == "" {
			diags.AddError(
				"Invalid Workflows instance config",
				fmt.Sprintf("identity_provider.%s is required when identity_provider.type = oauth2.", f.name),
			)
		}
	}
}

func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflows_instance"
}

func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := fmt.Sprintf("Workflows instance resource schema. %s", core.ResourceRegionFallbackDocstring)
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.WorkflowsExperiment, core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Required:    true,
				// Server rejects displayName on UpdateInstance (FieldNotAllowed),
				// so any change must recreate the instance.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtMost(25),
				},
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtMost(256),
				},
			},
			"version": schema.StringAttribute{
				Description: schemaDescriptions["version"],
				Required:    true,
				Validators: []validator.String{
					workflowsUtils.Airflow3Version(),
				},
			},
			"enable_stackit_example_dags": schema.BoolAttribute{
				Description: schemaDescriptions["enable_stackit_example_dags"],
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enable_airflow_example_dags": schema.BoolAttribute{
				Description: schemaDescriptions["enable_airflow_example_dags"],
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"observability_id": schema.StringAttribute{
				Description: schemaDescriptions["observability_id"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network": schema.SingleNestedAttribute{
				Description: schemaDescriptions["network"],
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					// No update endpoint exists for `network` — adding, removing, or
					// changing the block requires recreating the instance.
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: schemaDescriptions["network.id"],
						Required:    true,
						Validators: []validator.String{
							validate.UUID(),
							validate.NoSeparator(),
						},
					},
				},
			},
			"identity_provider": schema.SingleNestedAttribute{
				Description: schemaDescriptions["identity_provider"],
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: schemaDescriptions["identity_provider.type"],
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(identityProviderTypeOAuth2),
						},
					},
					"name": schema.StringAttribute{
						Description: schemaDescriptions["identity_provider.name"],
						Optional:    true,
					},
					"client_id": schema.StringAttribute{
						Description: schemaDescriptions["identity_provider.client_id"],
						Optional:    true,
					},
					"client_secret": schema.StringAttribute{
						Description: schemaDescriptions["identity_provider.client_secret"],
						Optional:    true,
						Sensitive:   true,
					},
					"scope": schema.StringAttribute{
						Description: schemaDescriptions["identity_provider.scope"],
						Optional:    true,
					},
					"discovery_endpoint": schema.StringAttribute{
						Description: schemaDescriptions["identity_provider.discovery_endpoint"],
						Optional:    true,
						Validators: []validator.String{
							workflowsUtils.URLHTTPSOnly(),
						},
					},
					"api_audience": schema.SetAttribute{
						Description: schemaDescriptions["identity_provider.api_audience"],
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
					"resource": schema.StringAttribute{
						Description: schemaDescriptions["identity_provider.resource"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"roles_claim": schema.StringAttribute{
						Description: schemaDescriptions["identity_provider.roles_claim"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"endpoints": schema.SingleNestedAttribute{
				Description: schemaDescriptions["endpoints"],
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					// Endpoints don't change on update; reuse state to keep plan output quiet.
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Description: schemaDescriptions["endpoints.url"],
						Computed:    true,
					},
					"redirect_url": schema.StringAttribute{
						Description: schemaDescriptions["endpoints.redirect_url"],
						Computed:    true,
					},
				},
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status_message": schema.StringAttribute{
				Description: schemaDescriptions["status_message"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: schemaDescriptions["created_at"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Workflows instance", fmt.Sprintf("Building API payload: %v", err))
		return
	}

	createResp, err := r.client.DefaultAPI.CreateInstance(ctx, projectID, region).CreateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Workflows instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	if createResp == nil || createResp.Id == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Workflows instance", "Create API response: incomplete response (id missing)")
		return
	}
	instanceID := createResp.Id

	// Persist identifiers before the wait so cancellation/timeouts don't orphan state.
	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectID,
		"region":      region,
		"instance_id": instanceID,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Workflows instance", fmt.Sprintf("Waiting for instance to become active: %v", err))
		return
	}

	if err := mapFields(ctx, waitResp, &model, region); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Workflows instance", fmt.Sprintf("Processing response: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Workflows instance created", map[string]any{"instance_id": instanceID})
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()

	if instanceID == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)

	instance, err := r.client.DefaultAPI.GetInstance(ctx, projectID, region, instanceID).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Workflows instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	if err := mapFields(ctx, instance, &model, region); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Workflows instance", fmt.Sprintf("Processing response: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Info(ctx, "Workflows instance read", map[string]any{"instance_id": instanceID})
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { //nolint:gocritic // function signature required by Terraform
	var plan, state Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := plan.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(plan.Region)
	instanceID := plan.InstanceID.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)

	// Capture planned values that mapFields would overwrite from the server
	// response BEFORE persist runs, so sub-step gating later in Update still
	// compares plan vs prior-state (not plan vs persisted-state).
	plannedObservabilityID := plan.ObservabilityID
	plannedIdentityProvider := plan.IdentityProvider

	// Persist state after each successful sub-step. mapFields carries forward
	// the prior client_secret from plan.IdentityProvider; that's correct AFTER
	// the IdP sub-step has succeeded, but BEFORE it runs we must use the
	// PRIOR state's secret — otherwise a partial-failure scenario (instance
	// PATCH succeeds, IdP PATCH then 5xxs) would land the new planned secret
	// in state while the server still holds the old one, and the API never
	// returns the secret to detect the drift.
	idpStepRan := false
	persist := func(waitResp *workflows.Instance) bool {
		if !idpStepRan {
			savedIdP := plan.IdentityProvider
			plan.IdentityProvider = state.IdentityProvider
			defer func() { plan.IdentityProvider = savedIdP }()
		}
		if err := mapFields(ctx, waitResp, &plan, region); err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows instance", fmt.Sprintf("Processing response: %v", err))
			return false
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		return !resp.Diagnostics.HasError()
	}

	// Each sub-step writes state twice: once with the API response (so a later
	// wait timeout doesn't lose the user's input — e.g. a rotated client_secret),
	// then again with the settled wait response (refreshes status/endpoints).
	if instanceFieldsChanged(&plan, &state) {
		payload := toUpdateInstancePayload(&plan, &state)
		apiResp, err := r.client.DefaultAPI.UpdateInstance(ctx, projectID, region, instanceID).UpdateInstancePayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows instance", fmt.Sprintf("Calling UpdateInstance: %v", err))
			return
		}
		if !persist(apiResp) {
			return
		}
		waitResp, err := wait.UpdateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows instance", fmt.Sprintf("Waiting for instance update: %v", err))
			return
		}
		if !persist(waitResp) {
			return
		}
	}

	if !plannedIdentityProvider.Equal(state.IdentityProvider) {
		// Restore the planned IdP onto `plan` so toUpdateIdentityProviderPayload
		// reads the user's intent (prior persist may have substituted state.IdP).
		plan.IdentityProvider = plannedIdentityProvider
		payload, err := toUpdateIdentityProviderPayload(ctx, &plan, &state)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows instance", fmt.Sprintf("Building identity provider payload: %v", err))
			return
		}
		apiResp, err := r.client.DefaultAPI.UpdateIdentityProvider(ctx, projectID, region, instanceID).UpdateIdentityProviderPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows instance", fmt.Sprintf("Calling UpdateIdentityProvider: %v", err))
			return
		}
		idpStepRan = true
		if !persist(apiResp) {
			return
		}
		waitResp, err := wait.UpdateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows instance", fmt.Sprintf("Waiting for identity provider update: %v", err))
			return
		}
		if !persist(waitResp) {
			return
		}
	}

	if !plannedObservabilityID.Equal(state.ObservabilityID) {
		payload := &workflows.UpdateObservabilityPayload{
			ObservabilityId: conversion.ClearableString(plannedObservabilityID, state.ObservabilityID),
		}
		apiResp, err := r.client.DefaultAPI.UpdateObservability(ctx, projectID, region, instanceID).UpdateObservabilityPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows instance", fmt.Sprintf("Calling UpdateObservability: %v", err))
			return
		}
		if !persist(apiResp) {
			return
		}
		waitResp, err := wait.UpdateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows instance", fmt.Sprintf("Waiting for observability update: %v", err))
			return
		}
		if !persist(waitResp) {
			return
		}
	}

	tflog.Info(ctx, "Workflows instance updated", map[string]any{"instance_id": instanceID})
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)

	if err := r.client.DefaultAPI.DeleteInstance(ctx, projectID, region, instanceID).Execute(); err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Workflows instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	if _, err := wait.DeleteInstanceWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID).WaitWithContext(ctx); err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Workflows instance", fmt.Sprintf("Waiting for deletion: %v", err))
		return
	}
	tflog.Info(ctx, "Workflows instance deleted", map[string]any{"instance_id": instanceID})
}

// The expected format of the resource import identifier is: project_id,region,instance_id
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing Workflows instance", fmt.Sprintf("Invalid import ID %q: expected format is `project_id`,`region`,`instance_id`", req.ID))
		return
	}
	if !uuidRE.MatchString(idParts[0]) || !uuidRE.MatchString(idParts[2]) {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing Workflows instance", fmt.Sprintf("Invalid import ID %q: project_id and instance_id must be UUIDs", req.ID))
		return
	}
	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"region":      idParts[1],
		"instance_id": idParts[2],
	})
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Workflows instance state imported")
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func boolOrFalse(b *bool) types.Bool {
	if b == nil {
		return types.BoolValue(false)
	}
	return types.BoolPointerValue(b)
}

func toCreatePayload(ctx context.Context, model *Model) (*workflows.CreateInstancePayload, error) {
	if model == nil {
		return nil, errors.New("missing model")
	}

	idp, err := buildIdentityProvider(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("building identity provider: %w", err)
	}

	// Description: empty literal is treated as "not set" so a Create with
	// `description = ""` does not seed "" into server state and then drift to
	// null on the next Update.
	payload := &workflows.CreateInstancePayload{
		DisplayName:              model.DisplayName.ValueString(),
		Description:              conversion.NilIfEmpty(model.Description),
		Version:                  model.Version.ValueString(),
		EnableStackitExampleDags: conversion.BoolValueToPointer(model.EnableStackitExampleDags),
		EnableAirflowExampleDags: conversion.BoolValueToPointer(model.EnableAirflowExampleDags),
		ObservabilityId:          conversion.StringValueToPointer(model.ObservabilityID),
		IdentityProvider:         idp,
	}

	network, err := buildNetwork(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("building network: %w", err)
	}
	payload.Network = network

	return payload, nil
}

// instanceFieldsChanged returns true when at least one UpdateInstance-routable
// field differs. display_name is intentionally NOT checked here — the server
// rejects it on update, so the schema marks it RequiresReplace.
func instanceFieldsChanged(plan, state *Model) bool {
	return !plan.Description.Equal(state.Description) ||
		!plan.Version.Equal(state.Version) ||
		!plan.EnableStackitExampleDags.Equal(state.EnableStackitExampleDags) ||
		!plan.EnableAirflowExampleDags.Equal(state.EnableAirflowExampleDags)
}

// toUpdateInstancePayload assembles an UpdateInstance PATCH payload. The
// server treats description == "" as a clear, so we send "" when the user
// removed a previously-set description. DisplayName is intentionally omitted —
// server rejects it on update.
func toUpdateInstancePayload(plan, state *Model) *workflows.UpdateInstancePayload {
	return &workflows.UpdateInstancePayload{
		Description:              conversion.ClearableString(plan.Description, state.Description),
		Version:                  conversion.StringValueToPointer(plan.Version),
		EnableStackitExampleDags: conversion.BoolValueToPointer(plan.EnableStackitExampleDags),
		EnableAirflowExampleDags: conversion.BoolValueToPointer(plan.EnableAirflowExampleDags),
	}
}

func buildNetwork(ctx context.Context, model *Model) (*workflows.Network, error) {
	if model.Network.IsNull() || model.Network.IsUnknown() {
		return nil, nil
	}
	var nm networkModel
	diags := model.Network.As(ctx, &nm, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting network object: %w", core.DiagsToError(diags))
	}
	return &workflows.Network{Id: conversion.StringValueToPointer(nm.ID)}, nil
}

func buildIdentityProvider(ctx context.Context, model *Model) (*workflows.IdentityProvider, error) {
	if model.IdentityProvider.IsNull() || model.IdentityProvider.IsUnknown() {
		return nil, errors.New("identity_provider is required")
	}
	var ipm identityProviderModel
	diags := model.IdentityProvider.As(ctx, &ipm, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting identity_provider object: %w", core.DiagsToError(diags))
	}

	// Only oauth2 is reachable: the schema validator rejects any other value at
	// plan time. The default branch is defensive in case the validator and the
	// builder ever drift.
	switch ipm.Type.ValueString() {
	case identityProviderTypeOAuth2:
		audience, err := conversion.StringSetToSlice(ipm.APIAudience)
		if err != nil {
			return nil, fmt.Errorf("api_audience: %w", err)
		}
		oauth := &workflows.OAuth2IdentityProvider{
			Type:              workflows.OAUTH2IDENTITYPROVIDERTYPE_OAUTH2,
			Name:              ipm.Name.ValueString(),
			ClientId:          ipm.ClientID.ValueString(),
			ClientSecret:      ipm.ClientSecret.ValueString(),
			Scope:             ipm.Scope.ValueString(),
			DiscoveryEndpoint: ipm.DiscoveryEndpoint.ValueString(),
			ApiAudience:       audience,
			Resource:          conversion.StringValueToPointer(ipm.Resource),
			RolesClaim:        conversion.StringValueToPointer(ipm.RolesClaim),
		}
		wrapped := workflows.OAuth2IdentityProviderAsIdentityProvider(oauth)
		return &wrapped, nil
	default:
		return nil, fmt.Errorf("unsupported identity_provider type %q", ipm.Type.ValueString())
	}
}

// toUpdateIdentityProviderPayload builds the PATCH payload for the IdP.
//
// client_secret is always sent (when present in plan). The server requires
// re-supplying the secret whenever client_id or discovery_endpoint change as a
// credential-leak defense — without it, an attacker who could rotate just the
// URL would inherit the existing secret. Resource/roles_claim use "empty
// string clears" semantics, so a user removing the field translates to "".
//
// Note: OAuth2IdentityProviderPatch has no `Type` field in the OAS — the
// server's discriminated union resolver infers the variant from the absence of
// StackIT-specific fields.
func toUpdateIdentityProviderPayload(ctx context.Context, plan, state *Model) (*workflows.UpdateIdentityProviderPayload, error) {
	if plan.IdentityProvider.IsNull() || plan.IdentityProvider.IsUnknown() {
		return nil, errors.New("identity_provider is required")
	}
	var ipm identityProviderModel
	diags := plan.IdentityProvider.As(ctx, &ipm, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting identity_provider object: %w", core.DiagsToError(diags))
	}

	priorResource, priorRolesClaim := types.StringNull(), types.StringNull()
	if !state.IdentityProvider.IsNull() && !state.IdentityProvider.IsUnknown() {
		var prior identityProviderModel
		if diags := state.IdentityProvider.As(ctx, &prior, basetypes.ObjectAsOptions{}); !diags.HasError() {
			priorResource = prior.Resource
			priorRolesClaim = prior.RolesClaim
		}
	}

	switch ipm.Type.ValueString() {
	case identityProviderTypeOAuth2:
		audience, err := conversion.StringSetToSlice(ipm.APIAudience)
		if err != nil {
			return nil, fmt.Errorf("api_audience: %w", err)
		}
		patch := &workflows.OAuth2IdentityProviderPatch{
			Name:              conversion.StringValueToPointer(ipm.Name),
			ClientId:          conversion.StringValueToPointer(ipm.ClientID),
			ClientSecret:      conversion.StringValueToPointer(ipm.ClientSecret),
			Scope:             conversion.StringValueToPointer(ipm.Scope),
			DiscoveryEndpoint: conversion.StringValueToPointer(ipm.DiscoveryEndpoint),
			ApiAudience:       audience,
			Resource:          conversion.ClearableString(ipm.Resource, priorResource),
			RolesClaim:        conversion.ClearableString(ipm.RolesClaim, priorRolesClaim),
		}
		wrapped := workflows.OAuth2IdentityProviderPatchAsUpdateIdentityProviderPayload(patch)
		return &wrapped, nil
	default:
		return nil, fmt.Errorf("unsupported identity_provider type %q", ipm.Type.ValueString())
	}
}

func mapFields(ctx context.Context, instance *workflows.Instance, model *Model, region string) error {
	if instance == nil {
		return errors.New("instance is nil")
	}
	if model == nil {
		return errors.New("model is nil")
	}

	var instanceID string
	switch {
	case model.InstanceID.ValueString() != "":
		instanceID = model.InstanceID.ValueString()
	case instance.Id != "":
		instanceID = instance.Id
	default:
		return errors.New("instance id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), region, instanceID)
	model.InstanceID = types.StringValue(instanceID)
	model.Region = types.StringValue(region)
	model.DisplayName = types.StringValue(instance.DisplayName)
	model.Description = types.StringPointerValue(instance.Description)
	model.Version = types.StringValue(instance.Version)
	// The server returns null for the example-dag flags on every instance we've
	// observed (Airflow 2 and 3). To avoid a perpetual diff against the schema
	// default of `false`, treat null as `false`.
	model.EnableStackitExampleDags = boolOrFalse(instance.EnableStackitExampleDags)
	model.EnableAirflowExampleDags = boolOrFalse(instance.EnableAirflowExampleDags)
	model.ObservabilityID = types.StringPointerValue(instance.ObservabilityId)
	model.Status = types.StringValue(string(instance.Status))
	model.StatusMessage = types.StringPointerValue(instance.StatusMessage)
	model.CreatedAt = types.StringValue(instance.CreatedAt.Format(time.RFC3339))

	if err := mapNetwork(ctx, instance, model); err != nil {
		return fmt.Errorf("mapping network: %w", err)
	}
	if err := mapIdentityProvider(ctx, instance, model); err != nil {
		return fmt.Errorf("mapping identity_provider: %w", err)
	}
	if err := mapEndpoints(ctx, instance, model); err != nil {
		return fmt.Errorf("mapping endpoints: %w", err)
	}

	return nil
}

func mapNetwork(ctx context.Context, instance *workflows.Instance, model *Model) error {
	if instance.Network == nil {
		model.Network = types.ObjectNull(networkTypes)
		return nil
	}
	val, diags := types.ObjectValueFrom(ctx, networkTypes, networkModel{
		ID: types.StringPointerValue(instance.Network.Id),
	})
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Network = val
	return nil
}

// mapIdentityProvider carries the existing client_secret forward — the API
// never returns it. The plan/state value loaded into `model` before mapFields
// is the source of truth for the secret.
func mapIdentityProvider(ctx context.Context, instance *workflows.Instance, model *Model) error {
	existingClientSecret := types.StringNull()
	if !model.IdentityProvider.IsNull() && !model.IdentityProvider.IsUnknown() {
		var prior identityProviderModel
		if diags := model.IdentityProvider.As(ctx, &prior, basetypes.ObjectAsOptions{}); !diags.HasError() {
			existingClientSecret = prior.ClientSecret
		}
	}

	ipm := identityProviderModel{
		APIAudience:  types.SetNull(types.StringType),
		ClientSecret: existingClientSecret,
	}

	switch {
	case instance.IdentityProvider.OAuth2IdentityProvider != nil:
		oauth := instance.IdentityProvider.OAuth2IdentityProvider
		ipm.Type = types.StringValue(string(oauth.Type))
		ipm.Name = types.StringValue(oauth.Name)
		ipm.ClientID = types.StringValue(oauth.ClientId)
		ipm.Scope = types.StringValue(oauth.Scope)
		ipm.DiscoveryEndpoint = types.StringValue(oauth.DiscoveryEndpoint)
		ipm.Resource = types.StringPointerValue(oauth.Resource)
		ipm.RolesClaim = types.StringPointerValue(oauth.RolesClaim)
		// Distinguish nil (API field omitted → null) from [] (empty list).
		if oauth.ApiAudience != nil {
			audience, diags := types.SetValueFrom(ctx, types.StringType, oauth.ApiAudience)
			if diags.HasError() {
				return fmt.Errorf("api_audience: %w", core.DiagsToError(diags))
			}
			ipm.APIAudience = audience
		}
	case instance.IdentityProvider.StackITIdentityProvider != nil:
		// The schema only accepts oauth2 in config; if the server returns a
		// stackit-typed IdP, writing `type = "stackit"` into state would brick
		// every subsequent plan against the OneOf validator. Refuse with an
		// actionable message instead.
		return fmt.Errorf("server returned a STACKIT identity provider, which this provider version does not support; upgrade the provider")
	default:
		return fmt.Errorf("server returned an unknown identity_provider variant; upgrade the provider to a version that supports it")
	}

	val, diags := types.ObjectValueFrom(ctx, identityProviderTypes, ipm)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.IdentityProvider = val
	return nil
}

func mapEndpoints(ctx context.Context, instance *workflows.Instance, model *Model) error {
	val, diags := types.ObjectValueFrom(ctx, endpointsTypes, endpointsModel{
		URL:         types.StringValue(instance.Endpoints.Url),
		RedirectURL: types.StringValue(instance.Endpoints.RedirectUrl),
	})
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Endpoints = val
	return nil
}
