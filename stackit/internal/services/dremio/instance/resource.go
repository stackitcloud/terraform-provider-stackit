package dremio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	dremioSdk "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi"
	dremioWaiter "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi/wait"

	dremioUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dremio/utils"
)

var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
	_ resource.ResourceWithModifyPlan  = &instanceResource{}
)

type Model struct {
	Id types.String `tfsdk:"id"`

	ProjectId  types.String `tfsdk:"project_id"`
	Region     types.String `tfsdk:"region"`
	InstanceId types.String `tfsdk:"instance_id"`

	// Required Fields
	DisplayName types.String `tfsdk:"display_name"`

	// Optional Fields
	Description types.String `tfsdk:"description"`

	// Read-only Fields
	Endpoints types.Object `tfsdk:"endpoints"` // see EndpointsModel
}

// InstanceModel maps the resource schema data.
type InstanceModel struct {
	Model

	Authentication *AuthenticationModel `tfsdk:"authentication"`
	Timeouts       timeouts.Value       `tfsdk:"timeouts"`
}

// AuthenticationModel maps the nested authentication block.
type AuthenticationModel struct {
	// Required Fields
	Type types.String `tfsdk:"type"`

	// Optional Fields
	AzureAD *AzureADModel `tfsdk:"azuread"`
	OAuth   *OAuthModel   `tfsdk:"oauth"`
}

type AzureADModel struct {
	// Required Fields
	AuthorityUrl types.String `tfsdk:"authority_url"`
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`

	RedirectUrl types.String `tfsdk:"redirect_url"`
}

type OAuthModel struct {
	// Required Fields
	AuthorityUrl types.String    `tfsdk:"authority_url"`
	ClientId     types.String    `tfsdk:"client_id"`
	ClientSecret types.String    `tfsdk:"client_secret"`
	JwtClaims    *JwtClaimsModel `tfsdk:"jwt_claims"`

	// Optional Fields
	Scope      types.String         `tfsdk:"scope"`
	Parameters []AuthParameterModel `tfsdk:"parameters"`

	// Read-only Fields
	RedirectUrl types.String `tfsdk:"redirect_url"`
}

type JwtClaimsModel struct {
	// Required Fields
	UserName types.String `tfsdk:"user_name"`
}

type AuthParameterModel struct {
	// Required Fields
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type EndpointsModel struct {
	ArrowFlight types.String `tfsdk:"arrow_flight"`
	Catalog     types.String `tfsdk:"catalog"`
	Ui          types.String `tfsdk:"ui"`
}

var endpointsAttrTypes = map[string]attr.Type{
	"arrow_flight": types.StringType,
	"catalog":      types.StringType,
	"ui":           types.StringType,
}

var descriptions = map[string]string{ //nolint:gosec // no hardcoded credentials in here
	"main":                       "Manages a STACKIT Dremio instance.",
	"id":                         "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`\".",
	"project_id":                 "STACKIT Project ID to which the resource is associated.",
	"instance_id":                "The Dremio instance ID.",
	"region":                     "The STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"display_name":               "The display name is a short name chosen by the user to identify the resource.",
	"description":                "The description is a longer text chosen by the user to provide more context for the resource.",
	"endpoints":                  "The available endpoints of the Dremio instance.",
	"endpoints_arrow_flight":     "The arrow flight endpoint of the Dremio instance.",
	"endpoints_catalog":          "The Apache Iceberg endpoint of the Dremio instance.",
	"endpoints_ui":               "The UI endpoint of the Dremio instance.",
	"authentication":             "Dremio instance authentication settings. A change here triggers a Dremio restart and will incur downtime.",
	"authentication_type":        "Type of authentication (local-only, azuread, oauth).",
	"azuread":                    "Azure Active Directory authentication configuration.",
	"azuread_authority_url":      "The Azure AD authority URL.",
	"azuread_client_id":          "The Azure AD client ID.",
	"azuread_client_secret":      "The Azure AD client secret.",
	"azuread_redirect_url":       "The Azure AD redirect URL.",
	"oauth":                      "OIDC authentication configuration.",
	"oauth_authority_url":        "The Issuer location URI, where the OIDC provider configuration can be found.",
	"oauth_client_id":            "The client ID assigned by the Identity Provider.",
	"oauth_client_secret":        "The client secret generated by the Identity Provider.",
	"oauth_scope":                "A list of space-separated scopes. The `openid` scope is always required; other scopes can vary by provider.",
	"oauth_redirect_url":         "The URL where the Dremio instance is hosted. The URL must match the redirect URL set in the Identity Provider.",
	"oauth_jwt_claims":           "Maps fields from the JWT token to fields Dremio requires.",
	"oauth_jwt_claims_user_name": "Mapped user name claim (e.g. email).",
	"oauth_parameters":           "Any additional parameters the Identity Provider requires.",
	"oauth_parameters_name":      "Parameter name.",
	"oauth_parameters_value":     "Parameter value.",
}

