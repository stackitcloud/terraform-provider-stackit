package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/telemetryrouter/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &telemetryRouterInstanceResource{}
	_ resource.ResourceWithConfigure   = &telemetryRouterInstanceResource{}
	_ resource.ResourceWithImportState = &telemetryRouterInstanceResource{}
	_ resource.ResourceWithModifyPlan  = &telemetryRouterInstanceResource{}
)

var schemaDescriptions = map[string]string{
	"id":                    "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`\".",
	"instance_id":           "The TelemetryRouter instance ID",
	"region":                "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"project_id":            "STACKIT project ID associated with the TelemetryRouter instance",
	"display_name":          "The display name of the TelemetryRouter instance",
	"description":           "The description of the TelemetryRouter instance",
	"filter":                "The TelemetryRouter global filter settings",
	"filter.attributes":     "The TelemetryRouter global filter attributes",
	"filter.attributes.key": "The TelemetryRouter global filter attribute key",
	"filter.attributes.level": fmt.Sprintf(
		"The TelemetryRouter global filter attribute level, possible values: %s",
		tfutils.FormatPossibleValues("resource", "scope", "logRecord"),
	),
	"filter.attributes.matcher": fmt.Sprintf(
		"The TelemetryRouter global filter attribute matcher, possible values: %s",
		tfutils.FormatPossibleValues("=", "!="),
	),
	"filter.attributes.values": "The TelemetryRouter global filter attributes",
	"creation_time":            "The date and time the creation of the TelemetryRouter instance was initiated",
	"uri":                      "The TelemetryRouter instance's URI",
	"status": fmt.Sprintf(
		"The status of the TelemetryRouter instance, possible values: %s",
		tfutils.FormatPossibleValues("active", "deleting", "reconciling"),
	),
}

type Model struct {
	ID           types.String `tfsdk:"id"` // Required by Terraform
	InstanceID   types.String `tfsdk:"instance_id"`
	Region       types.String `tfsdk:"region"`
	ProjectID    types.String `tfsdk:"project_id"`
	DisplayName  types.String `tfsdk:"display_name"`
	Description  types.String `tfsdk:"description"`
	Filter       types.Object `tfsdk:"filter"`
	CreationTime types.String `tfsdk:"creation_time"`
	URI          types.String `tfsdk:"uri"`
	Status       types.String `tfsdk:"status"`
}

// Struct corresponding to Model.Filter
type filter struct {
	Attributes types.List `tfsdk:"attributes"`
}

// Types corresponding to filter
var filterTypes = map[string]attr.Type{
	"attributes": basetypes.ListType{ElemType: types.ObjectType{AttrTypes: attributeTypes}},
}

// Struct corresponding to a single attribute
type attribute struct {
	Key     types.String `tfsdk:"key"`
	Level   types.String `tfsdk:"level"`
	Matcher types.String `tfsdk:"matcher"`
	Values  types.List   `tfsdk:"values"`
}

// Types coresponding to attributes
var attributeTypes = map[string]attr.Type{
	"key":     basetypes.StringType{},
	"level":   basetypes.StringType{},
	"matcher": basetypes.StringType{},
	"values":  basetypes.ListType{ElemType: types.StringType},
}

type telemetryRouterInstanceResource struct {
	client       *telemetryrouter.APIClient
	providerData core.ProviderData
}

func NewTelemetryRouterInstanceResource() resource.Resource {
	return &telemetryRouterInstanceResource{}
}

func (r *telemetryRouterInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	r.providerData = providerData
	tflog.Info(ctx, "TelemetryRouter client configured")
}

func (r *telemetryRouterInstanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *telemetryRouterInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_telemetryrouter_instance"
}

