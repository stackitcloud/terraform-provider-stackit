package volumeattach

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &volumeAttachResource{}
	_ resource.ResourceWithConfigure   = &volumeAttachResource{}
	_ resource.ResourceWithImportState = &volumeAttachResource{}
)

type Model struct {
	Id        types.String `tfsdk:"id"` // needed by TF
	ProjectId types.String `tfsdk:"project_id"`
	ServerId  types.String `tfsdk:"server_id"`
	VolumeId  types.String `tfsdk:"volume_id"`
}

// NewVolumeAttachResource is a helper function to simplify the provider implementation.
func NewVolumeAttachResource() resource.Resource {
	return &volumeAttachResource{}
}

// volumeAttachResource is the resource implementation.
type volumeAttachResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *volumeAttachResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_volume_attach"
}

// Configure adds the provider configured client to the resource.
func (r *volumeAttachResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *volumeAttachResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Volume attachment resource schema. Attaches a volume to a server. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`server_id`,`volume_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the volume attachment is associated.",
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
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"volume_id": schema.StringAttribute{
				Description: "The volume ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *volumeAttachResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "server_id", serverId)
	volumeId := model.VolumeId.ValueString()
	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	// Create new Volume attachment

	payload := iaas.AddVolumeToServerPayload{
		DeleteOnTermination: sdkUtils.Ptr(false),
	}
	_, err := r.client.AddVolumeToServer(ctx, projectId, serverId, volumeId).AddVolumeToServerPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error attaching volume to server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.AddVolumeToServerWaitHandler(ctx, r.client, projectId, serverId, volumeId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error attaching volume to server", fmt.Sprintf("volume attachment waiting: %v", err))
		return
	}

	model.Id = utils.BuildInternalTerraformId(projectId, serverId, volumeId)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Volume attachment created")
}

// Read refreshes the Terraform state with the latest data.
func (r *volumeAttachResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "server_id", serverId)
	volumeId := model.VolumeId.ValueString()
	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	_, err := r.client.GetAttachedVolume(ctx, projectId, serverId, volumeId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading volume attachment", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Volume attachment read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *volumeAttachResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update is not supported, all fields require replace
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *volumeAttachResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "server_id", serverId)
	volumeId := model.VolumeId.ValueString()
	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	// Remove volume from server
	err := r.client.RemoveVolumeFromServer(ctx, projectId, serverId, volumeId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error removing volume from server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.RemoveVolumeFromServerWaitHandler(ctx, r.client, projectId, serverId, volumeId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error removing volume from server", fmt.Sprintf("volume removal waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Volume attachment deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,server_id
func (r *volumeAttachResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing volume attachment",
			fmt.Sprintf("Expected import identifier with format: [project_id],[server_id],[volume_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	serverId := idParts[1]
	volumeId := idParts[2]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "volume_id", volumeId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("volume_id"), volumeId)...)
	tflog.Info(ctx, "Volume attachment state imported")
}