func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

type instanceResource struct {
	client       *dremioSdk.APIClient
	providerData core.ProviderData
}

func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dremio_instance"
}

func (r *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel InstanceModel
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel InstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &providerData, features.DremioExperiment, "stackit_dremio_instance", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := dremioUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Dremio instance client configured")
}

func (r *instanceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: features.AddExperimentDescription(descriptions["main"], features.DremioExperiment, core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Required:    true,
			},
			"authentication": schema.SingleNestedAttribute{
				Description: descriptions["authentication"],
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: descriptions["authentication_type"],
						Required:    true,
					},
					"azuread": schema.SingleNestedAttribute{
						Description: descriptions["azuread"],
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"authority_url": schema.StringAttribute{
								Description: descriptions["azuread_authority_url"],
								Required:    true,
							},
							"client_id": schema.StringAttribute{
								Description: descriptions["azuread_client_id"],
								Required:    true,
							},
							"client_secret": schema.StringAttribute{
								Description: descriptions["azuread_client_secret"],
								Required:    true,
								Sensitive:   true,
							},
							"redirect_url": schema.StringAttribute{
								Description: descriptions["azuread_redirect_url"],
								Computed:    true,
							},
						},
					},
					"oauth": schema.SingleNestedAttribute{
						Description: descriptions["oauth"],
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"authority_url": schema.StringAttribute{
								Description: descriptions["oauth_authority_url"],
								Required:    true,
							},
							"client_id": schema.StringAttribute{
								Description: descriptions["oauth_client_id"],
								Required:    true,
							},
							"client_secret": schema.StringAttribute{
								Description: descriptions["oauth_client_secret"],
								Required:    true,
								Sensitive:   true,
							},
							"jwt_claims": schema.SingleNestedAttribute{
								Description: descriptions["oauth_jwt_claims"],
								Required:    true,
								Attributes: map[string]schema.Attribute{
									"user_name": schema.StringAttribute{
										Description: descriptions["oauth_jwt_claims_user_name"],
										Required:    true,
									},
								},
							},
							"parameters": schema.ListNestedAttribute{
								Description: descriptions["oauth_parameters"],
								Optional:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"name": schema.StringAttribute{
											Description: descriptions["oauth_parameters_name"],
											Required:    true,
										},
										"value": schema.StringAttribute{
											Description: descriptions["oauth_parameters_value"],
											Required:    true,
										},
									},
								},
							},
							"redirect_url": schema.StringAttribute{
								Description: descriptions["oauth_redirect_url"],
								Computed:    true,
							},
							"scope": schema.StringAttribute{
								Description: descriptions["oauth_scope"],
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
				},
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Computed:    true, // Must be computed if a default is applied
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"endpoints": schema.SingleNestedAttribute{
				Description: descriptions["endpoints"],
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"arrow_flight": schema.StringAttribute{
						Description: descriptions["endpoints_arrow_flight"],
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"catalog": schema.StringAttribute{
						Description: descriptions["endpoints_catalog"],
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"ui": schema.StringAttribute{
						Description: descriptions["endpoints_ui"],
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"timeouts": timeouts.AttributesAll(ctx),
		},
	}
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model InstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	waiterTimeout := dremioWaiter.CreateDremioWaitHandler(ctx, r.client.DefaultAPI, "", "", "").GetTimeout()
	createTimeout, diags := model.Timeouts.Create(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// prepare the payload struct for the create instance request
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new Dremio instance
	instanceResp, err := r.client.DefaultAPI.CreateDremioInstance(ctx, projectId, region).CreateDremioInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if instanceResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Empty response", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceResp.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	_, err = dremioWaiter.CreateDremioWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Dremio instance", fmt.Sprintf("Dremio instance creation waiting: %v", err))
		return
	}

	err = mapFields(instanceResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Dremio instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Dremio instance created")
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model InstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	if instanceId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	instanceResp, err := r.client.DefaultAPI.GetDremioInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading dremio instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(instanceResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading dremio instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Dremio instance read")
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model, state InstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	waiterTimeout := dremioWaiter.UpdateDremioWaitHandler(ctx, r.client.DefaultAPI, "", "", "").GetTimeout()
	updateTimeout, diags := model.Timeouts.Update(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	instanceId := state.InstanceId.ValueString()
	instanceResp, err := r.client.DefaultAPI.UpdateDremioInstance(ctx, projectId, region, instanceId).UpdateDremioInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if instanceResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Empty response", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceResp.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	_, err = dremioWaiter.UpdateDremioWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Dremio instance", fmt.Sprintf("Dremio instance updating waiting: %v", err))
		return
	}

	err = mapFields(instanceResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Dremio instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Dremio instance updated")
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model InstanceModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	waiterTimeout := dremioWaiter.DeleteDremioWaitHandler(ctx, r.client.DefaultAPI, "", "", "").GetTimeout()
	deleteTimeout, diags := model.Timeouts.Delete(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	err := r.client.DefaultAPI.DeleteDremioInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Dremio instance", fmt.Sprintf("Calling API: %v", err))
	}

	ctx = core.LogResponse(ctx)

	_, err = dremioWaiter.DeleteDremioWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Dremio instance", fmt.Sprintf("Dremio instance deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Dremio instance deleted")
}

func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing dremio instance",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[instance_id], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"region":      idParts[1],
		"instance_id": idParts[2],
	})

	tflog.Info(ctx, "Dremio instance state imported")
}

