package publicipassociate

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
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &publicIpAssociateResource{}
	_ resource.ResourceWithConfigure   = &publicIpAssociateResource{}
	_ resource.ResourceWithImportState = &publicIpAssociateResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	ProjectId          types.String `tfsdk:"project_id"`
	PublicIpId         types.String `tfsdk:"public_ip_id"`
	Ip                 types.String `tfsdk:"ip"`
	NetworkInterfaceId types.String `tfsdk:"network_interface_id"`
}

// NewPublicIpAssociateResource is a helper function to simplify the provider implementation.
func NewPublicIpAssociateResource() resource.Resource {
	return &publicIpAssociateResource{}
}

// publicIpAssociateResource is the resource implementation.
type publicIpAssociateResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *publicIpAssociateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_ip_associate"
}

// Configure adds the provider configured client to the resource.
func (r *publicIpAssociateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	core.LogAndAddWarning(ctx, &resp.Diagnostics, "The `stackit_public_ip_associate` resource should not be used together with the `stackit_public_ip` resource for the same public IP or for the same network interface.",
		"Using both resources together for the same public IP or network interface WILL lead to conflicts, as they both have control of the public IP and network interface association.")

	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *publicIpAssociateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main": "Associates an existing public IP to a network interface. " +
			"This is useful for situations where you have a pre-allocated public IP or unable to use the `stackit_public_ip` resource to create a new public IP. " +
			"Must have a `region` specified in the provider configuration.",
		"warning_message": "The `stackit_public_ip_associate` resource should not be used together with the `stackit_public_ip` resource for the same public IP or for the same network interface. \n" +
			"Using both resources together for the same public IP or network interface WILL lead to conflicts, as they both have control of the public IP and network interface association.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("%s\n\n!> %s", descriptions["main"], descriptions["warning_message"]),
		Description:         fmt.Sprintf("%s\n\n%s", descriptions["main"], descriptions["warning_message"]),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`public_ip_id`,`network_interface_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the public IP is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"public_ip_id": schema.StringAttribute{
				Description: "The public IP ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"ip": schema.StringAttribute{
				Description: "The IP address.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.IP(false),
				},
			},
			"network_interface_id": schema.StringAttribute{
				Description: "The ID of the network interface (or virtual IP) to which the public IP should be attached to.",
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
func (r *publicIpAssociateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	publicIpId := model.PublicIpId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "public_ip_id", publicIpId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error associating public IP to network interface", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing public IP
	updatedPublicIp, err := r.client.UpdatePublicIP(ctx, projectId, publicIpId).UpdatePublicIPPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error associating public IP to network interface", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(updatedPublicIp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error associating public IP to network interface", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "public IP associated to network interface")
}

// Read refreshes the Terraform state with the latest data.
func (r *publicIpAssociateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	publicIpId := model.PublicIpId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "public_ip_id", publicIpId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	publicIpResp, err := r.client.GetPublicIP(ctx, projectId, publicIpId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading public IP association", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(publicIpResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading public IP association", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "public IP associate read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *publicIpAssociateResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update is not supported, all fields require replace
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *publicIpAssociateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	publicIpId := model.PublicIpId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "public_ip_id", publicIpId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	payload := &iaas.UpdatePublicIPPayload{
		NetworkInterface: iaas.NewNullableString(nil),
	}

	_, err := r.client.UpdatePublicIP(ctx, projectId, publicIpId).UpdatePublicIPPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting public IP association", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "public IP association deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,public_ip_id
func (r *publicIpAssociateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing public IP associate",
			fmt.Sprintf("Expected import identifier with format: [project_id],[public_ip_id],[network_interface_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	publicIpId := idParts[1]
	networkInterfaceId := idParts[2]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "public_ip_id", publicIpId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("public_ip_id"), publicIpId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_interface_id"), networkInterfaceId)...)
	tflog.Info(ctx, "public IP state imported")
}

func mapFields(publicIpResp *iaas.PublicIp, model *Model) error {
	if publicIpResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var publicIpId string
	if model.PublicIpId.ValueString() != "" {
		publicIpId = model.PublicIpId.ValueString()
	} else if publicIpResp.Id != nil {
		publicIpId = *publicIpResp.Id
	} else {
		return fmt.Errorf("public IP id not present")
	}

	if publicIpResp.NetworkInterface != nil {
		model.NetworkInterfaceId = types.StringPointerValue(publicIpResp.GetNetworkInterface())
	} else {
		model.NetworkInterfaceId = types.StringNull()
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		publicIpId,
		model.NetworkInterfaceId.ValueString(),
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	model.PublicIpId = types.StringValue(publicIpId)
	model.Ip = types.StringPointerValue(publicIpResp.Ip)

	return nil
}

func toCreatePayload(model *Model) (*iaas.UpdatePublicIPPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &iaas.UpdatePublicIPPayload{
		NetworkInterface: iaas.NewNullableString(conversion.StringValueToPointer(model.NetworkInterfaceId)),
	}, nil
}
