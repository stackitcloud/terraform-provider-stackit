package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha/wait"
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

	SupportedSourceTypes = []string{"volume", "image"}
)

type Model struct {
	Id               types.String `tfsdk:"id"` // needed by TF
	ProjectId        types.String `tfsdk:"project_id"`
	ServerId         types.String `tfsdk:"server_id"`
	MachineType      types.String `tfsdk:"machine_type"`
	Name             types.String `tfsdk:"name"`
	InitialNetwork   types.Object `tfsdk:"initial_network"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	BootVolume       types.Object `tfsdk:"boot_volume"`
	ImageId          types.String `tfsdk:"image_id"`
	KeypairName      types.String `tfsdk:"keypair_name"`
	Labels           types.Map    `tfsdk:"labels"`
	ServerGroup      types.String `tfsdk:"server_group"`
	UserData         types.String `tfsdk:"user_data"`
	CreatedAt        types.String `tfsdk:"created_at"`
	LaunchedAt       types.String `tfsdk:"launched_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

// Struct corresponding to Model.InitialNetwork
type initialNetworkModel struct {
	NetworkId           types.String `tfsdk:"network_id"`
	NetworkInterfaceIds types.List   `tfsdk:"network_interface_ids"`
}

// Types corresponding to initialNetworkModel
var initialNetworkTypes = map[string]attr.Type{
	"network_id":            basetypes.StringType{},
	"network_interface_ids": basetypes.ListType{ElemType: types.StringType},
}

// Struct corresponding to Model.BootVolume
type bootVolumeModel struct {
	DeleteOnTermination types.Bool   `tfsdk:"delete_on_termination"`
	PerformanceClass    types.String `tfsdk:"performance_class"`
	Size                types.Int64  `tfsdk:"size"`
	SourceType          types.String `tfsdk:"source_type"`
	SourceId            types.String `tfsdk:"source_id"`
}

// Types corresponding to bootVolumeModel
var bootVolumeTypes = map[string]attr.Type{
	"delete_on_termination": basetypes.BoolType{},
	"performance_class":     basetypes.StringType{},
	"size":                  basetypes.Int64Type{},
	"source_type":           basetypes.StringType{},
	"source_id":             basetypes.StringType{},
}

// NewServerResource is a helper function to simplify the provider implementation.
func NewServerResource() resource.Resource {
	return &serverResource{}
}

// serverResource is the resource implementation.
type serverResource struct {
	client *iaasalpha.APIClient
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

	var apiClient *iaasalpha.APIClient
	var err error
	if providerData.IaaSCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "iaas_custom_endpoint", providerData.IaaSCustomEndpoint)
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "iaasalpha client configured")
}