func mapFields(instanceResp *dremioSdk.DremioResponse, model *InstanceModel, region string) error {
	if instanceResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	err := mapModelFields(instanceResp, &model.Model, region)
	if err != nil {
		return fmt.Errorf("failed to map Model fields")
	}

	if model.Authentication == nil {
		model.Authentication = new(AuthenticationModel)
	}
	err = mapAuthentication(instanceResp, model.Authentication)
	if err != nil {
		return fmt.Errorf("failed to map Authentication fields")
	}

	return nil
}

// Maps instance fields to the provider's internal model
func mapModelFields(instanceResp *dremioSdk.DremioResponse, model *Model, region string) error {
	if instanceResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.InstanceId = types.StringValue(instanceResp.Id)

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		model.InstanceId.ValueString(),
	)

	model.DisplayName = types.StringValue(instanceResp.DisplayName)
	model.Description = types.StringPointerValue(instanceResp.Description)

	endpoints := &EndpointsModel{
		ArrowFlight: types.StringValue(instanceResp.Endpoints.ArrowFlight),
		Catalog:     types.StringValue(instanceResp.Endpoints.Catalog),
		Ui:          types.StringValue(instanceResp.Endpoints.Ui),
	}
	endpointsObj, diags := types.ObjectValueFrom(context.Background(), endpointsAttrTypes, endpoints)
	if diags.HasError() {
		return fmt.Errorf("failed to parse endpoints")
	}
	model.Endpoints = endpointsObj

	return nil
}

func mapAuthentication(instanceResp *dremioSdk.DremioResponse, auth *AuthenticationModel) error {
	if instanceResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if auth == nil {
		return fmt.Errorf("auth input is nil")
	}

	auth.Type = types.StringValue(string(instanceResp.Authentication.Type))

	if instanceResp.Authentication.Azuread != nil {
		if auth.AzureAD == nil {
			auth.AzureAD = new(AzureADModel)
		}

		azureADResp := instanceResp.Authentication.Azuread
		azureADModel := auth.AzureAD

		azureADModel.AuthorityUrl = types.StringValue(azureADResp.AuthorityUrl)
		azureADModel.ClientId = types.StringValue(azureADResp.ClientId)
		azureADModel.RedirectUrl = types.StringPointerValue(azureADResp.RedirectUrl)
	}

	if instanceResp.Authentication.Oauth != nil {
		if auth.OAuth == nil {
			auth.OAuth = new(OAuthModel)
		}
		oauthResp := instanceResp.Authentication.Oauth
		oauthModel := auth.OAuth

		oauthModel.AuthorityUrl = types.StringValue(oauthResp.AuthorityUrl)
		oauthModel.ClientId = types.StringValue(oauthResp.ClientId)
		oauthModel.Scope = types.StringPointerValue(oauthResp.Scope)
		oauthModel.RedirectUrl = types.StringPointerValue(oauthResp.RedirectUrl)
		oauthModel.JwtClaims = &JwtClaimsModel{UserName: types.StringValue(oauthResp.JwtClaims.UserName)}

		if len(oauthResp.Parameters) > 0 {
			var params []AuthParameterModel
			for _, p := range oauthResp.Parameters {
				params = append(params, AuthParameterModel{
					Name:  types.StringValue(p.Name),
					Value: types.StringValue(p.Value),
				})
			}
			oauthModel.Parameters = params
		}
	}

	return nil
}

