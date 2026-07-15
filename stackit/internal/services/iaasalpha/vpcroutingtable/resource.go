package vpcroutingtable

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vpcRoutingTableResource{}
	_ resource.ResourceWithConfigure   = &vpcRoutingTableResource{}
	_ resource.ResourceWithImportState = &vpcRoutingTableResource{}
	_ resource.ResourceWithModifyPlan  = &vpcRoutingTableResource{}
)

var schemaDescriptions = map[string]string{
	"id":               "Terraform's internal resource ID. It is structured as \"`project_id`,`vpc_id`,`region`,`routing_table_id`\".",
	"project_id":       "STACKIT project ID to which the regional routing table is associated.",
	"vpc_id":           "The vpc ID to which the regional routing table is associated.",
	"region":           "The resource region. If not defined, the provider region is used.",
	"routing_table_id": "The regional routing tables ID.",
	"name":             "The name of the regional routing table.",
	"description":      "Description of the regional routing table.",
	"labels":           "Labels are key-value string pairs which can be attached to a resource container",
	"dynamic_routes":   "This controls whether dynamic routes are propagated to this regional routing table",
	"system_routes":    "This allows installation of automatic system routes for connectivity between projects in the same VPC.",
}

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	Region         types.String `tfsdk:"region"`
	ProjectId      types.String `tfsdk:"project_id"`
	VpcId          types.String `tfsdk:"vpc_id"`
	RoutingTableId types.String `tfsdk:"routing_table_id"`

	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Labels        types.Map    `tfsdk:"labels"`
	DynamicRoutes types.Bool   `tfsdk:"dynamic_routes"`
	SystemRoutes  types.Bool   `tfsdk:"system_routes"`
}

// NewVpcRoutingTableResource is a helper function to simplify the provider implementation.
func NewVpcRoutingTableResource() resource.Resource {
	return &vpcRoutingTableResource{}
}

// vpcRoutingTableResource is the resource implementation.
type vpcRoutingTableResource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *vpcRoutingTableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_routing_table"
}

// Configure adds the provider configured client to the resource.
func (r *vpcRoutingTableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &r.providerData, features.VpcExperiment, "stackit_vpc_routing_table", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := iaasAlphaUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "IaaS v2alpha client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *vpcRoutingTableResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}

	var configModel Model
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

func (r *vpcRoutingTableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "VPC Regional routing table resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.VpcExperiment, core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"vpc_id": schema.StringAttribute{
				Description: schemaDescriptions["vpc_id"],
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
				Description: schemaDescriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(127),
				},
			},
			"routing_table_id": schema.StringAttribute{
				Description: schemaDescriptions["routing_table_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the regional routing table.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"labels": schema.MapAttribute{
				Description: schemaDescriptions["labels"],
				ElementType: types.StringType,
				Optional:    true,
			},
			"dynamic_routes": schema.BoolAttribute{
				Description: schemaDescriptions["dynamic_routes"],
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"system_routes": schema.BoolAttribute{
				Description: schemaDescriptions["system_routes"],
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *vpcRoutingTableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating vpc routing table", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	routingTable, err := r.client.DefaultAPI.AddVPCRoutingTable(ctx, projectId, vpcId, region).AddVPCRoutingTablePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating vpc routing table", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if routingTable.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating vpc routing table", "response did not return a valid ID")
		return
	}
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":       projectId,
		"vpc_id":           vpcId,
		"region":           region,
		"routing_table_id": routingTable.Id,
	})

	// Map response body to schema
	err = mapFields(ctx, routingTable, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating vpc routing table.", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC routing table created")
}

// Read refreshes the Terraform state with the latest data.
func (r *vpcRoutingTableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()

	if routingTableId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)

	routingTableResp, err := r.client.DefaultAPI.GetVPCRoutingTable(ctx, projectId, vpcId, region, routingTableId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading vpc routing table",
			fmt.Sprintf("vpc routing table with ID %q does not exist in project %q.", routingTableId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, routingTableResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading vpc routing table", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC routing table read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *vpcRoutingTableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)

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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating vpc routing table", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	routingTable, err := r.client.DefaultAPI.UpdateVPCRoutingTable(ctx, projectId, vpcId, region, routingTableId).UpdateVPCRoutingTablePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating vpc routing table", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, routingTable, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating vpc routing table", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC routing table updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *vpcRoutingTableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)

	// Delete existing routing table
	err := r.client.DefaultAPI.DeleteVPCRoutingTable(ctx, projectId, vpcId, region, routingTableId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting vpc routing table", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Routing table deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,vpc_id,region,routing_table_id
func (r *vpcRoutingTableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing routing table",
			fmt.Sprintf("Expected import identifier with format: [project_id],[vpc_id],[region],[routing_table_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":       idParts[0],
		"vpc_id":           idParts[1],
		"region":           idParts[2],
		"routing_table_id": idParts[3],
	})
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table state imported")
}

func mapFields(ctx context.Context, routingTable *iaas.VPCRoutingTable, model *Model, region string) error {
	if routingTable == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var routingTableId string
	if model.RoutingTableId.ValueString() != "" {
		routingTableId = model.RoutingTableId.ValueString()
	} else if routingTable.Id != nil {
		routingTableId = *routingTable.Id
	} else {
		return fmt.Errorf("routing table id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.VpcId.ValueString(), region, routingTableId)

	labels, err := iaasUtils.MapLabels(ctx, routingTable.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.RoutingTableId = types.StringValue(routingTableId)
	model.Name = types.StringValue(routingTable.Name)
	model.Description = types.StringPointerValue(routingTable.Description)
	model.Labels = labels
	model.Region = types.StringValue(region)
	model.SystemRoutes = types.BoolPointerValue(routingTable.SystemRoutes)
	model.DynamicRoutes = types.BoolPointerValue(routingTable.DynamicRoutes)
	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.AddVPCRoutingTablePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.AddVPCRoutingTablePayload{
		Description:   conversion.StringValueToPointer(model.Description),
		Name:          model.Name.ValueString(),
		Labels:        labels,
		SystemRoutes:  conversion.BoolValueToPointer(model.SystemRoutes),
		DynamicRoutes: conversion.BoolValueToPointer(model.DynamicRoutes),
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.UpdateVPCRoutingTablePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.UpdateVPCRoutingTablePayload{
		Description:   conversion.StringValueToPointer(model.Description),
		Name:          conversion.StringValueToPointer(model.Name),
		Labels:        labels,
		DynamicRoutes: conversion.BoolValueToPointer(model.DynamicRoutes),
		SystemRoutes:  conversion.BoolValueToPointer(model.SystemRoutes),
	}, nil
}
