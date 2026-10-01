package iplists

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	lbipListsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/lbiplists/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	lbiplists "github.com/stackitcloud/stackit-sdk-go/services/lbiplists/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

var (
	_ resource.Resource                = &ipListsServiceResource{}
	_ resource.ResourceWithConfigure   = &ipListsServiceResource{}
	_ resource.ResourceWithImportState = &ipListsServiceResource{}
	_ resource.ResourceWithModifyPlan  = &ipListsServiceResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	ProjectId          types.String `tfsdk:"project_id"`
	Region             types.String `tfsdk:"region"`
	Name               types.String `tfsdk:"name"`
	Labels             types.Map    `tfsdk:"labels"`
	NumberOfIPs        types.Int32  `tfsdk:"number_of_ips"`
	ContentHash        types.String `tfsdk:"content_hash"`
	FileContent        types.String `tfsdk:"file_content"`
	FileContentVersion types.Int64  `tfsdk:"file_content_version"`
}

type ipListsServiceResource struct {
	client       *lbiplists.APIClient
	providerData core.ProviderData
}

func NewIPListsServiceResource() resource.Resource {
	return &ipListsServiceResource{}
}

// Metadata implements [resource.Resource].
func (r *ipListsServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancer_ip_list"
}

// ModifyPlan implements [resource.ResourceWithModifyPlan].
func (r *ipListsServiceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

// Configure implements [resource.ResourceWithConfigure].
func (r *ipListsServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &r.providerData, features.LoadbalancerIPList, "stackit_loadbalancer_ip_list", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := lbipListsUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Load Balancer IP Lists client configured")
}

// Schema implements [resource.Resource].
func (r *ipListsServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                 "Load Balancer IP Lists resource schema." + core.ResourceRegionFallbackDocstring,
		"id":                   "Terraform's internal resource ID. It is structured as \"`project_id`,region,`name`\".",
		"project_id":           "STACKIT project ID to which the Load Balancer is associated.",
		"region":               "STACKIT region.",
		"labels":               "User-defined metadata as key-value pairs. Should not exceed 64 entries.",
		"name":                 "Name of the Load Balancer IP List.",
		"number_of_ips":        "The number of IP addresses in this IP list.",
		"content_hash":         "Unique SHA-256 hex fingerprint used to verify the stored IP list file content.",
		"file_content":         "Raw upload payload as a plain UTF-8 string. Newline-separated .txt list of IPv4 CIDR entries, or CSV with cidr,labels columns (any other columns are ignored). Write-only - never stored in state and never returned by the API. To rotate the content, update this value AND increment file_content_version. Changing this field alone will NOT trigger an update.",
		"file_content_version": "User-managed rotation counter for the file_content. Must be incremented every time file_content is changed. Terraform diffs this field to detect changes in the defined IP list - changing file_content alone will NOT trigger an update because it is write-only and never stored in state.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddExperimentDescription("Load Balancer IP Lists resource schema.", features.LoadbalancerIPList, core.Resource),
		Description:         descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
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
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,49}[0-9a-z])?$`), "must start and end with an alphanumeric character, may contain hyphens, and be 1-51 characters long",
					),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(64),
				},
			},
			"number_of_ips": schema.Int32Attribute{
				Description: descriptions["number_of_ips"],
				Computed:    true,
			},
			"content_hash": schema.StringAttribute{
				Description: descriptions["content_hash"],
				Computed:    true,
			},
			"file_content": schema.StringAttribute{
				Description: descriptions["file_content"],
				WriteOnly:   true,
				Required:    true,
			},
			"file_content_version": schema.Int64Attribute{
				Description: descriptions["file_content_version"],
				Required:    true,
			},
		},
	}
}

// ImportState implements [resource.ResourceWithImportState].
func (r *ipListsServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing Load Balancer IP list",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[name]  Got: %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"name":       idParts[2],
	})
	tflog.Info(ctx, "Load Balancer IP list state imported")
}

// Create implements [resource.Resource].
func (r *ipListsServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var planModel Model
	diags := req.Plan.Get(ctx, &planModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The config model - this has to be used because Terraform doesn't include write-only field values in the
	// plan and state models - for security measures. Write-only values should be only kept in the config model
	// so that they never end up in the state (or plan).
	var configModel Model
	diags = req.Config.Get(ctx, &configModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := planModel.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(planModel.Region)
	name := planModel.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	payload, err := toUploadPayload(ctx, &planModel, &configModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Load Balancer IP list", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.DefaultAPI.UploadIPList(ctx, projectId, region, name).UploadIPListPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Load Balancer IP list", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, createResp, &planModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Load Balancer IP list", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, planModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Load Balancer IP list created")
}

// Read implements [resource.Resource].
func (r *ipListsServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	ipList, err := r.client.DefaultAPI.GetIPList(ctx, projectId, region, name).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Load Balancer IP list", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, ipList, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Load Balancer IP list", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Load Balancer IP list read")
}

// Update implements [resource.Resource].
func (r *ipListsServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var planModel Model
	diags := req.Plan.Get(ctx, &planModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The config model - this has to be used because Terraform doesn't include write-only field values in the
	// plan and state models - for security measures. Write-only values should be only kept in the config model
	// so that they never end up in the state (or plan).
	var configModel Model
	diags = req.Config.Get(ctx, &configModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := planModel.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(planModel.Region)
	name := planModel.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	// Upload is the only update verb of the API and requires fileContent to be set. Uploads with identical
	// stored content are a no-op on the server side, so we always upload the current config content.
	payload, err := toUploadPayload(ctx, &planModel, &configModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Load Balancer IP list", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateResp, err := r.client.DefaultAPI.UploadIPList(ctx, projectId, region, name).UploadIPListPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Load Balancer IP list", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, updateResp, &planModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Load Balancer IP list", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, planModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Load Balancer IP list updated")
}

// Delete implements [resource.Resource].
func (r *ipListsServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	_, err := r.client.DefaultAPI.DeleteIPList(ctx, projectId, region, name).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Load Balancer IP list", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "Load Balancer IP list deleted")
}

// toUploadPayload builds the UploadIPList payload. Everything is read from the plan model, except for the
// write-only file_content which is only present in the config model and must never end up in the state
func toUploadPayload(ctx context.Context, planModel, configModel *Model) (*lbiplists.UploadIPListPayload, error) {
	if planModel == nil {
		return nil, fmt.Errorf("nil plan model")
	}
	if configModel == nil {
		return nil, fmt.Errorf("nil config model")
	}

	payload := &lbiplists.UploadIPListPayload{
		Name:        planModel.Name.ValueString(),
		FileContent: configModel.FileContent.ValueString(),
	}

	labels, err := utils.LabelsToPayload(ctx, planModel.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting labels: %w", err)
	}
	payload.SetLabels(labels)

	return payload, nil
}

func mapFields(ctx context.Context, ipList *lbiplists.GetIPListResponse, model *Model, region string) error {
	if ipList == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	name := ipList.Name
	if name == "" {
		if model.Name.ValueString() == "" {
			return fmt.Errorf("name not present")
		}
		name = model.Name.ValueString()
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, name)
	model.Name = types.StringValue(name)
	model.Region = types.StringValue(region)

	model.ContentHash = types.StringPointerValue(ipList.ContentHash)
	model.NumberOfIPs = types.Int32PointerValue(ipList.NumberOfIps)

	respLabels, _ := ipList.GetLabelsOk()
	labels, err := utils.MapLabels(ctx, respLabels, model.Labels)
	if err != nil {
		return fmt.Errorf("mapping labels: %w", err)
	}
	model.Labels = labels

	return nil
}
