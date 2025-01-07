package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &serverResource{}
	_ resource.ResourceWithConfigure   = &serverResource{}
	_ resource.ResourceWithImportState = &serverResource{}

	supportedSourceTypes = []string{"volume", "image"}
	desiredStatusOptions = []string{modelStateActive, modelStateInactive, modelStateDeallocated}
)

const (
	modelStateActive      = "active"
	modelStateInactive    = "inactive"
	modelStateDeallocated = "deallocated"
)

type Model struct {
	Id               types.String `tfsdk:"id"` // needed by TF
	ProjectId        types.String `tfsdk:"project_id"`
	ServerId         types.String `tfsdk:"server_id"`
	MachineType      types.String `tfsdk:"machine_type"`
	Name             types.String `tfsdk:"name"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	BootVolume       types.Object `tfsdk:"boot_volume"`
	ImageId          types.String `tfsdk:"image_id"`
	KeypairName      types.String `tfsdk:"keypair_name"`
	Labels           types.Map    `tfsdk:"labels"`
	AffinityGroup    types.String `tfsdk:"affinity_group"`
	UserData         types.String `tfsdk:"user_data"`
	CreatedAt        types.String `tfsdk:"created_at"`
	LaunchedAt       types.String `tfsdk:"launched_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	DesiredStatus    types.String `tfsdk:"desired_status"`
}

// Struct corresponding to Model.BootVolume
type bootVolumeModel struct {
	PerformanceClass types.String `tfsdk:"performance_class"`
	Size             types.Int64  `tfsdk:"size"`
	SourceType       types.String `tfsdk:"source_type"`
	SourceId         types.String `tfsdk:"source_id"`
}

// Types corresponding to bootVolumeModel
var bootVolumeTypes = map[string]attr.Type{
	"performance_class": basetypes.StringType{},
	"size":              basetypes.Int64Type{},
	"source_type":       basetypes.StringType{},
	"source_id":         basetypes.StringType{},
}

// NewServerResource is a helper function to simplify the provider implementation.
func NewServerResource() resource.Resource {
	return &serverResource{}
}

// serverResource is the resource implementation.
type serverResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

// ConfigValidators validates the resource configuration
func (r *serverResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("image_id"),
			path.MatchRoot("boot_volume"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("image_id"),
			path.MatchRoot("boot_volume"),
		),
	}
}

