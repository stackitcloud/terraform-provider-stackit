package sca

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sca "github.com/stackitcloud/stackit-sdk-go/services/sca/v1alphaapi"
	scaWait "github.com/stackitcloud/stackit-sdk-go/services/sca/v1alphaapi/wait"
)

var (
	_ resource.Resource                   = &applicationResource{}
	_ resource.ResourceWithConfigure      = &applicationResource{}
	_ resource.ResourceWithImportState    = &applicationResource{}
	_ resource.ResourceWithValidateConfig = &applicationResource{}
	_ resource.ResourceWithModifyPlan     = &applicationResource{}
)

func NewApplicationResource() resource.Resource {
	return &applicationResource{}
}

type applicationResource struct {
	client       sca.DefaultAPI
	providerData core.ProviderData
}

type ResourceModel struct {
	Model
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *applicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sca_application"
}

func (r *applicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.providerData = providerData
	r.client = clients.ScaV1AlphaClient

	tflog.Info(ctx, "SCA application client configured")
}

func (r *applicationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	if req.Config.Raw.IsNull() {
		return
	}

	var configModel ResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// if public ingress is flipped, urls may change
	if !req.State.Raw.IsNull() {
		var stateModel ResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
		if !resp.Diagnostics.HasError() {
			planEnabled := isPublicIngressEnabled(ctx, planModel.Network, &resp.Diagnostics)
			stateEnabled := isPublicIngressEnabled(ctx, stateModel.Network, &resp.Diagnostics)
			if planEnabled != stateEnabled {
				planModel.Urls = types.ListUnknown(types.StringType)
			}
		}
	}

	// region
	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
}

func (r *applicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: descriptionResource,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptionId,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Description: descriptionApplicationId,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptionProjectId,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"environment_id": schema.StringAttribute{
				Description: descriptionEnvironmentId,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: descriptionDisplayName,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"stopped": schema.BoolAttribute{
				Description: descriptionStopped,
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"containers": schema.ListNestedAttribute{
				Description: descriptionContainers,
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: descriptionContainerName,
							Required:    true,
						},
						"image": schema.StringAttribute{
							Description: descriptionContainerImage,
							Required:    true,
						},
						"cpu_millis": schema.Int32Attribute{
							Description: descriptionContainerCpu,
							Required:    true,
						},
						"memory_mb": schema.Int32Attribute{
							Description: descriptionContainerMemory,
							Required:    true,
						},
						"command": schema.ListAttribute{
							Description: descriptionContainerCommand,
							Optional:    true,
							ElementType: types.StringType,
						},
						"args": schema.ListAttribute{
							Description: descriptionContainerArgs,
							Optional:    true,
							ElementType: types.StringType,
						},
						"env": schema.SingleNestedAttribute{
							Description: descriptionContainerEnv,
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"from_value": schema.MapAttribute{
									Description: descriptionEnvFromValue,
									Optional:    true,
									ElementType: types.StringType,
								},
								"from_secret_ref": schema.MapAttribute{
									Description: descriptionEnvFromSecret,
									Optional:    true,
									ElementType: types.StringType,
								},
							},
						},
					},
				},
			},
			"network": schema.SingleNestedAttribute{
				Description: descriptionNetwork,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"public": schema.SingleNestedAttribute{
						Description: descriptionNetworkPublic,
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: descriptionNetworkPublicEnabled,
								Required:    true,
							},
							"port": schema.Int32Attribute{
								Description: descriptionNetworkPublicPort,
								Required:    true,
							},
							"acl": schema.ListAttribute{
								Description: descriptionNetworkPublicAcl,
								Optional:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
			},
			"scaling": schema.SingleNestedAttribute{
				Description: descriptionScaling,
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"manual": schema.SingleNestedAttribute{
						Description: descriptionScalingManual,
						Optional:    true,
						Validators: []validator.Object{
							objectvalidator.ExactlyOneOf(
								path.MatchRelative().AtParent().AtName("auto"),
							),
						},
						Attributes: map[string]schema.Attribute{
							"instances": schema.Int32Attribute{
								Description: descriptionScalingInstances,
								Required:    true,
							},
						},
					},
					"auto": schema.SingleNestedAttribute{
						Description: descriptionScalingAuto,
						Optional:    true,
						Validators: []validator.Object{
							objectvalidator.ExactlyOneOf(
								path.MatchRelative().AtParent().AtName("manual"),
							),
						},
						Attributes: map[string]schema.Attribute{
							"min_instances": schema.Int32Attribute{
								Description: descriptionScalingMinInstances,
								Required:    true,
							},
							"max_instances": schema.Int32Attribute{
								Description: descriptionScalingMaxInstances,
								Required:    true,
							},
							"allow_scale_to_zero": schema.BoolAttribute{
								Description: descriptionScalingScaleToZero,
								Optional:    true,
							},
							"http_rule": schema.SingleNestedAttribute{
								Description: descriptionScalingHttpRule,
								Optional:    true,
								Validators: []validator.Object{
									objectvalidator.AtLeastOneOf(
										path.MatchRelative().AtParent().AtName("native_rules"),
									),
								},
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Description: descriptionRuleName,
										Required:    true,
									},
									"concurrency": schema.Int32Attribute{
										Description: descriptionRuleConcurrency,
										Optional:    true,
										Validators: []validator.Int32{
											int32validator.AtLeastOneOf(
												path.MatchRelative().AtParent().AtName("rps"),
											),
										},
									},
									"rps": schema.Int32Attribute{
										Description: descriptionRuleRps,
										Optional:    true,
										Validators: []validator.Int32{
											int32validator.AtLeastOneOf(
												path.MatchRelative().AtParent().AtName("concurrency"),
											),
										},
									},
								},
							},
							"native_rules": schema.ListNestedAttribute{
								Description: descriptionScalingNativeRules,
								Optional:    true,
								Validators: []validator.List{
									listvalidator.AtLeastOneOf(
										path.MatchRelative().AtParent().AtName("http_rule"),
									),
								},
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"name": schema.StringAttribute{
											Description: descriptionRuleName,
											Required:    true,
										},
										"trigger": schema.StringAttribute{
											Description: descriptionRuleTrigger,
											Required:    true,
										},
										"parameters": schema.MapAttribute{
											Description: descriptionRuleParameters,
											Required:    true,
											ElementType: types.StringType,
										},
										"secrets_mapping": schema.MapAttribute{
											Description: descriptionRuleSecretsMapping,
											Optional:    true,
											ElementType: types.StringType,
										},
									},
								},
							},
						},
					},
				},
			},
			"region": schema.StringAttribute{
				Description: descriptionRegion,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"urls": schema.ListAttribute{
				Description: descriptionUrls,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: descriptionStatus,
				Computed:    true,
			},
			"timeouts": timeouts.AttributesAll(ctx),
		},
	}
}

