package database

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdk "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	sqlserverflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &databaseResource{}
	_ resource.ResourceWithConfigure   = &databaseResource{}
	_ resource.ResourceWithImportState = &databaseResource{}
	_ resource.ResourceWithModifyPlan  = &databaseResource{}
)

func NewDatabaseResource() resource.Resource {
	return &databaseResource{}
}

type databaseResource struct {
	client       *sdk.APIClient
	providerData core.ProviderData
}

type Model struct {
	SharedModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflex_database"
}

func (r *databaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	apiClient := sqlserverflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
}

func (r *databaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *databaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: descriptionResource,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptionId,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptionProjectId,
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
				Description: descriptionRegion,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptionInstanceId,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptionName,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"collation": schema.StringAttribute{
				Description: descriptionCollation,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"compatibility": schema.Int64Attribute{
				Description: descriptionCompatibility,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"owner": schema.StringAttribute{
				Description: descriptionOwner,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database_id": schema.Int64Attribute{
				Description: descriptionDatabaseId,
				Computed:    true,
			},
			"timeouts": timeouts.AttributesAll(ctx),
		},
	}
}

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := model.Timeouts.Create(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := model.Region.ValueString()

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating database", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	ctx = core.InitProviderContext(ctx)

	_, err = r.client.DefaultAPI.CreateDatabase(ctx, projectId, region, instanceId).CreateDatabasePayload(*payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating database", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceId,
		"name":        model.Name,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.DefaultAPI.GetDatabase(ctx, projectId, region, instanceId, model.Name.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error reading database after creation", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(apiResp, &model.SharedModel, region)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping resource", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SqlserverFlex database created")
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
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

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	name := model.Name.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = core.InitProviderContext(ctx)

	apiResp, err := r.client.DefaultAPI.GetDatabase(ctx, projectId, region, instanceId, name).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading database", fmt.Sprintf("calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(apiResp, &model.SharedModel, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading database", fmt.Sprintf("Calling API: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SqlserverFlex database read")
}

func (r *databaseResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic,revive // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating database", "Database cannot be updated")
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := model.Timeouts.Delete(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	name := model.Name.ValueString()
	region := model.Region.ValueString()

	ctx = core.InitProviderContext(ctx)

	err := r.client.DefaultAPI.DeleteDatabase(ctx, projectId, region, instanceId, name).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting database", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "SqlserverFlex database deleted")
}

func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id,region,instance_id,name. Got: %q", req.ID),
		)
		return
	}
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"region":      idParts[1],
		"instance_id": idParts[2],
		"name":        idParts[3],
	})
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SqlserverFlex database state imported")
}

func mapFields(apiResp *sdk.GetDatabaseResponse, model *SharedModel, region string) error {
	if apiResp == nil {
		return fmt.Errorf("response is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		model.InstanceId.ValueString(),
		apiResp.Name,
	)

	model.Region = types.StringValue(region)
	model.Name = types.StringValue(apiResp.Name)
	model.Collation = types.StringValue(apiResp.CollationName)
	model.Compatibility = types.Int64Value(int64(apiResp.CompatibilityLevel))
	model.Owner = types.StringValue(apiResp.Owner)
	model.DatabaseId = types.Int64Value(apiResp.Id)

	return nil
}

func toCreatePayload(model *Model) (*sdk.CreateDatabasePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	payload := &sdk.CreateDatabasePayload{
		Name:  model.Name.ValueString(),
		Owner: model.Owner.ValueString(),
	}

	if !model.Collation.IsNull() && !model.Collation.IsUnknown() {
		payload.Collation = model.Collation.ValueStringPointer()
	}
	if !model.Compatibility.IsNull() && !model.Compatibility.IsUnknown() {
		compatibility := int32(model.Compatibility.ValueInt64()) // nolint:gosec // user input is controlled by schema
		payload.Compatibility = &compatibility
	}
	return payload, nil
}
