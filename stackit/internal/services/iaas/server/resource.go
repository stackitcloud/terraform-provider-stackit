package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

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
	Id                types.String `tfsdk:"id"` // needed by TF
	ProjectId         types.String `tfsdk:"project_id"`
	ServerId          types.String `tfsdk:"server_id"`
	MachineType       types.String `tfsdk:"machine_type"`
	Name              types.String `tfsdk:"name"`
	AvailabilityZone  types.String `tfsdk:"availability_zone"`
	BootVolume        types.Object `tfsdk:"boot_volume"`
	ImageId           types.String `tfsdk:"image_id"`
	NetworkInterfaces types.List   `tfsdk:"network_interfaces"`
	KeypairName       types.String `tfsdk:"keypair_name"`
	Labels            types.Map    `tfsdk:"labels"`
	AffinityGroup     types.String `tfsdk:"affinity_group"`
	UserData          types.String `tfsdk:"user_data"`
	CreatedAt         types.String `tfsdk:"created_at"`
	LaunchedAt        types.String `tfsdk:"launched_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
	DesiredStatus     types.String `tfsdk:"desired_status"`
}

// Struct corresponding to Model.BootVolume
type bootVolumeModel struct {
	Id                  types.String `tfsdk:"id"`
	PerformanceClass    types.String `tfsdk:"performance_class"`
	Size                types.Int64  `tfsdk:"size"`
	SourceType          types.String `tfsdk:"source_type"`
	SourceId            types.String `tfsdk:"source_id"`
	DeleteOnTermination types.Bool   `tfsdk:"delete_on_termination"`
}

// Types corresponding to bootVolumeModel
var bootVolumeTypes = map[string]attr.Type{
	"performance_class":     basetypes.StringType{},
	"size":                  basetypes.Int64Type{},
	"source_type":           basetypes.StringType{},
	"source_id":             basetypes.StringType{},
	"delete_on_termination": basetypes.BoolType{},
	"id":                    basetypes.StringType{},
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

func (r serverResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// convert boot volume model
	var bootVolume = &bootVolumeModel{}
	if !(model.BootVolume.IsNull() || model.BootVolume.IsUnknown()) {
		diags := model.BootVolume.As(ctx, bootVolume, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return
		}
	}

	if !bootVolume.DeleteOnTermination.IsUnknown() && !bootVolume.DeleteOnTermination.IsNull() && !bootVolume.SourceType.IsUnknown() && !bootVolume.SourceType.IsNull() {
		if bootVolume.SourceType != types.StringValue("image") {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring server", "You can only provide `delete_on_termination` for `source_type` `image`.")
		}
	}
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
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
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
					"id": schema.StringAttribute{
						Description: "The ID of the boot volume",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
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
						Description: "The type of the source. " + utils.FormatPossibleValues(supportedSourceTypes...),
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
					"delete_on_termination": schema.BoolAttribute{
						Description: "Delete the volume during the termination of the server. Only allowed when `source_type` is `image`.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
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
			"network_interfaces": schema.ListAttribute{
				Description: "The IDs of network interfaces which should be attached to the server. Updating it will recreate the server.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						validate.UUID(),
						validate.NoSeparator(),
					),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
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
				Description: "The desired status of the server resource. " + utils.FormatPossibleValues(desiredStatusOptions...),
				Optional:    true,
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
func (d desiredStateModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) { //nolint: gocritic //signature is defined by terraform api
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
	_, err = wait.CreateServerWaitHandler(ctx, r.client, projectId, serverId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server", fmt.Sprintf("server creation waiting: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "server_id", serverId)

	// Get Server with details
	serverReq := r.client.GetServer(ctx, projectId, serverId)
	serverReq = serverReq.Details(true)
	server, err = serverReq.Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server", fmt.Sprintf("get server details: %v", err))
	}

	// Map response body to schema
	err = mapFields(ctx, server, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating server", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	if err := updateServerStatus(ctx, r.client, server.Status, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creting server", fmt.Sprintf("update server state: %v", err))
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
	wait.APIClientInterface
	StartServerExecute(ctx context.Context, projectId string, serverId string) error
	StopServerExecute(ctx context.Context, projectId string, serverId string) error
	DeallocateServerExecute(ctx context.Context, projectId string, serverId string) error
}

func startServer(ctx context.Context, client serverControlClient, projectId, serverId string) error {
	tflog.Debug(ctx, "starting server to enter active state")
	if err := client.StartServerExecute(ctx, projectId, serverId); err != nil {
		return fmt.Errorf("cannot start server: %w", err)
	}
	_, err := wait.StartServerWaitHandler(ctx, client, projectId, serverId).WaitWithContext(ctx)
	if err != nil {
		return fmt.Errorf("cannot check started server: %w", err)
	}
	return nil
}

func stopServer(ctx context.Context, client serverControlClient, projectId, serverId string) error {
	tflog.Debug(ctx, "stopping server to enter inactive state")
	if err := client.StopServerExecute(ctx, projectId, serverId); err != nil {
		return fmt.Errorf("cannot stop server: %w", err)
	}
	_, err := wait.StopServerWaitHandler(ctx, client, projectId, serverId).WaitWithContext(ctx)
	if err != nil {
		return fmt.Errorf("cannot check stopped server: %w", err)
	}
	return nil
}

func deallocatServer(ctx context.Context, client serverControlClient, projectId, serverId string) error {
	tflog.Debug(ctx, "deallocating server to enter shelved state")
	if err := client.DeallocateServerExecute(ctx, projectId, serverId); err != nil {
		return fmt.Errorf("cannot deallocate server: %w", err)
	}
	_, err := wait.DeallocateServerWaitHandler(ctx, client, projectId, serverId).WaitWithContext(ctx)
	if err != nil {
		return fmt.Errorf("cannot check deallocated server: %w", err)
	}
	return nil
}

// updateServerStatus applies the appropriate server state changes for the actual current and the intended state
func updateServerStatus(ctx context.Context, client serverControlClient, currentState *string, model *Model) error {
	if currentState == nil {
		tflog.Warn(ctx, "no current state available, not updating server state")
		return nil
	}
	switch *currentState {
	case wait.ServerActiveStatus:
		switch strings.ToUpper(model.DesiredStatus.ValueString()) {
		case wait.ServerInactiveStatus:
			if err := stopServer(ctx, client, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				return err
			}

		case wait.ServerDeallocatedStatus:

			if err := deallocatServer(ctx, client, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				return err
			}
		default:
			tflog.Debug(ctx, fmt.Sprintf("nothing to do for status value %q", model.DesiredStatus.ValueString()))
			if _, err := client.GetServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				return err
			}
		}
	case wait.ServerInactiveStatus:
		switch strings.ToUpper(model.DesiredStatus.ValueString()) {
		case wait.ServerActiveStatus:
			if err := startServer(ctx, client, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				return err
			}
		case wait.ServerDeallocatedStatus:
			if err := deallocatServer(ctx, client, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				return err
			}

		default:
			tflog.Debug(ctx, fmt.Sprintf("nothing to do for status value %q", model.DesiredStatus.ValueString()))
			if _, err := client.GetServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				return err
			}
		}
	case wait.ServerDeallocatedStatus:
		switch strings.ToUpper(model.DesiredStatus.ValueString()) {
		case wait.ServerActiveStatus:
			if err := startServer(ctx, client, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				return err
			}

		case wait.ServerInactiveStatus:
			if err := stopServer(ctx, client, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				return err
			}
		default:
			tflog.Debug(ctx, fmt.Sprintf("nothing to do for status value %q", model.DesiredStatus.ValueString()))
			if _, err := client.GetServerExecute(ctx, model.ProjectId.ValueString(), model.ServerId.ValueString()); err != nil {
				return err
			}
		}
	default:
		tflog.Debug(ctx, "not updating server state")
	}

	return nil
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

	serverReq := r.client.GetServer(ctx, projectId, serverId)
	serverReq = serverReq.Details(true)
	serverResp, err := serverReq.Execute()
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

func (r *serverResource) updateServerAttributes(ctx context.Context, model, stateModel *Model) (*iaas.Server, error) {
	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, model, stateModel.Labels)
	if err != nil {
		return nil, fmt.Errorf("Creating API payload: %w", err)
	}
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()

	var updatedServer *iaas.Server
	// Update existing server
	updatedServer, err = r.client.UpdateServer(ctx, projectId, serverId).UpdateServerPayload(*payload).Execute()
	if err != nil {
		return nil, fmt.Errorf("Calling API: %w", err)
	}

	// Update machine type
	modelMachineType := conversion.StringValueToPointer(model.MachineType)
	if modelMachineType != nil && updatedServer.MachineType != nil && *modelMachineType != *updatedServer.MachineType {
		payload := iaas.ResizeServerPayload{
			MachineType: modelMachineType,
		}
		err := r.client.ResizeServer(ctx, projectId, serverId).ResizeServerPayload(payload).Execute()
		if err != nil {
			return nil, fmt.Errorf("Resizing the server, calling API: %w", err)
		}

		_, err = wait.ResizeServerWaitHandler(ctx, r.client, projectId, serverId).WaitWithContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("server resize waiting: %w", err)
		}
		// Update server model because the API doesn't return a server object as response
		updatedServer.MachineType = modelMachineType
	}
	return updatedServer, nil
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

	if model.DesiredStatus.ValueString() == modelStateDeallocated {
		// if the target state is "deallocated", we have to perform the server update first
		// and then shelve it afterwards. A shelved server cannot be updated
		_, err = r.updateServerAttributes(ctx, &model, &stateModel)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", err.Error())
			return
		}

		if err := updateServerStatus(ctx, r.client, server.Status, &model); err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", err.Error())
			return
		}
	} else {
		// potentially unfreeze first and update afterwards
		if err := updateServerStatus(ctx, r.client, server.Status, &model); err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", err.Error())
			return
		}

		_, err = r.updateServerAttributes(ctx, &model, &stateModel)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", err.Error())
			return
		}
	}

	// Re-fetch the server data, to get the details values.
	serverReq := r.client.GetServer(ctx, projectId, serverId)
	serverReq = serverReq.Details(true)
	updatedServer, err := serverReq.Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", fmt.Sprintf("Calling API: %v", err))
		return
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), serverId)

	labels, err := iaasUtils.MapLabels(ctx, serverResp.Labels, model.Labels)
	if err != nil {
		return err
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
	if serverResp.Nics != nil {
		var respNics []string
		for _, nic := range *serverResp.Nics {
			respNics = append(respNics, *nic.NicId)
		}

		var modelNics []string
		for _, modelNic := range model.NetworkInterfaces.Elements() {
			modelNicString, ok := modelNic.(types.String)
			if !ok {
				return fmt.Errorf("type assertion for network interfaces failed")
			}
			modelNics = append(modelNics, modelNicString.ValueString())
		}

		var filteredNics []string
		for _, modelNic := range modelNics {
			for _, nic := range respNics {
				if nic == modelNic {
					filteredNics = append(filteredNics, nic)
					break
				}
			}
		}

		// Sorts the filteredNics based on the modelNics order
		resultNics := utils.ReconcileStringSlices(modelNics, filteredNics)

		if len(resultNics) != 0 {
			nicTF, diags := types.ListValueFrom(ctx, types.StringType, resultNics)
			if diags.HasError() {
				return fmt.Errorf("failed to map networkInterfaces: %w", core.DiagsToError(diags))
			}

			model.NetworkInterfaces = nicTF
		} else {
			model.NetworkInterfaces = types.ListNull(types.StringType)
		}
	} else {
		model.NetworkInterfaces = types.ListNull(types.StringType)
	}

	if serverResp.BootVolume != nil {
		// convert boot volume model
		var bootVolumeModel = &bootVolumeModel{}
		if !(model.BootVolume.IsNull() || model.BootVolume.IsUnknown()) {
			diags := model.BootVolume.As(ctx, bootVolumeModel, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return fmt.Errorf("failed to map bootVolume: %w", core.DiagsToError(diags))
			}
		}

		// Only the id and delete_on_termination is returned via response.
		// Take the other values from the model.
		bootVolume, diags := types.ObjectValue(bootVolumeTypes, map[string]attr.Value{
			"id":                    types.StringPointerValue(serverResp.BootVolume.Id),
			"delete_on_termination": types.BoolPointerValue(serverResp.BootVolume.DeleteOnTermination),
			"source_id":             bootVolumeModel.SourceId,
			"size":                  bootVolumeModel.Size,
			"source_type":           bootVolumeModel.SourceType,
			"performance_class":     bootVolumeModel.PerformanceClass,
		})
		if diags.HasError() {
			return fmt.Errorf("failed to map bootVolume: %w", core.DiagsToError(diags))
		}
		model.BootVolume = bootVolume
	} else {
		model.BootVolume = types.ObjectNull(bootVolumeTypes)
	}

	model.ServerId = types.StringValue(serverId)
	model.MachineType = types.StringPointerValue(serverResp.MachineType)

	// Proposed fix: If the server is deallocated, it has no availability zone anymore
	// reactivation will then _change_ the availability zone again, causing terraform
	// to destroy and recreate the resource, which is not intended. So we skip the zone
	// when the server is deallocated to retain the original zone until the server
	// is activated again
	if serverResp.Status != nil && *serverResp.Status != wait.ServerDeallocatedStatus {
		model.AvailabilityZone = types.StringPointerValue(serverResp.AvailabilityZone)
	}

	if serverResp.UserData != nil && len(*serverResp.UserData) > 0 {
		model.UserData = types.StringValue(string(*serverResp.UserData))
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
		if !bootVolume.DeleteOnTermination.IsNull() && !bootVolume.DeleteOnTermination.IsUnknown() && bootVolume.DeleteOnTermination.ValueBool() {
			// it is set and true, adjust payload
			bootVolumePayload.DeleteOnTermination = conversion.BoolValueToPointer(bootVolume.DeleteOnTermination)
		}
	}

	var userData *[]byte
	if !model.UserData.IsNull() && !model.UserData.IsUnknown() {
		src := []byte(model.UserData.ValueString())
		encodedUserData := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
		base64.StdEncoding.Encode(encodedUserData, src)
		userData = &encodedUserData
	}

	var network *iaas.CreateServerPayloadNetworking
	if !model.NetworkInterfaces.IsNull() && !model.NetworkInterfaces.IsUnknown() {
		var nicIds []string
		for _, nic := range model.NetworkInterfaces.Elements() {
			nicString, ok := nic.(types.String)
			if !ok {
				return nil, fmt.Errorf("type assertion failed")
			}
			nicIds = append(nicIds, nicString.ValueString())
		}

		network = &iaas.CreateServerPayloadNetworking{
			CreateServerNetworkingWithNics: &iaas.CreateServerNetworkingWithNics{
				NicIds: &nicIds,
			},
		}
	}

	return &iaas.CreateServerPayload{
		AffinityGroup:    conversion.StringValueToPointer(model.AffinityGroup),
		AvailabilityZone: conversion.StringValueToPointer(model.AvailabilityZone),
		BootVolume:       bootVolumePayload,
		ImageId:          conversion.StringValueToPointer(model.ImageId),
		KeypairName:      conversion.StringValueToPointer(model.KeypairName),
		Labels:           &labels,
		Name:             conversion.StringValueToPointer(model.Name),
		Networking:       network,
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