// Build UpdateDremioInstancePayload from provider's model
func toUpdatePayload(model *InstanceModel) (*dremioSdk.UpdateDremioInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	authentication, err := parseAuthentication(model)
	if err != nil {
		return nil, fmt.Errorf("failed to parse authentication: %w", err)
	}

	return &dremioSdk.UpdateDremioInstancePayload{
		Authentication: authentication,
		Description:    model.Description.ValueStringPointer(),
		DisplayName:    model.DisplayName.ValueStringPointer(),
	}, nil
}

// Build CreateDremioInstancePayload from provider's model
func toCreatePayload(model *InstanceModel) (*dremioSdk.CreateDremioInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	authentication, err := parseAuthentication(model)
	if err != nil {
		return nil, fmt.Errorf("failed to parse authentication: %w", err)
	}

	return &dremioSdk.CreateDremioInstancePayload{
		Authentication: authentication,
		Description:    model.Description.ValueStringPointer(),
		DisplayName:    model.DisplayName.ValueString(),
	}, nil
}

func parseAuthentication(model *InstanceModel) (*dremioSdk.Authentication, error) {
	// API only saves the block of the stated type. The other one is omitted.
	// Keeping the block in TF leads to inconsistent state. Therefore we have
	// make sure the type matches the existing block.

	switch model.Authentication.Type.ValueString() {
	case string(dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY):
		if !(model.Authentication.OAuth == nil) || !(model.Authentication.AzureAD == nil) {
			return nil, fmt.Errorf("can't state idp config if auth type is %q", dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY)
		}
		return &dremioSdk.Authentication{
			Azuread: nil,
			Oauth:   nil,
			Type:    dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY,
		}, nil
	case string(dremioSdk.AUTHENTICATIONTYPE_OAUTH):
		if !(model.Authentication.AzureAD == nil) {
			return nil, fmt.Errorf("can't state azure idp config if auth type is %q", dremioSdk.AUTHENTICATIONTYPE_OAUTH)
		}
		if model.Authentication.OAuth == nil {
			return nil, fmt.Errorf("missing oauth idp config")
		}

		oAuthParams := []dremioSdk.AuthParameters{}
		if len(model.Authentication.OAuth.Parameters) > 0 {
			parameters := model.Authentication.OAuth.Parameters
			for _, param := range parameters {
				oAuthParams = append(oAuthParams, dremioSdk.AuthParameters{
					Name:  param.Name.ValueString(),
					Value: param.Value.ValueString(),
				})
			}
		}

		oAuthPayload := &dremioSdk.Oauth{
			AuthorityUrl: model.Authentication.OAuth.AuthorityUrl.ValueString(),
			ClientId:     model.Authentication.OAuth.ClientId.ValueString(),
			ClientSecret: model.Authentication.OAuth.ClientSecret.ValueStringPointer(),
			JwtClaims: dremioSdk.OauthJwtClaims{
				UserName: model.Authentication.OAuth.JwtClaims.UserName.ValueString(),
			},
			RedirectUrl: model.Authentication.OAuth.RedirectUrl.ValueStringPointer(),
			Scope:       model.Authentication.OAuth.Scope.ValueStringPointer(),
			Parameters:  oAuthParams,
		}

		return &dremioSdk.Authentication{
			Azuread: nil,
			Oauth:   oAuthPayload,
			Type:    dremioSdk.AUTHENTICATIONTYPE_OAUTH,
		}, nil
	case string(dremioSdk.AUTHENTICATIONTYPE_AZUREAD):
		if !(model.Authentication.OAuth == nil) {
			return nil, fmt.Errorf("can't state oauth idp config if auth type is %q", dremioSdk.AUTHENTICATIONTYPE_AZUREAD)
		}
		if model.Authentication.AzureAD == nil {
			return nil, fmt.Errorf("missing azuread config")
		}

		azureAdPayload := &dremioSdk.Azuread{
			AuthorityUrl: model.Authentication.AzureAD.AuthorityUrl.ValueString(),
			ClientId:     model.Authentication.AzureAD.ClientId.ValueString(),
			ClientSecret: model.Authentication.AzureAD.ClientSecret.ValueStringPointer(),
			RedirectUrl:  model.Authentication.AzureAD.RedirectUrl.ValueStringPointer(),
		}
		return &dremioSdk.Authentication{
			Azuread: azureAdPayload,
			Oauth:   nil,
			Type:    dremioSdk.AUTHENTICATIONTYPE_AZUREAD,
		}, nil
	default:
		return nil, fmt.Errorf("unknown authentication type: %s", model.Authentication.Type)
	}
}
