package accesstoken

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/logs"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logs/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &logsAccessTokenResource{}
	_ resource.ResourceWithConfigure   = &logsAccessTokenResource{}
	_ resource.ResourceWithImportState = &logsAccessTokenResource{}
	_ resource.ResourceWithModifyPlan  = &logsAccessTokenResource{}
)

var schemaDescriptions = map[string]string{
	"id":              "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`,`access_token_id`\".",
	"access_token_id": "The access token ID",
	"instance_id":     "The Logs instance ID associated with the access token",
	"region":          "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"project_id":      "STACKIT project ID associated with the Logs access token",
	"access_token":    "The generated access token",
	"creator":         "The user who created the access token",
	"description":     "The description of the access token",
	"display_name":    "The displayed name of the access token",
	"expires":         "Indicates if the access token can expire",
	"valid_until":     "The date and time until an access token is valid to (inclusively)",
	"lifetime":        "A lifetime period for an access token in days. If unset the token will not expire.",
	"permissions":     "The access permissions granted to the access token. Possible values: `read`, `write`.",
	"status": fmt.Sprintf(
		"The status of the access token, possible values: %s",
		tfutils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(logs.AllowedAccessTokenStatusEnumValues)...),
	),
}

type Model struct {
	ID            types.String `tfsdk:"id"` // Required by Terraform
	AccessTokenID types.String `tfsdk:"access_token_id"`
	InstanceID    types.String `tfsdk:"instance_id"`
	Region        types.String `tfsdk:"region"`
	ProjectID     types.String `tfsdk:"project_id"`
	Creator       types.String `tfsdk:"creator"`
	Description   types.String `tfsdk:"description"`
	DisplayName   types.String `tfsdk:"display_name"`
	AccessToken   types.String `tfsdk:"access_token"`
	Expires       types.Bool   `tfsdk:"expires"`
	ValidUntil    types.String `tfsdk:"valid_until"`
	Lifetime      types.Int64  `tfsdk:"lifetime"`
	Permissions   types.List   `tfsdk:"permissions"`
	Status        types.String `tfsdk:"status"`
}

type logsAccessTokenResource struct {
	client       *logs.APIClient
	providerData core.ProviderData
}

func NewLogsAccessTokenResource() resource.Resource {
	return &logsAccessTokenResource{}
}

func (r *logsAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_logs_access_token", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	r.client = utils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Logs client configured")
}

func (r *logsAccessTokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
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

	tfutils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *logsAccessTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logs_access_token"
}

func (r *logsAccessTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Logs access token resource schema.", core.Resource),
		Description:         fmt.Sprintf("Logs access token resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access_token_id": schema.StringAttribute{
				Description: schemaDescriptions["access_token_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
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
			"creator": schema.StringAttribute{
				Description: schemaDescriptions["creator"],
				Computed:    true,
			},
			"access_token": schema.StringAttribute{
				Description: schemaDescriptions["access_token"],
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Optional:    true,
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Required:    true,
			},
			"expires": schema.BoolAttribute{
				Description: schemaDescriptions["expires"],
				Computed:    true,
			},
			"valid_until": schema.StringAttribute{
				Description: schemaDescriptions["valid_until"],
				Computed:    true,
			},
			"lifetime": schema.Int64Attribute{
				Description: schemaDescriptions["lifetime"],
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.ListAttribute{
				Description: schemaDescriptions["permissions"],
				ElementType: types.StringType,
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
		},
	}
}

func (r *logsAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	instanceId := model.InstanceID.ValueString()
	projectId := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(ctx, resp.Diagnostics, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Logs access token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.CreateAccessToken(ctx, projectId, region, instanceId).CreateAccessTokenPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Logs access token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Logs access token", "Got empty credential id")
		return
	}
	accessTokenId := *createResp.Id
	ctx = tflog.SetField(ctx, "access_token_id", accessTokenId)

	err = mapFields(ctx, createResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Logs access token", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Logs instance created")
}

func (r *logsAccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()
	accessTokenID := model.AccessTokenID.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "access_token_id", accessTokenID)

	accessTokenResponse, err := r.client.GetAccessToken(ctx, projectID, region, instanceID, accessTokenID).Execute()
	if err != nil {
		tfutils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading Logs access token",
			fmt.Sprintf("Calling API: %v", err),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectID),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, accessTokenResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Logs access token", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Logs access token read", map[string]interface{}{
		"access_token_id": accessTokenID,
	})
}

func (r *logsAccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := model.Region.ValueString()
	instanceID := model.InstanceID.ValueString()
	accessTokenID := model.AccessTokenID.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "access_token_id", accessTokenID)

	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Logs access token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	err = r.client.UpdateAccessToken(ctx, projectID, region, instanceID, accessTokenID).UpdateAccessTokenPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Logs access token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	accessTokenResponse, err := r.client.GetAccessToken(ctx, projectID, region, instanceID, accessTokenID).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Logs access token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, accessTokenResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Logs access token", fmt.Sprintf("Processing response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Logs access token updated", map[string]interface{}{
		"access_token_id": accessTokenID,
	})
}

