package exportpolicy

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	sfsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &exportPolicyResource{}
	_ resource.ResourceWithConfigure   = &exportPolicyResource{}
	_ resource.ResourceWithImportState = &exportPolicyResource{}
	_ resource.ResourceWithModifyPlan  = &exportPolicyResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	ExportPolicyId types.String `tfsdk:"policy_id"`
	Name           types.String `tfsdk:"name"`
	Rules          types.List   `tfsdk:"rules"`
	Region         types.String `tfsdk:"region"`
}

type rulesModel struct {
	Description types.String `tfsdk:"description"`
	IpAcl       types.List   `tfsdk:"ip_acl"`
	Order       types.Int64  `tfsdk:"order"`
	ReadOnly    types.Bool   `tfsdk:"read_only"`
	SetUuid     types.Bool   `tfsdk:"set_uuid"`
	SuperUser   types.Bool   `tfsdk:"super_user"`
}

// Types corresponding to rulesModel
var rulesTypes = map[string]attr.Type{
	"description": types.StringType,
	"ip_acl":      types.ListType{ElemType: types.StringType},
	"order":       types.Int64Type,
	"read_only":   types.BoolType,
	"set_uuid":    types.BoolType,
	"super_user":  types.BoolType,
}

func NewExportPolicyResource() resource.Resource {
	return &exportPolicyResource{}
}

type exportPolicyResource struct {
	client       *sfs.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *exportPolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	// If rules were completely removed from the config this is not recognized by terraform
	// since this field is optional and computed therefore this plan modifier is needed.
	utils.CheckListRemoval(ctx, configModel.Rules, planModel.Rules, path.Root("rules"), types.ObjectType{AttrTypes: rulesTypes}, true, resp)
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

// Metadata returns the resource type name.
func (r *exportPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sfs_export_policy"
}

// Configure adds the provider configured client to the resource.
func (r *exportPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !resourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_sfs_export_policy", core.Resource)
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	apiClient := sfsUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "SFS client configured")
}

