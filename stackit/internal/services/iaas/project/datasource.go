package project

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSourceWithConfigure = &projectDataSource{}
)

type DatasourceModel struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	AreaId         types.String `tfsdk:"area_id"`
	InternetAccess types.Bool   `tfsdk:"internet_access"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`

	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	State types.String `tfsdk:"state"`
}

// NewProjectDataSource is a helper function to simplify the provider implementation.
func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

// projectDatasource is the data source implementation.
type projectDataSource struct {
	client *iaas.APIClient
}

func (d *projectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Metadata returns the data source type name.
func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iaas_project"
}

// Schema defines the schema for the datasource.
func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":            "Project details. Must have a `region` specified in the provider configuration.",
		"id":              "Terraform's internal resource ID. It is structured as \"`project_id`\".",
		"project_id":      "STACKIT project ID.",
		"area_id":         "The area ID to which the project belongs to.",
		"internet_access": "Specifies if the project has internet_access",
		"state":           "Specifies the state of the project.",
		"created_at":      "Date-time when the project was created.",
		"updated_at":      "Date-time when the project was last updated.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: descriptions["main"],
		Description:         descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"area_id": schema.StringAttribute{
				Description: descriptions["area_id"],
				Computed:    true,
			},
			"internet_access": schema.BoolAttribute{
				Description: descriptions["internet_access"],
				Computed:    true,
			},
			// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
			"state": schema.StringAttribute{
				DeprecationMessage: "Deprecated: Will be removed in May 2026. Use the `status` field instead.",
				Description:        descriptions["state"],
				Computed:           true,
			},
			"status": schema.StringAttribute{
				Description: descriptions["status"],
				Computed:    true,
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

// Read refreshes the Terraform state with the latest data.
func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DatasourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "project_id", projectId)

	projectResp, err := d.client.GetProjectDetailsExecute(ctx, projectId)
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading project",
			fmt.Sprintf("Project with ID %q does not exists.", projectId),
			nil,
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapDataSourceFields(projectResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Process API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "project read")
}

func mapDataSourceFields(projectResp *iaas.Project, model *DatasourceModel) error {
	if projectResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var projectId string
	if model.ProjectId.ValueString() != "" {
		projectId = model.ProjectId.ValueString()
	} else if projectResp.Id != nil {
		projectId = *projectResp.Id
	} else {
		return fmt.Errorf("project id is not present")
	}

	model.Id = utils.BuildInternalTerraformId(projectId)
	model.ProjectId = types.StringValue(projectId)

	var areaId basetypes.StringValue
	if projectResp.AreaId != nil {
		if projectResp.AreaId.String != nil {
			areaId = types.StringPointerValue(projectResp.AreaId.String)
		} else if projectResp.AreaId.StaticAreaID != nil {
			areaId = types.StringValue(string(*projectResp.AreaId.StaticAreaID))
		}
	}

	var createdAt basetypes.StringValue
	if projectResp.CreatedAt != nil {
		createdAtValue := *projectResp.CreatedAt
		createdAt = types.StringValue(createdAtValue.Format(time.RFC3339))
	}

	var updatedAt basetypes.StringValue
	if projectResp.UpdatedAt != nil {
		updatedAtValue := *projectResp.UpdatedAt
		updatedAt = types.StringValue(updatedAtValue.Format(time.RFC3339))
	}

	model.AreaId = areaId
	model.InternetAccess = types.BoolPointerValue(projectResp.InternetAccess)
	model.State = types.StringPointerValue(projectResp.Status)
	model.Status = types.StringPointerValue(projectResp.Status)
	model.CreatedAt = createdAt
	model.UpdatedAt = updatedAt
	return nil
}