// Schema defines the schema for the resource.
func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Server resource schema. Must have a `region` specified in the provider configuration."),
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
				Description: "Name of the machine type the server will belong to.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
			},
			"initial_network": schema.SingleNestedAttribute{
				Description: "The initial networking setup for the server. A network ID or a list of network interfaces IDs can be provided",
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"network_id": schema.StringAttribute{
						Description: "The network ID",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.ConflictsWith(
								path.MatchRoot("initial_networking.network_interface_ids"),
							),
						},
					},
					"network_interface_ids": schema.ListAttribute{
						Description: "List of network interface IDs",
						Optional:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.RequiresReplace(),
						},
						Validators: []validator.List{
							listvalidator.ConflictsWith(
								path.MatchRoot("initial_networking.network_id"),
							),
						},
					},
				},
				Validators: []validator.Object{
					objectvalidator.AtLeastOneOf(
						path.MatchRoot("initial_networking.network_id"),
						path.MatchRoot("initial_networking.network_interface_ids"),
					),
				},
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zone of the server.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
					"delete_on_termination": schema.BoolAttribute{
						Description: "Delete the volume during the termination of the server. Defaults to `false`.",
						Optional:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
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
						Description: "The type of the source. " + utils.SupportedValuesDocumentation(SupportedSourceTypes),
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
			"server_group": schema.StringAttribute{
				Description: "The server group the server is assigned to.",
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
				Description: "User data that is provided to the server. Must be base64 encoded and is passed via cloud-init to the server.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date-time when the server was created",
				Computed:    true,
			},
			"launched_at": schema.StringAttribute{
				Description: "Date-time when the server was launched",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date-time when the server was updated",
				Computed:    true,
			},
		},
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
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Server created")
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

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing server
	updatedServer, err := r.client.V1alpha1UpdateServer(ctx, projectId, serverId).V1alpha1UpdateServerPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Update machine type
	modelMachineType := conversion.StringValueToPointer(model.MachineType)
	if modelMachineType != nil && updatedServer.MachineType != nil && *modelMachineType != *updatedServer.MachineType {
		payload := iaasalpha.ResizeServerPayload{
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

func mapFields(ctx context.Context, serverResp *iaasalpha.Server, model *Model) error {
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
		return fmt.Errorf("Server id not present")
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
	model.AvailabilityZone = types.StringPointerValue(serverResp.AvailabilityZone)
	model.Name = types.StringPointerValue(serverResp.Name)
	model.Labels = labels
	model.ImageId = types.StringPointerValue(serverResp.Image)
	model.KeypairName = types.StringPointerValue(serverResp.Keypair)
	model.ServerGroup = types.StringPointerValue(serverResp.ServerGroup)
	model.CreatedAt = createdAt
	model.UpdatedAt = updatedAt
	model.LaunchedAt = launchedAt
	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaasalpha.CreateServerPayload, error) {
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

	var initialNetwork = &initialNetworkModel{}
	if !(model.InitialNetwork.IsNull() || model.InitialNetwork.IsUnknown()) {
		diags := model.InitialNetwork.As(ctx, initialNetwork, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("convert initial network object to struct: %w", core.DiagsToError(diags))
		}
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	var bootVolumePayload *iaasalpha.CreateServerPayloadBootVolume
	if !bootVolume.SourceId.IsNull() && !bootVolume.SourceType.IsNull() {
		bootVolumePayload = &iaasalpha.CreateServerPayloadBootVolume{
			DeleteOnTermination: conversion.BoolValueToPointer(bootVolume.DeleteOnTermination),
			PerformanceClass:    conversion.StringValueToPointer(bootVolume.PerformanceClass),
			Size:                conversion.Int64ValueToPointer(bootVolume.Size),
			Source: &iaasalpha.BootVolumeSource{
				Id:   conversion.StringValueToPointer(bootVolume.SourceId),
				Type: conversion.StringValueToPointer(bootVolume.SourceType),
			},
		}
	}

	var initialNetworkPayload *iaasalpha.CreateServerPayloadNetworking
	if !initialNetwork.NetworkId.IsNull() {
		initialNetworkPayload = &iaasalpha.CreateServerPayloadNetworking{
			CreateServerNetworking: &iaasalpha.CreateServerNetworking{
				NetworkId: conversion.StringValueToPointer(initialNetwork.NetworkId),
			},
		}
	} else if !initialNetwork.NetworkInterfaceIds.IsNull() {
		nicIds, err := conversion.StringListToPointer(initialNetwork.NetworkInterfaceIds)
		if err != nil {
			return nil, fmt.Errorf("converting list of network interface IDs to string list pointer: %w", err)
		}
		initialNetworkPayload = &iaasalpha.CreateServerPayloadNetworking{
			CreateServerNetworkingWithNics: &iaasalpha.CreateServerNetworkingWithNics{
				NicIds: nicIds,
			},
		}
	}

	var userData *string
	if !model.UserData.IsNull() && !model.UserData.IsUnknown() {
		encodedUserData := base64.StdEncoding.EncodeToString([]byte(model.UserData.ValueString()))
		userData = &encodedUserData
	}

	return &iaasalpha.CreateServerPayload{
		AvailabilityZone: conversion.StringValueToPointer(model.AvailabilityZone),
		BootVolume:       bootVolumePayload,
		Image:            conversion.StringValueToPointer(model.ImageId),
		Keypair:          conversion.StringValueToPointer(model.KeypairName),
		Networking:       initialNetworkPayload,
		Labels:           &labels,
		Name:             conversion.StringValueToPointer(model.Name),
		MachineType:      conversion.StringValueToPointer(model.MachineType),
		UserData:         userData,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaasalpha.V1alpha1UpdateServerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaasalpha.V1alpha1UpdateServerPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: &labels,
	}, nil
}
