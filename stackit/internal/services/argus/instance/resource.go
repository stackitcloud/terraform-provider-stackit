package argus

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
	"github.com/stackitcloud/stackit-sdk-go/services/argus/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
)

type Model struct {
	Id                                 types.String `tfsdk:"id"` // needed by TF
	ProjectId                          types.String `tfsdk:"project_id"`
	InstanceId                         types.String `tfsdk:"instance_id"`
	Name                               types.String `tfsdk:"name"`
	PlanName                           types.String `tfsdk:"plan_name"`
	PlanId                             types.String `tfsdk:"plan_id"`
	Parameters                         types.Map    `tfsdk:"parameters"`
	DashboardURL                       types.String `tfsdk:"dashboard_url"`
	IsUpdatable                        types.Bool   `tfsdk:"is_updatable"`
	GrafanaURL                         types.String `tfsdk:"grafana_url"`
	GrafanaPublicReadAccess            types.Bool   `tfsdk:"grafana_public_read_access"`
	GrafanaInitialAdminPassword        types.String `tfsdk:"grafana_initial_admin_password"`
	GrafanaInitialAdminUser            types.String `tfsdk:"grafana_initial_admin_user"`
	MetricsRetentionDays               types.Int64  `tfsdk:"metrics_retention_days"`
	MetricsRetentionDays5mDownsampling types.Int64  `tfsdk:"metrics_retention_days_5m_downsampling"`
	MetricsRetentionDays1hDownsampling types.Int64  `tfsdk:"metrics_retention_days_1h_downsampling"`
	MetricsURL                         types.String `tfsdk:"metrics_url"`
	MetricsPushURL                     types.String `tfsdk:"metrics_push_url"`
	TargetsURL                         types.String `tfsdk:"targets_url"`
	AlertingURL                        types.String `tfsdk:"alerting_url"`
	LogsURL                            types.String `tfsdk:"logs_url"`
	LogsPushURL                        types.String `tfsdk:"logs_push_url"`
	JaegerTracesURL                    types.String `tfsdk:"jaeger_traces_url"`
	JaegerUIURL                        types.String `tfsdk:"jaeger_ui_url"`
	OtlpTracesURL                      types.String `tfsdk:"otlp_traces_url"`
	ZipkinSpansURL                     types.String `tfsdk:"zipkin_spans_url"`
	ACL                                types.Set    `tfsdk:"acl"`
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client *argus.APIClient
}

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_argus_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	tflog.Info(ctx, "Argus instance client configured")
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Argus instance resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the instance is associated.",
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
				Description: "The Argus instance ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Argus instance.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(200),
				},
			},
			"plan_name": schema.StringAttribute{
				Description: "Specifies the Argus plan. E.g. `Monitoring-Medium-EU01`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(200),
				},
			},
			"plan_id": schema.StringAttribute{
				Description: "The Argus plan ID.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"parameters": schema.MapAttribute{
				Description: "Additional parameters.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_url": schema.StringAttribute{
				Description: "Specifies Argus instance dashboard URL.",
				Computed:    true,
			},
			"is_updatable": schema.BoolAttribute{
				Description: "Specifies if the instance can be updated.",
				Computed:    true,
			},
			"grafana_public_read_access": schema.BoolAttribute{
				Description: "If true, anyone can access Grafana dashboards without logging in.",
				Computed:    true,
			},
			"grafana_url": schema.StringAttribute{
				Description: "Specifies Grafana URL.",
				Computed:    true,
			},
			"grafana_initial_admin_user": schema.StringAttribute{
				Description: "Specifies an initial Grafana admin username.",
				Computed:    true,
			},
			"grafana_initial_admin_password": schema.StringAttribute{
				Description: "Specifies an initial Grafana admin password.",
				Computed:    true,
				Sensitive:   true,
			},
			"metrics_retention_days": schema.Int64Attribute{
				Description: "Specifies for how many days the raw metrics are kept.",
				Computed:    true,
			},
			"metrics_retention_days_5m_downsampling": schema.Int64Attribute{
				Description: "Specifies for how many days the 5m downsampled metrics are kept. must be less than the value of the general retention. Default is set to `0` (disabled).",
				Computed:    true,
			},
			"metrics_retention_days_1h_downsampling": schema.Int64Attribute{
				Description: "Specifies for how many days the 1h downsampled metrics are kept. must be less than the value of the 5m downsampling retention. Default is set to `0` (disabled).",
				Computed:    true,
			},
			"metrics_url": schema.StringAttribute{
				Description: "Specifies metrics URL.",
				Computed:    true,
			},
			"metrics_push_url": schema.StringAttribute{
				Description: "Specifies URL for pushing metrics.",
				Computed:    true,
			},
			"targets_url": schema.StringAttribute{
				Description: "Specifies Targets URL.",
				Computed:    true,
			},
			"alerting_url": schema.StringAttribute{
				Description: "Specifies Alerting URL.",
				Computed:    true,
			},
			"logs_url": schema.StringAttribute{
				Description: "Specifies Logs URL.",
				Computed:    true,
			},
			"logs_push_url": schema.StringAttribute{
				Description: "Specifies URL for pushing logs.",
				Computed:    true,
			},
			"jaeger_traces_url": schema.StringAttribute{
				Computed: true,
			},
			"jaeger_ui_url": schema.StringAttribute{
				Computed: true,
			},
			"otlp_traces_url": schema.StringAttribute{
				Computed: true,
			},
			"zipkin_spans_url": schema.StringAttribute{
				Computed: true,
			},
			"acl": schema.SetAttribute{
				Description: "The access control list for this instance. Each entry is an IP address range that is permitted to access, in CIDR notation.",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						validate.CIDR(),
					),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	acl := []string{}
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	err := r.loadPlanId(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Loading service plan: %v", err))
		return
	}
	// Generate API request body from model
	createPayload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	createResp, err := r.client.CreateInstance(ctx, projectId).CreateInstancePayload(*createPayload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	instanceId := createResp.InstanceId
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client, *instanceId, projectId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to instance populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create ACL
	err = updateACL(ctx, projectId, *instanceId, acl, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating ACL: %v", err))
		return
	}
	aclList, err := r.client.ListACL(ctx, *instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API to list ACL data: %v", err))
		return
	}

	// Map response body to schema
	err = mapACLField(aclList, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response for the ACL: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Argus instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	instanceResp, err := r.client.GetInstance(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	aclList, err := r.client.ListACL(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API for ACL data: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, instanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Map response body to schema
	err = mapACLField(aclList, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API response for the ACL: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Argus instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()

	acl := []string{}
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	err := r.loadPlanId(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Loading service plan: %v", err))
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing instance
	_, err = r.client.UpdateInstance(ctx, instanceId, projectId).UpdateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.UpdateInstanceWaitHandler(ctx, r.client, instanceId, projectId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update ACL
	err = updateACL(ctx, projectId, instanceId, acl, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Updating ACL: %v", err))
		return
	}
	aclList, err := r.client.ListACL(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Calling API to list ACL data: %v", err))
		return
	}

	// Map response body to schema
	err = mapACLField(aclList, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API response for the ACL: %v", err))
		return
	}

	// Set state to ACL populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Argus instance updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()

	// Delete existing instance
	_, err := r.client.DeleteInstance(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client, instanceId, projectId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Argus instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	tflog.Info(ctx, "Argus instance state imported")
}

func mapFields(ctx context.Context, r *argus.GetInstanceResponse, model *Model) error {
	if r == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if r.Id != nil {
		instanceId = *r.Id
	} else {
		return fmt.Errorf("instance id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		instanceId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.InstanceId = types.StringValue(instanceId)
	model.PlanName = types.StringPointerValue(r.PlanName)
	model.PlanId = types.StringPointerValue(r.PlanId)
	model.Name = types.StringPointerValue(r.Name)

	ps := r.Parameters
	if ps == nil {
		model.Parameters = types.MapNull(types.StringType)
	} else {
		params := make(map[string]attr.Value, len(*ps))
		for k, v := range *ps {
			params[k] = types.StringValue(v)
		}
		res, diags := types.MapValueFrom(ctx, types.StringType, params)
		if diags.HasError() {
			return fmt.Errorf("parameter mapping %s", diags.Errors())
		}
		model.Parameters = res
	}

	model.IsUpdatable = types.BoolPointerValue(r.IsUpdatable)
	model.DashboardURL = types.StringPointerValue(r.DashboardUrl)
	if r.Instance != nil {
		i := *r.Instance
		model.GrafanaURL = types.StringPointerValue(i.GrafanaUrl)
		model.GrafanaPublicReadAccess = types.BoolPointerValue(i.GrafanaPublicReadAccess)
		model.GrafanaInitialAdminPassword = types.StringPointerValue(i.GrafanaAdminPassword)
		model.GrafanaInitialAdminUser = types.StringPointerValue(i.GrafanaAdminUser)
		model.MetricsRetentionDays = types.Int64Value(int64(*i.MetricsRetentionTimeRaw))
		model.MetricsRetentionDays5mDownsampling = types.Int64Value(int64(*i.MetricsRetentionTime5m))
		model.MetricsRetentionDays1hDownsampling = types.Int64Value(int64(*i.MetricsRetentionTime1h))
		model.MetricsURL = types.StringPointerValue(i.MetricsUrl)
		model.MetricsPushURL = types.StringPointerValue(i.PushMetricsUrl)
		model.TargetsURL = types.StringPointerValue(i.TargetsUrl)
		model.AlertingURL = types.StringPointerValue(i.AlertingUrl)
		model.LogsURL = types.StringPointerValue(i.LogsUrl)
		model.LogsPushURL = types.StringPointerValue(i.LogsPushUrl)
		model.JaegerTracesURL = types.StringPointerValue(i.JaegerTracesUrl)
		model.JaegerUIURL = types.StringPointerValue(i.JaegerUiUrl)
		model.OtlpTracesURL = types.StringPointerValue(i.OtlpTracesUrl)
		model.ZipkinSpansURL = types.StringPointerValue(i.ZipkinSpansUrl)
	}

	return nil
}

func mapACLField(aclList *argus.ListACLResponse, model *Model) error {
	if aclList == nil {
		return fmt.Errorf("mapping ACL: invalid API response")
	}

	if aclList.Acl == nil || len(*aclList.Acl) == 0 {
		if !(model.ACL.IsNull() || model.ACL.IsUnknown() || model.ACL.Equal(types.SetValueMust(types.StringType, []attr.Value{}))) {
			model.ACL = types.SetNull(types.StringType)
		}
		return nil
	}

	acl := []attr.Value{}
	for _, cidr := range *aclList.Acl {
		acl = append(acl, types.StringValue(cidr))
	}
	aclTF, diags := types.SetValue(types.StringType, acl)
	if diags.HasError() {
		return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
	}
	model.ACL = aclTF
	return nil
}

func toCreatePayload(model *Model) (*argus.CreateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	elements := model.Parameters.Elements()
	pa := make(map[string]interface{}, len(elements))
	for k := range elements {
		pa[k] = elements[k].String()
	}
	return &argus.CreateInstancePayload{
		Name:      conversion.StringValueToPointer(model.Name),
		PlanId:    conversion.StringValueToPointer(model.PlanId),
		Parameter: &pa,
	}, nil
}

func updateACL(ctx context.Context, projectId, instanceId string, acl []string, client *argus.APIClient) error {
	payload := argus.UpdateACLPayload{
		Acl: utils.Ptr(acl),
	}

	_, err := client.UpdateACL(ctx, instanceId, projectId).UpdateACLPayload(payload).Execute()
	if err != nil {
		return fmt.Errorf("updating ACL: %w", err)
	}

	return nil
}

func toUpdatePayload(model *Model) (*argus.UpdateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	elements := model.Parameters.Elements()
	pa := make(map[string]interface{}, len(elements))
	for k, v := range elements {
		pa[k] = v.String()
	}
	return &argus.UpdateInstancePayload{
		Name:      conversion.StringValueToPointer(model.Name),
		PlanId:    conversion.StringValueToPointer(model.PlanId),
		Parameter: &pa,
	}, nil
}

func (r *instanceResource) loadPlanId(ctx context.Context, model *Model) error {
	projectId := model.ProjectId.ValueString()
	res, err := r.client.ListPlans(ctx, projectId).Execute()
	if err != nil {
		return err
	}

	planName := model.PlanName.ValueString()
	avl := ""
	plans := *res.Plans
	for i := range plans {
		p := plans[i]
		if p.Name == nil {
			continue
		}
		if strings.EqualFold(*p.Name, planName) && p.PlanId != nil {
			model.PlanId = types.StringPointerValue(p.PlanId)
			break
		}
		avl = fmt.Sprintf("%s\n- %s", avl, *p.Name)
	}
	if model.PlanId.ValueString() == "" {
		return fmt.Errorf("couldn't find plan_name '%s', available names are: %s", planName, avl)
	}
	return nil
}
