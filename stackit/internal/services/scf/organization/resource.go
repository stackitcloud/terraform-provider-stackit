package organization

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/scf"
	"github.com/stackitcloud/stackit-sdk-go/services/scf/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	scfUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/scf/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &scfOrganizationResource{}
	_ resource.ResourceWithConfigure   = &scfOrganizationResource{}
	_ resource.ResourceWithImportState = &scfOrganizationResource{}
	_ resource.ResourceWithModifyPlan  = &scfOrganizationResource{}
)

type Model struct {
	Id         types.String `tfsdk:"id"` // Required by Terraform
	CreateAt   types.String `tfsdk:"created_at"`
	Name       types.String `tfsdk:"name"`
	PlatformId types.String `tfsdk:"platform_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	QuotaId    types.String `tfsdk:"quota_id"`
	OrgId      types.String `tfsdk:"org_id"`
	Region     types.String `tfsdk:"region"`
	Status     types.String `tfsdk:"status"`
	Suspended  types.Bool   `tfsdk:"suspended"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

// NewScfOrganizationResource is a helper function to create a new scf organization resource.
func NewScfOrganizationResource() resource.Resource {
	return &scfOrganizationResource{}
}

// scfOrganizationResource implements the resource interface for scf organization.
type scfOrganizationResource struct {
	client       *scf.APIClient
	providerData core.ProviderData
}

// descriptions for the attributes in the Schema
var descriptions = map[string]string{
	"id":          "Terraform's internal resource ID, structured as \"`project_id`,`region`,`org_id`\".",
	"created_at":  "The time when the organization was created",
	"name":        "The name of the organization",
	"platform_id": "The ID of the platform associated with the organization",
	"project_id":  "The ID of the project associated with the organization",
	"quota_id":    "The ID of the quota associated with the organization",
	"region":      "The resource region. If not defined, the provider region is used",
	"status":      "The status of the organization (e.g., deleting, delete_failed)",
	"suspended":   "A boolean indicating whether the organization is suspended",
	"org_id":      "The ID of the Cloud Foundry Organization",
	"updated_at":  "The time when the organization was last updated",
}

func (s *scfOrganizationResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	var ok bool
	s.providerData, ok = conversion.ParseProviderData(ctx, request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	apiClient := scfUtils.ConfigureClient(ctx, &s.providerData, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	s.client = apiClient
	tflog.Info(ctx, "scf client configured")
}

func (s *scfOrganizationResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_scf_organization"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *scfOrganizationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (s *scfOrganizationResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "STACKIT Cloud Foundry organization resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: descriptions["created_at"],
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"platform_id": schema.StringAttribute{
				Description: descriptions["platform_id"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: descriptions["org_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"quota_id": schema.StringAttribute{
				Description: descriptions["quota_id"],
				Optional:    true,
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
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: descriptions["status"],
				Computed:    true,
			},
			"suspended": schema.BoolAttribute{
				Description: descriptions["suspended"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: descriptions["updated_at"],
				Computed:    true,
			},
		},
	}
}

func (s *scfOrganizationResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the planned values for the resource.
	var model Model
	diags := request.Plan.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	// Set logging context with the project ID and instance ID.
	region := model.Region.ValueString()
	projectId := model.ProjectId.ValueString()
	orgName := model.Name.ValueString()
	quotaId := model.QuotaId.ValueString()
	suspended := model.Suspended.ValueBool()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "org_name", orgName)
	ctx = tflog.SetField(ctx, "region", region)
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating scf organization", fmt.Sprintf("Creating API payload: %v\n", err))
		return
	}

	// Create the new scf organization via the API client.
	scfOrgCreateResponse, err := s.client.CreateOrganization(ctx, projectId, region).
		CreateOrganizationPayload(payload).
		Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating scf organization", fmt.Sprintf("Calling API to create org: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	orgId := *scfOrgCreateResponse.Guid

	// Apply the org quota if provided
	if quotaId != "" {
		applyOrgQuota, err := s.client.ApplyOrganizationQuota(ctx, projectId, region, orgId).ApplyOrganizationQuotaPayload(
			scf.ApplyOrganizationQuotaPayload{
				QuotaId: &quotaId,
			}).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &response.Diagnostics, "Error creating scf organization", fmt.Sprintf("Calling API to apply quota: %v", err))
			return
		}
		model.QuotaId = types.StringPointerValue(applyOrgQuota.QuotaId)
	}

	if suspended {
		_, err := s.client.UpdateOrganization(ctx, projectId, region, orgId).UpdateOrganizationPayload(

			scf.UpdateOrganizationPayload{
				Suspended: &suspended,
			}).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &response.Diagnostics, "Error creating scf organization", fmt.Sprintf("Calling API to update suspended: %v", err))
			return
		}
	}

	// Load the newly created scf organization
	scfOrgResponse, err := s.client.GetOrganization(ctx, projectId, region, orgId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating scf organization", fmt.Sprintf("Calling API to load created org: %v", err))
		return
	}

	err = mapFields(scfOrgResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating scf organization", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set the state with fully populated data.
	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Scf organization created")
}

// Read refreshes the Terraform state with the latest scf organization data.
func (s *scfOrganizationResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := request.State.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	// Extract the project ID and instance id of the model
	projectId := model.ProjectId.ValueString()
	orgId := model.OrgId.ValueString()
	// Extract the region
	region := s.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "org_id", orgId)
	ctx = tflog.SetField(ctx, "region", region)
	// Read the current scf organization via guid
	scfOrgResponse, err := s.client.GetOrganization(ctx, projectId, region, orgId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading scf organization", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(scfOrgResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading scf organization", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = response.State.Set(ctx, &model)
	response.Diagnostics.Append(diags...)
	tflog.Info(ctx, fmt.Sprintf("read scf organization %s", orgId))
}

