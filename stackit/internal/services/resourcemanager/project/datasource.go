package project

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	resourcemanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &projectDataSource{}
)

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
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := resourcemanagerUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
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
		"creation_time":       "Date-time at which the folder was created.",
		"update_time":         "Date-time at which the folder was last modified.",
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
			"creation_time": schema.StringAttribute{
				Description: descriptions["creation_time"],
				Computed:    true,
			},
			"update_time": schema.StringAttribute{
				Description: descriptions["update_time"],
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "container_id", containerId)

	if containerId == "" && projectId == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", "Either container_id or project_id must be set")
		return
	}

	// set project identifier. If projectId is provided, it takes precedence over containerId
	var identifier = containerId
	identifierType := "Container"
	if projectId != "" {
		identifier = projectId
		identifierType = "Project"
	}

	projectResp, err := d.client.GetProject(ctx, identifier).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading project",
			fmt.Sprintf("%s with ID %q does not exist.", identifierType, identifier),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("%s with ID %q not found or forbidden access", identifierType, identifier),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapProjectFields(ctx, projectResp, &model, &resp.State)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Resource Manager project read")
}
