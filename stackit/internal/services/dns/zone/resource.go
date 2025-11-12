package dns

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	dnsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stackitcloud/stackit-sdk-go/services/dns/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &zoneResource{}
	_ resource.ResourceWithConfigure   = &zoneResource{}
	_ resource.ResourceWithImportState = &zoneResource{}
)

type Model struct {
	Id                types.String `tfsdk:"id"` // needed by TF
	ZoneId            types.String `tfsdk:"zone_id"`
	ProjectId         types.String `tfsdk:"project_id"`
	Name              types.String `tfsdk:"name"`
	DnsName           types.String `tfsdk:"dns_name"`
	Description       types.String `tfsdk:"description"`
	Acl               types.String `tfsdk:"acl"`
	Active            types.Bool   `tfsdk:"active"`
	ContactEmail      types.String `tfsdk:"contact_email"`
	DefaultTTL        types.Int64  `tfsdk:"default_ttl"`
	ExpireTime        types.Int64  `tfsdk:"expire_time"`
	IsReverseZone     types.Bool   `tfsdk:"is_reverse_zone"`
	NegativeCache     types.Int64  `tfsdk:"negative_cache"`
	PrimaryNameServer types.String `tfsdk:"primary_name_server"`
	Primaries         types.List   `tfsdk:"primaries"`
	RecordCount       types.Int64  `tfsdk:"record_count"`
	RefreshTime       types.Int64  `tfsdk:"refresh_time"`
	RetryTime         types.Int64  `tfsdk:"retry_time"`
	SerialNumber      types.Int64  `tfsdk:"serial_number"`
	Type              types.String `tfsdk:"type"`
	Visibility        types.String `tfsdk:"visibility"`
	State             types.String `tfsdk:"state"`
}

// NewZoneResource is a helper function to simplify the provider implementation.
func NewZoneResource() resource.Resource {
	return &zoneResource{}
}

// zoneResource is the resource implementation.
type zoneResource struct {
	client *dns.APIClient
}

// Metadata returns the resource type name.
func (r *zoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

// Configure adds the provider configured client to the resource.
func (r *zoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := dnsUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "DNS zone client configured")
}

