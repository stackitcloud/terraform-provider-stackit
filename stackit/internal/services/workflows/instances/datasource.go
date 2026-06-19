package instances

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	workflowsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/workflows/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var _ datasource.DataSource = &instancesDataSource{}

type Model struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	Instances types.List   `tfsdk:"instances"`
}

type instanceSummary struct {
	InstanceID  types.String `tfsdk:"instance_id"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	Version     types.String `tfsdk:"version"`
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

var instanceSummaryTypes = map[string]attr.Type{
	"instance_id":  basetypes.StringType{},
	"display_name": basetypes.StringType{},
	"description":  basetypes.StringType{},
	"version":      basetypes.StringType{},
	"status":       basetypes.StringType{},
	"created_at":   basetypes.StringType{},
}

type instancesDataSource struct {
	client       *workflows.APIClient
	providerData core.ProviderData
}

func NewWorkflowsInstancesDataSource() datasource.DataSource {
	return &instancesDataSource{}
}

// Metadata returns the data source type name.
func (d *instancesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflows_instances"
}

// Configure adds the provider configured client to the data source.
func (d *instancesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.providerData = providerData

	features.CheckExperimentEnabled(ctx, &d.providerData, features.WorkflowsExperiment, "stackit_workflows_instances", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := workflowsUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
}

// Schema defines the schema for the data source.
func (d *instancesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := fmt.Sprintf("Lists all Workflows instances in a project. %s", core.DatasourceRegionFallbackDocstring)
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.WorkflowsExperiment, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data-source ID. It is structured as \"`project_id`,`region`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID.",
				Required:    true,
				Validators:  []validator.String{validate.UUID(), validate.NoSeparator()},
			},
			"region": schema.StringAttribute{
				Description: "STACKIT region name. If not defined, the provider region is used.",
				Optional:    true,
				Computed:    true,
			},
			"instances": schema.ListNestedAttribute{
				Description: "All Workflows instances in this project + region.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"instance_id":  schema.StringAttribute{Description: "The Workflows instance ID.", Computed: true},
						"display_name": schema.StringAttribute{Description: "Display name of the instance.", Computed: true},
						"description":  schema.StringAttribute{Description: "User-provided description.", Computed: true},
						"version":      schema.StringAttribute{Description: "Workflows version.", Computed: true},
						"status":       schema.StringAttribute{Description: "Lifecycle status.", Computed: true},
						"created_at":   schema.StringAttribute{Description: "Creation timestamp (RFC 3339).", Computed: true},
					},
				},
			},
		},
	}
}

// Read reads the data source and writes its result to Terraform state.
func (d *instancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)

	listResp, err := d.client.DefaultAPI.ListInstances(ctx, projectID, region).Execute()
	if err != nil {
		tfutils.LogError(ctx, &resp.Diagnostics, err, "Error listing Workflows instances", fmt.Sprintf("Project %q region %q", projectID, region), nil)
		return
	}
	ctx = core.LogResponse(ctx)

	model.Region = types.StringValue(region)
	model.ID = types.StringValue(fmt.Sprintf("%s,%s", projectID, region))
	objType := types.ObjectType{AttrTypes: instanceSummaryTypes}
	elements := make([]attr.Value, 0, len(listResp.GetInstances()))
	for i := range listResp.Instances {
		inst := listResp.Instances[i]
		obj, diags := types.ObjectValueFrom(ctx, instanceSummaryTypes, instanceSummary{
			InstanceID:  types.StringValue(inst.Id),
			DisplayName: types.StringValue(inst.DisplayName),
			Description: types.StringPointerValue(inst.Description),
			Version:     types.StringValue(inst.Version),
			Status:      types.StringValue(string(inst.Status)),
			CreatedAt:   types.StringValue(inst.CreatedAt.Format(time.RFC3339)),
		})
		if diags.HasError() {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing Workflows instances", fmt.Sprintf("Mapping instance %s: %v", inst.Id, diags.Errors()))
			return
		}
		elements = append(elements, obj)
	}
	list, diags := types.ListValue(objType, elements)
	if diags.HasError() {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing Workflows instances", fmt.Sprintf("Building list: %v", diags.Errors()))
		return
	}
	model.Instances = list

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Info(ctx, "Workflows instances listed", map[string]any{"count": len(elements)})
}
