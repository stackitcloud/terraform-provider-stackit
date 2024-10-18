package networkinterfaceattach

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkInterfaceAttachResource{}
	_ resource.ResourceWithConfigure   = &networkInterfaceAttachResource{}
	_ resource.ResourceWithImportState = &networkInterfaceAttachResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	ProjectId          types.String `tfsdk:"project_id"`
	ServerId           types.String `tfsdk:"server_id"`
	NetworkInterfaceId types.String `tfsdk:"network_interface_id"`
}

// NewNetworkInterfaceAttachResource is a helper function to simplify the provider implementation.
func NewNetworkInterfaceAttachResource() resource.Resource {
	return &networkInterfaceAttachResource{}
}

// networkInterfaceAttachResource is the resource implementation.
type networkInterfaceAttachResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *networkInterfaceAttachResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_network_interface_attach"
}

// Configure adds the provider configured client to the resource.
func (r *networkInterfaceAttachResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_server_network_interface_attach", "resource")
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
func (r *networkInterfaceAttachResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Network interface attachment resource schema. Attaches a network interface to a server. Must have a `region` specified in the provider configuration."),
		Description:         "Network interface attachment resource schema. Attaches a network interface to a server. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`server_id`,`network_interface_id`\".",
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

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "server_id", serverId)
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	// Create new network interface attachment
	err := r.client.AddNICToServer(ctx, projectId, serverId, networkInterfaceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error attaching network interface to server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	idParts := []string{
		projectId,
		serverId,
		networkInterfaceId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

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
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "server_id", serverId)
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	nics, err := r.client.ListServerNICs(ctx, projectId, serverId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network interface attachment", fmt.Sprintf("Calling API: %v", err))
		return
	}

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

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "server_id", serverId)
	network_interfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "network_interface_id", network_interfaceId)

	// Remove network_interface from server
	err := r.client.RemoveNICFromServer(ctx, projectId, serverId, network_interfaceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error removing network interface from server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Network interface attachment deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,server_id
func (r *networkInterfaceAttachResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network_interface attachment",
			fmt.Sprintf("Expected import identifier with format: [project_id],[server_id],[network_interface_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	serverId := idParts[1]
	network_interfaceId := idParts[2]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "network_interface_id", network_interfaceId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_interface_id"), network_interfaceId)...)
	tflog.Info(ctx, "Network interface attachment state imported")
}