func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id,region,environment_id,application_id. Got: %q", req.ID),
		)
		return
	}
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":     idParts[0],
		"region":         idParts[1],
		"environment_id": idParts[2],
		"application_id": idParts[3],
	})
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "SCA application state imported")
}

func (r *applicationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model ResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if model.Containers.IsNull() || model.Containers.IsUnknown() {
		return
	}

	var containers []containerModel
	model.Containers.ElementsAs(ctx, &containers, false)
	for i, c := range containers {
		if c.Env.IsNull() || c.Env.IsUnknown() {
			continue
		}

		var envData envModel
		c.Env.As(ctx, &envData, basetypes.ObjectAsOptions{})

		if envData.FromValue.IsNull() || envData.FromSecretRef.IsNull() {
			continue
		}

		for key := range envData.FromValue.Elements() {
			if _, exists := envData.FromSecretRef.Elements()[key]; exists {
				resp.Diagnostics.AddError(
					"Overlapping Environment Variables",
					fmt.Sprintf("Container at index %d defines %q in both 'from_value' and 'from_secret_ref'.", i, key),
				)
			}
		}
	}
}

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model ResourceModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := model.Timeouts.Create(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	environmentId := model.EnvironmentId.ValueString()
	region := model.Region.ValueString()

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating application", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	ctx = core.InitProviderContext(ctx)

	appResp, err := r.client.CreateApplication(ctx, projectId, environmentId).CreateApplicationPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating application", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id":     projectId,
		"region":         region,
		"environment_id": environmentId,
		"application_id": *appResp.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := scaWait.CreateApplicationWaitHandler(ctx, r.client, projectId, environmentId, *appResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating application", fmt.Sprintf("Application creation waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating application", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "SCA application created")
}

func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var state ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	projectId := state.ProjectId.ValueString()
	environmentId := state.EnvironmentId.ValueString()
	appId := state.ApplicationId.ValueString()
	region := r.providerData.GetRegionWithOverride(state.Region)

	ctx = core.InitProviderContext(ctx)

	appResp, err := r.client.GetApplication(ctx, projectId, environmentId, appId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading application", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, appResp, &state, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading application", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "SCA application read")
}