func (r *logsAccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := model.Region.ValueString()
	instanceID := model.InstanceID.ValueString()
	accessTokenID := model.AccessTokenID.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "access_token_id", accessTokenID)

	err := r.client.DeleteAccessToken(ctx, projectID, region, instanceID, accessTokenID).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Logs access token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Logs access token deleted")
}

func (r *logsAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing Logs access token", fmt.Sprintf("Invalid import ID %q: expected format is `project_id`,`region`,`instance_id`,`access_token_id`", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_token_id"), idParts[3])...)

	core.LogAndAddWarning(ctx, &resp.Diagnostics,
		"Logs access token imported with empty token",
		"The token is not imported as they are only available upon creation of a new access token. The token field will be empty.",
	)

	tflog.Info(ctx, "Logs access token state imported")
}

func toCreatePayload(ctx context.Context, diagnostics diag.Diagnostics, model *Model) (*logs.CreateAccessTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	payload := &logs.CreateAccessTokenPayload{
		Description: conversion.StringValueToPointer(model.Description),
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
		Lifetime:    conversion.Int64ValueToPointer(model.Lifetime),
	}

	if !(tfutils.IsUndefined(model.Permissions)) {
		var permissions []string
		permissionDiags := model.Permissions.ElementsAs(ctx, &permissions, false)
		diagnostics.Append(permissionDiags...)
		if !permissionDiags.HasError() {
			payload.Permissions = &permissions
		}
	}

	return payload, nil
}

func mapFields(ctx context.Context, accessToken *logs.AccessToken, model *Model) error {
	if accessToken == nil {
		return fmt.Errorf("access token is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	var accessTokenID string
	if model.AccessTokenID.ValueString() != "" {
		accessTokenID = model.AccessTokenID.ValueString()
	} else if accessToken.Id != nil {
		accessTokenID = *accessToken.Id
	} else {
		return fmt.Errorf("access token id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), model.Region.ValueString(), model.InstanceID.ValueString(), accessTokenID)
	model.AccessTokenID = types.StringValue(accessTokenID)
	model.Region = types.StringValue(model.Region.ValueString())
	model.Creator = types.StringPointerValue(accessToken.Creator)
	model.Description = types.StringPointerValue(accessToken.Description)
	model.DisplayName = types.StringPointerValue(accessToken.DisplayName)
	model.Expires = types.BoolPointerValue(accessToken.Expires)
	model.Status = types.StringValue(string(*accessToken.Status))

	model.ValidUntil = types.StringNull()
	if accessToken.ValidUntil != nil {
		model.ValidUntil = types.StringValue(accessToken.ValidUntil.Format(time.RFC3339))
	}

	if accessToken.AccessToken != nil {
		model.AccessToken = types.StringValue(*accessToken.AccessToken)
	}

	permissionList := types.ListNull(types.StringType)
	var diags diag.Diagnostics
	if accessToken.Permissions != nil && len(*accessToken.Permissions) > 0 {
		permissionList, diags = types.ListValueFrom(ctx, types.StringType, accessToken.Permissions)
		if diags.HasError() {
			return fmt.Errorf("mapping permissions: %w", core.DiagsToError(diags))
		}
	}
	model.Permissions = permissionList

	return nil
}

func toUpdatePayload(model *Model) (*logs.UpdateAccessTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	payload := &logs.UpdateAccessTokenPayload{
		Description: conversion.StringValueToPointer(model.Description),
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
	}

	return payload, nil
}
