package serviceaccountattach

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
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkInterfaceAttachResource{}
	_ resource.ResourceWithConfigure   = &networkInterfaceAttachResource{}
	_ resource.ResourceWithImportState = &networkInterfaceAttachResource{}
)

type Model struct {
	Id                  types.String `tfsdk:"id"` // needed by TF
	ProjectId           types.String `tfsdk:"project_id"`
	ServerId            types.String `tfsdk:"server_id"`
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
}

// NewServiceAccountAttachResource is a helper function to simplify the provider implementation.
func NewServiceAccountAttachResource() resource.Resource {
	return &networkInterfaceAttachResource{}
}

// networkInterfaceAttachResource is the resource implementation.
type networkInterfaceAttachResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *networkInterfaceAttachResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_service_account_attach"
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
			config.WithRegion(providerData.GetRegion()),
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
	description := "Service account attachment resource schema. Attaches a service account to a server. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`server_id`,`service_account_email`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the service account attachment is associated.",
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
			"service_account_email": schema.StringAttribute{
				Description: "The service account email.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)

	// Create new service account attachment
	_, err := r.client.AddServiceAccountToServer(ctx, projectId, serverId, serviceAccountEmail).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error attaching service account to server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	idParts := []string{
		projectId,
		serverId,
		serviceAccountEmail,
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
	tflog.Info(ctx, "Service account attachment created")
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
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)

	serviceAccounts, err := r.client.ListServerServiceAccounts(ctx, projectId, serverId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account attachment", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if serviceAccounts == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account attachment", "List of service accounts attached to the server is nil")
		return
	}

	if serviceAccounts.Items != nil {
		for _, mail := range *serviceAccounts.Items {
			if mail != serviceAccountEmail {
				continue
			}
			// Set refreshed state
			diags = resp.State.Set(ctx, model)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			tflog.Info(ctx, "Service account attachment read")
			return
		}
	}

	// no matching service account was found, the attachment no longer exists
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
	service_accountId := model.ServiceAccountEmail.ValueString()
	ctx = tflog.SetField(ctx, "service_account_email", service_accountId)

	// Remove service_account from server
	_, err := r.client.RemoveServiceAccountFromServer(ctx, projectId, serverId, service_accountId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error removing service account from server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Service account attachment deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,server_id
func (r *networkInterfaceAttachResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing service_account attachment",
			fmt.Sprintf("Expected import identifier with format: [project_id],[server_id],[service_account_email]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	serverId := idParts[1]
	service_accountId := idParts[2]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "service_account_email", service_accountId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_account_email"), service_accountId)...)
	tflog.Info(ctx, "Service account attachment state imported")
}
