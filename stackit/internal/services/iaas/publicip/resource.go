package publicip

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

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
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &publicIpResource{}
	_ resource.ResourceWithConfigure   = &publicIpResource{}
	_ resource.ResourceWithImportState = &publicIpResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	ProjectId          types.String `tfsdk:"project_id"`
	PublicIpId         types.String `tfsdk:"public_ip_id"`
	Ip                 types.String `tfsdk:"ip"`
	NetworkInterfaceId types.String `tfsdk:"network_interface_id"`
	Labels             types.Map    `tfsdk:"labels"`
}

// NewPublicIpResource is a helper function to simplify the provider implementation.
func NewPublicIpResource() resource.Resource {
	return &publicIpResource{}
}

// publicIpResource is the resource implementation.
type publicIpResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *publicIpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_ip"
}

// Configure adds the provider configured client to the resource.
func (r *publicIpResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *publicIpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Public IP resource schema. Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on resource level."
	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`public_ip_id`\".",
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
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				Description: "Associates the public IP with a network interface or a virtual IP (ID). If you are using this resource with a Kubernetes Load Balancer or any other resource which associates a network interface implicitly, use the lifecycle `ignore_changes` property in this field to prevent unintentional removal of the network interface due to drift in the Terraform state",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *publicIpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating public IP", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new public IP

	publicIp, err := r.client.CreatePublicIP(ctx, projectId).CreatePublicIPPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating public IP", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = tflog.SetField(ctx, "public_ip_id", *publicIp.Id)

	// Map response body to schema
	err = mapFields(ctx, publicIp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating public IP", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Public IP created")
}

// Read refreshes the Terraform state with the latest data.
func (r *publicIpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	publicIpId := model.PublicIpId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "public_ip_id", publicIpId)

	publicIpResp, err := r.client.GetPublicIP(ctx, projectId, publicIpId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading public IP", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, publicIpResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading public IP", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "public IP read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *publicIpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	publicIpId := model.PublicIpId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "public_ip_id", publicIpId)

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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating public IP", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing public IP
	updatedPublicIp, err := r.client.UpdatePublicIP(ctx, projectId, publicIpId).UpdatePublicIPPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating public IP", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, updatedPublicIp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating public IP", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "public IP updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *publicIpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	publicIpId := model.PublicIpId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "public_ip_id", publicIpId)

	// Delete existing publicIp
	err := r.client.DeletePublicIP(ctx, projectId, publicIpId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting public IP", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "public IP deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,public_ip_id
func (r *publicIpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing public IP",
			fmt.Sprintf("Expected import identifier with format: [project_id],[public_ip_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	publicIpId := idParts[1]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "public_ip_id", publicIpId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("public_ip_id"), publicIpId)...)
	tflog.Info(ctx, "public IP state imported")
}

func mapFields(ctx context.Context, publicIpResp *iaas.PublicIp, model *Model) error {
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), publicIpId)

	labels, err := iaasUtils.MapLabels(ctx, publicIpResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.PublicIpId = types.StringValue(publicIpId)
	model.Ip = types.StringPointerValue(publicIpResp.Ip)
	if publicIpResp.NetworkInterface != nil {
		model.NetworkInterfaceId = types.StringPointerValue(publicIpResp.GetNetworkInterface())
	} else {
		model.NetworkInterfaceId = types.StringNull()
	}
	model.Labels = labels
	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreatePublicIPPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.CreatePublicIPPayload{
		Labels:           &labels,
		Ip:               conversion.StringValueToPointer(model.Ip),
		NetworkInterface: iaas.NewNullableString(conversion.StringValueToPointer(model.NetworkInterfaceId)),
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.UpdatePublicIPPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.UpdatePublicIPPayload{
		Labels:           &labels,
		NetworkInterface: iaas.NewNullableString(conversion.StringValueToPointer(model.NetworkInterfaceId)),
	}, nil
}
