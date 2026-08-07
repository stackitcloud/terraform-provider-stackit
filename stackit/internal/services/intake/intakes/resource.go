package intakes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	intakeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/intake/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	intake "github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &intakesResource{}
	_ resource.ResourceWithConfigure   = &intakesResource{}
	_ resource.ResourceWithImportState = &intakesResource{}
	_ resource.ResourceWithModifyPlan  = &intakesResource{}
)

// Model is the internal model of the terraform resource
type Model struct {
	Id                  types.String `tfsdk:"id"` // needed by TF
	ProjectId           types.String `tfsdk:"project_id"`
	RunnerId            types.String `tfsdk:"runner_id"`
	IntakeId            types.String `tfsdk:"intake_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Labels              types.Map    `tfsdk:"labels"`
	Region              types.String `tfsdk:"region"`
	Uri                 types.String `tfsdk:"uri"`
	CreateTime          types.String `tfsdk:"create_time"`
	DremioPAT           types.String `tfsdk:"dremio_personal_access_token"`
	DremioTokenEndpoint types.String `tfsdk:"dremio_token_endpoint"`
	CatalogAuthType     types.String `tfsdk:"catalog_auth_type"`
	CatalogNamespace    types.String `tfsdk:"catalog_namespace"`
	CatalogPartitioning types.String `tfsdk:"catalog_partitioning"`
	CatalogPartitionBy  types.List   `tfsdk:"catalog_partition_by"`
	CatalogTableName    types.String `tfsdk:"catalog_table_name"`
	CatalogUri          types.String `tfsdk:"catalog_uri"`
	CatalogWarehouse    types.String `tfsdk:"catalog_warehouse"`
}

// NewIntakesResource is a helper function to simplify the provider implementation.
func NewIntakesResource() resource.Resource {
	return &intakesResource{}
}

// intakesResource is the resource implementation.
type intakesResource struct {
	client       *intake.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *intakesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intakes"
}

// Configure adds the provider configured client to the resource.
func (r *intakesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := intakeUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	r.providerData = providerData
	tflog.Info(ctx, "Intakes client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *intakesResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

// Schema defines the schema for the data source
func (r *intakesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{ //nolint:gosec // descriptions
		"main":                         "Manages STACKIT Intake.",
		"id":                           "Terraform's internal resource identifier. It is structured as `project_id`,`region`,`intake_id`.",
		"project_id":                   "STACKIT Project ID to which the intake is associated.",
		"runner_id":                    "The runner ID.",
		"intake_id":                    "The intake ID.",
		"name":                         "The name of the intake.",
		"region":                       "The resource region. If not defined, the provider region is used.",
		"description":                  "The description of the intake.",
		"labels":                       "User-defined labels.",
		"uri":                          "The URI of the intake.",
		"create_time":                  "The creation time of the intake.",
		"dremio_personal_access_token": "The Dremio personal access token.",
		"dremio_token_endpoint":        "The Dremio token endpoint.",
		"catalog_auth_type":            "The catalog authentication type.",
		"catalog_namespace":            "The catalog namespace.",
		"catalog_partitioning":         "The catalog partitioning.",
		"catalog_partition_by":         "The catalog partition by.",
		"catalog_table_name":           "The catalog table name.",
		"catalog_uri":                  "The catalog URI.",
		"catalog_warehouse":            "The catalog warehouse.",
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
			"runner_id": schema.StringAttribute{
				Description: descriptions["runner_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"intake_id": schema.StringAttribute{
				Description: descriptions["intake_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"uri": schema.StringAttribute{
				Description: descriptions["uri"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_time": schema.StringAttribute{
				Description: descriptions["create_time"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dremio_personal_access_token": schema.StringAttribute{
				Description: descriptions["dremio_personal_access_token"],
				Optional:    true,
				Sensitive:   true,
			},
			"dremio_token_endpoint": schema.StringAttribute{
				Description: descriptions["dremio_token_endpoint"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_auth_type": schema.StringAttribute{
				Description: descriptions["catalog_auth_type"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_namespace": schema.StringAttribute{
				Description: descriptions["catalog_namespace"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_partitioning": schema.StringAttribute{
				Description: descriptions["catalog_partitioning"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_partition_by": schema.ListAttribute{
				Description: descriptions["catalog_partition_by"],
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_table_name": schema.StringAttribute{
				Description: descriptions["catalog_table_name"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_uri": schema.StringAttribute{
				Description: descriptions["catalog_uri"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_warehouse": schema.StringAttribute{
				Description: descriptions["catalog_warehouse"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *intakesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating intake", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	intakeResp, err := r.client.DefaultAPI.CreateIntake(ctx, projectId, region).CreateIntakePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating intake", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id": projectId,
		"region":     region,
		"intake_id":  intakeResp.Id,
	})

	if resp.Diagnostics.HasError() {
		return
	}

	_, err = wait.CreateIntakeWaitHandler(ctx, r.client.DefaultAPI, projectId, region, intakeResp.GetId()).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating intake", fmt.Sprintf("Intake creation waiting: %v", err))
		return
	}

	err = mapFields(ctx, intakeResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating intake", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Intake created")
}

// Read refreshes the Terraform state with the latest data.
func (r *intakesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	intakeId := model.IntakeId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "intake_id", intakeId)

	intakeResp, err := r.client.DefaultAPI.GetIntake(ctx, projectId, region, intakeId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading intake", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, intakeResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading intake", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Intake read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *intakesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model, state Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	intakeId := model.IntakeId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "intake_id", intakeId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating intake", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	intakeResp, err := r.client.DefaultAPI.UpdateIntake(ctx, projectId, region, intakeId).UpdateIntakePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating intake", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.UpdateIntakeWaitHandler(ctx, r.client.DefaultAPI, projectId, region, intakeId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating intake", fmt.Sprintf("Intake update waiting: %v", err))
		return
	}

	err = mapFields(ctx, intakeResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating intake", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Intake updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *intakesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	intakeId := model.IntakeId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "intake_id", intakeId)

	err := r.client.DefaultAPI.DeleteIntake(ctx, projectId, region, intakeId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			tflog.Info(ctx, "Intake already deleted")
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting intake", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteIntakeWaitHandler(ctx, r.client.DefaultAPI, projectId, region, intakeId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting intake", fmt.Sprintf("Intake deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Intake deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the Intake resource import identifier is: [project_id],[region],[intake_id]
func (r *intakesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing intake",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[intake_id], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"intake_id":  idParts[2],
	})

	tflog.Info(ctx, "Intake state imported")
}

// Maps intake fields to the provider internal model
func mapFields(ctx context.Context, intakeResp *intake.IntakeResponse, model *Model, region string) error {
	if intakeResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		intakeResp.Id,
	)

	labels, err := utils.MapLabels(ctx, &intakeResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.IntakeId = types.StringValue(intakeResp.Id)
	model.RunnerId = types.StringValue(intakeResp.IntakeRunnerId)
	model.Name = types.StringValue(intakeResp.DisplayName)
	model.Labels = labels
	model.Description = types.StringPointerValue(intakeResp.Description)
	model.Region = types.StringValue(region)
	model.Uri = types.StringValue(intakeResp.Uri)
	model.CreateTime = types.StringValue(intakeResp.CreateTime.String())

	model.CatalogNamespace = types.StringPointerValue(intakeResp.Catalog.Namespace)
	model.CatalogTableName = types.StringPointerValue(intakeResp.Catalog.TableName)
	model.CatalogUri = types.StringValue(intakeResp.Catalog.Uri)
	model.CatalogWarehouse = types.StringValue(intakeResp.Catalog.Warehouse)

	if intakeResp.Catalog.Partitioning != nil {
		model.CatalogPartitioning = types.StringValue(string(*intakeResp.Catalog.Partitioning))
	} else {
		model.CatalogPartitioning = types.StringNull()
	}

	if intakeResp.Catalog.PartitionBy != nil {
		partitionByList, diags := types.ListValueFrom(ctx, types.StringType, intakeResp.Catalog.PartitionBy)
		if diags.HasError() {
			return fmt.Errorf("converting partition_by list: %v", diags)
		}
		model.CatalogPartitionBy = partitionByList
	} else {
		model.CatalogPartitionBy = types.ListNull(types.StringType)
	}

	if intakeResp.Catalog.Auth != nil {
		model.CatalogAuthType = types.StringValue(string(intakeResp.Catalog.Auth.Type))
		if intakeResp.Catalog.Auth.Dremio != nil {
			model.DremioTokenEndpoint = types.StringValue(intakeResp.Catalog.Auth.Dremio.TokenEndpoint)
		} else {
			model.DremioTokenEndpoint = types.StringNull()
		}
	} else {
		model.CatalogAuthType = types.StringNull()
		model.DremioTokenEndpoint = types.StringNull()
	}

	if model.DremioPAT.IsUnknown() {
		model.DremioPAT = types.StringNull()
	}

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*intake.CreateIntakePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := utils.LabelsToPayload(ctx, model.Labels)
	if err != nil {
		return nil, err
	}

	partitionBy, err := conversion.StringListToSlice(model.CatalogPartitionBy)
	if err != nil {
		return nil, err
	}

	var partitioning *intake.PartitioningType
	if !model.CatalogPartitioning.IsNull() && !model.CatalogPartitioning.IsUnknown() {
		p, err := intake.NewPartitioningTypeFromValue(model.CatalogPartitioning.ValueString())
		if err != nil {
			return nil, err
		}
		partitioning = p
	}

	var auth *intake.CatalogAuth
	if !model.CatalogAuthType.IsNull() && !model.CatalogAuthType.IsUnknown() {
		authType := model.CatalogAuthType.ValueString()
		auth = &intake.CatalogAuth{
			Type: intake.CatalogAuthType(authType),
		}
		if authType == "dremio" {
			auth.Dremio = intake.NewDremioAuth(model.DremioPAT.ValueString(), model.DremioTokenEndpoint.ValueString())
		}
	}

	return &intake.CreateIntakePayload{
		Description:    conversion.StringValueToPointer(model.Description),
		DisplayName:    model.Name.ValueString(),
		IntakeRunnerId: model.RunnerId.ValueString(),
		Labels:         labels,
		Catalog: intake.IntakeCatalog{
			Auth:         auth,
			Namespace:    conversion.StringValueToPointer(model.CatalogNamespace),
			PartitionBy:  partitionBy,
			Partitioning: partitioning,
			TableName:    conversion.StringValueToPointer(model.CatalogTableName),
			Uri:          model.CatalogUri.ValueString(),
			Warehouse:    model.CatalogWarehouse.ValueString(),
		},
	}, nil
}

// Build UpdateIntakePayload from provider's model
func toUpdatePayload(ctx context.Context, model *Model) (*intake.UpdateIntakePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	payload := &intake.UpdateIntakePayload{}
	payload.DisplayName = conversion.StringValueToPointer(model.Name)
	payload.Description = conversion.StringValueToPointer(model.Description)

	labels, err := utils.LabelsToPayload(ctx, model.Labels)
	if err != nil {
		return nil, err
	}
	payload.Labels = labels

	return payload, nil
}
