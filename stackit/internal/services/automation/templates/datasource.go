package templates

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/experimental/paginate"
	automation "github.com/stackitcloud/stackit-sdk-go/services/automation/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	automationUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/automation/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = new(templatesDataSource)
	_ datasource.DataSourceWithConfigure = new(templatesDataSource)
)

type model struct {
	ID        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	Templates []template   `tfsdk:"templates"`
}

type template struct {
	Id          types.String `tfsdk:"id"`
	CreateTime  types.String `tfsdk:"create_time"`
	Description types.String `tfsdk:"description"`
	Name        types.String `tfsdk:"name"`
}

type templatesDataSource struct {
	client       *automation.APIClient
	providerData core.ProviderData
}

func NewAutomationTemplatesDataSource() datasource.DataSource {
	return new(templatesDataSource)
}

func (d *templatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automation_templates"
}

func (d *templatesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	features.CheckBetaResourcesEnabled(ctx, &d.providerData, &resp.Diagnostics, "stackit_automation_templates", core.Datasource)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := automationUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Automation templates client configured")
}

func (d *templatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Automation templates datasource schema.", core.Datasource),
		Description:         "Automation templates datasource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source ID, structured as \"`project_id`,`region`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "Automation templates data source region. If undefined, the provider region is used.",
				Optional:    true,
				Computed:    true,
			},
			"templates": schema.ListNestedAttribute{
				Description: "List of available templates.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Template ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the template.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the template.",
							Computed:    true,
						},
						"create_time": schema.StringAttribute{
							Description: "Create timestamp of the template.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *templatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	model.Region = types.StringValue(region)
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": projectId,
		"region":     region,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	templatesReq := d.client.DefaultAPI.ListVolumeTemplates(ctx, projectId, region)
	templates, err := paginate.All(templatesReq)

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Reading templates", fmt.Sprintf("Calling ListVolumeTemplates: %v", err))
		return
	}
	if templates == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Reading templates", "ListVolumeTemplates returned an empty response")
		return
	}

	ctx = core.LogResponse(ctx)

	if err := mapFields(templates, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Reading templates", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Automation templates read")
}

func mapFields(templates []automation.Template, m *model) error {
	if templates == nil {
		return fmt.Errorf("nil response")
	}
	if m == nil {
		return fmt.Errorf("nil model")
	}

	m.ID = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), m.Region.ValueString())
	m.Templates = make([]template, 0, len(templates))

	slices.SortFunc(templates, func(a, b automation.Template) int {
		return strings.Compare(a.Id, b.Id)
	})

	for _, respTemplate := range templates {
		m.Templates = append(m.Templates, template{
			Id:          types.StringValue(respTemplate.Id),
			Name:        types.StringValue(respTemplate.Name),
			Description: types.StringValue(respTemplate.Description),
			CreateTime:  types.StringValue(respTemplate.CreateTime.Format(time.RFC3339)),
		})
	}
	return nil
}