// Configure adds the provider configured client to the resource.
func (r *serverResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !resourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_server", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	var apiClient *iaas.APIClient
	var err error
	if providerData.IaaSCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "iaas_custom_endpoint", providerData.IaaSCustomEndpoint)
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescription,
		Description:         "Server resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`server_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the server is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"server_id": schema.StringAttribute{
				Description: "The server ID.",
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
				Description: "The name of the server.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
			},
			"machine_type": schema.StringAttribute{
				MarkdownDescription: "Name of the type of the machine for the server. Possible values are documented in [Virtual machine flavors](https://docs.stackit.cloud/stackit/en/virtual-machine-flavors-75137231.html)",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zone of the server.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Optional: true,
				Computed: true,
			},
			"boot_volume": schema.SingleNestedAttribute{
				Description: "The boot volume for the server",
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"performance_class": schema.StringAttribute{
						Description: "The performance class of the server.",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
							stringvalidator.LengthAtMost(63),
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
								"must match expression"),
						},
					},
					"size": schema.Int64Attribute{
						Description: "The size of the boot volume in GB. Must be provided when `source_type` is `image`.",
						Optional:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
					},
					"source_type": schema.StringAttribute{
						Description: "The type of the source. " + utils.SupportedValuesDocumentation(supportedSourceTypes),
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"source_id": schema.StringAttribute{
						Description: "The ID of the source, either image ID or volume ID",
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"image_id": schema.StringAttribute{
				Description: "The image ID to be used for an ephemeral disk on the server.",
				Optional:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"keypair_name": schema.StringAttribute{
				Description: "The name of the keypair used during server creation.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
			"affinity_group": schema.StringAttribute{
				Description: "The affinity group the server is assigned to.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(36),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
						"must match expression"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_data": schema.StringAttribute{
				Description: "User data that is passed via cloud-init to the server.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date-time when the server was created",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"launched_at": schema.StringAttribute{
				Description: "Date-time when the server was launched",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date-time when the server was updated",
				Computed:    true,
			},
			"desired_status": schema.StringAttribute{
				Description: "The desired status of the server resource. Defaults to 'active' " + utils.SupportedValuesDocumentation(desiredStatusOptions),
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(desiredStatusOptions...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					desiredStateModifier{},
				},
			},
		},
	}
}

var _ planmodifier.String = desiredStateModifier{}

type desiredStateModifier struct {
}

// Description implements planmodifier.String.
func (d desiredStateModifier) Description(context.Context) string {
	return "validates desired state transition"
}

// MarkdownDescription implements planmodifier.String.
func (d desiredStateModifier) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

// PlanModifyString implements planmodifier.String.
func (d desiredStateModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Retrieve values from plan
	var (
		planState    types.String
		currentState types.String
	)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("desired_status"), &planState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("desired_status"), &currentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if currentState.ValueString() == modelStateDeallocated && planState.ValueString() == modelStateInactive {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error changing server state", "Server state change from deallocated to inactive is not possible")
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new server

	server, err := r.client.CreateServer(ctx, projectId).CreateServerPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	serverId := *server.Id
	server, err = wait.CreateServerWaitHandler(ctx, r.client, projectId, serverId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server", fmt.Sprintf("server creation waiting: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "server_id", serverId)

	// Map response body to schema
	err = mapFields(ctx, server, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	updateServerStatus(ctx, r.client, server.Status, model, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server created")
}

// serverControlClient provides a mockable interface for the necessary
// client operations in [updateServerStatus]
type serverControlClient interface {
	StartServerExecute(ctx context.Context, projectId string, serverId string) error
	StopServerExecute(ctx context.Context, projectId string, serverId string) error
	DeallocateServerExecute(ctx context.Context, projectId string, serverId string) error
	GetServerExecute(ctx context.Context, projectId string, serverId string) (*iaas.Server, error)
}

// updateServerStatus applies the appropriate server state changes for the actual current and the intended state
func updateServerStatus(ctx context.Context, client serverControlClient, currentState *string, model Model, diag *diag.Diagnostics) {
	if currentState == nil {
		tflog.Warn(ctx, "no current state available, not updating server state")
		return
	}
	switch *currentState {
	case wait.ServerActiveStatus:
		switch strings.ToUpper(model.DesiredStatus.ValueString()) {
		case wait.ServerInactiveStatus:
			tflog.Debug(ctx, "stopping server to enter inactive state")
			if err := client.StopServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				core.LogAndAddError(ctx, diag, "Error creating the server", fmt.Sprintf("cannot stop server: %v", err))
			}
		case wait.ServerDeallocatedStatus:
			tflog.Debug(ctx, "deallocating server to enter deallocated state")
			if err := client.DeallocateServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				core.LogAndAddError(ctx, diag, "Error creating the server", fmt.Sprintf("cannot deallocate server: %v", err))
			}
		default:
			tflog.Debug(ctx, fmt.Sprintf("nothing to do for status value %q", model.DesiredStatus.ValueString()))
		}
	case wait.ServerInactiveStatus:
		switch strings.ToUpper(model.DesiredStatus.ValueString()) {
		case wait.ServerActiveStatus:
			tflog.Debug(ctx, "starting server to enter active state")
			if err := client.StartServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				core.LogAndAddError(ctx, diag, "Error creating the server", fmt.Sprintf("cannot start server: %v", err))
			}

		case wait.ServerDeallocatedStatus:
			tflog.Debug(ctx, "deallocating server to enter deallocated state")
			if err := client.DeallocateServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				core.LogAndAddError(ctx, diag, "Error creating the server", fmt.Sprintf("cannot deallocate server: %v", err))
			}

		default:
			tflog.Debug(ctx, fmt.Sprintf("nothing to do for status value %q", model.DesiredStatus.ValueString()))
		}
	case wait.ServerDeallocatedStatus:
		switch strings.ToUpper(model.DesiredStatus.ValueString()) {
		case wait.ServerActiveStatus:
			tflog.Debug(ctx, "starting server to enter active state")
			if err := client.StartServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				core.LogAndAddError(ctx, diag, "Error creating the server", fmt.Sprintf("cannot start server: %v", err))
			}
		case wait.ServerInactiveStatus:
			tflog.Debug(ctx, "stopping server to enter inactive state")
			if err := client.StopServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				core.LogAndAddError(ctx, diag, "Error creating the server", fmt.Sprintf("cannot stop server: %v", err))
			}
		default:
			tflog.Debug(ctx, fmt.Sprintf("nothing to do for status value %q", model.DesiredStatus.ValueString()))
		}
	default:
		tflog.Debug(ctx, "not updating server state")
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 10 * time.Second
	state, err := backoff.Retry(ctx, func() (status string, err error) {
		server, err := client.GetServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString())
		if err != nil {
			return "", backoff.Permanent(err)
		}

		// state not yet available
		if server.Status == nil {
			tflog.Debug(ctx, "server state undefined")
			return "", fmt.Errorf("unknown server state")
		}

		// don't care about effective state
		if model.DesiredStatus.IsNull() || model.DesiredStatus.IsUnknown() {
			return strings.ToLower(*server.Status), nil
		}

		// require selected state, but not yet reached
		if strings.ToLower(*server.Status) != model.DesiredStatus.ValueString() {
			tflog.Debug(ctx, "target state not yet reached", map[string]any{"serverstate": *server.Status, "desired status": model.DesiredStatus})
			return "", fmt.Errorf("wrong state, expected %s but got %s", *server.Status, model.DesiredStatus)
		}

		// desired state reached
		return strings.ToLower(*server.Status), nil
	}, backoff.WithMaxElapsedTime(10*time.Minute))
	if err != nil {
		core.LogAndAddError(ctx, diag, "Error getting server status", fmt.Sprintf("cannot get server status: %v", err))
	} else {
		model.DesiredStatus = types.StringValue(state)
	}
	return
}

