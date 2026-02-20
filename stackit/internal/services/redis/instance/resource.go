package redis

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	redisUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/redis/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/redis"
	"github.com/stackitcloud/stackit-sdk-go/services/redis/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	InstanceId         types.String `tfsdk:"instance_id"`
	ProjectId          types.String `tfsdk:"project_id"`
	CfGuid             types.String `tfsdk:"cf_guid"`
	CfSpaceGuid        types.String `tfsdk:"cf_space_guid"`
	DashboardUrl       types.String `tfsdk:"dashboard_url"`
	ImageUrl           types.String `tfsdk:"image_url"`
	Name               types.String `tfsdk:"name"`
	CfOrganizationGuid types.String `tfsdk:"cf_organization_guid"`
	Parameters         types.Object `tfsdk:"parameters"`
	Version            types.String `tfsdk:"version"`
	PlanName           types.String `tfsdk:"plan_name"`
	PlanId             types.String `tfsdk:"plan_id"`
}

// Struct corresponding to DataSourceModel.Parameters
type parametersModel struct {
	SgwAcl                types.String `tfsdk:"sgw_acl"`
	DownAfterMilliseconds types.Int64  `tfsdk:"down_after_milliseconds"`
	EnableMonitoring      types.Bool   `tfsdk:"enable_monitoring"`
	FailoverTimeout       types.Int64  `tfsdk:"failover_timeout"`
	Graphite              types.String `tfsdk:"graphite"`
	LazyfreeLazyEviction  types.String `tfsdk:"lazyfree_lazy_eviction"`
	LazyfreeLazyExpire    types.String `tfsdk:"lazyfree_lazy_expire"`
	LuaTimeLimit          types.Int64  `tfsdk:"lua_time_limit"`
	MaxDiskThreshold      types.Int64  `tfsdk:"max_disk_threshold"`
	Maxclients            types.Int64  `tfsdk:"maxclients"`
	MaxmemoryPolicy       types.String `tfsdk:"maxmemory_policy"`
	MaxmemorySamples      types.Int64  `tfsdk:"maxmemory_samples"`
	MetricsFrequency      types.Int64  `tfsdk:"metrics_frequency"`
	MetricsPrefix         types.String `tfsdk:"metrics_prefix"`
	MinReplicasMaxLag     types.Int64  `tfsdk:"min_replicas_max_lag"`
	MonitoringInstanceId  types.String `tfsdk:"monitoring_instance_id"`
	NotifyKeyspaceEvents  types.String `tfsdk:"notify_keyspace_events"`
	Snapshot              types.String `tfsdk:"snapshot"`
	Syslog                types.List   `tfsdk:"syslog"`
	TlsCiphers            types.List   `tfsdk:"tls_ciphers"`
	TlsCiphersuites       types.String `tfsdk:"tls_ciphersuites"`
	TlsProtocols          types.String `tfsdk:"tls_protocols"`
}

