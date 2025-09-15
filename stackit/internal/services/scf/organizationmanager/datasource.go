package organizationmanager

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	}
}

func (s *scfOrganizationManagerDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := request.Config.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

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

	err = mapFieldsRead(ScfOrgManager, &model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading scf organization manager", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = response.State.Set(ctx, &model)
	response.Diagnostics.Append(diags...)
	tflog.Info(ctx, fmt.Sprintf("read scf organization manager %s", orgId))
}
