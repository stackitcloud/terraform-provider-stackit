package organizationmanager

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/scf"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	scfUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/scf/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &scfOrganizationManagerResource{}
	_ resource.ResourceWithConfigure   = &scfOrganizationManagerResource{}
	_ resource.ResourceWithImportState = &scfOrganizationManagerResource{}
	_ resource.ResourceWithModifyPlan  = &scfOrganizationManagerResource{}
)

type Model struct {
	Id         types.String `tfsdk:"id"` // Required by Terraform
	Region     types.String `tfsdk:"region"`
	PlatformId types.String `tfsdk:"platform_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	OrgId      types.String `tfsdk:"org_id"`
	UserId     types.String `tfsdk:"user_id"`
	UserName   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	CreateAt   types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

// NewScfOrganizationManagerResource is a helper function to create a new scf organization manager resource.
func NewScfOrganizationManagerResource() resource.Resource {
	return &scfOrganizationManagerResource{}
}

// scfOrganizationManagerResource implements the resource interface for scf organization manager.
type scfOrganizationManagerResource struct {
	client       *scf.APIClient
	providerData core.ProviderData
}

// descriptions for the attributes in the Schema
var descriptions = map[string]string{
	"id":          "Terraform's internal resource ID, structured as \"`project_id`,`region`,`org_id`,`user_id`\".",
	"region":      "The region where the organization of the organization manager is located. If not defined, the provider region is used",
	"platform_id": "The ID of the platform associated with the organization of the organization manager",
	"project_id":  "The ID of the project associated with the organization of the organization manager",
	"org_id":      "The ID of the Cloud Foundry Organization",
	"user_id":     "The ID of the organization manager user",
	"username":    "An auto-generated organization manager user name",
	"password":    "An auto-generated password",
	"created_at":  "The time when the organization manager was created",
	"updated_at":  "The time when the organization manager was last updated",
}

func (s *scfOrganizationManagerResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) { // nolint:gocritic // function signature required by Terraform
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

func (s *scfOrganizationManagerResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) { // nolint:gocritic // function signature required by Terraform
	response.TypeName = request.ProviderTypeName + "_scf_organization_manager"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *scfOrganizationManagerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (s *scfOrganizationManagerResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) { // nolint:gocritic // function signature required by Terraform
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"platform_id": schema.StringAttribute{
				Description: descriptions["platform_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
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
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: descriptions["org_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: descriptions["user_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"username": schema.StringAttribute{
				Description: descriptions["username"],
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"password": schema.StringAttribute{
				Description: descriptions["password"],
				Computed:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"created_at": schema.StringAttribute{
				Description: descriptions["created_at"],
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: descriptions["updated_at"],
				Computed:    true,
			},
		},
		Description: "STACKIT Cloud Foundry organization manager resource schema.",
	}
}

func (s *scfOrganizationManagerResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the planned values for the resource.
	var model Model
	diags := request.Plan.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	// Set logging context with the project ID and username.
	projectId := model.ProjectId.ValueString()
	orgId := model.OrgId.ValueString()
	userName := model.UserName.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "username", userName)
	ctx = tflog.SetField(ctx, "region", region)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Create the new scf organization manager via the API client.
	scfOrgManagerCreateResponse, err := s.client.CreateOrgManagerExecute(ctx, projectId, region, orgId)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating scf organization manager", fmt.Sprintf("Calling API to create org manager: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFieldsCreate(scfOrgManagerCreateResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error creating scf organization manager", fmt.Sprintf("Mapping fields: %v", err))
		return
	}

	// Set the state with fully populated data.
	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Scf organization manager created")
}

func (s *scfOrganizationManagerResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := request.State.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	// Extract the project ID, region and org id of the model
	projectId := model.ProjectId.ValueString()
	orgId := model.OrgId.ValueString()
	region := s.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "org_id", orgId)
	ctx = tflog.SetField(ctx, "region", region)

	// Read the current scf organization manager via orgId
	scfOrgManager, err := s.client.GetOrgManagerExecute(ctx, projectId, region, orgId)
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			core.LogAndAddWarning(ctx, &response.Diagnostics, "SCF Organization manager not found", "SCF Organization manager not found, remove from state")
			response.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading scf organization manager", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFieldsRead(scfOrgManager, &model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading scf organization manager", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = response.State.Set(ctx, &model)
	response.Diagnostics.Append(diags...)
	tflog.Info(ctx, fmt.Sprintf("read scf organization manager %s", orgId))
}

func (s *scfOrganizationManagerResource) Update(ctx context.Context, _ resource.UpdateRequest, response *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// organization manager cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &response.Diagnostics, "Error updating organization manager", "Organization Manager can't be updated")
}

func (s *scfOrganizationManagerResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "org_id", orgId)
	ctx = tflog.SetField(ctx, "region", region)

	// Call API to delete the existing scf organization manager.
	_, err := s.client.DeleteOrgManagerExecute(ctx, projectId, region, orgId)
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusGone {
			tflog.Info(ctx, "Scf organization manager was already deleted")
			return
		}
		core.LogAndAddError(ctx, &response.Diagnostics, "Error deleting scf organization manager", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "Scf organization manager deleted")
}

func (s *scfOrganizationManagerResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) { // nolint:gocritic // function signature required by Terraform
	// Split the import identifier to extract project ID, region org ID and user ID.
	idParts := strings.Split(request.ID, core.Separator)

	// Ensure the import identifier format is correct.
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &response.Diagnostics,
			"Error importing scf organization manager",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[org_id],[user_id]  Got: %q", request.ID),
		)
		return
	}

	projectId := idParts[0]
	region := idParts[1]
	orgId := idParts[2]
	userId := idParts[3]
	// Set the project id, region organization id and user id in the state
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("region"), region)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("org_id"), orgId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("user_id"), userId)...)
	tflog.Info(ctx, "Scf organization manager state imported")
}

func mapFieldsCreate(response *scf.OrgManagerResponse, model *Model) error {
	if response == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
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

	var orgId string
	if response.OrgId != nil {
		orgId = *response.OrgId
	} else if model.OrgId.ValueString() != "" {
		orgId = model.OrgId.ValueString()
	} else {
		return fmt.Errorf("org id is not present")
	}

	var userId string
	if response.Guid != nil {
		userId = *response.Guid
	} else if model.UserId.ValueString() != "" {
		userId = model.UserId.ValueString()
	} else {
		return fmt.Errorf("user id is not present")
	}

	model.Id = utils.BuildInternalTerraformId(projectId, region, orgId, userId)
	model.Region = types.StringValue(region)
	model.PlatformId = types.StringPointerValue(response.PlatformId)
	model.ProjectId = types.StringValue(projectId)
	model.OrgId = types.StringValue(orgId)
	model.UserId = types.StringValue(userId)
	model.UserName = types.StringPointerValue(response.Username)
	model.Password = types.StringPointerValue(response.Password)
	model.CreateAt = types.StringValue(response.CreatedAt.String())
	model.UpdatedAt = types.StringValue(response.UpdatedAt.String())
	return nil
}

func mapFieldsRead(response *scf.OrgManager, model *Model) error {
	if response == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
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

	var orgId string
	if response.OrgId != nil {
		orgId = *response.OrgId
	} else if model.OrgId.ValueString() != "" {
		orgId = model.OrgId.ValueString()
	} else {
		return fmt.Errorf("org id is not present")
	}

	var userId string
	if response.Guid != nil {
		userId = *response.Guid
		if model.UserId.ValueString() != "" && userId != model.UserId.ValueString() {
			return fmt.Errorf("user id mismatch in response and model")
		}
	} else if model.UserId.ValueString() != "" {
		userId = model.UserId.ValueString()
	} else {
		return fmt.Errorf("user id is not present")
	}

	model.Id = utils.BuildInternalTerraformId(projectId, region, orgId, userId)
	model.Region = types.StringValue(region)
	model.PlatformId = types.StringPointerValue(response.PlatformId)
	model.ProjectId = types.StringValue(projectId)
	model.OrgId = types.StringValue(orgId)
	model.UserId = types.StringValue(userId)
	model.UserName = types.StringPointerValue(response.Username)
	model.CreateAt = types.StringValue(response.CreatedAt.String())
	model.UpdatedAt = types.StringValue(response.UpdatedAt.String())
	return nil
}
