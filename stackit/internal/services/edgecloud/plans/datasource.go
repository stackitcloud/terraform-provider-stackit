package plan

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/edge"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	edgeutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/edgecloud/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &plansDataSource{}
)

// DataSourceModel maps the data source schema data.
type DataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Plans     types.List   `tfsdk:"plans"`
}

// planTypes defines the attribute types for a single plan object within the list.
var planTypes = map[string]attr.Type{
	"id":             types.StringType,
	"name":           types.StringType,
	"description":    types.StringType,
	"max_edge_hosts": types.Int64Type,
}

// NewPlansDataSource creates a new plan data source.
func NewPlansDataSource() datasource.DataSource {
	return &plansDataSource{}
}

// plansDataSource is the datasource implementation.
type plansDataSource struct {
	client *edge.APIClient
}

// Configure sets up the API client for the Edge Cloud plans data source.
func (d *plansDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_edgecloud_plans", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	d.client = edgeutils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "edge cloud client configured")
}

// Metadata provides metadata for the Edge Cloud plans data source.
func (d *plansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edgecloud_plans"
}

// Schema defines the schema for the Edge Cloud plans data source.
func (d *plansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Edge Cloud is in private Beta and not generally available.\n You can contact support if you are interested in trying it out.", core.Datasource),
		Description:         "The Edge Cloud Plans datasource lists all valid plans for a given project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source ID, `project_id` is used here.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID the Plans belongs to.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"plans": schema.ListNestedAttribute{
				Description: "A list of Edge Cloud Plans.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the plan.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the plan.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the plan.",
							Computed:    true,
						},
						"max_edge_hosts": schema.Int64Attribute{
							Description: "Maximum number of Edge Cloud hosts that can be used.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read fetches the list of Edge Cloud plans and populates the data source.
func (d *plansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var state DataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := state.ProjectId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Fetch all Plans for the project
	plansResp, err := d.client.ListPlansProject(ctx, projectId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Error reading Edge Cloud plans:",
			fmt.Sprintf("Calling API: %v", err),
			map[int]string{
				http.StatusNotFound: fmt.Sprintf("Project %q not found", projectId),
			},
		)
		return
	}

	ctx = core.LogResponse(ctx)

	// Process the API response and build the list
	var plansList []attr.Value
	if plansResp.ValidPlans != nil {
		for _, plan := range *plansResp.ValidPlans {
			planAttrs, err := mapPlanToAttrs(&plan)
			if err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Edge Cloud plans", fmt.Sprintf("Could not process plans: %v", err))
				return
			}

			planObjectValue, diags := types.ObjectValue(planTypes, planAttrs)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			plansList = append(plansList, planObjectValue)
		}
	}

	state.Id = types.StringValue(projectId)

	planListValue, diags := types.ListValue(types.ObjectType{AttrTypes: planTypes}, plansList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Plans = planListValue

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read all Edge Cloud plans")
}

// mapPlanToAttrs maps a single edge.Plan to a map of Terraform attributes.
func mapPlanToAttrs(plan *edge.Plan) (map[string]attr.Value, error) {
	if plan == nil || plan.Id == nil || plan.Name == nil || plan.MaxEdgeHosts == nil {
		return nil, fmt.Errorf("received nil or incomplete plan from API")
	}

	attrs := map[string]attr.Value{
		"id":             types.StringValue(plan.GetId()),
		"name":           types.StringValue(plan.GetName()),
		"description":    types.StringValue(plan.GetDescription()),
		"max_edge_hosts": types.Int64Value(plan.GetMaxEdgeHosts()),
	}

	return attrs, nil
}
