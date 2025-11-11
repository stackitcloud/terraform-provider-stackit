package cdn

import (
	"context"
	"fmt"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	cdnUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

type distributionDataSource struct {
	client *cdn.APIClient
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &distributionDataSource{}
)

func NewDistributionDataSource() datasource.DataSource {
	return &distributionDataSource{}
}

func (d *distributionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_cdn_distribution", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := cdnUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

func (r *distributionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_distribution"
}

func (r *distributionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	backendOptions := []string{"http"}
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("CDN distribution data source schema.", core.Datasource),
		Description:         "CDN distribution data source schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
			},
			"distribution_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: schemaDescriptions["status"],
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: schemaDescriptions["created_at"],
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: schemaDescriptions["updated_at"],
			},
			"errors": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: schemaDescriptions["errors"],
			},
			"domains": schema.ListNestedAttribute{
				Computed:    true,
				Description: schemaDescriptions["domains"],
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: schemaDescriptions["domain_name"],
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: schemaDescriptions["domain_status"],
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: schemaDescriptions["domain_type"],
						},
						"errors": schema.ListAttribute{
							Computed:    true,
							Description: schemaDescriptions["domain_errors"],
							ElementType: types.StringType,
						},
					},
				},
			},
			"config": schema.SingleNestedAttribute{
				Computed:    true,
				Description: schemaDescriptions["config"],
				Attributes: map[string]schema.Attribute{
					"backend": schema.SingleNestedAttribute{
						Computed:    true,
						Description: schemaDescriptions["config_backend"],
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Computed:    true,
								Description: schemaDescriptions["config_backend_type"] + utils.FormatPossibleValues(backendOptions...),
							},
							"origin_url": schema.StringAttribute{
								Computed:    true,
								Description: schemaDescriptions["config_backend_origin_url"],
							},
							"origin_request_headers": schema.MapAttribute{
								Computed:    true,
								Description: schemaDescriptions["config_backend_origin_request_headers"],
								ElementType: types.StringType,
							},
							"geofencing": schema.MapAttribute{
								Description: "A map of URLs to a list of countries where content is allowed.",
								Computed:    true,
								ElementType: types.ListType{
									ElemType: types.StringType,
								},
							},
						},
					},
					"regions": schema.ListAttribute{
						Computed:    true,
						Description: schemaDescriptions["config_regions"],
						ElementType: types.StringType,
					},
					"blocked_countries": schema.ListAttribute{
						Optional:    true,
						Description: schemaDescriptions["config_blocked_countries"],
						ElementType: types.StringType,
					},
					"optimizer": schema.SingleNestedAttribute{
						Description: schemaDescriptions["config_optimizer"],
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Computed: true,
							},
						},
					},
				},
			},
		},
	}
}

func (r *distributionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	distributionId := model.DistributionId.ValueString()
	distributionResp, err := r.client.GetDistributionExecute(ctx, projectId, distributionId)
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading CDN distribution",
			fmt.Sprintf("Unable to access CDN distribution %q.", distributionId),
			map[int]string{},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	err = mapFields(ctx, distributionResp.Distribution, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN distribution", fmt.Sprintf("Error processing API response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
