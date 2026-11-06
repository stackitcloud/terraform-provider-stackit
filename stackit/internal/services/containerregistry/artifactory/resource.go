package artifactory

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	registry "github.com/stackitcloud/stackit-sdk-go/services/registry/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/registry/v1api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &artifactoryResource{}
	_ resource.ResourceWithConfigure   = &artifactoryResource{}
	_ resource.ResourceWithImportState = &artifactoryResource{}
	_ resource.ResourceWithModifyPlan  = &artifactoryResource{}
)

type Model struct {
	Id            types.String `tfsdk:"id"`
	ProjectId     types.String `tfsdk:"project_id"`
	Region        types.String `tfsdk:"region"`
	ArtifactoryId types.String `tfsdk:"artifactory_id"`
	Name          types.String `tfsdk:"name"`
	CreateTime    types.String `tfsdk:"create_time"`
	State         types.String `tfsdk:"state"`
	Url           types.String `tfsdk:"url"`
	Proxy         types.Object `tfsdk:"proxy"`
}

type proxyModel struct {
	Description  types.String `tfsdk:"description"`
	RegistryType types.String `tfsdk:"registry_type"`
	Url          types.String `tfsdk:"url"`
	Credentials  types.Object `tfsdk:"credentials"`
}

type credentialsModel struct {
	AccessKey             types.String `tfsdk:"access_key"`
	AccessSecret          types.String `tfsdk:"access_secret"`
	AccessSecretWo        types.String `tfsdk:"access_secret_wo"`
	AccessSecretWoVersion types.Int64  `tfsdk:"access_secret_wo_version"`
}

var credentialsTypes = map[string]attr.Type{
	"access_key":               types.StringType,
	"access_secret":            types.StringType,
	"access_secret_wo":         types.StringType,
	"access_secret_wo_version": types.Int64Type,
}

var proxyTypes = map[string]attr.Type{
	"description":   types.StringType,
	"registry_type": types.StringType,
	"url":           types.StringType,
	"credentials":   types.ObjectType{AttrTypes: credentialsTypes},
}

var descriptions = map[string]string{ //nolint:gosec // linter false positive

	"id": "Terraform's internal resource ID, structured as `project_id`,`region`,`artifactory_id`.",

	"project_id": "STACKIT project ID to which the container registry artifactory is associated.",

	"region": "STACKIT region where the container registry artifactory is located. If not defined, the provider default_region is used.",

	"artifactory_id": "The unique identifier of the container registry artifactory.",

	"name": "A user-chosen name to distinguish multiple STACKIT artifactories.",

	"create_time": "The date and time the creation of the STACKIT artifactory was triggered.",

	"state": "The current state of the STACKIT artifactory.",

	"url": "The base URL of the STACKIT artifactory.",

	"proxy": "Proxy configuration if this artifactory is created as a proxy-cache for an upstream registry.",

	"proxy.registry_type": "Which replication adapter to use for this registry.",

	"proxy.url": "Upstream registry URL to proxy.",

	"proxy.description": "Description of the upstream registry.",

	"proxy.credentials": "Credentials for the upstream registry.",

	"proxy.credentials.access_key": "Access key or username for the upstream registry.",

	"proxy.credentials.access_secret": "Access secret or password for the upstream registry. Write-only argument `access_secret_wo` should be preferred.",

	"proxy.credentials.access_secret_wo": "Write-only access secret or password for the upstream registry - never stored in state and never returned by the API. To rotate the secret, update this value AND increment `access_secret_wo_version`. Changing this field alone will NOT trigger an update.",

	"proxy.credentials.access_secret_wo_version": "User-managed rotation counter for `access_secret_wo`. Must be incremented every time `access_secret_wo` is updated.",
}

func NewArtifactoryResource() resource.Resource {
	return &artifactoryResource{}
}

