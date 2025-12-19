package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/logs"
	"github.com/stackitcloud/stackit-sdk-go/services/logs/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logs/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &logsInstanceResource{}
	_ resource.ResourceWithConfigure   = &logsInstanceResource{}
	_ resource.ResourceWithImportState = &logsInstanceResource{}
	_ resource.ResourceWithModifyPlan  = &logsInstanceResource{}
)

var schemaDescriptions = map[string]string{
	"id":              "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`\".",
	"instance_id":     "The Logs instance ID",
	"region":          "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"project_id":      "STACKIT project ID associated with the logs instance",
	"acl":             "ACL entries for the logs instance",
	"created":         "Time when the distribution was created",
	"datasource_url":  "Logs instance datasource URL, can be used in Grafana as datasource URL",
	"description":     "The description of the Logs instance",
	"display_name":    "The displayed name of the Logs instance",
	"ingest_otlp_url": "The Logs instance's ingest logs via OTLP URL",
	"ingest_url":      "The logs instance's ingest logs URL",
	"query_range_url": "The Logs instance's query range URL",
	"query_url":       "The Logs instance's query URL",
	"retention_days":  "The log retention time in days",
	"status":          "The status of the Logs instance",
}

type Model struct {
	ID            types.String `tfsdk:"id"`              // Required by Terraform
	InstanceID    types.String `tfsdk:"instance_id"`     // The Logs instance ID
	Region        types.String `tfsdk:"region"`          // STACKIT region name the resource is located in
	ProjectID     types.String `tfsdk:"project_id"`      // ProjectID associated with the logs instance
	ACL           types.List   `tfsdk:"acl"`             // ACL entries for the logs instance
	Created       types.String `tfsdk:"created"`         // When the instance was created
	DatasourceURL types.String `tfsdk:"datasource_url"`  // Logs instance datasource URL, can be used in Grafana as datasource URL
	Description   types.String `tfsdk:"description"`     // The description of the Logs instance
	DisplayName   types.String `tfsdk:"display_name"`    // The displayed name of the Logs instance
	IngestOTLPURL types.String `tfsdk:"ingest_otlp_url"` // The Logs instance's ingest logs via OTLP URL
	IngestURL     types.String `tfsdk:"ingest_url"`      // The logs instance's ingest logs URL
	QueryRangeURL types.String `tfsdk:"query_range_url"` // The Logs instance's query range URL
	QueryURL      types.String `tfsdk:"query_url"`       // The Logs instance's query URL
	RetentionDays types.Int64  `tfsdk:"retention_days"`  // The log retention time in days
	Status        types.String `tfsdk:"status"`          // The status of the Logs instance
}

type logsInstanceResource struct {
	client       *logs.APIClient
	providerData core.ProviderData
}

func NewLogsInstanceResource() resource.Resource {
	return &logsInstanceResource{}
}

func (r *logsInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_logs_instance", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	r.providerData = providerData
	tflog.Info(ctx, "Logs client configured")
}

func (r *logsInstanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *logsInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logs_instance"
}

func (r *logsInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Logs instance resource schema.", core.Resource),
		Description:         fmt.Sprintf("Logs instance resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Computed:    true,
				Validators:  []validator.String{validate.UUID()},
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
			},
			"acl": schema.ListAttribute{
				Description: schemaDescriptions["acl"],
				ElementType: types.StringType,
				Optional:    true,
			},
			"created": schema.StringAttribute{
				Description: schemaDescriptions["created"],
				Computed:    true,
			},
			"datasource_url": schema.StringAttribute{
				Description: schemaDescriptions["datasource_url"],
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Optional:    true,
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Required:    true,
			},
			"ingest_otlp_url": schema.StringAttribute{
				Description: schemaDescriptions["ingest_otlp_url"],
				Computed:    true,
			},
			"ingest_url": schema.StringAttribute{
				Description: schemaDescriptions["ingest_url"],
				Computed:    true,
			},
			"query_range_url": schema.StringAttribute{
				Description: schemaDescriptions["query_range_url"],
				Computed:    true,
			},
			"query_url": schema.StringAttribute{
				Description: schemaDescriptions["query_url"],
				Computed:    true,
			},
			"retention_days": schema.Int64Attribute{
				Description: schemaDescriptions["retention_days"],
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
		},
	}
}

func (r *logsInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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

	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Logs Instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	regionId := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "region", regionId)
	createResp, err := r.client.CreateLogsInstance(ctx, projectId, regionId).CreateLogsInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Logs Instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.CreateLogsInstanceWaitHandler(ctx, r.client, projectId, regionId, *createResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Logs Instance", fmt.Sprintf("Waiting for Logs Instance to become active: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Logs Instance", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Logs instance created")
}