// Schema defines the schema for the resource.
func (r *exportPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "SFS export policy resource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddBetaDescription(description, core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`policy_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the export policy is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"policy_id": schema.StringAttribute{
				Description: "Export policy ID",
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
				Description: "Name of the export policy.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"rules": schema.ListNestedAttribute{
				Computed: true,
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "Description of the Rule",
						},
						"ip_acl": schema.ListAttribute{
							ElementType: types.StringType,
							Required:    true,
							Description: `IP access control list; IPs must have a subnet mask (e.g. "172.16.0.0/24" for a range of IPs, or "172.16.0.250/32" for a specific IP).`,
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
								listvalidator.ValueStringsAre(validate.CIDR()),
							},
						},
						"order": schema.Int64Attribute{
							Description: "Order of the rule within a Share Export Policy. The order is used so that when a client IP matches multiple rules, the first rule is applied",
							Required:    true,
						},
						"read_only": schema.BoolAttribute{
							Description: "Flag to indicate if client IPs matching this rule can only mount the share in read only mode",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"set_uuid": schema.BoolAttribute{
							Description: "Flag to honor set UUID",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"super_user": schema.BoolAttribute{
							Description: "Flag to indicate if client IPs matching this rule have root access on the Share",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
					},
				},
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
func (r *exportPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { //nolint:gocritic // defined by terraform api
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	var rules = []rulesModel{}
	if !(model.Rules.IsNull() || model.Rules.IsUnknown()) {
		diags = model.Rules.ElementsAs(ctx, &rules, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	payload, err := toCreatePayload(&model, rules)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating export policy", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.CreateShareExportPolicy(ctx, projectId, region).CreateShareExportPolicyPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating export policy", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp == nil || createResp.ShareExportPolicy == nil || createResp.ShareExportPolicy.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating export policy", "response did not contain an ID")
		return
	}
	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
		"policy_id":  *createResp.ShareExportPolicy.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	// get export policy
	getResp, err := r.client.GetShareExportPolicy(ctx, projectId, region, *createResp.ShareExportPolicy.Id).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating export policy", fmt.Sprintf("Calling API to get export policy: %v", err))
		return
	}

	err = mapFields(ctx, getResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating export policy", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS export policy created")
}

// Read refreshes the Terraform state with the latest data.
func (r *exportPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { //nolint:gocritic // defined by terraform api
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	exportPolicyId := model.ExportPolicyId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "policy_id", exportPolicyId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	// get export policy
	exportPolicyResp, err := r.client.GetShareExportPolicy(ctx, projectId, region, exportPolicyId).Execute()
	if err != nil {
		var openapiError *oapierror.GenericOpenAPIError
		if errors.As(err, &openapiError) {
			if openapiError.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading export policy", fmt.Sprintf("Calling API to get export policy: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// map export policy
	err = mapFields(ctx, exportPolicyResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading export policy", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "SFS export policy read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *exportPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { //nolint:gocritic // defined by terraform api
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	exportPolicyId := model.ExportPolicyId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "policy_id", exportPolicyId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	var rules = []rulesModel{}
	if !(model.Rules.IsNull() || model.Rules.IsUnknown()) {
		diags = model.Rules.ElementsAs(ctx, &rules, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	payload, err := toUpdatePayload(&model, rules)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating export policy", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	_, err = r.client.UpdateShareExportPolicy(ctx, projectId, region, exportPolicyId).UpdateShareExportPolicyPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating export policy", fmt.Sprintf("Calling API to update export policy: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// get export policy
	exportPolicyResp, err := r.client.GetShareExportPolicy(ctx, projectId, region, exportPolicyId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating export policy", fmt.Sprintf("Calling API to get export policy: %v", err))
		return
	}

	// map export policy
	err = mapFields(ctx, exportPolicyResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating export policy", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "SFS export policy update")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *exportPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { //nolint:gocritic // defined by terraform api
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	exportPolicyId := model.ExportPolicyId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "policy_id", exportPolicyId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	_, err := r.client.DeleteShareExportPolicy(ctx, projectId, region, exportPolicyId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting export policy", fmt.Sprintf("Calling API: %v", err))
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "SFS export policy delete")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the export policy resource import identifier is: project_id,region,policy_id
func (r *exportPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing export policy",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[policy_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_id"), idParts[2])...)

	tflog.Info(ctx, "SFS export policy state import")
}

// Maps bar fields to the provider's internal model
func mapFields(ctx context.Context, resp *sfs.GetShareExportPolicyResponse, model *Model, region string) error {
	if resp == nil || resp.ShareExportPolicy == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var exportPolicyId string
	if model.ExportPolicyId.ValueString() != "" {
		exportPolicyId = model.ExportPolicyId.ValueString()
	} else if resp.ShareExportPolicy.Id != nil {
		exportPolicyId = *resp.ShareExportPolicy.Id
	} else {
		return fmt.Errorf("export policy id not present")
	}

	// iterate over Rules from response
	if resp.ShareExportPolicy.Rules != nil {
		rulesList := []attr.Value{}
		for _, rule := range *resp.ShareExportPolicy.Rules {
			var ipAcl basetypes.ListValue
			if rule.IpAcl != nil {
				var diags diag.Diagnostics
				ipAcl, diags = types.ListValueFrom(ctx, types.StringType, rule.IpAcl)
				if diags.HasError() {
					return fmt.Errorf("failed to map ip acls: %w", core.DiagsToError(diags))
				}
			} else {
				ipAcl = types.ListNull(types.StringType)
			}

			rulesValues := map[string]attr.Value{
				"description": types.StringPointerValue(rule.GetDescription()),
				"ip_acl":      ipAcl,
				"order":       types.Int64PointerValue(rule.Order),
				"read_only":   types.BoolPointerValue(rule.ReadOnly),
				"set_uuid":    types.BoolPointerValue(rule.SetUuid),
				"super_user":  types.BoolPointerValue(rule.SuperUser),
			}

			ruleModel, diags := types.ObjectValue(rulesTypes, rulesValues)
			if diags.HasError() {
				return fmt.Errorf("converting rule to TF types: %w", core.DiagsToError(diags))
			}

			rulesList = append(rulesList, ruleModel)
		}

		convertedRulesList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: rulesTypes}, rulesList)
		if diags.HasError() {
			return fmt.Errorf("mapping rules list: %w", core.DiagsToError(diags))
		}

		model.Rules = convertedRulesList
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		exportPolicyId,
	)
	model.ExportPolicyId = types.StringValue(exportPolicyId)
	model.Name = types.StringPointerValue(resp.ShareExportPolicy.Name)
	model.Region = types.StringValue(region)

	return nil
}

// Build CreateBarPayload from provider's model
func toCreatePayload(model *Model, rules []rulesModel) (*sfs.CreateShareExportPolicyPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if rules == nil {
		return nil, fmt.Errorf("nil rules")
	}

	// iterate over rules
	var tempRules []sfs.CreateShareExportPolicyRequestRule
	for _, rule := range rules {
		// convert list
		convertedList, err := conversion.StringListToPointer(rule.IpAcl)
		if err != nil {
			return nil, fmt.Errorf("conversion of rule failed")
		}
		tempRule := sfs.CreateShareExportPolicyRequestRule{
			Description: sfs.NewNullableString(conversion.StringValueToPointer(rule.Description)),
			IpAcl:       convertedList,
			Order:       conversion.Int64ValueToPointer(rule.Order),
			ReadOnly:    conversion.BoolValueToPointer(rule.ReadOnly),
			SetUuid:     conversion.BoolValueToPointer(rule.SetUuid),
			SuperUser:   conversion.BoolValueToPointer(rule.SuperUser),
		}
		tempRules = append(tempRules, tempRule)
	}

	// name and rules
	result := &sfs.CreateShareExportPolicyPayload{
		Name: model.Name.ValueStringPointer(),
	}

	// Rules should only be set if tempRules has value. Otherwise, the payload would contain `{ "rules": null }` what should be prevented
	if tempRules != nil {
		result.Rules = &tempRules
	}

	return result, nil
}

func toUpdatePayload(model *Model, rules []rulesModel) (*sfs.UpdateShareExportPolicyPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if rules == nil {
		return nil, fmt.Errorf("nil rules")
	}

	// iterate over rules
	tempRules := make([]sfs.UpdateShareExportPolicyBodyRule, len(rules))
	for i, rule := range rules {
		// convert list
		convertedList, err := conversion.StringListToPointer(rule.IpAcl)
		if err != nil {
			return nil, fmt.Errorf("conversion of rule failed")
		}
		tempRule := sfs.UpdateShareExportPolicyBodyRule{
			Description: sfs.NewNullableString(conversion.StringValueToPointer(rule.Description)),
			IpAcl:       convertedList,
			Order:       conversion.Int64ValueToPointer(rule.Order),
			ReadOnly:    conversion.BoolValueToPointer(rule.ReadOnly),
			SetUuid:     conversion.BoolValueToPointer(rule.SetUuid),
			SuperUser:   conversion.BoolValueToPointer(rule.SuperUser),
		}
		tempRules[i] = tempRule
	}

	// only rules
	result := &sfs.UpdateShareExportPolicyPayload{
		// Rules should *+never** result in a payload where they are defined as null, e.g. `{ "rules": null }`. Instead,
		// they should either be set to an array (with values or empty) or they shouldn't be present in the payload.
		Rules: &tempRules,
	}
	return result, nil
}
