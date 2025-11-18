package platform

import (
	"context"
	"fmt"
	"net/http"

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
	_ datasource.DataSource              = &scfPlatformDataSource{}
	_ datasource.DataSourceWithConfigure = &scfPlatformDataSource{}
)

// NewScfPlatformDataSource creates a new instance of the ScfPlatformDataSource.
func NewScfPlatformDataSource() datasource.DataSource {
	return &scfPlatformDataSource{}
}

// scfPlatformDataSource is the datasource implementation.
type scfPlatformDataSource struct {
	client       *scf.APIClient
	providerData core.ProviderData
}

func (s *scfPlatformDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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
	tflog.Info(ctx, "scf client configured for platform")
}

func (s *scfPlatformDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) { // nolint:gocritic // function signature required by Terraform
	response.TypeName = request.ProviderTypeName + "_scf_platform"
}

type Model struct {
	Id          types.String `tfsdk:"id"` // Required by Terraform
	PlatformId  types.String `tfsdk:"platform_id"`
	ProjectId   types.String `tfsdk:"project_id"`
	SystemId    types.String `tfsdk:"system_id"`
	DisplayName types.String `tfsdk:"display_name"`
	Region      types.String `tfsdk:"region"`
	ApiUrl      types.String `tfsdk:"api_url"`
	ConsoleUrl  types.String `tfsdk:"console_url"`
}

// descriptions for the attributes in the Schema
var descriptions = map[string]string{
	"id":           "Terraform's internal resource ID, structured as \"`project_id`,`region`,`platform_id`\".",
	"platform_id":  "The unique id of the platform",
	"project_id":   "The ID of the project associated with the platform",
	"system_id":    "The ID of the platform System",
	"display_name": "The name of the platform",
	"region":       "The region where the platform is located. If not defined, the provider region is used",
	"api_url":      "The CF API Url of the platform",
	"console_url":  "The Stratos URL of the platform",
}

func (s *scfPlatformDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) { // nolint:gocritic // function signature required by Terraform
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"platform_id": schema.StringAttribute{
				Description: descriptions["platform_id"],
				Required:    true,
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
			"system_id": schema.StringAttribute{
				Description: descriptions["system_id"],
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
			},
			"api_url": schema.StringAttribute{
				Description: descriptions["api_url"],
				Computed:    true,
			},
			"console_url": schema.StringAttribute{
				Description: descriptions["console_url"],
				Computed:    true,
			},
		},
		Description: "STACKIT Cloud Foundry Platform datasource schema.",
	}
}

func (s *scfPlatformDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := request.Config.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	// Extract the project ID region and platform id of the model
	projectId := model.ProjectId.ValueString()
	platformId := model.PlatformId.ValueString()
	region := s.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "platform_id", platformId)
	ctx = tflog.SetField(ctx, "region", region)

	// Read the scf platform
	scfPlatformResponse, err := s.client.GetPlatformExecute(ctx, projectId, region, platformId)
	if err != nil {
		utils.LogError(
			ctx,
			&response.Diagnostics,
			err,
			"Reading scf platform",
			fmt.Sprintf("Platform with ID %q does not exist in project %q.", platformId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Platform with ID %q not found or forbidden access", platformId),
			},
		)
		response.State.RemoveResource(ctx)
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(scfPlatformResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading scf platform", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = response.State.Set(ctx, &model)
	response.Diagnostics.Append(diags...)
	tflog.Info(ctx, fmt.Sprintf("read scf Platform %s", platformId))
}

// mapFields maps a SCF Platform response to the model.
func mapFields(response *scf.Platforms, model *Model) error {
	if response == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var projectId string
	if model.ProjectId.ValueString() == "" {
		return fmt.Errorf("project id is not present")
	}
	projectId = model.ProjectId.ValueString()

	var region string
	if response.Region != nil {
		region = *response.Region
	} else if model.Region.ValueString() != "" {
		region = model.Region.ValueString()
	} else {
		return fmt.Errorf("region is not present")
	}

	var platformId string
	if response.Guid != nil {
		platformId = *response.Guid
	} else if model.PlatformId.ValueString() != "" {
		platformId = model.PlatformId.ValueString()
	} else {
		return fmt.Errorf("platform id is not present")
	}

	// Build the ID
	model.Id = utils.BuildInternalTerraformId(projectId, region, platformId)
	model.PlatformId = types.StringValue(platformId)
	model.ProjectId = types.StringValue(projectId)
	model.SystemId = types.StringPointerValue(response.SystemId)
	model.DisplayName = types.StringPointerValue(response.DisplayName)
	model.Region = types.StringValue(region)
	model.ApiUrl = types.StringPointerValue(response.ApiUrl)
	model.ConsoleUrl = types.StringPointerValue(response.ConsoleUrl)
	return nil
}