func (r *logsInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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

	instanceResponse, err := r.client.GetLogsInstance(ctx, projectID, region, instanceID).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading logs instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, instanceResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading logs instance", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Logs Instance read")
}

func (r *logsInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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

	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Logs Instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateResp, err := r.client.UpdateLogsInstance(ctx, projectID, region, instanceID).UpdateLogsInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Logs Instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, updateResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Logs Instance", fmt.Sprintf("Processing response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Logs Instance updated")
}

func (r *logsInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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

	err := r.client.DeleteLogsInstance(ctx, projectID, region, instanceID).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Logs Instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteLogsInstanceWaitHandler(ctx, r.client, projectID, region, instanceID).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Logs Instance", fmt.Sprintf("Waiting for Logs Instance to be deleted: %v", err))
		return
	}

	tflog.Info(ctx, "Logs Instance deleted")
}

func (r *logsInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing Logs Instance", fmt.Sprintf("Invalid import ID %q: expected format is `project_id`,`region`,`instance_id`", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	tflog.Info(ctx, "Logs Instance state imported")
}

func toCreatePayload(model *Model) (*logs.CreateLogsInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	var acls []string
	for i, acl := range model.ACL.Elements() {
		aclString, ok := acl.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected acl at index %d to be of type %T, got %T", i, types.String{}, acl)
		}
		acls = append(acls, aclString.ValueString())
	}
	var payloadACLs *[]string
	if len(acls) > 0 {
		payloadACLs = &acls
	}

	payload := &logs.CreateLogsInstancePayload{
		Acl:           payloadACLs,
		Description:   conversion.StringValueToPointer(model.Description),
		DisplayName:   conversion.StringValueToPointer(model.DisplayName),
		RetentionDays: conversion.Int64ValueToPointer(model.RetentionDays),
	}

	return payload, nil
}

func mapFields(ctx context.Context, instance *logs.LogsInstance, model *Model) error {
	if instance == nil {
		return fmt.Errorf("instance is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	if instance.Status == nil {
		return fmt.Errorf("instance status is nil")
	}
	if instance.Created == nil {
		return fmt.Errorf("instance created is nil")
	}
	var instanceID string
	if model.InstanceID.ValueString() != "" {
		instanceID = model.InstanceID.ValueString()
	} else if instance.Id != nil {
		instanceID = *instance.Id
	} else {
		return fmt.Errorf("instance id not present")
	}

	aclList := types.ListNull(types.StringType)
	var diags diag.Diagnostics
	if instance.Acl != nil && len(*instance.Acl) > 0 {
		aclList, diags = types.ListValueFrom(ctx, types.StringType, instance.Acl)
		if diags.HasError() {
			return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
		}
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), model.Region.ValueString(), instanceID)
	model.InstanceID = types.StringValue(instanceID)
	model.ACL = aclList
	model.Created = types.StringValue(instance.Created.String())
	model.DatasourceURL = types.StringPointerValue(instance.DatasourceUrl)
	model.Description = types.StringPointerValue(instance.Description)
	model.DisplayName = types.StringPointerValue(instance.DisplayName)
	model.IngestOTLPURL = types.StringPointerValue(instance.IngestOtlpUrl)
	model.IngestURL = types.StringPointerValue(instance.IngestUrl)
	model.QueryRangeURL = types.StringPointerValue(instance.QueryRangeUrl)
	model.QueryURL = types.StringPointerValue(instance.QueryUrl)
	model.RetentionDays = types.Int64PointerValue(instance.RetentionDays)
	model.Status = types.StringValue(string(*instance.Status))

	return nil
}

func toUpdatePayload(model *Model) (*logs.UpdateLogsInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	var acls []string
	for i, acl := range model.ACL.Elements() {
		aclString, ok := acl.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected acl at index %d to be of type %T, got %T", i, types.String{}, acl)
		}
		acls = append(acls, aclString.ValueString())
	}
	var payloadACLs *[]string
	if len(acls) > 0 {
		payloadACLs = &acls
	}

	payload := &logs.UpdateLogsInstancePayload{
		Acl:           payloadACLs,
		Description:   conversion.StringValueToPointer(model.Description),
		DisplayName:   conversion.StringValueToPointer(model.DisplayName),
		RetentionDays: conversion.Int64ValueToPointer(model.RetentionDays),
	}

	return payload, nil
}
