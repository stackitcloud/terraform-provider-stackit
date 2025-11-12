package dns

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stackitcloud/stackit-sdk-go/services/dns/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	dnsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &recordSetResource{}
	_ resource.ResourceWithConfigure   = &recordSetResource{}
	_ resource.ResourceWithImportState = &recordSetResource{}
)

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	RecordSetId types.String `tfsdk:"record_set_id"`
	ZoneId      types.String `tfsdk:"zone_id"`
	ProjectId   types.String `tfsdk:"project_id"`
	Active      types.Bool   `tfsdk:"active"`
	Comment     types.String `tfsdk:"comment"`
	Name        types.String `tfsdk:"name"`
	Records     types.List   `tfsdk:"records"`
	TTL         types.Int64  `tfsdk:"ttl"`
	Type        types.String `tfsdk:"type"`
	Error       types.String `tfsdk:"error"`
	State       types.String `tfsdk:"state"`
	FQDN        types.String `tfsdk:"fqdn"`
}

// NewRecordSetResource is a helper function to simplify the provider implementation.
func NewRecordSetResource() resource.Resource {
	return &recordSetResource{}
}

// recordSetResource is the resource implementation.
type recordSetResource struct {
	client *dns.APIClient
}

// Metadata returns the resource type name.
func (r *recordSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record_set"
}

// Configure adds the provider configured client to the resource.
func (r *recordSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := dnsUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "DNS record set client configured")
}

// Schema defines the schema for the resource.
func (r *recordSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "DNS Record Set Resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`zone_id`,`record_set_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the dns record set is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"zone_id": schema.StringAttribute{
				Description: "The zone ID to which is dns record set is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"record_set_id": schema.StringAttribute{
				Description: "The rr set id.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the record which should be a valid domain according to rfc1035 Section 2.3.4. E.g. `example.com`",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"fqdn": schema.StringAttribute{
				Description: "Fully qualified domain name (FQDN) of the record set.",
				Computed:    true,
			},
			"records": schema.ListAttribute{
				Description: "Records.",
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(validate.RecordSet()),
				},
			},
			"ttl": schema.Int64Attribute{
				Description: "Time to live. E.g. 3600",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(60),
					int64validator.AtMost(99999999),
				},
			},
			"type": schema.StringAttribute{
				Description: "The record set type. E.g. `A` or `CNAME`",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"active": schema.BoolAttribute{
				Description: "Specifies if the record set is active or not. Defaults to `true`",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"comment": schema.StringAttribute{
				Description: "Comment.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"error": schema.StringAttribute{
				Description: "Error shows error in case create/update/delete failed.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(2000),
				},
			},
			"state": schema.StringAttribute{
				Description: "Record set state.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *recordSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	zoneId := model.ZoneId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "zone_id", zoneId)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating record set", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new recordset
	recordSetResp, err := r.client.CreateRecordSet(ctx, projectId, zoneId).CreateRecordSetPayload(*payload).Execute()
	if err != nil || recordSetResp.Rrset == nil || recordSetResp.Rrset.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating record set", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	recordSetId := *recordSetResp.Rrset.Id
	model.RecordSetId = types.StringValue(recordSetId)
	model.Id = utils.BuildInternalTerraformId(projectId, zoneId, recordSetId)

	// Set all unknown/null fields to null before saving state
	if err := utils.SetModelFieldsToNull(ctx, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating record set", fmt.Sprintf("Setting model fields to null: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !utils.ShouldWait() {
		tflog.Info(ctx, "Skipping wait; async mode for Crossplane/Upjet")
		return
	}

	waitResp, err := wait.CreateRecordSetWaitHandler(ctx, r.client, projectId, zoneId, *recordSetResp.Rrset.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating record set", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating record set", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "DNS record set created")
}

// Read refreshes the Terraform state with the latest data.
func (r *recordSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	zoneId := model.ZoneId.ValueString()
	recordSetId := model.RecordSetId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "zone_id", zoneId)
	ctx = tflog.SetField(ctx, "record_set_id", recordSetId)

	recordSetResp, err := r.client.GetRecordSet(ctx, projectId, zoneId, recordSetId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusGone) {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading record set", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if recordSetResp != nil && recordSetResp.Rrset.State != nil && *recordSetResp.Rrset.State == dns.RECORDSETSTATE_DELETE_SUCCEEDED {
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, recordSetResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading record set", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "DNS record set read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *recordSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	zoneId := model.ZoneId.ValueString()
	recordSetId := model.RecordSetId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "zone_id", zoneId)
	ctx = tflog.SetField(ctx, "record_set_id", recordSetId)

	// Generate API request body from model
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating record set", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update recordset
	_, err = r.client.PartialUpdateRecordSet(ctx, projectId, zoneId, recordSetId).PartialUpdateRecordSetPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating record set", err.Error())
		return
	}

	if !utils.ShouldWait() {
		tflog.Info(ctx, "Skipping wait; async mode for Crossplane/Upjet")
		return
	}

	waitResp, err := wait.PartialUpdateRecordSetWaitHandler(ctx, r.client, projectId, zoneId, recordSetId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating record set", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating record set", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "DNS record set updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *recordSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	zoneId := model.ZoneId.ValueString()
	recordSetId := model.RecordSetId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "zone_id", zoneId)
	ctx = tflog.SetField(ctx, "record_set_id", recordSetId)

	// Delete existing record set
	_, err := r.client.DeleteRecordSet(ctx, projectId, zoneId, recordSetId).Execute()
	if err != nil {
		// If resource is already gone (404 or 410), treat as success for idempotency
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusGone) {
			tflog.Info(ctx, "Record set already deleted")
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting record set", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if !utils.ShouldWait() {
		tflog.Info(ctx, "Skipping wait; async mode for Crossplane/Upjet")
		return
	}

	_, err = wait.DeleteRecordSetWaitHandler(ctx, r.client, projectId, zoneId, recordSetId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting record set", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "DNS record set deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,zone_id,record_set_id
func (r *recordSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing record set",
			fmt.Sprintf("Expected import identifier with format [project_id],[zone_id],[record_set_id], got %q", req.ID),
		)
		return
	}

	var model Model
	model.ProjectId = types.StringValue(idParts[0])
	model.ZoneId = types.StringValue(idParts[1])
	model.RecordSetId = types.StringValue(idParts[2])
	model.Id = utils.BuildInternalTerraformId(idParts[0], idParts[1], idParts[2])

	if err := utils.SetModelFieldsToNull(ctx, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing zone", fmt.Sprintf("Setting model fields to null: %v", err))
		return
	}

	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	tflog.Info(ctx, "DNS record set state imported")
}

