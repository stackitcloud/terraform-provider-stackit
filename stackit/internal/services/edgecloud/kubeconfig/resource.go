package kubeconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	edgeCloud "github.com/stackitcloud/stackit-sdk-go/services/edge"
	edgeCloudWait "github.com/stackitcloud/stackit-sdk-go/services/edge/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	edgeCloudUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/edgecloud/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource               = &kubeconfigResource{}
	_ resource.ResourceWithConfigure  = &kubeconfigResource{}
	_ resource.ResourceWithModifyPlan = &kubeconfigResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	InstanceName   types.String `tfsdk:"instance_name"`
	InstanceId     types.String `tfsdk:"instance_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	KubeconfigId   types.String `tfsdk:"kubeconfig_id"` // uuid generated internally because kubeconfig has no identifier
	Kubeconfig     types.String `tfsdk:"kubeconfig"`
	Expiration     types.Int64  `tfsdk:"expiration"`
	RecreateBefore types.Int64  `tfsdk:"recreate_before"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
	CreationTime   types.String `tfsdk:"creation_time"`
	Region         types.String `tfsdk:"region"`
}

var descriptions = map[string]string{
	"main":          "Edge Cloud Instance kubeconfig resource schema. Allows managing edge hosts and edge cluster configuration resources via kubernetes API.",
	"id":            "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_name` or `instance_id`,`kubeconfig_id`\".",
	"kubeconfig_id": "Internally generated UUID to identify a kubeconfig resource in Terraform, since the Edge Cloud API doesn't return a kubeconfig identifier",
	"instance_name": "Name of the Edge Cloud instance.",
	"instance_id":   "ID of the Edge Cloud instance.",
	"project_id":    "STACKIT project ID to which the Edge Cloud instance is associated.",
	"kubeconfig":    "Raw kubeconfig.",
	// kubeconfig uses a service token, that's why it has the same min and max duration
	"expiration":      fmt.Sprintf("Expiration time of the kubeconfig, in seconds. Minimum is %d, Maximum is %d. Defaults to `3600`", edgeCloudUtils.TokenMinDuration, edgeCloudUtils.TokenMaxDuration),
	"recreate_before": "Number of seconds before expiration to trigger recreation of the kubeconfig at.",
	"expires_at":      "Timestamp when the kubeconfig expires",
	"creation_time":   "Date-time when the kubeconfig was created",
	"region":          "The resource region. If not defined, the provider region is used.",
}

// NewKubeconfigResource is a helper function to simplify the provider implementation.
func NewKubeconfigResource() resource.Resource {
	return &kubeconfigResource{}
}

// kubeconfigResource is the resource implementation.
type kubeconfigResource struct {
	client       *edgeCloud.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *kubeconfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edgecloud_kubeconfig"
}

// Configure adds the provider configured client to the resource.
func (r *kubeconfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_edgecloud_kubeconfig", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := edgeCloudUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Edge Cloud kubeconfig client configured")
}