// Types corresponding to parametersModel
var parametersTypes = map[string]attr.Type{
	"sgw_acl":                 basetypes.StringType{},
	"down_after_milliseconds": basetypes.Int64Type{},
	"enable_monitoring":       basetypes.BoolType{},
	"failover_timeout":        basetypes.Int64Type{},
	"graphite":                basetypes.StringType{},
	"lazyfree_lazy_eviction":  basetypes.StringType{},
	"lazyfree_lazy_expire":    basetypes.StringType{},
	"lua_time_limit":          basetypes.Int64Type{},
	"max_disk_threshold":      basetypes.Int64Type{},
	"maxclients":              basetypes.Int64Type{},
	"maxmemory_policy":        basetypes.StringType{},
	"maxmemory_samples":       basetypes.Int64Type{},
	"metrics_frequency":       basetypes.Int64Type{},
	"metrics_prefix":          basetypes.StringType{},
	"min_replicas_max_lag":    basetypes.Int64Type{},
	"monitoring_instance_id":  basetypes.StringType{},
	"notify_keyspace_events":  basetypes.StringType{},
	"snapshot":                basetypes.StringType{},
	"syslog":                  basetypes.ListType{ElemType: types.StringType},
	"tls_ciphers":             basetypes.ListType{ElemType: types.StringType},
	"tls_ciphersuites":        basetypes.StringType{},
	"tls_protocols":           basetypes.StringType{},
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client *redis.APIClient
}

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := redisUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Redis instance client configured")
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "Redis instance resource schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the Redis instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"name":        "Instance name.",
		"version":     "The service version.",
		"plan_name":   "The selected plan name.",
		"plan_id":     "The selected plan ID.",
		"parameters":  "Configuration parameters. Please note that removing a previously configured field from your Terraform configuration won't replace its value in the API. To update a previously configured field, explicitly set a new value for it.",
	}

	parametersDescriptions := map[string]string{
		"sgw_acl":                 "Comma separated list of IP networks in CIDR notation which are allowed to access this instance.",
		"down_after_milliseconds": "The number of milliseconds after which the instance is considered down.",
		"enable_monitoring":       "Enable monitoring.",
		"failover_timeout":        "The failover timeout in milliseconds.",
		"graphite":                "Graphite server URL (host and port). If set, monitoring with Graphite will be enabled.",
		"lazyfree_lazy_eviction":  "The lazy eviction enablement (yes or no).",
		"lazyfree_lazy_expire":    "The lazy expire enablement (yes or no).",
		"lua_time_limit":          "The Lua time limit.",
		"max_disk_threshold":      "The maximum disk threshold in MB. If the disk usage exceeds this threshold, the instance will be stopped.",
		"maxclients":              "The maximum number of clients.",
		"maxmemory_policy":        "The policy to handle the maximum memory (volatile-lru, noeviction, etc).",
		"maxmemory_samples":       "The maximum memory samples.",
		"metrics_frequency":       "The frequency in seconds at which metrics are emitted.",
		"metrics_prefix":          "The prefix for the metrics. Could be useful when using Graphite monitoring to prefix the metrics with a certain value, like an API key",
		"min_replicas_max_lag":    "The minimum replicas maximum lag.",
		"monitoring_instance_id":  "The ID of the STACKIT monitoring instance.",
		"notify_keyspace_events":  "The notify keyspace events.",
		"snapshot":                "The snapshot configuration.",
		"syslog":                  "List of syslog servers to send logs to.",
		"tls_ciphers":             "List of TLS ciphers to use.",
		"tls_ciphersuites":        "TLS cipher suites to use.",
		"tls_protocols":           "TLS protocol to use.",
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
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
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
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Required:    true,
			},
			"plan_name": schema.StringAttribute{
				Description: descriptions["plan_name"],
				Required:    true,
			},
			"plan_id": schema.StringAttribute{
				Description: descriptions["plan_id"],
				Computed:    true,
			},
			"parameters": schema.SingleNestedAttribute{
				Description: descriptions["parameters"],
				Attributes: map[string]schema.Attribute{
					"sgw_acl": schema.StringAttribute{
						Description: parametersDescriptions["sgw_acl"],
						Optional:    true,
						Computed:    true,
					},
					"down_after_milliseconds": schema.Int64Attribute{
						Description: parametersDescriptions["down_after_milliseconds"],
						Optional:    true,
						Computed:    true,
					},
					"enable_monitoring": schema.BoolAttribute{
						Description: parametersDescriptions["enable_monitoring"],
						Optional:    true,
						Computed:    true,
					},
					"failover_timeout": schema.Int64Attribute{
						Description: parametersDescriptions["failover_timeout"],
						Optional:    true,
						Computed:    true,
					},
					"graphite": schema.StringAttribute{
						Description: parametersDescriptions["graphite"],
						Optional:    true,
						Computed:    true,
					},
					"lazyfree_lazy_eviction": schema.StringAttribute{
						Description: parametersDescriptions["lazyfree_lazy_eviction"],
						Optional:    true,
						Computed:    true,
					},
					"lazyfree_lazy_expire": schema.StringAttribute{
						Description: parametersDescriptions["lazyfree_lazy_expire"],
						Optional:    true,
						Computed:    true,
					},
					"lua_time_limit": schema.Int64Attribute{
						Description: parametersDescriptions["lua_time_limit"],
						Optional:    true,
						Computed:    true,
					},
					"max_disk_threshold": schema.Int64Attribute{
						Description: parametersDescriptions["max_disk_threshold"],
						Optional:    true,
						Computed:    true,
					},
					"maxclients": schema.Int64Attribute{
						Description: parametersDescriptions["maxclients"],
						Optional:    true,
						Computed:    true,
					},
					"maxmemory_policy": schema.StringAttribute{
						Description: parametersDescriptions["maxmemory_policy"],
						Optional:    true,
						Computed:    true,
					},
					"maxmemory_samples": schema.Int64Attribute{
						Description: parametersDescriptions["maxmemory_samples"],
						Optional:    true,
						Computed:    true,
					},
					"metrics_frequency": schema.Int64Attribute{
						Description: parametersDescriptions["metrics_frequency"],
						Optional:    true,
						Computed:    true,
					},
					"metrics_prefix": schema.StringAttribute{
						Description: parametersDescriptions["metrics_prefix"],
						Optional:    true,
						Computed:    true,
					},
					"min_replicas_max_lag": schema.Int64Attribute{
						Description: parametersDescriptions["min_replicas_max_lag"],
						Optional:    true,
						Computed:    true,
					},
					"monitoring_instance_id": schema.StringAttribute{
						Description: parametersDescriptions["monitoring_instance_id"],
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							validate.UUID(),
							validate.NoSeparator(),
						},
					},
					"notify_keyspace_events": schema.StringAttribute{
						Description: parametersDescriptions["notify_keyspace_events"],
						Optional:    true,
						Computed:    true,
					},
					"snapshot": schema.StringAttribute{
						Description: parametersDescriptions["snapshot"],
						Optional:    true,
						Computed:    true,
					},
					"syslog": schema.ListAttribute{
						Description: parametersDescriptions["syslog"],
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
					},
					"tls_ciphers": schema.ListAttribute{
						Description: parametersDescriptions["tls_ciphers"],
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
					},
					"tls_ciphersuites": schema.StringAttribute{
						Description: parametersDescriptions["tls_ciphersuites"],
						Optional:    true,
						Computed:    true,
					},
					"tls_protocols": schema.StringAttribute{
						Description: parametersDescriptions["tls_protocols"],
						Optional:    true,
						Computed:    true,
					},
				},
				Optional: true,
				Computed: true,
			},
			"cf_guid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cf_space_guid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"image_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cf_organization_guid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	var parameters *parametersModel
	if !(model.Parameters.IsNull() || model.Parameters.IsUnknown()) {
		parameters = &parametersModel{}
		diags = model.Parameters.As(ctx, parameters, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	err := r.loadPlanId(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Loading service plan: %v", err))
		return
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, parameters)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new instance
	createResp, err := r.client.CreateInstance(ctx, projectId).CreateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp.InstanceId == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", "Got empty instance id")
		return
	}
	instanceId := *createResp.InstanceId
	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"instance_id": instanceId,
	})
	if resp.Diagnostics.HasError() {
		return
	}
	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Redis instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	instanceResp, err := r.client.GetInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusGone) {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(instanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Compute and store values not present in the API response
	err = loadPlanNameAndVersion(ctx, r.client, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Loading service plan details: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Redis instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	var parameters *parametersModel
	if !(model.Parameters.IsNull() || model.Parameters.IsUnknown()) {
		parameters = &parametersModel{}
		diags = model.Parameters.As(ctx, parameters, basetypes.ObjectAsOptions{})
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
	payload, err := toUpdatePayload(&model, parameters)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing instance
	err = r.client.PartialUpdateInstance(ctx, projectId, instanceId).PartialUpdateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.PartialUpdateInstanceWaitHandler(ctx, r.client, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Redis instance updated")
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

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// Delete existing instance
	err := r.client.DeleteInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "Redis instance deleted")
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

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"instance_id": idParts[1],
	})
	tflog.Info(ctx, "Redis instance state imported")
}

func mapFields(instance *redis.Instance, model *Model) error {
	if instance == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if instance.InstanceId != nil {
		instanceId = *instance.InstanceId
	} else {
		return fmt.Errorf("instance id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), instanceId)
	model.InstanceId = types.StringValue(instanceId)
	model.PlanId = types.StringPointerValue(instance.PlanId)
	model.CfGuid = types.StringPointerValue(instance.CfGuid)
	model.CfSpaceGuid = types.StringPointerValue(instance.CfSpaceGuid)
	model.DashboardUrl = types.StringPointerValue(instance.DashboardUrl)
	model.ImageUrl = types.StringPointerValue(instance.ImageUrl)
	model.Name = types.StringPointerValue(instance.Name)
	model.CfOrganizationGuid = types.StringPointerValue(instance.CfOrganizationGuid)

	if instance.Parameters == nil {
		model.Parameters = types.ObjectNull(parametersTypes)
	} else {
		parameters, err := mapParameters(*instance.Parameters)
		if err != nil {
			return fmt.Errorf("mapping parameters: %w", err)
		}
		model.Parameters = parameters
	}
	return nil
}

func mapParameters(params map[string]interface{}) (types.Object, error) {
	attributes := map[string]attr.Value{}
	for attribute := range parametersTypes {
		var valueInterface interface{}
		var ok bool

		// This replacement is necessary because Terraform does not allow hyphens in attribute names
		// And the API uses hyphens in some of the attribute names, which would cause a mismatch
		// The following attributes have hyphens in the API but underscores in the schema
		hyphenAttributes := []string{
			"down_after_milliseconds",
			"failover_timeout",
			"lazyfree_lazy_eviction",
			"lazyfree_lazy_expire",
			"lua_time_limit",
			"maxmemory_policy",
			"maxmemory_samples",
			"notify_keyspace_events",
			"tls_ciphers",
			"tls_ciphersuites",
			"tls_protocols",
		}
		if slices.Contains(hyphenAttributes, attribute) {
			alteredAttribute := strings.ReplaceAll(attribute, "_", "-")
			valueInterface, ok = params[alteredAttribute]
		} else {
			valueInterface, ok = params[attribute]
		}
		if !ok {
			// All fields are optional, so this is ok
			// Set the value as nil, will be handled accordingly
			valueInterface = nil
		}

		var value attr.Value
		switch parametersTypes[attribute].(type) {
		default:
			return types.ObjectNull(parametersTypes), fmt.Errorf("found unexpected attribute type '%T'", parametersTypes[attribute])
		case basetypes.StringType:
			if valueInterface == nil {
				value = types.StringNull()
			} else {
				valueString, ok := valueInterface.(string)
				if !ok {
					return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' of type %T, failed to assert as string", attribute, valueInterface)
				}
				value = types.StringValue(valueString)
			}
		case basetypes.BoolType:
			if valueInterface == nil {
				value = types.BoolNull()
			} else {
				valueBool, ok := valueInterface.(bool)
				if !ok {
					return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' of type %T, failed to assert as bool", attribute, valueInterface)
				}
				value = types.BoolValue(valueBool)
			}
		case basetypes.Int64Type:
			if valueInterface == nil {
				value = types.Int64Null()
			} else {
				// This may be int64, int32, int or float64
				// We try to assert all 4
				var valueInt64 int64
				switch temp := valueInterface.(type) {
				default:
					return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' of type %T, failed to assert as int", attribute, valueInterface)
				case int64:
					valueInt64 = temp
				case int32:
					valueInt64 = int64(temp)
				case int:
					valueInt64 = int64(temp)
				case float64:
					valueInt64 = int64(temp)
				}
				value = types.Int64Value(valueInt64)
			}
		case basetypes.ListType: // Assumed to be a list of strings
			if valueInterface == nil {
				value = types.ListNull(types.StringType)
			} else {
				// This may be []string{} or []interface{}
				// We try to assert all 2
				var valueList []attr.Value
				switch temp := valueInterface.(type) {
				default:
					return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' of type %T, failed to assert as array of interface", attribute, valueInterface)
				case []string:
					for _, x := range temp {
						valueList = append(valueList, types.StringValue(x))
					}
				case []interface{}:
					for _, x := range temp {
						xString, ok := x.(string)
						if !ok {
							return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' with element '%s' of type %T, failed to assert as string", attribute, x, x)
						}
						valueList = append(valueList, types.StringValue(xString))
					}
				}
				temp2, diags := types.ListValue(types.StringType, valueList)
				if diags.HasError() {
					return types.ObjectNull(parametersTypes), fmt.Errorf("failed to map %s: %w", attribute, core.DiagsToError(diags))
				}
				value = temp2
			}
		}
		attributes[attribute] = value
	}

	output, diags := types.ObjectValue(parametersTypes, attributes)
	if diags.HasError() {
		return types.ObjectNull(parametersTypes), fmt.Errorf("failed to create object: %w", core.DiagsToError(diags))
	}
	return output, nil
}

func toCreatePayload(model *Model, parameters *parametersModel) (*redis.CreateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payloadParams, err := toInstanceParams(parameters)
	if err != nil {
		return nil, fmt.Errorf("converting parameters: %w", err)
	}

	return &redis.CreateInstancePayload{
		InstanceName: conversion.StringValueToPointer(model.Name),
		Parameters:   payloadParams,
		PlanId:       conversion.StringValueToPointer(model.PlanId),
	}, nil
}

func toUpdatePayload(model *Model, parameters *parametersModel) (*redis.PartialUpdateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payloadParams, err := toInstanceParams(parameters)
	if err != nil {
		return nil, fmt.Errorf("converting parameters: %w", err)
	}

	return &redis.PartialUpdateInstancePayload{
		Parameters: payloadParams,
		PlanId:     conversion.StringValueToPointer(model.PlanId),
	}, nil
}

func toInstanceParams(parameters *parametersModel) (*redis.InstanceParameters, error) {
	if parameters == nil {
		return nil, nil
	}
	payloadParams := &redis.InstanceParameters{}

	payloadParams.SgwAcl = conversion.StringValueToPointer(parameters.SgwAcl)
	payloadParams.DownAfterMilliseconds = conversion.Int64ValueToPointer(parameters.DownAfterMilliseconds)
	payloadParams.EnableMonitoring = conversion.BoolValueToPointer(parameters.EnableMonitoring)
	payloadParams.FailoverTimeout = conversion.Int64ValueToPointer(parameters.FailoverTimeout)
	payloadParams.Graphite = conversion.StringValueToPointer(parameters.Graphite)
	payloadParams.LazyfreeLazyEviction = redis.InstanceParametersGetLazyfreeLazyEvictionAttributeType(conversion.StringValueToPointer(parameters.LazyfreeLazyEviction))
	payloadParams.LazyfreeLazyExpire = redis.InstanceParametersGetLazyfreeLazyExpireAttributeType(conversion.StringValueToPointer(parameters.LazyfreeLazyExpire))
	payloadParams.LuaTimeLimit = conversion.Int64ValueToPointer(parameters.LuaTimeLimit)
	payloadParams.MaxDiskThreshold = conversion.Int64ValueToPointer(parameters.MaxDiskThreshold)
	payloadParams.Maxclients = conversion.Int64ValueToPointer(parameters.Maxclients)
	payloadParams.MaxmemoryPolicy = redis.InstanceParametersGetMaxmemoryPolicyAttributeType(conversion.StringValueToPointer(parameters.MaxmemoryPolicy))
	payloadParams.MaxmemorySamples = conversion.Int64ValueToPointer(parameters.MaxmemorySamples)
	payloadParams.MetricsFrequency = conversion.Int64ValueToPointer(parameters.MetricsFrequency)
	payloadParams.MetricsPrefix = conversion.StringValueToPointer(parameters.MetricsPrefix)
	payloadParams.MinReplicasMaxLag = conversion.Int64ValueToPointer(parameters.MinReplicasMaxLag)
	payloadParams.MonitoringInstanceId = conversion.StringValueToPointer(parameters.MonitoringInstanceId)
	payloadParams.NotifyKeyspaceEvents = conversion.StringValueToPointer(parameters.NotifyKeyspaceEvents)
	payloadParams.Snapshot = conversion.StringValueToPointer(parameters.Snapshot)
	payloadParams.TlsCiphersuites = conversion.StringValueToPointer(parameters.TlsCiphersuites)
	payloadParams.TlsProtocols = redis.InstanceParametersGetTlsProtocolsAttributeType(conversion.StringValueToPointer(parameters.TlsProtocols))

	var err error
	payloadParams.Syslog, err = conversion.StringListToPointer(parameters.Syslog)
	if err != nil {
		return nil, fmt.Errorf("converting syslog: %w", err)
	}

	payloadParams.TlsCiphers, err = conversion.StringListToPointer(parameters.TlsCiphers)
	if err != nil {
		return nil, fmt.Errorf("converting tls_ciphers: %w", err)
	}

	return payloadParams, nil
}

func (r *instanceResource) loadPlanId(ctx context.Context, model *Model) error {
	projectId := model.ProjectId.ValueString()
	res, err := r.client.ListOfferings(ctx, projectId).Execute()
	if err != nil {
		return fmt.Errorf("getting Redis offerings: %w", err)
	}

	version := model.Version.ValueString()
	planName := model.PlanName.ValueString()
	availableVersions := ""
	availablePlanNames := ""
	isValidVersion := false
	for _, offer := range *res.Offerings {
		if !strings.EqualFold(*offer.Version, version) {
			availableVersions = fmt.Sprintf("%s\n- %s", availableVersions, *offer.Version)
			continue
		}
		isValidVersion = true

		for _, plan := range *offer.Plans {
			if plan.Name == nil {
				continue
			}
			if strings.EqualFold(*plan.Name, planName) && plan.Id != nil {
				model.PlanId = types.StringPointerValue(plan.Id)
				return nil
			}
			availablePlanNames = fmt.Sprintf("%s\n- %s", availablePlanNames, *plan.Name)
		}
	}

	if !isValidVersion {
		return fmt.Errorf("couldn't find version '%s', available versions are: %s", version, availableVersions)
	}
	return fmt.Errorf("couldn't find plan_name '%s' for version %s, available names are: %s", planName, version, availablePlanNames)
}

func loadPlanNameAndVersion(ctx context.Context, client *redis.APIClient, model *Model) error {
	projectId := model.ProjectId.ValueString()
	planId := model.PlanId.ValueString()
	res, err := client.ListOfferings(ctx, projectId).Execute()
	if err != nil {
		return fmt.Errorf("getting Redis offerings: %w", err)
	}

	for _, offer := range *res.Offerings {
		for _, plan := range *offer.Plans {
			if strings.EqualFold(*plan.Id, planId) && plan.Id != nil {
				model.PlanName = types.StringPointerValue(plan.Name)
				model.Version = types.StringPointerValue(offer.Version)
				return nil
			}
		}
	}

	return fmt.Errorf("couldn't find plan_name and version for plan_id '%s'", planId)
}