// Update attempts to update the resource.
func (s *scfOrganizationResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := request.Plan.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	region := model.Region.ValueString()
	projectId := model.ProjectId.ValueString()
	orgId := model.OrgId.ValueString()
	name := model.Name.ValueString()
	quotaId := model.QuotaId.ValueString()
	suspended := model.Suspended.ValueBool()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "org_id", orgId)
	ctx = tflog.SetField(ctx, "region", region)

	org, err := s.client.GetOrganization(ctx, projectId, region, orgId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error retrieving organization state", fmt.Sprintf("Getting organization state: %v", err))
		return
	}

	// handle a change of the organization name or the suspended flag
	if name != org.GetName() || suspended != org.GetSuspended() {
		updatedOrg, err := s.client.UpdateOrganization(ctx, projectId, region, orgId).UpdateOrganizationPayload(
			scf.UpdateOrganizationPayload{
				Name:      &name,
				Suspended: &suspended,
			}).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &response.Diagnostics, "Error updating organization", fmt.Sprintf("Processing API payload: %v", err))
			return
		}
		org = updatedOrg

		ctx = core.LogResponse(ctx)

	}

	// handle a quota change of the org
	if quotaId != org.GetQuotaId() {
		applyOrgQuota, err := s.client.ApplyOrganizationQuota(ctx, projectId, region, orgId).ApplyOrganizationQuotaPayload(
			scf.ApplyOrganizationQuotaPayload{
				QuotaId: &quotaId,
			}).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &response.Diagnostics, "Error applying organization quota", fmt.Sprintf("Processing API payload: %v", err))
			return
		}
		org.QuotaId = applyOrgQuota.QuotaId
	}

	err = mapFields(org, &model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error updating organization", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "organization updated")
}

// Delete deletes the git instance and removes it from the Terraform state on success.
func (s *scfOrganizationResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve current state of the resource.
	var model Model
	diags := request.State.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	orgId := model.OrgId.ValueString()

	// Extract the region
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "org_id", orgId)
	ctx = tflog.SetField(ctx, "region", region)

	// Call API to delete the existing scf organization.
	_, err := s.client.DeleteOrganization(ctx, projectId, region, orgId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error deleting scf organization", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteOrganizationWaitHandler(ctx, s.client, projectId, model.Region.ValueString(), orgId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error waiting for scf org deletion", fmt.Sprintf("SCFOrganization deleting waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Scf organization deleted")
}

func (s *scfOrganizationResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	// Split the import identifier to extract project ID and email.
	idParts := strings.Split(request.ID, core.Separator)

	// Ensure the import identifier format is correct.
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &response.Diagnostics,
			"Error importing scf organization",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[org_id]  Got: %q", request.ID),
		)
		return
	}

	projectId := idParts[0]
	region := idParts[1]
	orgId := idParts[2]
	// Set the project id and organization id in the state
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("region"), region)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("org_id"), orgId)...)
	tflog.Info(ctx, "Scf organization state imported")
}

// mapFields maps a SCF Organization response to the model.
func mapFields(response *scf.Organization, model *Model) error {
	if response == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var orgId string
	if response.Guid != nil {
		orgId = *response.Guid
	} else if model.OrgId.ValueString() != "" {
		orgId = model.OrgId.ValueString()
	} else {
		return fmt.Errorf("org id is not present")
	}

	var projectId string
	if response.ProjectId != nil {
		projectId = *response.ProjectId
	} else if model.ProjectId.ValueString() != "" {
		projectId = model.ProjectId.ValueString()
	} else {
		return fmt.Errorf("project id is not present")
	}

	var region string
	if response.Region != nil {
		region = *response.Region
	} else if model.Region.ValueString() != "" {
		region = model.Region.ValueString()
	} else {
		return fmt.Errorf("region is not present")
	}

	// Build the ID by combining the project ID and organization id and assign the model's fields.
	model.Id = utils.BuildInternalTerraformId(projectId, region, orgId)
	model.ProjectId = types.StringValue(projectId)
	model.Region = types.StringValue(region)
	model.PlatformId = types.StringPointerValue(response.PlatformId)
	model.OrgId = types.StringValue(orgId)
	model.Name = types.StringPointerValue(response.Name)
	model.Status = types.StringPointerValue(response.Status)
	model.Suspended = types.BoolPointerValue(response.Suspended)
	model.QuotaId = types.StringPointerValue(response.QuotaId)
	model.CreateAt = types.StringValue(response.CreatedAt.String())
	model.UpdatedAt = types.StringValue(response.UpdatedAt.String())
	return nil
}

// toCreatePayload creates the payload to create a scf organization instance
func toCreatePayload(model *Model) (scf.CreateOrganizationPayload, error) {
	if model == nil {
		return scf.CreateOrganizationPayload{}, fmt.Errorf("nil model")
	}

	payload := scf.CreateOrganizationPayload{
		Name: model.Name.ValueStringPointer(),
	}
	if !model.PlatformId.IsNull() && !model.PlatformId.IsUnknown() {
		payload.PlatformId = model.PlatformId.ValueStringPointer()
	}
	return payload, nil
}