func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model ResourceModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	environmentId := model.EnvironmentId.ValueString()
	region := model.Region.ValueString()
	appId := model.ApplicationId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "environment_id", environmentId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "application_id", appId)

	patchPayload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating application", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	_, err = r.client.UpdateApplication(ctx, projectId, environmentId, appId).UpdateApplicationPayload(*patchPayload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating application", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "Triggered update application")

	waitResp, err := scaWait.UpdateApplicationWaitHandler(ctx, r.client, projectId, environmentId, appId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating application", fmt.Sprintf("Application update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating application", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var state ResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	projectId := state.ProjectId.ValueString()
	environmentId := state.EnvironmentId.ValueString()
	appId := state.ApplicationId.ValueString()

	ctx = core.InitProviderContext(ctx)

	_, err := r.client.DeleteApplication(ctx, projectId, environmentId, appId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting application", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	_, err = scaWait.DeleteApplicationWaitHandler(ctx, r.client, projectId, environmentId, appId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting application", fmt.Sprintf("Application deletion waiting: %v", err))
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "SCA application deleted")
}

func mapFields(ctx context.Context, app *sca.Application, m *ResourceModel, region string) error {
	if app == nil {
		return fmt.Errorf("response input is nil")
	}
	if m == nil {
		return fmt.Errorf("model input is nil")
	}

	var appId string
	if m.ApplicationId.ValueString() != "" {
		appId = m.ApplicationId.ValueString()
	} else if app.Id != nil {
		appId = *app.Id
	} else {
		return fmt.Errorf("application id not present")
	}

	m.Id = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), region, m.EnvironmentId.ValueString(), appId)
	m.ApplicationId = types.StringValue(*app.Id)
	m.DisplayName = types.StringValue(app.DisplayName)
	m.Urls = types.ListNull(types.StringType)
	m.Status = types.StringNull()

	if app.Stopped != nil {
		m.Stopped = types.BoolValue(*app.Stopped)
	}

	if err := mapContainers(ctx, app.Containers, m); err != nil {
		return fmt.Errorf("mapping containers: %w", err)
	}

	if err := mapNetwork(ctx, app.Network, m); err != nil {
		return fmt.Errorf("mapping network: %w", err)
	}

	if err := mapScaling(ctx, app.Scaling, m); err != nil {
		return fmt.Errorf("mapping scaling: %w", err)
	}

	if err := mapRuntimeStatus(ctx, app.RuntimeStatus, m); err != nil {
		return fmt.Errorf("mapping runtime status: %w", err)
	}

	return nil
}

func mapContainers(ctx context.Context, apiContainers []sca.Container, m *ResourceModel) error {
	var tfContainers []attr.Value

	for _, c := range apiContainers {
		// Map Command and Args lists (or null lists)
		var diags diag.Diagnostics
		cmdList := types.ListNull(types.StringType)
		if len(c.Command) > 0 {
			cmdList, diags = types.ListValueFrom(ctx, types.StringType, c.Command)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}
		argsList := types.ListNull(types.StringType)
		if len(c.Args) > 0 {
			argsList, diags = types.ListValueFrom(ctx, types.StringType, c.Args)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}

		// Map Environments variables by type (or null obj)
		envObj := types.ObjectNull(envTypes)
		if len(c.EnvironmentVariables) > 0 {
			plaintextEnv := make(map[string]string)
			secretEnv := make(map[string]string)

			for _, env := range c.EnvironmentVariables {
				if env.Origin == nil {
					continue
				}
				switch string(*env.Origin) {
				case string(sca.ENVVARTYPE_ENV_FROM_SOURCE_TYPE_MANUAL):
					plaintextEnv[env.Key] = env.Value
				case string(sca.ENVVARTYPE_ENV_FROM_SOURCE_TYPE_SECRET):
					secretEnv[env.Key] = env.Value
				default:
					tflog.Warn(ctx, "Encountered unknown environment variable origin type from API", map[string]any{
						"key":  env.Key,
						"type": string(*env.Origin),
					})
				}
			}

			plainTextMap := types.MapNull(types.StringType)
			if len(plaintextEnv) > 0 {
				plainTextMap, diags = types.MapValueFrom(ctx, types.StringType, plaintextEnv)
				if diags.HasError() {
					return core.DiagsToError(diags)
				}
			}

			secretEnvMap := types.MapNull(types.StringType)
			if len(secretEnv) > 0 {
				secretEnvMap, diags = types.MapValueFrom(ctx, types.StringType, secretEnv)
				if diags.HasError() {
					return core.DiagsToError(diags)
				}
			}

			envObj, diags = types.ObjectValue(envTypes, map[string]attr.Value{
				"from_value":      plainTextMap,
				"from_secret_ref": secretEnvMap,
			})
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}

		// Build container model (simple fields)
		cModel := containerModel{
			Name:      types.StringValue(c.Name),
			Image:     types.StringValue(c.Image),
			CPUMillis: types.Int32PointerValue(c.Cpu),
			MemoryMB:  types.Int32PointerValue(c.Memory),
			Command:   cmdList,
			Args:      argsList,
			Env:       envObj,
		}
		tfContainerObj, diags := types.ObjectValueFrom(ctx, containerTypes, cModel)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
		tfContainers = append(tfContainers, tfContainerObj)
	}

	// Create the final list of containers
	listVal, diags := types.ListValue(types.ObjectType{AttrTypes: containerTypes}, tfContainers)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	m.Containers = listVal
	return nil
}