func mapFields(ctx context.Context, recordSetResp *dns.RecordSetResponse, model *Model) error {
	if recordSetResp == nil || recordSetResp.Rrset == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	recordSet := recordSetResp.Rrset

	var recordSetId string
	if model.RecordSetId.ValueString() != "" {
		recordSetId = model.RecordSetId.ValueString()
	} else if recordSet.Id != nil {
		recordSetId = *recordSet.Id
	} else {
		return fmt.Errorf("record set id not present")
	}

	if recordSet.Records == nil {
		model.Records = types.ListNull(types.StringType)
	} else {
		respRecords := []string{}

		for _, record := range *recordSet.Records {
			respRecords = append(respRecords, *record.Content)
		}

		modelRecords, err := utils.ListValuetoStringSlice(model.Records)
		if err != nil {
			return err
		}

		reconciledRecords := utils.ReconcileStringSlices(modelRecords, respRecords)

		recordsTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledRecords)
		if diags.HasError() {
			return fmt.Errorf("failed to map records: %w", core.DiagsToError(diags))
		}

		model.Records = recordsTF
	}
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), model.ZoneId.ValueString(), recordSetId,
	)
	model.RecordSetId = types.StringPointerValue(recordSet.Id)
	model.Active = types.BoolPointerValue(recordSet.Active)
	model.Comment = types.StringPointerValue(recordSet.Comment)
	model.Error = types.StringPointerValue(recordSet.Error)
	if model.Name.IsNull() || model.Name.IsUnknown() {
		model.Name = types.StringPointerValue(recordSet.Name)
	}
	model.FQDN = types.StringPointerValue(recordSet.Name)
	model.State = types.StringValue(string(recordSet.GetState()))
	model.TTL = types.Int64PointerValue(recordSet.Ttl)
	model.Type = types.StringValue(string(recordSet.GetType()))
	return nil
}

func toCreatePayload(model *Model) (*dns.CreateRecordSetPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	records := []dns.RecordPayload{}
	for i, record := range model.Records.Elements() {
		recordString, ok := record.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected record at index %d to be of type %T, got %T", i, types.String{}, record)
		}
		records = append(records, dns.RecordPayload{
			Content: conversion.StringValueToPointer(recordString),
		})
	}

	return &dns.CreateRecordSetPayload{
		Comment: conversion.StringValueToPointer(model.Comment),
		Name:    conversion.StringValueToPointer(model.Name),
		Records: &records,
		Ttl:     conversion.Int64ValueToPointer(model.TTL),
		Type:    dns.CreateRecordSetPayloadGetTypeAttributeType(conversion.StringValueToPointer(model.Type)),
	}, nil
}

func toUpdatePayload(model *Model) (*dns.PartialUpdateRecordSetPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	records := []dns.RecordPayload{}
	for i, record := range model.Records.Elements() {
		recordString, ok := record.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected record at index %d to be of type %T, got %T", i, types.String{}, record)
		}
		records = append(records, dns.RecordPayload{
			Content: conversion.StringValueToPointer(recordString),
		})
	}

	return &dns.PartialUpdateRecordSetPayload{
		Comment: conversion.StringValueToPointer(model.Comment),
		Name:    conversion.StringValueToPointer(model.Name),
		Records: &records,
		Ttl:     conversion.Int64ValueToPointer(model.TTL),
	}, nil
}
