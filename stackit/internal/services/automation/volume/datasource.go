package volume

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	automation "github.com/stackitcloud/stackit-sdk-go/services/automation/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	automationUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/automation/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &volumeAutomationDataSource{}
)

// NewVolumeAutomationDataSource is a helper function to simplify the provider implementation.
func NewVolumeAutomationDataSource() datasource.DataSource {
	return &volumeAutomationDataSource{}
}

// volumeAutomationDataSource is the data source implementation.
type volumeAutomationDataSource struct {
	client       *automation.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *volumeAutomationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_automation"
}

// Configure adds the provider configured client to the data source.
func (d *volumeAutomationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_volume_automation", core.Datasource)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := automationUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.providerData = providerData
	d.client = apiClient
	tflog.Info(ctx, "Volume automation client configured.")
}

// Schema defines the schema for the data source.
func (d *volumeAutomationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Volume automation datasource schema. Must have a `region` specified in the provider configuration.", core.Datasource),
		Description:         "Volume automation datasource schema. Must have a `region` specified in the provider configuration.",
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
			"region": schema.StringAttribute{
				// the region cannot be looked up, so it has to be passed
				Optional:    true,
				Description: descriptions["region"],
			},
			"automation_id": schema.StringAttribute{
				Description: descriptions["automation_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"template_id": schema.StringAttribute{
				Description: descriptions["template_id"],
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Computed:    true,
			},
			"input": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: descriptions["input"],
				Computed:    true,
			},
			"triggers": schema.SingleNestedAttribute{
				Description: descriptions["triggers"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"schedule": schema.SingleNestedAttribute{
						Description: descriptions["schedule"],
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"rrule": schema.StringAttribute{
								Description: descriptions["rrule"],
								Computed:    true,
							},
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *volumeAutomationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	automationId := model.AutomationId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "automation_id", automationId)
	ctx = tflog.SetField(ctx, "region", region)

	automationResp, err := d.client.DefaultAPI.GetVolumeAutomation(ctx, projectId, region, automationId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading volume automation",
			fmt.Sprintf("Volume automation with ID %q does not exist in project %q.", automationId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, automationResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading volume automation", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Volume automation read.")
}
