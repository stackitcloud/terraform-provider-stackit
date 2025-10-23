package networkinterfaceattach

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkInterfaceAttachResource{}
	_ resource.ResourceWithConfigure   = &networkInterfaceAttachResource{}
	_ resource.ResourceWithImportState = &networkInterfaceAttachResource{}
	_ resource.ResourceWithModifyPlan  = &networkInterfaceAttachResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	ProjectId          types.String `tfsdk:"project_id"`
	Region             types.String `tfsdk:"region"`
	ServerId           types.String `tfsdk:"server_id"`
	NetworkInterfaceId types.String `tfsdk:"network_interface_id"`
}

// NewNetworkInterfaceAttachResource is a helper function to simplify the provider implementation.
func NewNetworkInterfaceAttachResource() resource.Resource {
	return &networkInterfaceAttachResource{}
}

// networkInterfaceAttachResource is the resource implementation.
type networkInterfaceAttachResource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *networkInterfaceAttachResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_network_interface_attach"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *networkInterfaceAttachResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	// skip initial empty configuration to avoid follow-up errors
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

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *networkInterfaceAttachResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *networkInterfaceAttachResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Network interface attachment resource schema. Attaches a network interface to a server. The attachment only takes full effect after server reboot."
	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`server_id`,`network_interface_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the network interface attachment is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
			"network_interface_id": schema.StringAttribute{
				Description: "The network interface ID.",
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
func (r *networkInterfaceAttachResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	serverId := model.ServerId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	// Create new network interface attachment
	err := r.client.AddNicToServer(ctx, projectId, region, serverId, networkInterfaceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error attaching network interface to server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	model.Id = utils.BuildInternalTerraformId(projectId, region, serverId, networkInterfaceId)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network interface attachment created")
}

// Read refreshes the Terraform state with the latest data.
func (r *networkInterfaceAttachResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	serverId := model.ServerId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	nics, err := r.client.ListServerNICs(ctx, projectId, region, serverId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network interface attachment", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if nics == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network interface attachment", "List of network interfaces attached to the server is nil")
		return
	}

	if nics.Items != nil {
		for _, nic := range *nics.Items {
			if nic.Id == nil || (nic.Id != nil && *nic.Id != networkInterfaceId) {
				continue
			}
			// Set refreshed state
			diags = resp.State.Set(ctx, model)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			tflog.Info(ctx, "Network interface attachment read")
			return
		}
	}

	// no matching network interface was found, the attachment no longer exists
	resp.State.RemoveResource(ctx)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkInterfaceAttachResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update is not supported, all fields require replace
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkInterfaceAttachResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	serverId := model.ServerId.ValueString()
	network_interfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "network_interface_id", network_interfaceId)

	// Remove network_interface from server
	err := r.client.RemoveNicFromServer(ctx, projectId, region, serverId, network_interfaceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error removing network interface from server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Network interface attachment deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,server_id
func (r *networkInterfaceAttachResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network_interface attachment",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[server_id],[network_interface_id]  Got: %q", req.ID),
		)
		return
	}

	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id":           idParts[0],
		"region":               idParts[1],
		"server_id":            idParts[2],
		"network_interface_id": idParts[3],
	})

	tflog.Info(ctx, "Network interface attachment state imported")
}