type artifactoryResource struct {
	client registry.DefaultAPI

	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.

func (r *artifactoryResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *artifactoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_containerregistry_artifactory"
}

func (r *artifactoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.providerData = providerData
	r.client = clients.ContainerRegistryV1Client
	tflog.Info(ctx, "Container Registry client configured")
}

func (r *artifactoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Container registry artifactory resource schema.",
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
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"artifactory_id": schema.StringAttribute{
				Description: descriptions["artifactory_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`),
						"Must consist of lowercase alphanumeric characters, optionally separated by single '.', '_', or '-' characters.",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"create_time": schema.StringAttribute{
				Description: descriptions["create_time"],
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: descriptions["state"],
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: descriptions["url"],
				Computed:    true,
			},
			"proxy": schema.SingleNestedAttribute{
				Description: descriptions["proxy"],
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"registry_type": schema.StringAttribute{
						Description: descriptions["proxy.registry_type"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"url": schema.StringAttribute{
						Description: descriptions["proxy.url"],
						Required:    true,
					},
					"description": schema.StringAttribute{
						Description: descriptions["proxy.description"],
						Optional:    true,
					},
					"credentials": schema.SingleNestedAttribute{
						Description: descriptions["proxy.credentials"],
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"access_key": schema.StringAttribute{
								Description: descriptions["proxy.credentials.access_key"],
								Required:    true,
							},
							"access_secret": schema.StringAttribute{
								Description: descriptions["proxy.credentials.access_secret"],
								Optional:    true,
								Sensitive:   true,
								Validators: []validator.String{
									stringvalidator.ConflictsWith(
										path.MatchRelative().AtParent().AtName("access_secret_wo"),
										path.MatchRelative().AtParent().AtName("access_secret_wo_version"),
									),
									stringvalidator.PreferWriteOnlyAttribute(
										path.MatchRoot("proxy").AtName("credentials").AtName("access_secret_wo"),
									),
								},
							},
							"access_secret_wo": schema.StringAttribute{
								Description: descriptions["proxy.credentials.access_secret_wo"],
								Optional:    true,
								WriteOnly:   true,
								Sensitive:   true,
								Validators: []validator.String{
									stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("access_secret")),
									stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("access_secret_wo_version")),
								},
							},
							"access_secret_wo_version": schema.Int64Attribute{
								Description: descriptions["proxy.credentials.access_secret_wo_version"],
								Optional:    true,
								Validators: []validator.Int64{
									int64validator.AlsoRequires(path.MatchRelative().AtParent().AtName("access_secret_wo")),
									int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("access_secret")),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *artifactoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configModel Model
	if req.Config.Schema != nil && req.Config.Raw.Type() != nil {
		diags = req.Config.Get(ctx, &configModel)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(ctx, &model, &configModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating artifactory", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.CreateArtifactory(ctx, projectId, region).CreateArtifactoryPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating artifactory", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp == nil || createResp.Id == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating artifactory", "Create API response: Incomplete response (id missing)")
		return
	}

	artifactoryId := createResp.Id
	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":     projectId,
		"region":         region,
		"artifactory_id": artifactoryId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := wait.CreateArtifactoryWaitHandler(ctx, r.client, projectId, region, artifactoryId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating artifactory", fmt.Sprintf("Waiting for artifactory to be ready: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating artifactory", fmt.Sprintf("Mapping API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Container Registry artifactory created")
}

func (r *artifactoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	artifactoryId := model.ArtifactoryId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "artifactory_id", artifactoryId)

	artifactory, err := r.client.GetArtifactory(ctx, projectId, region, artifactoryId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading artifactory", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if artifactory.State == registry.ARTIFACTORYSTATE_DELETED {
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapFields(ctx, artifactory, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading artifactory", fmt.Sprintf("Mapping API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Container Registry artifactory read")
}

func (r *artifactoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var plan Model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state Model
	if req.State.Schema != nil && req.State.Raw.Type() != nil {
		diags = req.State.Get(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var configModel Model
	if req.Config.Schema != nil && req.Config.Raw.Type() != nil {
		diags = req.Config.Get(ctx, &configModel)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	ctx = core.InitProviderContext(ctx)

	projectId := plan.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(plan.Region)
	artifactoryId := plan.ArtifactoryId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "artifactory_id", artifactoryId)

	payload, err := toUpdatePayload(ctx, &plan, &state, &configModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating artifactory", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	_, err = r.client.PatchArtifactory(ctx, projectId, region, artifactoryId).PatchArtifactoryPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating artifactory", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.UpdateArtifactoryWaitHandler(ctx, r.client, projectId, region, artifactoryId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating artifactory", fmt.Sprintf("Waiting for artifactory update: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &plan, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating artifactory", fmt.Sprintf("Mapping API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Container Registry artifactory updated")
}

func (r *artifactoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	artifactoryId := model.ArtifactoryId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "artifactory_id", artifactoryId)

	err := r.client.DeleteArtifactory(ctx, projectId, region, artifactoryId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting artifactory", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteArtifactoryWaitHandler(ctx, r.client, projectId, region, artifactoryId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting artifactory", fmt.Sprintf("Waiting for artifactory to be deleted: %v", err))
		return
	}

	tflog.Info(ctx, "Container Registry artifactory deleted")
}

func (r *artifactoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing artifactory",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[artifactory_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":     idParts[0],
		"region":         idParts[1],
		"artifactory_id": idParts[2],
	})
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	tflog.Info(ctx, "Container Registry artifactory state imported")
}

func toCreatePayload(ctx context.Context, model, configModel *Model) (*registry.CreateArtifactoryPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model input is nil")
	}

	payload := registry.NewCreateArtifactoryPayload(model.Name.ValueString())

	if !model.Proxy.IsNull() && !model.Proxy.IsUnknown() {
		var p proxyModel
		diags := model.Proxy.As(ctx, &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting proxy object: %v", diags.Errors())
		}

		proxyReq := registry.NewCreateProxyRequest(p.RegistryType.ValueString(), p.Url.ValueString())
		if !p.Description.IsNull() && !p.Description.IsUnknown() {
			proxyReq.SetDescription(p.Description.ValueString())
		}

		if !p.Credentials.IsNull() && !p.Credentials.IsUnknown() {
			var creds credentialsModel
			diags := p.Credentials.As(ctx, &creds, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, fmt.Errorf("converting credentials object: %v", diags.Errors())
			}

			var configCreds *credentialsModel
			if configModel != nil && !configModel.Proxy.IsNull() && !configModel.Proxy.IsUnknown() {
				var cp proxyModel
				if d := configModel.Proxy.As(ctx, &cp, basetypes.ObjectAsOptions{}); !d.HasError() && !cp.Credentials.IsNull() && !cp.Credentials.IsUnknown() {
					var cc credentialsModel
					if d2 := cp.Credentials.As(ctx, &cc, basetypes.ObjectAsOptions{}); !d2.HasError() {
						configCreds = &cc
					}
				}
			}

			secret := getAccessSecret(&creds, nil, configCreds, false)
			if secret == nil {
				return nil, fmt.Errorf("either access_secret or access_secret_wo must be provided in proxy.credentials")
			}

			proxyReq.SetCredentials(*registry.NewCredentials(creds.AccessKey.ValueString(), *secret))
		}

		payload.SetProxy(*proxyReq)
	}

	return payload, nil
}

func toUpdatePayload(ctx context.Context, model, stateModel, configModel *Model) (*registry.PatchArtifactoryPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model input is nil")
	}

	payload := registry.NewPatchArtifactoryPayload()

	if !model.Proxy.IsNull() && !model.Proxy.IsUnknown() {
		var p proxyModel
		diags := model.Proxy.As(ctx, &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting proxy object: %v", diags.Errors())
		}

		updateProxyReq := registry.NewUpdateProxyRequest()
		if !p.Url.IsNull() && !p.Url.IsUnknown() {
			updateProxyReq.SetUrl(p.Url.ValueString())
		}
		if !p.Description.IsNull() && !p.Description.IsUnknown() {
			updateProxyReq.SetDescription(p.Description.ValueString())
		}

		if !p.Credentials.IsNull() && !p.Credentials.IsUnknown() {
			var creds credentialsModel
			diags := p.Credentials.As(ctx, &creds, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, fmt.Errorf("converting credentials object: %v", diags.Errors())
			}

			var stateCreds *credentialsModel
			if stateModel != nil && !stateModel.Proxy.IsNull() && !stateModel.Proxy.IsUnknown() {
				var sp proxyModel
				if d := stateModel.Proxy.As(ctx, &sp, basetypes.ObjectAsOptions{}); !d.HasError() && !sp.Credentials.IsNull() && !sp.Credentials.IsUnknown() {
					var sc credentialsModel
					if d2 := sp.Credentials.As(ctx, &sc, basetypes.ObjectAsOptions{}); !d2.HasError() {
						stateCreds = &sc
					}
				}
			}

			var configCreds *credentialsModel
			if configModel != nil && !configModel.Proxy.IsNull() && !configModel.Proxy.IsUnknown() {
				var cp proxyModel
				if d := configModel.Proxy.As(ctx, &cp, basetypes.ObjectAsOptions{}); !d.HasError() && !cp.Credentials.IsNull() && !cp.Credentials.IsUnknown() {
					var cc credentialsModel
					if d2 := cp.Credentials.As(ctx, &cc, basetypes.ObjectAsOptions{}); !d2.HasError() {
						configCreds = &cc
					}
				}
			}

			secret := getAccessSecret(&creds, stateCreds, configCreds, true)
			if secret != nil {
				updateProxyReq.SetCredentials(*registry.NewCredentials(creds.AccessKey.ValueString(), *secret))
			}
		}

		payload.SetProxy(*updateProxyReq)
	}

	return payload, nil
}

func getAccessSecret(planCreds, stateCreds, configCreds *credentialsModel, isUpdate bool) *string {
	if planCreds == nil {
		return nil
	}
	// 1. Legacy field
	if !planCreds.AccessSecret.IsNull() && !planCreds.AccessSecret.IsUnknown() {
		return planCreds.AccessSecret.ValueStringPointer()
	}
	// 2. Write-only field
	if configCreds != nil && !configCreds.AccessSecretWo.IsNull() && !configCreds.AccessSecretWo.IsUnknown() {
		if !isUpdate {
			return configCreds.AccessSecretWo.ValueStringPointer()
		}
		// On update, only send if version was bumped (or previously not set)
		if stateCreds == nil || !planCreds.AccessSecretWoVersion.Equal(stateCreds.AccessSecretWoVersion) {
			return configCreds.AccessSecretWo.ValueStringPointer()
		}
	}
	return nil
}

func mapFields(ctx context.Context, artifactory *registry.Artifactory, model *Model, region string) error {
	if artifactory == nil {
		return fmt.Errorf("artifactory is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	var artifactoryId string
	if model.ArtifactoryId.ValueString() != "" {
		artifactoryId = model.ArtifactoryId.ValueString()
	} else if artifactory.Id != "" {
		artifactoryId = artifactory.Id
	} else {
		return fmt.Errorf("artifactory id not present")
	}

	model.Id = tfutils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, artifactoryId)
	model.Region = types.StringValue(region)
	model.ArtifactoryId = types.StringValue(artifactoryId)
	model.Name = types.StringValue(artifactory.Name)
	model.CreateTime = types.StringValue(artifactory.CreateTime.Format(time.RFC3339))
	model.State = types.StringValue(string(artifactory.State))
	model.Url = types.StringValue(artifactory.Url)

	if artifactory.Proxy == nil {
		model.Proxy = types.ObjectNull(proxyTypes)
	} else {
		var p proxyModel
		if !model.Proxy.IsNull() && !model.Proxy.IsUnknown() {
			diags := model.Proxy.As(ctx, &p, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})
			if diags.HasError() {
				return fmt.Errorf("converting proxy object: %v", diags.Errors())
			}
		}

		p.RegistryType = types.StringValue(artifactory.Proxy.RegistryType)
		p.Url = types.StringValue(artifactory.Proxy.Url)
		p.Description = types.StringPointerValue(artifactory.Proxy.Description)

		if artifactory.Proxy.Credentials == nil {
			p.Credentials = types.ObjectNull(credentialsTypes)
		} else {
			var creds credentialsModel
			if !p.Credentials.IsNull() && !p.Credentials.IsUnknown() {
				diags := p.Credentials.As(ctx, &creds, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})
				if diags.HasError() {
					return fmt.Errorf("converting credentials object: %v", diags.Errors())
				}
			}

			creds.AccessKey = types.StringValue(artifactory.Proxy.Credentials.AccessKey)
			if tfutils.IsUndefined(creds.AccessSecret) {
				creds.AccessSecret = types.StringNull()
			}
			creds.AccessSecretWo = types.StringNull()
			if tfutils.IsUndefined(creds.AccessSecretWoVersion) {
				creds.AccessSecretWoVersion = types.Int64Null()
			}

			credsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, creds)
			if diags.HasError() {
				return fmt.Errorf("mapping credentials: %w", core.DiagsToError(diags))
			}
			p.Credentials = credsObj
		}

		proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, p)
		if diags.HasError() {
			return fmt.Errorf("mapping proxy: %w", core.DiagsToError(diags))
		}
		model.Proxy = proxyObj
	}

	return nil
}
