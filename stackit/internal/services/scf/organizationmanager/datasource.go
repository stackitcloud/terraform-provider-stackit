package organizationmanager

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/scf"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	scfUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/scf/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &scfOrganizationManagerDataSource{}
	_ datasource.DataSourceWithConfigure = &scfOrganizationManagerDataSource{}
)

type DataSourceModel struct {
	Id         types.String `tfsdk:"id"` // Required by Terraform
	Region     types.String `tfsdk:"region"`
	PlatformId types.String `tfsdk:"platform_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	OrgId      types.String `tfsdk:"org_id"`
	UserId     types.String `tfsdk:"user_id"`
	UserName   types.String `tfsdk:"username"`
	CreateAt   types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

// NewScfOrganizationManagerDataSource creates a new instance of the scfOrganizationDataSource.
func NewScfOrganizationManagerDataSource() datasource.DataSource {
	return &scfOrganizationManagerDataSource{}
}

// scfOrganizationManagerDataSource is the datasource implementation.
type scfOrganizationManagerDataSource struct {
	client       *scf.APIClient
	providerData core.ProviderData
}

func (s *scfOrganizationManagerDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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
	tflog.Info(ctx, "scf client configured for scfOrganizationManagerDataSource")
}

func (s *scfOrganizationManagerDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) { // nolint:gocritic // function signature required by Terraform
	response.TypeName = request.ProviderTypeName + "_scf_organization_manager"
}

func (s *scfOrganizationManagerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) { // nolint:gocritic // function signature required by Terraform
	response.Schema = schema.Schema{
		Description: "STACKIT Cloud Foundry organization manager datasource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
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
			},
			"org_id": schema.StringAttribute{
				Description: descriptions["org_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
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
			"created_at": schema.StringAttribute{
				Description: descriptions["created_at"],
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: descriptions["updated_at"],
				Computed:    true,
			},
		},
	}
}

func (s *scfOrganizationManagerDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model DataSourceModel
	diags := request.Config.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	// Extract the project ID and instance id of the model
	projectId := model.ProjectId.ValueString()
	orgId := model.OrgId.ValueString()

	region := s.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "org_id", orgId)
	ctx = tflog.SetField(ctx, "region", region)
	// Read the current scf organization manager via orgId
	ScfOrgManager, err := s.client.GetOrgManagerExecute(ctx, projectId, region, orgId)
	if err != nil {
		utils.LogError(
			ctx,
			&response.Diagnostics,
			err,
			"Reading scf organization manager",
			fmt.Sprintf("Organization with ID %q does not exist in project %q.", orgId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Organization with ID %q not found or forbidden access", orgId),
			},
		)
		response.State.RemoveResource(ctx)
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFieldsDataSource(ScfOrgManager, &model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading scf organization manager", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = response.State.Set(ctx, &model)
	response.Diagnostics.Append(diags...)
	tflog.Info(ctx, fmt.Sprintf("read scf organization manager %s", orgId))
}

func mapFieldsDataSource(response *scf.OrgManager, model *DataSourceModel) error {
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
