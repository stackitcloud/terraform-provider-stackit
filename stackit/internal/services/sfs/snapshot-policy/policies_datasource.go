package snapshot_policy

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	sfsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &policiesDataSource{}
)

func NewSnapshotPoliciesDataSource() datasource.DataSource {
	return &policiesDataSource{}
}

type policiesDataSource struct {
	client       *sfs.APIClient
	providerData core.ProviderData
}

func (r *policiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sfs_snapshot_policies"
}

func (r *policiesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := sfsUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "SFS client configured.")
}

func (r *policiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "SFS snapshot policies datasource schema",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source ID. It is structured as \"`project_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the snapshot policy is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"immutable": schema.BoolAttribute{
				Description: "List only immutable snapshot policies.",
				Optional:    true,
			},
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of the Snapshot Policy.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the Snapshot Policy.",
						},
						"comment": schema.StringAttribute{
							Computed:    true,
							Description: "Comment of the Snapshot Policy.",
						},
						"enabled": schema.BoolAttribute{
							Computed:    true,
							Description: "Wether the Snapshot Policy is enabled.",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "Created At timestamp.",
						},
						"snapshot_schedules": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:    true,
										Description: "ID of the Snapshot Schedule.",
									},
									"name": schema.StringAttribute{
										Computed:    true,
										Description: "Name of the Snapshot Schedule.",
									},
									"created_at": schema.StringAttribute{
										Computed:    true,
										Description: "Created At timestamp.",
									},
									"interval": schema.StringAttribute{
										Computed:    true,
										Description: "Interval of the Snapshot Schedule (follows the cron schedule xpression in Unix-like systems).",
									},
									"prefix": schema.StringAttribute{
										Computed:    true,
										Description: "Prefix used for snapshots created by this policy.",
									},
									"retention_count": schema.Int64Attribute{
										Computed:    true,
										Description: "Retention Count.",
									},
									"retention_period": schema.StringAttribute{
										Computed:    true,
										Description: "Retention Period (ISO 8601 format or 'infinite').",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

type model struct {
	ID        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Immutable types.Bool   `tfsdk:"immutable"`
	Items     []policy     `tfsdk:"items"`
}

type policy struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Comment           types.String `tfsdk:"comment"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	CreatedAt         types.String `tfsdk:"created_at"`
	SnapshotSchedules []schedule   `tfsdk:"snapshot_schedules"`
}

type schedule struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	CreatedAt       types.String `tfsdk:"created_at"`
	Interval        types.String `tfsdk:"interval"`
	Prefix          types.String `tfsdk:"prefix"`
	RetentionCount  types.Int64  `tfsdk:"retention_count"`
	RetentionPeriod types.String `tfsdk:"retention_period"`
}

func (r *policiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)

	listRequest := r.client.DefaultAPI.ListSnapshotPolicies(ctx, projectId)
	if !utils.IsUndefined(model.Immutable) {
		listRequest = listRequest.Immutable(model.Immutable.ValueBool())
	
		title := `The "immutable" attribute of the "stackit_sfs_snapshot_policies" data source is in beta`
		content := `This attribute may be subject to breaking changes in the future. Use with caution.`
		tflog.Warn(ctx, fmt.Sprintf(`%s | %s`, title, content))
		diags.AddWarning(title, content)
	}
	policies, err := listRequest.Execute()
	if err != nil {
		core.LogAndAddError(ctx, &diags, "Error listing snapshot policies", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, policies, &model)
	if err != nil {
		core.LogAndAddError(ctx, &diags, "Error reading snapshot policies", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS snapshot policies read")
}

func mapFields(_ context.Context, resp *sfs.ListSnapshotPoliciesResponse, model *model) error {
	if resp == nil || resp.SnapshotPolicies == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	projectID := model.ProjectId.ValueString()

	model.ID = utils.BuildInternalTerraformId(projectID)

	for _, respPolicy := range resp.SnapshotPolicies {
		var createdAt types.String
		if respPolicy.CreatedAt != nil {
			createdAt = types.StringValue(respPolicy.CreatedAt.String())
		}
		modelPolicy := policy{
			ID:        types.StringPointerValue(respPolicy.Id),
			Name:      types.StringPointerValue(respPolicy.Name),
			Comment:   types.StringPointerValue(respPolicy.Comment),
			Enabled:   types.BoolPointerValue(respPolicy.Enabled),
			CreatedAt: createdAt,
		}
		for _, respSchedule := range respPolicy.SnapshotSchedules {
			var scheduleCreatedAt types.String
			if respPolicy.CreatedAt != nil {
				scheduleCreatedAt = types.StringValue(respSchedule.CreatedAt.String())
			}
			var retentionCount *int64
			if respSchedule.RetentionCount != nil {
				retentionCount = new(int64(*respSchedule.RetentionCount))
			}
			modelSchedule := schedule{
				ID:              types.StringPointerValue(respSchedule.Id),
				Name:            types.StringPointerValue(respSchedule.Name),
				CreatedAt:       scheduleCreatedAt,
				Interval:        types.StringPointerValue(respSchedule.Interval),
				Prefix:          types.StringPointerValue(respSchedule.Prefix),
				RetentionCount:  types.Int64PointerValue(retentionCount),
				RetentionPeriod: types.StringPointerValue(respSchedule.RetentionPeriod),
			}
			modelPolicy.SnapshotSchedules = append(modelPolicy.SnapshotSchedules, modelSchedule)
		}
		model.Items = append(model.Items, modelPolicy)
	}
	return nil
}