func mapNetwork(ctx context.Context, net sca.Network, m *ResourceModel) error {
	networkValues := map[string]attr.Value{
		"public": types.ObjectNull(publicNetworkTypes),
	}

	// If public ingress is enabled
	if net.PublicIngress {
		var diags diag.Diagnostics
		aclList := types.ListNull(types.StringType)
		if len(net.IngressAcl) > 0 {
			aclList, diags = types.ListValueFrom(ctx, types.StringType, net.IngressAcl)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}

		publicModel := publicNetworkModel{
			Enabled: types.BoolValue(net.PublicIngress),
			Port:    types.Int32PointerValue(net.Port),
			ACL:     aclList,
		}
		publicObj, diags := types.ObjectValueFrom(ctx, publicNetworkTypes, publicModel)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
		networkValues["public"] = publicObj
	}

	netObj, diags := types.ObjectValue(networkTypes, networkValues)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	m.Network = netObj
	return nil
}

func mapScaling(ctx context.Context, apiScaling sca.Scaling, m *ResourceModel) error {
	scalingValues := map[string]attr.Value{
		"manual": types.ObjectNull(manualScalingTypes),
		"auto":   types.ObjectNull(autoScalingTypes),
	}

	if apiScaling.Type == sca.SCALINGTYPE_SCALING_TYPE_MANUAL && apiScaling.ManualScaling != nil {
		manualModel := manualScalingModel{
			Instances: types.Int32Value(apiScaling.ManualScaling.Instances),
		}
		manualObj, diags := types.ObjectValueFrom(ctx, manualScalingTypes, manualModel)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
		scalingValues["manual"] = manualObj
	} else if apiScaling.Type == sca.SCALINGTYPE_SCALING_TYPE_AUTO && apiScaling.AutoScaling != nil {
		apiAuto := apiScaling.AutoScaling

		httpRuleVal := types.ObjectNull(httpRuleTypes)
		var tfNativeRules []attr.Value
		for _, apiRule := range apiAuto.Rules {
			switch apiRule.Type {
			case sca.RULETYPE_RULE_TYPE_HTTP:
				if apiRule.HttpRule == nil {
					continue
				}

				// concurrency and rps have 0 as default/disabled in the API
				concurrency := types.Int32Null()
				if apiRule.HttpRule.Concurrency != nil && *apiRule.HttpRule.Concurrency > 0 {
					concurrency = types.Int32Value(*apiRule.HttpRule.Concurrency)
				}
				rps := types.Int32Null()
				if apiRule.HttpRule.Rps != nil && *apiRule.HttpRule.Rps > 0 {
					rps = types.Int32Value(*apiRule.HttpRule.Rps)
				}

				httpModel := httpRuleModel{
					Name:        types.StringValue(apiRule.Name),
					Concurrency: concurrency,
					Rps:         rps,
				}
				obj, diags := types.ObjectValueFrom(ctx, httpRuleTypes, httpModel)
				if diags.HasError() {
					return core.DiagsToError(diags)
				}
				httpRuleVal = obj
			case sca.RULETYPE_RULE_TYPE_CUSTOM:
				if apiRule.CustomRule == nil {
					continue
				}

				paramsMap := make(map[string]string)
				for _, p := range apiRule.CustomRule.Parameters {
					paramsMap[p.Name] = p.Value
				}
				tfParams, diags := types.MapValueFrom(ctx, types.StringType, paramsMap)
				if diags.HasError() {
					return core.DiagsToError(diags)
				}
				secretsMap := make(map[string]string)
				for _, s := range apiRule.CustomRule.SecretsMapping {
					secretsMap[s.Parameter] = s.Secret
				}
				tfSecrets, diags := types.MapValueFrom(ctx, types.StringType, secretsMap)
				if diags.HasError() {
					return core.DiagsToError(diags)
				}

				nativeModel := nativeRuleModel{
					Name:           types.StringValue(apiRule.Name),
					Trigger:        types.StringValue(apiRule.CustomRule.Type),
					Parameters:     tfParams,
					SecretsMapping: tfSecrets,
				}
				obj, diags := types.ObjectValueFrom(ctx, nativeRuleTypes, nativeModel)
				if diags.HasError() {
					return core.DiagsToError(diags)
				}
				tfNativeRules = append(tfNativeRules, obj)
			default:
				tflog.Warn(ctx, "Encountered unknown scaling rule type from API", map[string]any{
					"name": apiRule.Name,
					"type": string(apiRule.Type),
				})
			}
		}
		nativeRulesList := types.ListNull(types.ObjectType{AttrTypes: nativeRuleTypes})
		if len(tfNativeRules) > 0 {
			var diags diag.Diagnostics
			nativeRulesList, diags = types.ListValue(types.ObjectType{AttrTypes: nativeRuleTypes}, tfNativeRules)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}

		autoModel := autoScalingModel{
			MinInstances:     types.Int32Value(apiAuto.MinInstances),
			MaxInstances:     types.Int32Value(apiAuto.MaxInstances),
			AllowScaleToZero: types.BoolPointerValue(apiAuto.AllowScaleToZero),
			HTTPRule:         httpRuleVal,
			NativeRules:      nativeRulesList,
		}

		autoObj, diags := types.ObjectValueFrom(ctx, autoScalingTypes, autoModel)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
		scalingValues["auto"] = autoObj
	}

	scaleObj, diags := types.ObjectValue(scalingTypes, scalingValues)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	m.Scaling = scaleObj
	return nil
}