// Schema defines the schema for the resource.
func (r *kubeconfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Edge Cloud is in private Beta and not generally available.\n You can contact support if you are interested in trying it out.", core.Resource),

		Description: descriptions["main"],
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
			"instance_name": schema.StringAttribute{
				Description: descriptions["instance_name"],
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.NoSeparator(),
					stringvalidator.ExactlyOneOf(path.MatchRoot("instance_id")),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.NoSeparator(),
					stringvalidator.ExactlyOneOf(path.MatchRoot("instance_name")),
				},
			},
			"kubeconfig_id": schema.StringAttribute{
				Description: descriptions["kubeconfig_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expiration": schema.Int64Attribute{
				Description: descriptions["expiration"],
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3600),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					// kubeconfig uses a service token, that's why it has the same min and max duration
					int64validator.Between(edgeCloudUtils.TokenMinDuration, edgeCloudUtils.TokenMaxDuration),
				},
			},
			"recreate_before": schema.Int64Attribute{
				Description: descriptions["recreate_before"],
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"kubeconfig": schema.StringAttribute{
				Description: descriptions["kubeconfig"],
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.StringAttribute{
				Description: descriptions["expires_at"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"creation_time": schema.StringAttribute{
				Description: descriptions["creation_time"],
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
		},
	}
}

func (r *kubeconfigResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	if req.Config.Raw.IsNull() {
		return
	}

	var configModel Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	if !req.State.Raw.IsNull() {
		var stateModel Model
		resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !utils.IsUndefined(stateModel.ExpiresAt) {
			recreateBefore := planModel.RecreateBefore
			if recreateBefore.IsUnknown() {
				recreateBefore = types.Int64Null()
			}

			shouldRecreate, err := edgeCloudUtils.CheckExpiration(stateModel.ExpiresAt, recreateBefore, time.Now())

			if err != nil {
				resp.Diagnostics.AddError("Error checking kubeconfig expiration in plan", err.Error())
				return
			}

			if shouldRecreate {
				tflog.Info(ctx, "Forcing kubeconfig recreation based on expiration/recreate_before window", map[string]any{
					"expires_at":      stateModel.ExpiresAt.ValueString(),
					"recreate_before": recreateBefore.String(),
				})

				planModel.ExpiresAt = types.StringUnknown()
				resp.RequiresReplace.Append(path.Root("expires_at"))
			}
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *kubeconfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	expirationSeconds := model.Expiration.ValueInt64()
	region := model.Region.ValueString()
	kubeconfigUUID := uuid.New().String()
	model.KubeconfigId = types.StringValue(kubeconfigUUID)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "kubeconfig_id", kubeconfigUUID)
	ctx = tflog.SetField(ctx, "region", region)

	var kubeconfigResp *edgeCloud.Kubeconfig
	var err error
	if !model.InstanceId.IsNull() {
		instanceId := model.InstanceId.ValueString()
		ctx = tflog.SetField(ctx, "instance_id", model.InstanceId)
		kubeconfigResp, err = edgeCloudWait.KubeconfigWaitHandler(ctx, r.client, projectId, region, instanceId, &expirationSeconds).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating kubeconfig", fmt.Sprintf("Kubeconfig creation waiting: %v", err))
			return
		}
		model.Id = types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, region, instanceId, kubeconfigUUID))
	} else if !model.InstanceName.IsNull() {
		instanceName := model.InstanceName.ValueString()
		ctx = tflog.SetField(ctx, "instance_name", model.InstanceName)
		kubeconfigResp, err = edgeCloudWait.KubeconfigByInstanceNameWaitHandler(ctx, r.client, projectId, region, instanceName, &expirationSeconds).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating kubeconfig", fmt.Sprintf("Kubeconfig creation waiting: %v", err))
			return
		}
		model.Id = types.StringValue(fmt.Sprintf("%s,%s,%s,%s", projectId, region, instanceName, kubeconfigUUID))
	}

	ctx = core.LogResponse(ctx)

	creationTime := time.Now()
	model.CreationTime = types.StringValue(creationTime.Format(time.RFC3339))
	expirationDuration := time.Duration(model.Expiration.ValueInt64()) * time.Second
	expiresAtTime := creationTime.Add(expirationDuration)
	model.ExpiresAt = types.StringValue(expiresAtTime.Format(time.RFC3339))

	if kubeconfigResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating kubeconfig", "API response is nil")
		return
	}

	if kubeconfigResp.Kubeconfig == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating kubeconfig", "API response kubeconfig field is nil")
		return
	}

	kubeconfig, err := marshalKubeconfig(*kubeconfigResp.Kubeconfig)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating kubeconfig", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	model.Kubeconfig = types.StringValue(kubeconfig)

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Edge Cloud kubeconfig created")
}

func (r *kubeconfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !model.InstanceId.IsNull() {
		ctx = tflog.SetField(ctx, "instance_id", model.InstanceId)
	} else if !model.InstanceName.IsNull() {
		ctx = tflog.SetField(ctx, "instance_name", model.InstanceName)
	}
	projectId := model.ProjectId.ValueString()
	kubeconfigUUID := model.KubeconfigId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "kubeconfig_id", kubeconfigUUID)
	ctx = tflog.SetField(ctx, "region", region)
	tflog.Info(ctx, "Edge Cloud kubeconfig read")
}

// Update only works for recreate_before, since this is a provider internal state value. Everything else requires recreation.
func (r *kubeconfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	ctx = tflog.SetField(ctx, "region", model.Region.ValueString())
	ctx = tflog.SetField(ctx, "project_id", model.ProjectId.ValueString())
	ctx = tflog.SetField(ctx, "instance_id", model.InstanceId.ValueString())
	ctx = tflog.SetField(ctx, "kubeconfig_id", model.KubeconfigId.ValueString())

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Edge Cloud kubeconfig updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *kubeconfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddWarning(ctx, &resp.Diagnostics, "Deleting kubeconfig", "Deleting this resource will only remove the values from the terraform state, it will not trigger a deletion or revoke the actual kubeconfig since kubernetes does not support the revocation of service tokens. The kubeconfig will still be valid until it expires.")

	// Retrieve values from plan
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !model.InstanceId.IsNull() {
		ctx = tflog.SetField(ctx, "instance_id", model.InstanceId)
	} else if !model.InstanceName.IsNull() {
		ctx = tflog.SetField(ctx, "instance_name", model.InstanceName)
	}

	ctx = tflog.SetField(ctx, "region", model.Region.ValueString())
	ctx = tflog.SetField(ctx, "project_id", model.ProjectId.ValueString())
	ctx = tflog.SetField(ctx, "kubeconfig_id", model.KubeconfigId.ValueString())

	tflog.Info(ctx, "Edge Cloud kubeconfig deleted from state")
}

func marshalKubeconfig(kubeconfigData map[string]any) (string, error) {
	// Check for empty/nil input
	if len(kubeconfigData) == 0 {
		return "", fmt.Errorf("received nil or empty kubeconfig data from the API")
	}

	// Marshal to JSON
	jsonDataBytes, err := json.Marshal(kubeconfigData)
	if err != nil {
		return "", fmt.Errorf("could not marshal kubeconfig map to JSON: %w", err)
	}

	return string(jsonDataBytes), nil
}