// // Read refreshes the Terraform state with the latest data.
func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)

	serverResp, err := r.client.GetServer(ctx, projectId, serverId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, serverResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "server read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)

	// Retrieve values from state
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		server *iaas.Server
		err    error
	)
	if server, err = r.client.GetServer(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()).Execute(); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error retrieving server state", fmt.Sprintf("Getting server state: %v", err))
	}

	// the server state must be updated before, otherwise setting the metadata might fail
	updateServerStatus(ctx, r.client, server.Status, model, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	var updatedServer *iaas.Server
	if model.DesiredStatus.ValueString() != modelStateDeallocated {
		// Update existing server
		updatedServer, err = r.client.UpdateServer(ctx, projectId, serverId).UpdateServerPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", fmt.Sprintf("Calling API: %v", err))
			return
		}
	} else {
		// we cannot update the metadata of a shelved server, read the server state again to update the model
		updatedServer, err = r.client.GetServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString())
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", fmt.Sprintf("Calling API: %v", err))
			return
		}
	}

	// Update machine type
	modelMachineType := conversion.StringValueToPointer(model.MachineType)
	if modelMachineType != nil && updatedServer.MachineType != nil && *modelMachineType != *updatedServer.MachineType {
		payload := iaas.ResizeServerPayload{
			MachineType: modelMachineType,
		}
		err := r.client.ResizeServer(ctx, projectId, serverId).ResizeServerPayload(payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", fmt.Sprintf("Resizing the server, calling API: %v", err))
		}

		_, err = wait.ResizeServerWaitHandler(ctx, r.client, projectId, serverId).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", fmt.Sprintf("server resize waiting: %v", err))
			return
		}
		// Update server model because the API doesn't return a server object as response
		updatedServer.MachineType = modelMachineType
	}

	err = mapFields(ctx, updatedServer, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "server updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)

	// Delete existing server
	err := r.client.DeleteServer(ctx, projectId, serverId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting server", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteServerWaitHandler(ctx, r.client, projectId, serverId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting server", fmt.Sprintf("server deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "server deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,server_id
func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing server",
			fmt.Sprintf("Expected import identifier with format: [project_id],[server_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	serverId := idParts[1]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverId)...)
	tflog.Info(ctx, "server state imported")
}

func mapFields(ctx context.Context, serverResp *iaas.Server, model *Model) error {
	if serverResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var serverId string
	if model.ServerId.ValueString() != "" {
		serverId = model.ServerId.ValueString()
	} else if serverResp.Id != nil {
		serverId = *serverResp.Id
	} else {
		return fmt.Errorf("server id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		serverId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	labels, diags := types.MapValueFrom(ctx, types.StringType, map[string]interface{}{})
	if diags.HasError() {
		return fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
	}
	if serverResp.Labels != nil && len(*serverResp.Labels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *serverResp.Labels)
		if diags.HasError() {
			return fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else if model.Labels.IsNull() {
		labels = types.MapNull(types.StringType)
	}
	var createdAt basetypes.StringValue
	if serverResp.CreatedAt != nil {
		createdAtValue := *serverResp.CreatedAt
		createdAt = types.StringValue(createdAtValue.Format(time.RFC3339))
	}
	var updatedAt basetypes.StringValue
	if serverResp.UpdatedAt != nil {
		updatedAtValue := *serverResp.UpdatedAt
		updatedAt = types.StringValue(updatedAtValue.Format(time.RFC3339))
	}
	var launchedAt basetypes.StringValue
	if serverResp.LaunchedAt != nil {
		launchedAtValue := *serverResp.LaunchedAt
		launchedAt = types.StringValue(launchedAtValue.Format(time.RFC3339))
	}

	model.ServerId = types.StringValue(serverId)
	model.MachineType = types.StringPointerValue(serverResp.MachineType)

	// Proposed fix: If the server is deallocated, it has no availability zone anymore
	// reactivation will then _change_ the availability zone again, causing terraform
	// to destroy and recreate the resouce, which is not intended. So we skip the zone
	// when the server is deallocated to retain the original zone until the server
	// is activated again
	if serverResp.Status != nil && *serverResp.Status != wait.ServerDeallocatedStatus {
		model.AvailabilityZone = types.StringPointerValue(serverResp.AvailabilityZone)
	}
	model.Name = types.StringPointerValue(serverResp.Name)
	model.Labels = labels
	model.ImageId = types.StringPointerValue(serverResp.ImageId)
	model.KeypairName = types.StringPointerValue(serverResp.KeypairName)
	model.AffinityGroup = types.StringPointerValue(serverResp.AffinityGroup)
	model.CreatedAt = createdAt
	model.UpdatedAt = updatedAt
	model.LaunchedAt = launchedAt

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateServerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var bootVolume = &bootVolumeModel{}
	if !(model.BootVolume.IsNull() || model.BootVolume.IsUnknown()) {
		diags := model.BootVolume.As(ctx, bootVolume, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("convert boot volume object to struct: %w", core.DiagsToError(diags))
		}
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	var bootVolumePayload *iaas.CreateServerPayloadBootVolume
	if !bootVolume.SourceId.IsNull() && !bootVolume.SourceType.IsNull() {
		bootVolumePayload = &iaas.CreateServerPayloadBootVolume{
			PerformanceClass: conversion.StringValueToPointer(bootVolume.PerformanceClass),
			Size:             conversion.Int64ValueToPointer(bootVolume.Size),
			Source: &iaas.BootVolumeSource{
				Id:   conversion.StringValueToPointer(bootVolume.SourceId),
				Type: conversion.StringValueToPointer(bootVolume.SourceType),
			},
		}
	}

	var userData *string
	if !model.UserData.IsNull() && !model.UserData.IsUnknown() {
		encodedUserData := base64.StdEncoding.EncodeToString([]byte(model.UserData.ValueString()))
		userData = &encodedUserData
	}

	return &iaas.CreateServerPayload{
		AvailabilityZone: conversion.StringValueToPointer(model.AvailabilityZone),
		BootVolume:       bootVolumePayload,
		ImageId:          conversion.StringValueToPointer(model.ImageId),
		KeypairName:      conversion.StringValueToPointer(model.KeypairName),
		Labels:           &labels,
		Name:             conversion.StringValueToPointer(model.Name),
		MachineType:      conversion.StringValueToPointer(model.MachineType),
		UserData:         userData,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.UpdateServerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.UpdateServerPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: &labels,
	}, nil
}