func mapRuntimeStatus(ctx context.Context, apiRuntimeStatus *sca.RuntimeStatus, m *ResourceModel) error {
	if apiRuntimeStatus == nil {
		return nil
	}

	if apiRuntimeStatus.CurrentStatus != nil {
		m.Status = types.StringValue(string(*apiRuntimeStatus.CurrentStatus))
	}

	if len(apiRuntimeStatus.Urls) > 0 {
		urlsList, diags := types.ListValueFrom(ctx, types.StringType, apiRuntimeStatus.Urls)
		if diags.HasError() {
			return fmt.Errorf("mapping urls: %w", core.DiagsToError(diags))
		}
		m.Urls = urlsList
	}
	return nil
}

func toCreatePayload(ctx context.Context, m *ResourceModel) (*sca.CreateApplicationPayload, error) {
	containers, err := toAPIContainers(ctx, m.Containers)
	if err != nil {
		return nil, err
	}

	network, err := toAPINetwork(ctx, m.Network)
	if err != nil {
		return nil, err
	}

	scaling, err := toAPIScaling(ctx, m.Scaling)
	if err != nil {
		return nil, err
	}

	payload := sca.CreateApplicationPayload{
		DisplayName: m.DisplayName.ValueString(),
		Containers:  containers,
	}

	if network != nil {
		payload.Network = *network
	}
	if scaling != nil {
		payload.Scaling = *scaling
	}

	if !m.Stopped.IsNull() && !m.Stopped.IsUnknown() {
		payload.Stopped = conversion.BoolValueToPointer(m.Stopped)
	}

	return &payload, nil
}

func toUpdatePayload(ctx context.Context, m *ResourceModel) (*sca.UpdateApplicationPayload, error) {
	containers, err := toAPIContainers(ctx, m.Containers)
	if err != nil {
		return nil, err
	}

	network, err := toAPINetwork(ctx, m.Network)
	if err != nil {
		return nil, err
	}

	scaling, err := toAPIScaling(ctx, m.Scaling)
	if err != nil {
		return nil, err
	}

	payload := sca.UpdateApplicationPayload{
		Containers: containers,
		Network:    network,
		Scaling:    scaling,
	}

	if !m.Stopped.IsNull() && !m.Stopped.IsUnknown() {
		payload.Stopped = conversion.BoolValueToPointer(m.Stopped)
	}

	return &payload, nil
}

