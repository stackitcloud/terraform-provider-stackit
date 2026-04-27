package accesstoken

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/telemetryrouter/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &telemetryRouterAccessTokenResource{}
	_ resource.ResourceWithConfigure   = &telemetryRouterAccessTokenResource{}
	_ resource.ResourceWithImportState = &telemetryRouterAccessTokenResource{}
	_ resource.ResourceWithModifyPlan  = &telemetryRouterAccessTokenResource{}
)

const (
	AccessTokenStatusActive   = "active"
	AccessTokenStatusExpired  = "expired"
	AccessTokenStatusDeleting = "deleting"
)

var schemaDescriptions = map[string]string{
	"id":              "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`,`access_token_id`\".",
	"access_token_id": "The access token ID",
	"instance_id":     "The TelemetryRouter instance ID associated with the access token",
	"region":          "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"project_id":      "STACKIT project ID associated with the TelemetryRouter access token",
	"display_name":    "The displayed name of the access token",
	"description":     "The description of the access token",
	"access_token":    "The generated access token",
	"creator_id":      "The user who created the access token",
	"expiration_time": "The date and time an access token will expire at (inclusively)",
	"ttl":             "The time-to-live (TTL) in days for the access token. If not set, token will not expire",
	"status": fmt.Sprintf(
		"The status of the access token. %s",
		tfutils.FormatPossibleValues(AccessTokenStatusActive, AccessTokenStatusExpired, AccessTokenStatusDeleting),
	),
}

type Model struct {
	ID             types.String `tfsdk:"id"` // Required by Terraform
	AccessTokenID  types.String `tfsdk:"access_token_id"`
	InstanceID     types.String `tfsdk:"instance_id"`
	Region         types.String `tfsdk:"region"`
	ProjectID      types.String `tfsdk:"project_id"`
	DisplayName    types.String `tfsdk:"display_name"`
	Description    types.String `tfsdk:"description"`
	AccessToken    types.String `tfsdk:"access_token"`
	CreatorID      types.String `tfsdk:"creator_id"`
	ExpirationTime types.String `tfsdk:"expiration_time"`
	Ttl            types.Int32  `tfsdk:"ttl"`
	Status         types.String `tfsdk:"status"`
}

type telemetryRouterAccessTokenResource struct {
	client       *telemetryrouter.APIClient
	providerData core.ProviderData
}

func NewTelemetryRouterAccessTokenResource() resource.Resource {
	return &telemetryRouterAccessTokenResource{}
}

func (r *telemetryRouterAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.client = utils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter client configured")
}

func (r *telemetryRouterAccessTokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *telemetryRouterAccessTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_telemetryrouter_access_token"
}

func (r *telemetryRouterAccessTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("TelemetryRouter access token resource schema. %s", core.ResourceRegionFallbackDocstring),
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
			"creator_id": schema.StringAttribute{
				Description: schemaDescriptions["creator_id"],
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
			"expiration_time": schema.StringAttribute{
				Description: schemaDescriptions["expiration_time"],
				Computed:    true,
			},
			"ttl": schema.Int32Attribute{
				Description: schemaDescriptions["ttl"],
				Optional:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
		},
	}
}

func (r *telemetryRouterAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter access token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.DefaultAPI.CreateAccessToken(ctx, projectId, region, instanceId).CreateAccessTokenPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter access token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp.Id == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter access token", "Got empty credential id")
		return
	}
	accessTokenId := createResp.Id
	ctx = tflog.SetField(ctx, "access_token_id", accessTokenId)

	err = mapCreateFields(ctx, createResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter access token", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter instance created")
}

func (r *telemetryRouterAccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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

	accessTokenResponse, err := r.client.DefaultAPI.GetAccessToken(ctx, projectID, region, instanceID, accessTokenID).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryRouter access token", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapGetFields(ctx, accessTokenResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryRouter access token", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter access token read", map[string]interface{}{
		"access_token_id": accessTokenID,
	})
}

func (r *telemetryRouterAccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter access token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	accessTokenResponse, err := r.client.DefaultAPI.UpdateAccessToken(ctx, projectID, region, instanceID, accessTokenID).UpdateAccessTokenPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter access token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapUpdateFields(ctx, accessTokenResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter access token", fmt.Sprintf("Processing response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter access token updated", map[string]interface{}{
		"access_token_id": accessTokenID,
	})
}

func (r *telemetryRouterAccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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

	err := r.client.DefaultAPI.DeleteAccessToken(ctx, projectID, region, instanceID, accessTokenID).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryRouter access token", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "TelemetryRouter access token deleted")
}

func (r *telemetryRouterAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing TelemetryRouter access token", fmt.Sprintf("Invalid import ID %q: expected format is `project_id`,`region`,`instance_id`,`access_token_id`", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_token_id"), idParts[3])...)

	core.LogAndAddWarning(ctx, &resp.Diagnostics,
		"TelemetryRouter access token imported with empty token",
		"The token is not imported as they are only available upon creation of a new access token. The token field will be empty.",
	)

	tflog.Info(ctx, "TelemetryRouter access token state imported")
}