func (r *telemetryRouterInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("TelemetryRouter instance resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
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
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Optional:    true,
			},
			"filter": schema.SingleNestedAttribute{
				Description: schemaDescriptions["filter"],
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"attributes": schema.ListNestedAttribute{
						Description: schemaDescriptions["filter.attributes"],
						Required:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Description: schemaDescriptions["filter.attributes.key"],
									Required:    true,
								},
								"level": schema.StringAttribute{
									Description: schemaDescriptions["filter.attributes.level"],
									Required:    true,
									Validators: []validator.String{
										stringvalidator.OneOf("resource", "scope", "logRecord"),
									},
								},
								"matcher": schema.StringAttribute{
									Description: schemaDescriptions["filter.attributes.matcher"],
									Required:    true,
									Validators: []validator.String{
										stringvalidator.OneOf("=", "!="),
									},
								},
								"values": schema.ListAttribute{
									Description: schemaDescriptions["filter.attributes.values"],
									ElementType: types.StringType,
									Required:    true,
								},
							},
						},
					},
				},
			},
			"creation_time": schema.StringAttribute{
				Description: schemaDescriptions["creation_time"],
				Computed:    true,
			},
			"uri": schema.StringAttribute{
				Description: schemaDescriptions["uri"],
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
		},
	}
}

func (r *telemetryRouterInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(ctx, resp.Diagnostics, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter Instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	regionId := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "region", regionId)
	createResp, err := r.client.DefaultAPI.CreateTelemetryRouter(ctx, projectId, regionId).CreateTelemetryRouterPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter Instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	waitResp, err := wait.CreateTelemetryRouterWaitHandler(ctx, r.client.DefaultAPI, projectId, regionId, createResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter Instance", fmt.Sprintf("Waiting for TelemetryRouter Instance to become active: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter Instance", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter instance created")
}

func (r *telemetryRouterInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)

	instanceResponse, err := r.client.DefaultAPI.GetTelemetryRouter(ctx, projectID, region, instanceID).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryRouter instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, instanceResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryRouter instance", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter Instance read", map[string]any{
		"instance_id": instanceID,
	})
}

func (r *telemetryRouterInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)

	payload, err := toUpdatePayload(ctx, resp.Diagnostics, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter Instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateResp, err := r.client.DefaultAPI.UpdateTelemetryRouter(ctx, projectID, region, instanceID).UpdateTelemetryRouterPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter Instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.UpdateTelemetryRouterWaitHandler(ctx, r.client.DefaultAPI, projectID, region, updateResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter Instance", fmt.Sprintf("Waiting for TelemetryRouter Instance to become active: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter Instance", fmt.Sprintf("Processing response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter Instance updated", map[string]any{
		"instance_id": instanceID,
	})
}

func (r *telemetryRouterInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)

	err := r.client.DefaultAPI.DeleteTelemetryRouter(ctx, projectID, region, instanceID).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryRouter Instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.DeleteTelemetryRouterWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryRouter Instance", fmt.Sprintf("Waiting for TelemetryRouter Instance to become deleted: %v", err))
		return
	}

	tflog.Info(ctx, "TelemetryRouter Instance deleted")
}

func (r *telemetryRouterInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing TelemetryRouter Instance", fmt.Sprintf("Invalid import ID %q: expected format is `project_id`,`region`,`instance_id`", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	tflog.Info(ctx, "TelemetryRouter Instance state imported")
}

func toCreatePayload(ctx context.Context, diags diag.Diagnostics, model *Model) (*telemetryrouter.CreateTelemetryRouterPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	payload := &telemetryrouter.CreateTelemetryRouterPayload{
		DisplayName: model.DisplayName.ValueString(),
		Description: conversion.StringValueToPointer(model.Description),
	}

	configFilter, err := toConfigFilter(ctx, diags, model)
	if err != nil {
		return nil, err
	}
	payload.Filter = configFilter

	return payload, nil
}