func toAPIContainers(ctx context.Context, list types.List) ([]sca.Container, error) {
	if list.IsNull() || list.IsUnknown() {
		return nil, fmt.Errorf("containers list cannot be null or unknown")
	}

	var tfContainers []containerModel
	if diags := list.ElementsAs(ctx, &tfContainers, false); diags.HasError() {
		return nil, fmt.Errorf("parsing containers: %v", diags.Errors())
	}

	var apiContainers []sca.Container
	for _, c := range tfContainers {
		apiContainer := sca.Container{
			Name:   c.Name.ValueString(),
			Image:  c.Image.ValueString(),
			Cpu:    conversion.Int32ValueToPointer(c.CPUMillis),
			Memory: conversion.Int32ValueToPointer(c.MemoryMB),
		}

		if !c.Command.IsNull() {
			var cmd []string
			c.Command.ElementsAs(ctx, &cmd, false)
			apiContainer.Command = cmd
		}

		if !c.Args.IsNull() {
			var args []string
			c.Args.ElementsAs(ctx, &args, false)
			apiContainer.Args = args
		}

		envs, err := toAPIEnv(ctx, c.Env)
		if err != nil {
			return nil, err
		}
		apiContainer.EnvironmentVariables = envs

		apiContainers = append(apiContainers, apiContainer)
	}
	return apiContainers, nil
}

func toAPIEnv(ctx context.Context, envObj types.Object) ([]sca.EnvVar, error) {
	if envObj.IsNull() || envObj.IsUnknown() {
		return nil, nil
	}

	var envData envModel
	if diags := envObj.As(ctx, &envData, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil, fmt.Errorf("parsing env block: %v", diags.Errors())
	}

	var apiEnvs []sca.EnvVar
	if !envData.FromValue.IsNull() {
		for k, v := range envData.FromValue.Elements() {
			if strVal, ok := v.(types.String); ok {
				apiEnvs = append(apiEnvs, sca.EnvVar{
					Key:    k,
					Value:  strVal.ValueString(),
					Origin: new(sca.ENVVARTYPE_ENV_FROM_SOURCE_TYPE_MANUAL),
				})
			}
		}
	}
	if !envData.FromSecretRef.IsNull() {
		for k, v := range envData.FromSecretRef.Elements() {
			if strVal, ok := v.(types.String); ok {
				apiEnvs = append(apiEnvs, sca.EnvVar{
					Key:    k,
					Value:  strVal.ValueString(),
					Origin: new(sca.ENVVARTYPE_ENV_FROM_SOURCE_TYPE_SECRET),
				})
			}
		}
	}

	return apiEnvs, nil
}

func toAPINetwork(ctx context.Context, obj types.Object) (*sca.Network, error) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, nil
	}

	var netTF networkModel
	if diags := obj.As(ctx, &netTF, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil, fmt.Errorf("parsing network: %v", diags.Errors())
	}

	apiNetwork := sca.Network{}

	if !netTF.Public.IsNull() && !netTF.Public.IsUnknown() {
		var publicAttrs struct {
			Enabled types.Bool  `tfsdk:"enabled"`
			Port    types.Int32 `tfsdk:"port"`
			Acl     types.List  `tfsdk:"acl"`
		}
		if diags := netTF.Public.As(ctx, &publicAttrs, basetypes.ObjectAsOptions{}); diags.HasError() {
			return nil, fmt.Errorf("parsing public network: %v", diags.Errors())
		}

		apiNetwork.PublicIngress = publicAttrs.Enabled.ValueBool()
		if !publicAttrs.Port.IsNull() {
			apiNetwork.Port = conversion.Int32ValueToPointer(publicAttrs.Port)
		}
		if !publicAttrs.Acl.IsNull() && !publicAttrs.Acl.IsUnknown() {
			var acl []string
			publicAttrs.Acl.ElementsAs(ctx, &acl, false)
			apiNetwork.IngressAcl = acl
		}
	}

	return &apiNetwork, nil
}

