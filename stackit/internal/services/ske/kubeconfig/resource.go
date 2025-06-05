package ske

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	skeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource               = &kubeconfigResource{}
	_ resource.ResourceWithConfigure  = &kubeconfigResource{}
	_ resource.ResourceWithModifyPlan = &kubeconfigResource{}
)

type Model struct {
	Id           types.String `tfsdk:"id"` // needed by TF
	ClusterName  types.String `tfsdk:"cluster_name"`
	ProjectId    types.String `tfsdk:"project_id"`
	KubeconfigId types.String `tfsdk:"kube_config_id"` // uuid generated internally because kubeconfig has no identifier
	Kubeconfig   types.String `tfsdk:"kube_config"`
	Expiration   types.Int64  `tfsdk:"expiration"`
	Refresh      types.Bool   `tfsdk:"refresh"`
	ExpiresAt    types.String `tfsdk:"expires_at"`
	CreationTime types.String `tfsdk:"creation_time"`
}

// NewKubeconfigResource is a helper function to simplify the provider implementation.
func NewKubeconfigResource() resource.Resource {
	return &kubeconfigResource{}
}

// kubeconfigResource is the resource implementation.
type kubeconfigResource struct {
	client *ske.APIClient
}

// Metadata returns the resource type name.
func (r *kubeconfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_kubeconfig"
}

// Configure adds the provider configured client to the resource.
func (r *kubeconfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := skeUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "SKE kubeconfig client configured")
}

