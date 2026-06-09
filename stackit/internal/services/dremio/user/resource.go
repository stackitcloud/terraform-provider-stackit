package dremio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	dremioSdk "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi"

	dremioUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dremio/utils"

	dremioWaiter "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi/wait"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
	_ resource.ResourceWithModifyPlan  = &userResource{}
)

type Model struct {
	Id types.String `tfsdk:"id"`

	ProjectId  types.String `tfsdk:"project_id"`
	Region     types.String `tfsdk:"region"`
	InstanceId types.String `tfsdk:"instance_id"`
	UserId     types.String `tfsdk:"user_id"`

	// Required Fields
	Email     types.String `tfsdk:"email"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Name      types.String `tfsdk:"name"`

	// Optional Fields
	Description types.String `tfsdk:"description"`

	// Read-only Fields
	State        types.String `tfsdk:"state"`
	ErrorMessage types.String `tfsdk:"error_message"`
}

type UserModel struct {
	Model

	// Required Fields
	Password types.String `tfsdk:"password"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

var descriptions = map[string]string{
	"main":          "Manages a STACKIT Dremio instances user.",
	"id":            "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`,`user_id`\".",
	"project_id":    "STACKIT Project ID to which the resource is associated.",
	"instance_id":   "The Dremio instance ID.",
	"region":        "The STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"user_id":       "The Dremio user ID.",
	"email":         "The email address of the user.",
	"first_name":    "The first name of the user.",
	"last_name":     "The last name of the user.",
	"name":          "The username of the user.",
	"password":      "The password of the user. Only used for creation and updates. Must be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, one number and one special character.",
	"description":   "The description of the user.",
	"state":         "The current state of the resource.",
	"error_message": "A message describing an actionable error the user can resolve. This field is empty if no such error exists.",
}

type userResource struct {
	client       *dremioSdk.APIClient
	providerData core.ProviderData
}

func NewUserResource() resource.Resource {
	return &userResource{}
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dremio_user"
}

func (r *userResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel UserModel
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel UserModel
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

func (r *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := dremioUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Dremio user client configured")
}

func (r *userResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				Description: descriptions["email"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"first_name": schema.StringAttribute{
				Description: descriptions["first_name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"last_name": schema.StringAttribute{
				Description: descriptions["last_name"],
				Required:    true,
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
			},
			"password": schema.StringAttribute{
				Description: descriptions["password"],
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Computed:    true, // Must be computed if a default is applied
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"error_message": schema.StringAttribute{
				Description: descriptions["error_message"],
				Optional:    true,
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: descriptions["state"],
				Computed:    true,
			},
			"user_id": schema.StringAttribute{
				Description: descriptions["user_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timeouts": timeouts.AttributesAll(ctx),
		},
	}
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	waiterTimeout := dremioWaiter.CreateDremioUserWaitHandler(ctx, r.client.DefaultAPI, "", "", "", "").GetTimeout()
	createTimeout, diags := model.Timeouts.Create(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// prepare the payload struct for the create user request
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new Dremio user
	userResp, err := r.client.DefaultAPI.CreateDremioUser(ctx, projectId, region, instanceId).CreateDremioUserPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id": projectId,
		"region":     region,
		"user_id":    userResp.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	_, err = dremioWaiter.CreateDremioUserWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId, userResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Dremio user", fmt.Sprintf("Dremio user creation waiting: %v", err))
		return
	}

	err = mapFields(userResp, &model.Model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Dremio user", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Dremio user created")
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	userId := model.UserId.ValueString()
	if userId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)

	userResp, err := r.client.DefaultAPI.GetDremioUser(ctx, projectId, region, instanceId, userId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading dremio user", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(userResp, &model.Model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading dremio user", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Dremio user read")
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// We don't allow updates on Dremio users.
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model UserModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	waiterTimeout := dremioWaiter.DeleteDremioUserWaitHandler(ctx, r.client.DefaultAPI, "", "", "", "").GetTimeout()
	deleteTimeout, diags := model.Timeouts.Delete(ctx, waiterTimeout+core.DefaultTimeoutMargin)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	userId := model.UserId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)

	err := r.client.DefaultAPI.DeleteDremioUser(ctx, projectId, region, instanceId, userId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Dremio user", fmt.Sprintf("Calling API: %v", err))
	}

	ctx = core.LogResponse(ctx)

	_, err = dremioWaiter.DeleteDremioUserWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId, userId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Dremio user", fmt.Sprintf("Dremio user deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Dremio user deleted")
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing dremio user",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[instance_id],[user_id] got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"region":      idParts[1],
		"instance_id": idParts[2],
		"user_id":     idParts[3],
	})

	tflog.Info(ctx, "Dremio user state imported")
}

func mapFields(userResp *dremioSdk.DremioUserResponse, model *Model, region string) error {
	if userResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.UserId = types.StringValue(userResp.Id)

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		model.InstanceId.ValueString(),
		model.UserId.ValueString(),
	)

	model.Description = types.StringPointerValue(userResp.Description)
	model.Email = types.StringValue(userResp.Email)
	model.FirstName = types.StringValue(userResp.FirstName)
	model.LastName = types.StringValue(userResp.LastName)
	model.Name = types.StringValue(userResp.Name)

	model.State = types.StringValue(string(userResp.State))
	model.ErrorMessage = types.StringPointerValue(userResp.ErrorMessage)

	return nil
}

func toCreatePayload(model *UserModel) (*dremioSdk.CreateDremioUserPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model input is nil")
	}

	payload := &dremioSdk.CreateDremioUserPayload{
		Description: model.Description.ValueStringPointer(),
		Email:       model.Email.ValueString(),
		FirstName:   model.FirstName.ValueString(),
		LastName:    model.LastName.ValueString(),
		Name:        model.Name.ValueString(),
		Password:    model.Password.ValueString(),
	}

	return payload, nil
}