func toCreatePayload(_ context.Context, _ diag.Diagnostics, model *Model) (*telemetryrouter.CreateAccessTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	return &telemetryrouter.CreateAccessTokenPayload{
		Description: model.Description.ValueStringPointer(),
		DisplayName: model.DisplayName.ValueString(),
		Ttl:         *telemetryrouter.NewNullableInt32(model.Ttl.ValueInt32Pointer()),
	}, nil
}

func mapCreateFields(ctx context.Context, accessToken *telemetryrouter.CreateAccessTokenResponse, model *Model) error {
	if accessToken == nil {
		return fmt.Errorf("access token is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	var accessTokenID string
	if model.AccessTokenID.ValueString() != "" {
		accessTokenID = model.AccessTokenID.ValueString()
	} else if accessToken.Id != "" {
		accessTokenID = accessToken.Id
	} else {
		return fmt.Errorf("access token id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), model.Region.ValueString(), model.InstanceID.ValueString(), accessTokenID)
	model.AccessTokenID = types.StringValue(accessTokenID)
	model.Region = types.StringValue(model.Region.ValueString())
	model.CreatorID = types.StringValue(accessToken.CreatorId)
	model.Description = types.StringPointerValue(accessToken.Description)
	model.DisplayName = types.StringValue(accessToken.DisplayName)
	model.Status = types.StringValue(accessToken.Status)

	model.ExpirationTime = types.StringNull()
	if accessToken.HasExpirationTime() && accessToken.ExpirationTime.Get() != nil {
		model.ExpirationTime = types.StringValue(accessToken.ExpirationTime.Get().Format(time.RFC3339))
	}

	if accessToken.AccessToken != "" {
		model.AccessToken = types.StringValue(accessToken.AccessToken)
	}

	return nil
}

func mapGetFields(ctx context.Context, accessToken *telemetryrouter.GetAccessTokenResponse, model *Model) error {
	if accessToken == nil {
		return fmt.Errorf("access token is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	var accessTokenID string
	if model.AccessTokenID.ValueString() != "" {
		accessTokenID = model.AccessTokenID.ValueString()
	} else if accessToken.Id != "" {
		accessTokenID = accessToken.Id
	} else {
		return fmt.Errorf("access token id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), model.Region.ValueString(), model.InstanceID.ValueString(), accessTokenID)
	model.AccessTokenID = types.StringValue(accessTokenID)
	model.Region = types.StringValue(model.Region.ValueString())
	model.CreatorID = types.StringValue(accessToken.CreatorId)
	model.Description = types.StringPointerValue(accessToken.Description)
	model.DisplayName = types.StringValue(accessToken.DisplayName)
	model.Status = types.StringValue(accessToken.Status)

	model.ExpirationTime = types.StringNull()
	if accessToken.HasExpirationTime() && accessToken.ExpirationTime.Get() != nil {
		model.ExpirationTime = types.StringValue(accessToken.ExpirationTime.Get().Format(time.RFC3339))
	}

	return nil
}

func mapUpdateFields(ctx context.Context, accessToken *telemetryrouter.UpdateAccessTokenResponse, model *Model) error {
	if accessToken == nil {
		return fmt.Errorf("access token is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	var accessTokenID string
	if model.AccessTokenID.ValueString() != "" {
		accessTokenID = model.AccessTokenID.ValueString()
	} else if accessToken.Id != "" {
		accessTokenID = accessToken.Id
	} else {
		return fmt.Errorf("access token id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), model.Region.ValueString(), model.InstanceID.ValueString(), accessTokenID)
	model.AccessTokenID = types.StringValue(accessTokenID)
	model.Region = types.StringValue(model.Region.ValueString())
	model.CreatorID = types.StringValue(accessToken.CreatorId)
	if accessToken.Description != nil && *accessToken.Description != "" {
		model.Description = types.StringPointerValue(accessToken.Description)
	}
	model.DisplayName = types.StringValue(accessToken.DisplayName)
	model.Status = types.StringValue(accessToken.Status)

	model.ExpirationTime = types.StringNull()
	if accessToken.HasExpirationTime() && accessToken.ExpirationTime.Get() != nil {
		model.ExpirationTime = types.StringValue(accessToken.ExpirationTime.Get().Format(time.RFC3339))
	}

	return nil
}

func toUpdatePayload(model *Model) (*telemetryrouter.UpdateAccessTokenPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	return &telemetryrouter.UpdateAccessTokenPayload{
		Description: *telemetryrouter.NewNullableString(conversion.StringValueToPointer(model.Description)),
		DisplayName: *telemetryrouter.NewNullableString(conversion.StringValueToPointer(model.DisplayName)),
	}, nil
}