// Schema defines the schema for the resource.
func (r *kubeconfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":           "SKE kubeconfig resource schema. Must have a `region` specified in the provider configuration.",
		"id":             "Terraform's internal resource ID. It is structured as \"`project_id`,`cluster_name`,`kube_config_id`\".",
		"kube_config_id": "Internally generated UUID to identify a kubeconfig resource in Terraform, since the SKE API doesnt return a kubeconfig identifier",
		"cluster_name":   "Name of the SKE cluster.",
		"project_id":     "STACKIT project ID to which the cluster is associated.",
		"kube_config":    "Raw short-lived admin kubeconfig.",
		"expiration":     "Expiration time of the kubeconfig, in seconds. Defaults to `3600`",
		"expires_at":     "Timestamp when the kubeconfig expires",
		"refresh":        "If set to true, the provider will check if the kubeconfig has expired and will generated a new valid one in-place",
		"creation_time":  "Date-time when the kubeconfig was created",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kube_config_id": schema.StringAttribute{
				Description: descriptions["kube_config_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				Description: descriptions["cluster_name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"expiration": schema.Int64Attribute{
				Description: descriptions["expiration"],
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3600), // the default value is not returned by the API so we set a default value here, otherwise we would have to compute the expiration based on the expires_at field
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"refresh": schema.BoolAttribute{
				Description: descriptions["refresh"],
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"kube_config": schema.StringAttribute{
				Description: descriptions["kube_config"],
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
		},
	}
}

// ModifyPlan will be called in the Plan phase and will check if the plan is a creation of the resource
// If so, show warning related to deprecated credentials endpoints
func (r *kubeconfigResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	if req.State.Raw.IsNull() {
		// Planned to create a kubeconfig
		core.LogAndAddWarning(ctx, &resp.Diagnostics, "Planned to create kubeconfig", "Once this resource is created, you will no longer be able to use the deprecated credentials endpoints and the kube_config field on the cluster resource will be empty for this cluster. For more info check How to Rotate SKE Credentials (https://docs.stackit.cloud/stackit/en/how-to-rotate-ske-credentials-200016334.html)")
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
	projectId := model.ProjectId.ValueString()
	clusterName := model.ClusterName.ValueString()
	kubeconfigUUID := uuid.New().String()

	model.KubeconfigId = types.StringValue(kubeconfigUUID)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "cluster_name", clusterName)
	ctx = tflog.SetField(ctx, "kube_config_id", kubeconfigUUID)

	err := r.createKubeconfig(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating kubeconfig", fmt.Sprintf("Creating kubeconfig: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SKE kubeconfig created")
}

// Read refreshes the Terraform state with the latest data.
// There is no GET kubeconfig endpoint.
// If the refresh field is set, Read will check the expiration date and will get a new valid kubeconfig if it has expired
// If kubeconfig creation time is before lastCompletionTime of the credentials rotation or
// before cluster creation time a new kubeconfig is created.
func (r *kubeconfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	clusterName := model.ClusterName.ValueString()
	kubeconfigUUID := model.KubeconfigId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "cluster_name", clusterName)
	ctx = tflog.SetField(ctx, "kube_config_id", kubeconfigUUID)

	cluster, err := r.client.GetClusterExecute(ctx, projectId, clusterName)
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading kubeconfig",
			fmt.Sprintf("Kubeconfig with ID %q or cluster with name %q does not exist in project %q.", kubeconfigUUID, clusterName, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// check if kubeconfig has expired
	hasExpired, err := checkHasExpired(&model, time.Now())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading kubeconfig", fmt.Sprintf("%v", err))
		return
	}

	clusterRecreation, err := checkClusterRecreation(cluster, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading kubeconfig", fmt.Sprintf("%v", err))
		return
	}

	credentialsRotation, err := checkCredentialsRotation(cluster, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading kubeconfig", fmt.Sprintf("%v", err))
		return
	}

	if hasExpired || clusterRecreation || credentialsRotation {
		err := r.createKubeconfig(ctx, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading kubeconfig", fmt.Sprintf("The existing kubeconfig is invalid, creating a new one: %v", err))
			return
		}

		// Set state to fully populated data
		diags = resp.State.Set(ctx, model)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Info(ctx, "SKE kubeconfig read")
}

func (r *kubeconfigResource) createKubeconfig(ctx context.Context, model *Model) error {
	// Generate API request body from model
	payload, err := toCreatePayload(model)
	if err != nil {
		return fmt.Errorf("creating API payload: %w", err)
	}
	// Create new kubeconfig
	kubeconfigResp, err := r.client.CreateKubeconfig(ctx, model.ProjectId.ValueString(), model.ClusterName.ValueString()).CreateKubeconfigPayload(*payload).Execute()
	if err != nil {
		return fmt.Errorf("calling API: %w", err)
	}

	// Map response body to schema
	err = mapFields(kubeconfigResp, model, time.Now())
	if err != nil {
		return fmt.Errorf("processing API payload: %w", err)
	}
	return nil
}

func (r *kubeconfigResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating kubeconfig", "Kubeconfig can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *kubeconfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddWarning(ctx, &resp.Diagnostics, "Deleting kubeconfig", "Deleted this resource will only remove the values from the terraform state, it will not trigger a deletion or revoke of the actual kubeconfig as this is not supported by the SKE API. The kubeconfig will still be valid until it expires.")

	// Retrieve values from plan
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	clusterName := model.ClusterName.ValueString()
	kubeconfigUUID := model.KubeconfigId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "cluster_name", clusterName)
	ctx = tflog.SetField(ctx, "kube_config_id", kubeconfigUUID)

	// kubeconfig is deleted automatically from the state
	tflog.Info(ctx, "SKE kubeconfig deleted")
}

func mapFields(kubeconfigResp *ske.Kubeconfig, model *Model, creationTime time.Time) error {
	if kubeconfigResp == nil {
		return fmt.Errorf("response is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), model.ClusterName.ValueString(), model.KubeconfigId.ValueString(),
	)

	if kubeconfigResp.Kubeconfig == nil {
		return fmt.Errorf("kubeconfig not present")
	}

	model.Kubeconfig = types.StringPointerValue(kubeconfigResp.Kubeconfig)
	model.ExpiresAt = types.StringValue(kubeconfigResp.ExpirationTimestamp.Format(time.RFC3339))
	// set creation time
	model.CreationTime = types.StringValue(creationTime.Format(time.RFC3339))
	return nil
}

func toCreatePayload(model *Model) (*ske.CreateKubeconfigPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	expiration := conversion.Int64ValueToPointer(model.Expiration)
	var expirationStringPtr *string
	if expiration != nil {
		expirationStringPtr = sdkUtils.Ptr(strconv.FormatInt(*expiration, 10))
	}

	return &ske.CreateKubeconfigPayload{
		ExpirationSeconds: expirationStringPtr,
	}, nil
}

// helper function to check if kubecondig has expired
func checkHasExpired(model *Model, currentTime time.Time) (bool, error) {
	expiresAt := model.ExpiresAt
	if model.Refresh.ValueBool() && !expiresAt.IsNull() {
		if expiresAt.IsUnknown() {
			return true, nil
		}
		expiresAt, err := time.Parse(time.RFC3339, expiresAt.ValueString())
		if err != nil {
			return false, fmt.Errorf("converting expiresAt field to timestamp: %w", err)
		}
		if expiresAt.Before(currentTime) {
			return true, nil
		}
	}
	return false, nil
}

// helper function to check if a credentials rotation was done
func checkCredentialsRotation(cluster *ske.Cluster, model *Model) (bool, error) {
	creationTimeValue := model.CreationTime
	if creationTimeValue.IsNull() || creationTimeValue.IsUnknown() {
		return false, nil
	}
	creationTime, err := time.Parse(time.RFC3339, creationTimeValue.ValueString())
	if err != nil {
		return false, fmt.Errorf("converting creationTime field to timestamp: %w", err)
	}
	if cluster.Status.CredentialsRotation.LastCompletionTime != nil {
		if creationTime.Before(*cluster.Status.CredentialsRotation.LastCompletionTime) {
			return true, nil
		}
	}
	return false, nil
}

// helper function to check if a cluster recreation was done
func checkClusterRecreation(cluster *ske.Cluster, model *Model) (bool, error) {
	creationTimeValue := model.CreationTime
	if creationTimeValue.IsNull() || creationTimeValue.IsUnknown() {
		return false, nil
	}
	creationTime, err := time.Parse(time.RFC3339, creationTimeValue.ValueString())
	if err != nil {
		return false, fmt.Errorf("converting creationTime field to timestamp: %w", err)
	}
	if cluster.Status.CreationTime != nil {
		if creationTime.Before(*cluster.Status.CreationTime) {
			return true, nil
		}
	}
	return false, nil
}
