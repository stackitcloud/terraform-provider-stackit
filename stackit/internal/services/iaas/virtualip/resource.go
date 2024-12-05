package virtualip

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	"github.com/stackitcloud/stackit-sdk-go/core/runtime"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
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
	_ resource.Resource                = &virtualIPResource{}
	_ resource.ResourceWithConfigure   = &virtualIPResource{}
	_ resource.ResourceWithImportState = &virtualIPResource{}
)

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	NetworkId   types.String `tfsdk:"network_id"`
	VirtualIpId types.String `tfsdk:"virtual_ip_id"`
	Name        types.String `tfsdk:"name"`
	IP          types.String `tfsdk:"ip"`
	Labels      types.Map    `tfsdk:"labels"`
}

// NewVirtualIPResource is a helper function to simplify the provider implementation.
func NewVirtualIPResource() resource.Resource {
	return &virtualIPResource{}
}

// networkResource is the resource implementation.
type virtualIPResource struct {
	client *iaasalpha.APIClient
}

// Metadata returns the resource type name.
func (r *virtualIPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_ip"
}

// Configure adds the provider configured client to the resource.
func (r *virtualIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_virtual_ip", "resource")
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
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the resource.
func (r *virtualIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Virtual IP resource schema. Must have a `region` specified in the provider configuration.",
		MarkdownDescription: features.AddBetaDescription("Virtual IP resource schema. Must have a `region` specified in the provider configuration."),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`network_id`,`virtual_ip_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the virtual IP is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_id": schema.StringAttribute{
				Description: "The network ID to which the virtual IP is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"virtual_ip_id": schema.StringAttribute{
				Description: "The virtual IP ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of virtual IP.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip": schema.StringAttribute{
				Description: "The IP address of the virtual IP.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.IP(),
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
func (r *virtualIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "network_id", networkId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating virtual IP", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new virtual IP
	virtualIp, err := r.client.CreateVirtualIP(ctx, projectId, networkId).CreateVirtualIPPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating virtual IP", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if virtualIp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating virtual IP.", "Empty response from API")
		return
	}

	_, err = wait.CreateVirtualIPWaitHandler(ctx, r.client, projectId, networkId, *virtualIp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting virtual IP", fmt.Sprintf("Waiting for deletion: %v", err))
		return
	}

	ctx = tflog.SetField(ctx, "virtual_ip_id", virtualIp.Id)

	// Map response body to schema
	err = mapFields(ctx, virtualIp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating virtual IP.", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Virtual IP created")
}

// Read refreshes the Terraform state with the latest data.
func (r *virtualIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	virtualIpId := model.VirtualIpId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "virtual_ip_id", virtualIpId)

	virtualIp, err := r.client.GetVirtualIP(ctx, projectId, networkId, virtualIpId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading virtual IP.", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, virtualIp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading virtual IP", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Virtual IP read")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *virtualIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	virtualIpId := model.VirtualIpId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "virtual_ip_id", virtualIpId)

	// Delete existing virtual IP
	err := r.client.DeleteVirtualIP(ctx, projectId, networkId, virtualIpId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting virtual IP", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.DeleteVirtualIPWaitHandler(ctx, r.client, projectId, networkId, virtualIpId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting virtual IP", fmt.Sprintf("Waiting for deletion: %v", err))
		return
	}

	tflog.Info(ctx, "Virtual IP deleted")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *virtualIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	networkAreaId := model.NetworkId.ValueString()
	networkAreaRouteId := model.VirtualIpId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkAreaId)
	ctx = tflog.SetField(ctx, "virtual_ip_id", networkAreaRouteId)

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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating virtual IP", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing virtual IP
	var httpResp *http.Response
	ctxWithHTTPResp := runtime.WithCaptureHTTPResponse(ctx, &httpResp)
	err = r.client.UpdateVirtualIP(ctxWithHTTPResp, projectId, networkAreaId, networkAreaRouteId).UpdateVirtualIPPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating virtual IP", fmt.Sprintf("Calling API: %v", err))
		return
	}
	requestId := httpResp.Header[wait.XRequestIDHeader][0]
	_, err = wait.ProjectRequestWaitHandler(ctx, r.client, projectId, requestId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating virtual IP", fmt.Sprintf("Waiting for update: %v", err))
		return
	}

	// Get updated virtual IP
	virtualIpResp, err := r.client.GetVirtualIP(ctx, projectId, networkAreaId, networkAreaRouteId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating virtual IP", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, virtualIpResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating virtual IP", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Virtual IP updated")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,network_id,virtual_ip_id
func (r *virtualIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing virtual IP",
			fmt.Sprintf("Expected import identifier with format: [project_id],[network_id],[virtual_ip_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	networkId := idParts[1]
	virtualIpId := idParts[2]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "virtual_ip_id", virtualIpId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_id"), networkId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("virtual_ip_id"), virtualIpId)...)
	tflog.Info(ctx, "Virtual IP state imported")
}

func mapFields(ctx context.Context, virtualIp *iaasalpha.VirtualIp, model *Model) error {
	if virtualIp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var virtualIpId string
	if model.VirtualIpId.ValueString() != "" {
		virtualIpId = model.VirtualIpId.ValueString()
	} else if virtualIp.Id != nil {
		virtualIpId = *virtualIp.Id
	} else {
		return fmt.Errorf("virtual IP id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		model.NetworkId.ValueString(),
		virtualIpId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	labels, diags := types.MapValueFrom(ctx, types.StringType, map[string]interface{}{})
	if diags.HasError() {
		return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
	}
	if virtualIp.Labels != nil && len(*virtualIp.Labels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *virtualIp.Labels)
		if diags.HasError() {
			return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else if model.Labels.IsNull() {
		labels = types.MapNull(types.StringType)
	}

	model.VirtualIpId = types.StringValue(virtualIpId)
	model.Name = types.StringPointerValue(virtualIp.Name)
	model.IP = types.StringPointerValue(virtualIp.Ip)
	model.Labels = labels

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaasalpha.CreateVirtualIPPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaasalpha.CreateVirtualIPPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Ip:     conversion.StringValueToPointer(model.IP),
		Labels: &labels,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaasalpha.UpdateVirtualIPPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaasalpha.UpdateVirtualIPPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: &labels,
	}, nil
}
