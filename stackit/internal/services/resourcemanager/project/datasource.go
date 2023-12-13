package project

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &projectDataSource{}
)

type ModelData struct {
	Id                types.String `tfsdk:"id"` // needed by TF
	ProjectId         types.String `tfsdk:"project_id"`
	ContainerId       types.String `tfsdk:"container_id"`
	ContainerParentId types.String `tfsdk:"parent_container_id"`
	Name              types.String `tfsdk:"name"`
	Labels            types.Map    `tfsdk:"labels"`
}

// NewProjectDataSource is a helper function to simplify the provider implementation.
func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

// projectDataSource is the data source implementation.
type projectDataSource struct {
	client *resourcemanager.APIClient
}

// Metadata returns the data source type name.
func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resourcemanager_project"
}

func (d *projectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var apiClient *resourcemanager.APIClient
	var err error

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if providerData.ResourceManagerCustomEndpoint != "" {
		apiClient, err = resourcemanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithServiceAccountEmail(providerData.ServiceAccountEmail),
			config.WithEndpoint(providerData.ResourceManagerCustomEndpoint),
		)
	} else {
		apiClient, err = resourcemanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithServiceAccountEmail(providerData.ServiceAccountEmail),
		)
	}
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	d.client = apiClient
	tflog.Info(ctx, "Resource Manager project client configured")
}

// Schema defines the schema for the data source.
func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                "Resource Manager project data source schema. To identify the project, you need to provider either project_id or container_id. If you provide both, project_id will be used.",
		"id":                  "Terraform's internal data source. ID. It is structured as \"`container_id`\".",
		"project_id":          "Project UUID identifier. This is the ID that can be used in most of the other resources to identify the project.",
		"container_id":        "Project container ID. Globally unique, user-friendly identifier.",
		"parent_container_id": "Parent resource identifier. Both container ID (user-friendly) and UUID are supported",
		"name":                "Project name.",
		"labels":              `Labels are key-value string pairs which can be attached to a resource container. A label key must match the regex [A-ZÄÜÖa-zäüöß0-9_-]{1,64}. A label value must match the regex ^$|[A-ZÄÜÖa-zäüöß0-9_-]{1,64}`,
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Optional:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"container_id": schema.StringAttribute{
				Description: descriptions["container_id"],
				Optional:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"parent_container_id": schema.StringAttribute{
				Description: descriptions["parent_container_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				ElementType: types.StringType,
				Computed:    true,
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`[A-ZÄÜÖa-zäüöß0-9_-]{1,64}`),
							"must match expression"),
					),
					mapvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`[A-ZÄÜÖa-zäüöß0-9_-]{1,64}`),
							"must match expression"),
					),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var state ModelData
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := state.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	containerId := state.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "container_id", containerId)

	if containerId == "" && projectId == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", "Either container_id or project_id must be set")
		return
	}

	// set project identifier. If projectId is provided, it takes precedence over containerId
	var identifier = containerId
	if projectId != "" {
		identifier = projectId
	}

	projectResp, err := d.client.GetProject(ctx, identifier).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapDataFields(ctx, projectResp, &state)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Resource Manager project read")
}

func mapDataFields(ctx context.Context, projectResp *resourcemanager.ProjectResponseWithParents, model *ModelData) (err error) {
	if projectResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var projectId string
	if model.ProjectId.ValueString() != "" {
		projectId = model.ProjectId.ValueString()
	} else if projectResp.ProjectId != nil {
		projectId = *projectResp.ProjectId
	} else {
		return fmt.Errorf("project id not present")
	}

	var containerId string
	if model.ContainerId.ValueString() != "" {
		containerId = model.ContainerId.ValueString()
	} else if projectResp.ContainerId != nil {
		containerId = *projectResp.ContainerId
	} else {
		return fmt.Errorf("container id not present")
	}

	var labels basetypes.MapValue
	if projectResp.Labels != nil {
		labels, err = conversion.ToTerraformStringMap(ctx, *projectResp.Labels)
		if err != nil {
			return fmt.Errorf("converting to StringValue map: %w", err)
		}
	} else {
		labels = types.MapNull(types.StringType)
	}

	model.Id = types.StringValue(containerId)
	model.ProjectId = types.StringValue(projectId)
	model.ContainerId = types.StringValue(containerId)
	model.ContainerParentId = types.StringPointerValue(projectResp.Parent.ContainerId)
	model.Name = types.StringPointerValue(projectResp.Name)
	model.Labels = labels
	return nil
}
