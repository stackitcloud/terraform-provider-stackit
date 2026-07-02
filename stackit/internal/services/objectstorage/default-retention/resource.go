package objectstorage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	objectstorageUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &defaultRetentionResource{}
	_ resource.ResourceWithConfigure   = &defaultRetentionResource{}
	_ resource.ResourceWithImportState = &defaultRetentionResource{}
	_ resource.ResourceWithModifyPlan  = &defaultRetentionResource{}
)

type model struct {
	Id         types.String `tfsdk:"id"` // needed by TF
	BucketName types.String `tfsdk:"bucket_name"`
	ProjectId  types.String `tfsdk:"project_id"`
	Region     types.String `tfsdk:"region"`
	Days       types.Int32  `tfsdk:"days"`
	Mode       types.String `tfsdk:"mode"`
}

// NewDefaultRetentionResource is a helper function to simplify the provider implementation.
func NewDefaultRetentionResource() resource.Resource {
	return &defaultRetentionResource{}
}

// defaultRetentionResource is the resource implementation.
type defaultRetentionResource struct {
	client       *objectstorage.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements [resource.ResourceWithModifyPlan].
func (r *defaultRetentionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic
	var configModel model
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel model
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

// ImportState implements [resource.ResourceWithImportState].
func (r *defaultRetentionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing default-retention",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[bucketName], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"region":      idParts[1],
		"bucket_name": idParts[2],
	})
	tflog.Info(ctx, "ObjectStorage default-retention state imported")
}

// Configure implements [resource.ResourceWithConfigure].
func (r *defaultRetentionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := objectstorageUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "ObjectStorage bucket client configured")
}

// Schema implements [resource.Resource].
func (r *defaultRetentionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "ObjectStorage default-retention resource schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`bucket_name`\".",
		"bucket_name": "The associated bucket's name. It must be DNS conform.",
		"project_id":  "STACKIT Project ID to which the default-retention is associated.",
		"region":      "The resource region. If not defined, the provider region is used.",
		"days":        "The number retention period in days.",
		"mode":        "The retention mode for default retention on a bucket.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket_name": schema.StringAttribute{
				Description: descriptions["bucket_name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"days": schema.Int32Attribute{
				Required:    true,
				Description: descriptions["days"],
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.RequiresReplace(),
				},
			},
			"mode": schema.StringAttribute{
				Required:    true,
				Description: descriptions["mode"],
				Validators: []validator.String{
					stringvalidator.OneOf(sdkUtils.EnumSliceToStringSlice(objectstorage.AllowedRetentionModeEnumValues)...),
					validate.NoSeparator(),
				},
			},
		},
	}
}

// Metadata implements [resource.Resource].
func (r *defaultRetentionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_default_retention"
}

// Create implements [resource.Resource].
func (r *defaultRetentionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic
	var model model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	bucketName := model.BucketName.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bucket_name", bucketName)
	ctx = tflog.SetField(ctx, "region", region)

	// Create default-retention
	apiRequest, err := toSetDefaultRetentionRequest(ctx, r.client.DefaultAPI, &model, projectId, bucketName, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error setting default-retention", fmt.Sprintf("Parsing model: %v", err))
	}
	result, err := apiRequest.Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error setting default-retention", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(result, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error setting default-retention", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage default-retention created")
}

// Delete implements [resource.Resource].
func (r *defaultRetentionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic
	var model model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	bucketName := model.BucketName.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bucket_name", bucketName)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete default-retention
	_, err := r.client.DefaultAPI.DeleteDefaultRetention(ctx, projectId, region, bucketName).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
			if oapiErr.StatusCode == http.StatusConflict {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting default-retention", "Encountered conflict")
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting default-retention", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "ObjectStorage default-retention deleted")
}

// Read implements [resource.Resource].
func (r *defaultRetentionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic
	var model model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	bucketName := model.BucketName.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bucket_name", bucketName)
	ctx = tflog.SetField(ctx, "region", region)

	// Read default-retention
	result, err := r.client.DefaultAPI.GetDefaultRetention(ctx, projectId, region, bucketName).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading default-retention", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(result, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading default-retention", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage default-retention read")
}

// Update implements [resource.Resource].
func (r *defaultRetentionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic
	var model model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	bucketName := model.BucketName.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bucket_name", bucketName)
	ctx = tflog.SetField(ctx, "region", region)

	// Update default-retention
	apiRequest, err := toSetDefaultRetentionRequest(ctx, r.client.DefaultAPI, &model, projectId, bucketName, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error setting default-retention", fmt.Sprintf("Parsing model: %v", err))
	}
	result, err := apiRequest.Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error setting default-retention", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(result, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error setting default-retention", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage default-retention set")
}

func toSetDefaultRetentionRequest(ctx context.Context, client objectstorage.DefaultAPI, m *model, projectId, bucketName, region string) (objectstorage.ApiSetDefaultRetentionRequest, error) {
	days := m.Days.ValueInt32()
	stringMode := m.Mode.ValueString()
	mode, err := objectstorage.NewRetentionModeFromValue(stringMode)
	if err != nil {
		return objectstorage.ApiSetDefaultRetentionRequest{}, fmt.Errorf("could not parse provided retention mode to enum: %w", err)
	}
	apiRequest := client.SetDefaultRetention(ctx, projectId, region, bucketName).SetDefaultRetentionPayload(objectstorage.SetDefaultRetentionPayload{
		Days: days,
		Mode: *mode,
	})
	return apiRequest, nil
}

func mapFields(res *objectstorage.DefaultRetentionResponse, m *model, region string) error {
	if res == nil {
		return fmt.Errorf("response input is nil")
	}
	if m == nil {
		return fmt.Errorf("model input is nil")
	}

	m.BucketName = types.StringValue(res.Bucket)
	m.Id = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), region, m.BucketName.ValueString())
	m.Region = types.StringValue(region)
	m.Days = types.Int32Value(res.Days)
	m.Mode = types.StringValue(string(res.Mode))
	return nil
}