func toConfigFilter(ctx context.Context, diags diag.Diagnostics, model *Model) (*telemetryrouter.ConfigFilter, error) {
	if !model.Filter.IsNull() && !model.Filter.IsUnknown() {
		var fltr filter
		diags.Append(model.Filter.As(ctx, &fltr, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, fmt.Errorf("converting filter object: %v", diags.Errors())
		}

		var attributes []attribute
		diags.Append(fltr.Attributes.ElementsAs(ctx, &attributes, false)...)
		if diags.HasError() {
			return nil, fmt.Errorf("converting attributes list: %v", diags.Errors())
		}

		configFilterAttributes := make([]telemetryrouter.ConfigFilterAttributes, 0, len(attributes))
		for _, item := range attributes {
			var values []string
			valuesDiags := item.Values.ElementsAs(ctx, &values, false)
			diags.Append(valuesDiags...)
			if !valuesDiags.HasError() {
				configFilterAttributes = append(configFilterAttributes, telemetryrouter.ConfigFilterAttributes{
					Key:     item.Key.ValueString(),
					Level:   telemetryrouter.ConfigFilterLevel(item.Level.ValueString()),
					Matcher: telemetryrouter.ConfigFilterMatcher(item.Matcher.ValueString()),
					Values:  values,
				})
			}
		}
		if len(configFilterAttributes) > 0 {
			return telemetryrouter.NewConfigFilter(
				configFilterAttributes,
			), nil
		}
	}

	return nil, nil
}

func mapFields(ctx context.Context, instance *telemetryrouter.TelemetryRouterResponse, model *Model) error {
	if instance == nil {
		return fmt.Errorf("instance is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	var instanceID string
	if model.InstanceID.ValueString() != "" {
		instanceID = model.InstanceID.ValueString()
	} else if instance.Id != "" {
		instanceID = instance.Id
	} else {
		return fmt.Errorf("instance id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), model.Region.ValueString(), instanceID)
	model.InstanceID = types.StringValue(instanceID)
	model.DisplayName = types.StringValue(instance.DisplayName)
	model.Description = types.StringPointerValue(instance.Description)
	model.CreationTime = types.StringValue(instance.CreationTime.Format(time.RFC3339))
	model.URI = types.StringValue(instance.Uri)
	model.Status = types.StringValue(instance.Status)

	if err := mapFilter(ctx, instance, model); err != nil {
		return fmt.Errorf("map filter: %w", err)
	}

	return nil
}

func mapFilter(ctx context.Context, instance *telemetryrouter.TelemetryRouterResponse, model *Model) error {
	if instance.Filter == nil {
		model.Filter = types.ObjectNull(filterTypes)
		return nil
	}

	attrList := []attr.Value{}
	for _, currentAttr := range instance.Filter.Attributes {
		values, diags := types.ListValueFrom(ctx, types.StringType, currentAttr.Values)
		if diags.HasError() {
			return fmt.Errorf("mapping filter values: %w", core.DiagsToError(diags))
		}
		attrModel, diags := types.ObjectValueFrom(ctx, attributeTypes, attribute{
			Key:     types.StringValue(currentAttr.Key),
			Level:   types.StringValue(string(currentAttr.Level)),
			Matcher: types.StringValue(string(currentAttr.Matcher)),
			Values:  values,
		})
		if diags.HasError() {
			return fmt.Errorf("mapping filter attributes: %w", core.DiagsToError(diags))
		}
		attrList = append(attrList, attrModel)
	}

	var attrConfigs basetypes.ListValue
	var diags diag.Diagnostics
	if len(attrList) == 0 {
		attrConfigs = types.ListNull(types.ObjectType{AttrTypes: attributeTypes})
	} else {
		attrConfigs, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, attrList)
		if diags.HasError() {
			return fmt.Errorf("mapping attributes: %w", core.DiagsToError(diags))
		}
	}

	filterValue, diags := types.ObjectValueFrom(ctx, filterTypes, filter{
		Attributes: attrConfigs,
	})
	if diags.HasError() {
		return fmt.Errorf("mapping filter: %w", core.DiagsToError(diags))
	}
	model.Filter = filterValue

	return nil
}

func toUpdatePayload(ctx context.Context, diags diag.Diagnostics, model *Model) (*telemetryrouter.UpdateTelemetryRouterPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	payload := &telemetryrouter.UpdateTelemetryRouterPayload{
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
		Description: conversion.StringValueToPointer(model.Description),
	}

	configFilter, err := toConfigFilter(ctx, diags, model)
	if err != nil {
		return nil, err
	}
	payload.Filter = configFilter

	return payload, nil
}