// Schema defines the schema for the resource.
func (r *zoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	primaryOptions := []string{"primary", "secondary"}

	resp.Schema = schema.Schema{
		Description: "DNS Zone resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`zone_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the dns zone is associated.",
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
				Description: "The zone ID.",
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
				Description: "The user given name of the zone.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"dns_name": schema.StringAttribute{
				Description: "The zone name. E.g. `example.com`",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(253),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the zone.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
			},
			"acl": schema.StringAttribute{
				Description: "The access control list. E.g. `0.0.0.0/0,::/0`",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(2000),
				},
			},
			"active": schema.BoolAttribute{
				Description: "",
				Optional:    true,
				Computed:    true,
			},
			"contact_email": schema.StringAttribute{
				Description: "A contact e-mail for the zone.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"default_ttl": schema.Int64Attribute{
				Description: "Default time to live. E.g. 3600.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(60, 99999999),
				},
			},
			"expire_time": schema.Int64Attribute{
				Description: "Expire time. E.g. 1209600.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(60, 99999999),
				},
			},
			"is_reverse_zone": schema.BoolAttribute{
				Description: "Specifies, if the zone is a reverse zone or not. Defaults to `false`",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"negative_cache": schema.Int64Attribute{
				Description: "Negative caching. E.g. 60",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(60, 99999999),
				},
			},
			"primaries": schema.ListAttribute{
				Description: `Primary name server for secondary zone. E.g. ["1.2.3.4"]`,
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
					listplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(10),
				},
			},
			"refresh_time": schema.Int64Attribute{
				Description: "Refresh time. E.g. 3600",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(60, 99999999),
				},
			},
			"retry_time": schema.Int64Attribute{
				Description: "Retry time. E.g. 600",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(60, 99999999),
				},
			},
			"type": schema.StringAttribute{
				Description: "Zone type. Defaults to `primary`. " + utils.FormatPossibleValues(primaryOptions...),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("primary"),
				Validators: []validator.String{
					stringvalidator.OneOf(primaryOptions...),
				},
			},
			"primary_name_server": schema.StringAttribute{
				Description: "Primary name server. FQDN.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(253),
				},
			},
			"serial_number": schema.Int64Attribute{
				Description: "Serial number. E.g. `2022111400`.",
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
					int64validator.AtMost(math.MaxInt32 - 1),
				},
			},
			"visibility": schema.StringAttribute{
				Description: "Visibility of the zone. E.g. `public`.",
				Computed:    true,
			},
			"record_count": schema.Int64Attribute{
				Description: "Record count how many records are in the zone.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "Zone state. E.g. `CREATE_SUCCEEDED`.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *zoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating zone", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new zone
	createResp, err := r.client.CreateZone(ctx, projectId).CreateZonePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating zone", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Save minimal state immediately after API call succeeds to ensure idempotency
	zoneId := *createResp.Zone.Id
	model.ZoneId = types.StringValue(zoneId)
	model.Id = utils.BuildInternalTerraformId(projectId, zoneId)

	// Set all unknown/null fields to null before saving state
	if err := utils.SetModelFieldsToNull(ctx, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating zone", fmt.Sprintf("Setting model fields to null: %v", err))
		return
	}

	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	if !utils.ShouldWait() {
		tflog.Info(ctx, "Skipping wait; async mode for Crossplane/Upjet")
		return
	}

	waitResp, err := wait.CreateZoneWaitHandler(ctx, r.client, projectId, zoneId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating zone", fmt.Sprintf("Zone creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating zone", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "DNS zone created")
}

// Read refreshes the Terraform state with the latest data.
func (r *zoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	zoneId := model.ZoneId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "zone_id", zoneId)

	zoneResp, err := r.client.GetZone(ctx, projectId, zoneId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusGone) {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading zone", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if zoneResp != nil && zoneResp.Zone.State != nil &&
		*zoneResp.Zone.State == dns.ZONESTATE_DELETE_SUCCEEDED {
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, zoneResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading zone", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "DNS zone read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *zoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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
	payload, err := toUpdatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating zone", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing zone
	_, err = r.client.PartialUpdateZone(ctx, projectId, zoneId).PartialUpdateZonePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating zone", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if !utils.ShouldWait() {
		tflog.Info(ctx, "Skipping wait; async mode for Crossplane/Upjet")
		return
	}

	waitResp, err := wait.PartialUpdateZoneWaitHandler(ctx, r.client, projectId, zoneId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating zone", fmt.Sprintf("Zone update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating zone", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "DNS zone updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *zoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	zoneId := model.ZoneId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "zone_id", zoneId)

	// Delete existing zone
	_, err := r.client.DeleteZone(ctx, projectId, zoneId).Execute()
	if err != nil {
		// If resource is already gone (404 or 410), treat as success for idempotency
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok &&
			(oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusGone) {
			tflog.Info(ctx, "DNS zone already deleted")
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting zone", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if !utils.ShouldWait() {
		tflog.Info(ctx, "Skipping wait; async mode for Crossplane/Upjet")
		return
	}

	_, err = wait.DeleteZoneWaitHandler(ctx, r.client, projectId, zoneId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting zone", fmt.Sprintf("Zone deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "DNS zone deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,zone_id
func (r *zoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing zone",
			fmt.Sprintf("Expected import identifier with format: [project_id],[zone_id]  Got: %q", req.ID),
		)
		return
	}

	var model Model
	model.ProjectId = types.StringValue(idParts[0])
	model.ZoneId = types.StringValue(idParts[1])
	model.Id = utils.BuildInternalTerraformId(idParts[0], idParts[1])

	if err := utils.SetModelFieldsToNull(ctx, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing zone", fmt.Sprintf("Setting model fields to null: %v", err))
		return
	}

	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	tflog.Info(ctx, "DNS zone state imported")
}

func mapFields(ctx context.Context, zoneResp *dns.ZoneResponse, model *Model) error {
	if zoneResp == nil || zoneResp.Zone == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	z := zoneResp.Zone

	var rc *int64
	if z.RecordCount != nil {
		recordCount64 := int64(*z.RecordCount)
		rc = &recordCount64
	} else {
		rc = nil
	}

	var zoneId string
	if model.ZoneId.ValueString() != "" {
		zoneId = model.ZoneId.ValueString()
	} else if z.Id != nil {
		zoneId = *z.Id
	} else {
		return fmt.Errorf("zone id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), zoneId)

	if z.Primaries == nil {
		model.Primaries = types.ListNull(types.StringType)
	} else {
		respPrimaries := *z.Primaries
		modelPrimaries, err := utils.ListValuetoStringSlice(model.Primaries)
		if err != nil {
			return err
		}

		reconciledPrimaries := utils.ReconcileStringSlices(modelPrimaries, respPrimaries)

		primariesTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledPrimaries)
		if diags.HasError() {
			return fmt.Errorf("failed to map zone primaries: %w", core.DiagsToError(diags))
		}

		model.Primaries = primariesTF
	}
	model.ZoneId = types.StringValue(zoneId)
	model.Description = types.StringPointerValue(z.Description)
	model.Acl = types.StringPointerValue(z.Acl)
	model.Active = types.BoolPointerValue(z.Active)
	model.ContactEmail = types.StringPointerValue(z.ContactEmail)
	model.DefaultTTL = types.Int64PointerValue(z.DefaultTTL)
	model.DnsName = types.StringPointerValue(z.DnsName)
	model.ExpireTime = types.Int64PointerValue(z.ExpireTime)
	model.IsReverseZone = types.BoolPointerValue(z.IsReverseZone)
	model.Name = types.StringPointerValue(z.Name)
	model.NegativeCache = types.Int64PointerValue(z.NegativeCache)
	model.PrimaryNameServer = types.StringPointerValue(z.PrimaryNameServer)
	model.RecordCount = types.Int64PointerValue(rc)
	model.RefreshTime = types.Int64PointerValue(z.RefreshTime)
	model.RetryTime = types.Int64PointerValue(z.RetryTime)
	model.SerialNumber = types.Int64PointerValue(z.SerialNumber)
	model.State = types.StringValue(string(z.GetState()))
	model.Type = types.StringValue(string(z.GetType()))
	model.Visibility = types.StringValue(string(z.GetVisibility()))
	return nil
}

func toCreatePayload(model *Model) (*dns.CreateZonePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelPrimaries := []string{}
	for _, primary := range model.Primaries.Elements() {
		primaryString, ok := primary.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelPrimaries = append(modelPrimaries, primaryString.ValueString())
	}
	return &dns.CreateZonePayload{
		Name:          conversion.StringValueToPointer(model.Name),
		DnsName:       conversion.StringValueToPointer(model.DnsName),
		ContactEmail:  conversion.StringValueToPointer(model.ContactEmail),
		Description:   conversion.StringValueToPointer(model.Description),
		Acl:           conversion.StringValueToPointer(model.Acl),
		Type:          dns.CreateZonePayloadGetTypeAttributeType(conversion.StringValueToPointer(model.Type)),
		DefaultTTL:    conversion.Int64ValueToPointer(model.DefaultTTL),
		ExpireTime:    conversion.Int64ValueToPointer(model.ExpireTime),
		RefreshTime:   conversion.Int64ValueToPointer(model.RefreshTime),
		RetryTime:     conversion.Int64ValueToPointer(model.RetryTime),
		NegativeCache: conversion.Int64ValueToPointer(model.NegativeCache),
		IsReverseZone: conversion.BoolValueToPointer(model.IsReverseZone),
		Primaries:     &modelPrimaries,
	}, nil
}

func toUpdatePayload(model *Model) (*dns.PartialUpdateZonePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &dns.PartialUpdateZonePayload{
		Name:          conversion.StringValueToPointer(model.Name),
		ContactEmail:  conversion.StringValueToPointer(model.ContactEmail),
		Description:   conversion.StringValueToPointer(model.Description),
		Acl:           conversion.StringValueToPointer(model.Acl),
		DefaultTTL:    conversion.Int64ValueToPointer(model.DefaultTTL),
		ExpireTime:    conversion.Int64ValueToPointer(model.ExpireTime),
		RefreshTime:   conversion.Int64ValueToPointer(model.RefreshTime),
		RetryTime:     conversion.Int64ValueToPointer(model.RetryTime),
		NegativeCache: conversion.Int64ValueToPointer(model.NegativeCache),
		Primaries:     nil, // API returns error if this field is set, even if nothing changes
	}, nil
}