func toAPIScaling(ctx context.Context, obj types.Object) (*sca.Scaling, error) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, nil
	}

	var scaleTF scalingModel
	if diags := obj.As(ctx, &scaleTF, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil, fmt.Errorf("parsing scaling: %v", diags.Errors())
	}

	apiScaling := sca.Scaling{}

	if !scaleTF.Manual.IsNull() && !scaleTF.Manual.IsUnknown() {
		apiScaling.Type = sca.SCALINGTYPE_SCALING_TYPE_MANUAL

		var manualAttrs struct {
			Instances types.Int32 `tfsdk:"instances"`
		}
		scaleTF.Manual.As(ctx, &manualAttrs, basetypes.ObjectAsOptions{})

		apiScaling.ManualScaling = &sca.ManualScaling{
			Instances: manualAttrs.Instances.ValueInt32(),
		}
	} else if !scaleTF.Auto.IsNull() && !scaleTF.Auto.IsUnknown() {
		apiScaling.Type = sca.SCALINGTYPE_SCALING_TYPE_AUTO

		var autoAttrs autoScalingModel
		scaleTF.Auto.As(ctx, &autoAttrs, basetypes.ObjectAsOptions{})

		apiAuto := sca.AutoScaling{
			MinInstances: autoAttrs.MinInstances.ValueInt32(),
			MaxInstances: autoAttrs.MaxInstances.ValueInt32(),
		}

		if !autoAttrs.AllowScaleToZero.IsNull() {
			apiAuto.AllowScaleToZero = conversion.BoolValueToPointer(autoAttrs.AllowScaleToZero)
		}

		var apiRules []sca.ScaleRule

		// Map single HTTP rule
		if !autoAttrs.HTTPRule.IsNull() && !autoAttrs.HTTPRule.IsUnknown() {
			var httpAttrs httpRuleModel
			autoAttrs.HTTPRule.As(ctx, &httpAttrs, basetypes.ObjectAsOptions{})

			apiRules = append(apiRules, sca.ScaleRule{
				Name: httpAttrs.Name.ValueString(),
				Type: sca.RULETYPE_RULE_TYPE_HTTP,
				HttpRule: &sca.HttpScaleRule{
					Concurrency: conversion.Int32ValueToPointer(httpAttrs.Concurrency),
					Rps:         conversion.Int32ValueToPointer(httpAttrs.Rps),
				},
			})
		}

		// Map native rules array
		if !autoAttrs.NativeRules.IsNull() && !autoAttrs.NativeRules.IsUnknown() {
			var tfRules []nativeRuleModel
			autoAttrs.NativeRules.ElementsAs(ctx, &tfRules, false)

			for _, r := range tfRules {
				customRule := sca.CustomScaleRule{
					Type: r.Trigger.ValueString(),
				}
				if !r.Parameters.IsNull() {
					for k, v := range r.Parameters.Elements() {
						if strVal, ok := v.(types.String); ok {
							customRule.Parameters = append(customRule.Parameters, sca.CustomRuleParameter{
								Name:  k,
								Value: strVal.ValueString(),
							})
						}
					}
				}
				if !r.SecretsMapping.IsNull() {
					for k, v := range r.SecretsMapping.Elements() {
						if strVal, ok := v.(types.String); ok {
							customRule.SecretsMapping = append(customRule.SecretsMapping, sca.CustomRuleSecretMapping{
								Parameter: k,
								Secret:    strVal.ValueString(),
							})
						}
					}
				}
				apiRules = append(apiRules, sca.ScaleRule{
					Name:       r.Name.ValueString(),
					Type:       sca.RULETYPE_RULE_TYPE_CUSTOM,
					CustomRule: &customRule,
				})
			}
		}

		if len(apiRules) > 0 {
			apiAuto.Rules = apiRules
		}
		apiScaling.AutoScaling = &apiAuto
	}
	return &apiScaling, nil
}

func isPublicIngressEnabled(ctx context.Context, netObj types.Object, diags *diag.Diagnostics) bool {
	if netObj.IsNull() || netObj.IsUnknown() {
		return false
	}

	var netModel networkModel
	diags.Append(netObj.As(ctx, &netModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() || netModel.Public.IsNull() || netModel.Public.IsUnknown() {
		return false
	}

	var pubModel publicNetworkModel
	diags.Append(netModel.Public.As(ctx, &pubModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() || pubModel.Enabled.IsNull() || pubModel.Enabled.IsUnknown() {
		return false
	}

	return pubModel.Enabled.ValueBool()
}
