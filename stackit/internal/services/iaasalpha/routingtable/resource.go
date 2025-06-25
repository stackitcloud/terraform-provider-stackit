package routingtable

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasalphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &routingTableResource{}
	_ resource.ResourceWithConfigure   = &routingTableResource{}
	_ resource.ResourceWithImportState = &routingTableResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	OrganizationId types.String `tfsdk:"organization_id"`
	RoutingTableId types.String `tfsdk:"routing_table_id"`
	Name           types.String `tfsdk:"name"`
	NetworkAreaId  types.String `tfsdk:"network_area_id"`
	Description    types.String `tfsdk:"description"`
	Labels         types.Map    `tfsdk:"labels"`
	Region         types.String `tfsdk:"region"`
}

// NewRoutingTableResource is a helper function to simplify the provider implementation.
func NewRoutingTableResource() resource.Resource {
	return &routingTableResource{}
}

// routingTableResource is the resource implementation.
type routingTableResource struct {
	client       *iaasalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *routingTableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_table"
}

// Configure adds the provider configured client to the resource.
func (r *routingTableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_routing_table", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := iaasalphaUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "IaaS alpha client configured")
}

func (r *routingTableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Routing table resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddBetaDescription(description),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`organization_id`,`region`,`network_area_id`,`routing_table_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT organization ID to which the routing table is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"routing_table_id": schema.StringAttribute{
				Description: "The routing tables ID.",
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
				Description: "The name of the routing table.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"network_area_id": schema.StringAttribute{
				Description: "The network area ID to which the routing table is associated.",
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
			"description": schema.StringAttribute{
				Description: "Description of the routing table.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(127),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *routingTableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating routing table", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	routingTable, err := r.client.AddRoutingTableToArea(ctx, organizationId, networkAreaId, region).AddRoutingTableToAreaPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating routing table", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, routingTable, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating routing table.", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table created")
}

// Read refreshes the Terraform state with the latest data.
func (r *routingTableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	routingTableId := model.RoutingTableId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	routingTableResp, err := r.client.GetRoutingTableOfArea(ctx, organizationId, networkAreaId, region, routingTableId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading routing table",
			fmt.Sprintf("routing table with ID %q does not exist in organization %q.", routingTableId, organizationId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Organization with ID %q not found or forbidden access", organizationId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, routingTableResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading routing table", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *routingTableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	routingTableId := model.RoutingTableId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating routing table", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	routingTable, err := r.client.UpdateRoutingTableOfArea(ctx, organizationId, networkAreaId, region, routingTableId).UpdateRoutingTableOfAreaPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating routing table", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, routingTable, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating routing table", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Routing table updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *routingTableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	routingTableId := model.RoutingTableId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	// Delete existing routing table
	err := r.client.DeleteRoutingTableFromArea(ctx, organizationId, networkAreaId, region, routingTableId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting routing table", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Routing table deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: organization_id,region,routing_table_id
func (r *routingTableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing routing table",
			fmt.Sprintf("Expected import identifier with format: [organization_id],[region],[network_area_id],[routing_table_id]  Got: %q", req.ID),
		)
		return
	}

	organizationId := idParts[0]
	region := idParts[1]
	networkAreaId := idParts[2]
	routingTableId := idParts[3]
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)
	ctx = tflog.SetField(ctx, "routing_table_id", routingTableId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), region)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_area_id"), networkAreaId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("routing_table_id"), routingTableId)...)
	tflog.Info(ctx, "Routing table state imported")
}

func mapFields(ctx context.Context, routingTable *iaasalpha.RoutingTable, model *Model, region string) error {
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

	model.Id = utils.BuildInternalTerraformId(model.OrganizationId.ValueString(), region, model.NetworkAreaId.ValueString(), routingTableId)

	labels, err := iaasUtils.MapLabels(ctx, routingTable.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.RoutingTableId = types.StringValue(routingTableId)
	model.Name = types.StringPointerValue(routingTable.Name)
	model.Description = types.StringPointerValue(routingTable.Description)
	model.Labels = labels
	model.Region = types.StringValue(region)
	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaasalpha.AddRoutingTableToAreaPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaasalpha.AddRoutingTableToAreaPayload{
		Description: conversion.StringValueToPointer(model.Description),
		Name:        conversion.StringValueToPointer(model.Name),
		Labels:      &labels,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaasalpha.UpdateRoutingTableOfAreaPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaasalpha.UpdateRoutingTableOfAreaPayload{
		Description: conversion.StringValueToPointer(model.Description),
		Name:        conversion.StringValueToPointer(model.Name),
		Labels:      &labels,
	}, nil
}
