package cdn

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
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
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_cdn_distribution", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	var apiClient *cdn.APIClient
	var err error
	if providerData.CdnCustomEndpoint != "" {
		apiClient, err = cdn.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.CdnCustomEndpoint),
		)
	} else {
		apiClient, err = cdn.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	d.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

func (r *distributionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_distribution"
}

func (r *distributionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"id":                                    "Terrform resource ID",
		"distribution_id":                       "CDN distribution ID",
		"project_id":                            "STACKIT project ID associated with the distribution",
		"status":                                "Status of the distribution",
		"created_at":                            "Time when the distribution was created",
		"updated_at":                            "Time when the distribution was last updated",
		"errors":                                "List of distribution errors",
		"domains":                               "List of configured domains for the distribution",
		"config":                                "The distribution configuration",
		"config_backend":                        "The configured backend for the distribution",
		"config_regions":                        "The configured regions where content will be hosted",
		"config_backend_type":                   "the ",
		"config_backend_origin_url":             "The configured backend type for the distribution",
		"config_backend_origin_request_headers": "The configured origin request headers for the backend",
		"domain_name":                           "The name of the domain",
		"domain_status":                         "The status of the domain",
		"domain_type":                           "The type of the domain. Each distribution has one domain of type \"managed\", and domains of type \"custom\" may be additionally created by the user",
		"domain_errors":                         "List of domain errors",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("CDN distribution data source schema."),
		Description:         "CDN distribution data source schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"distribution_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: descriptions["status"],
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: descriptions["created_at"],
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: descriptions["updated_at"],
			},
			"errors": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: descriptions["errors"],
			},
			"domains": schema.ListNestedAttribute{
				Computed:    true,
				Description: descriptions["domains"],
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: descriptions["domain_name"],
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: descriptions["domain_status"],
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: descriptions["domain_type"],
						},
						"errors": schema.ListAttribute{
							Computed:    true,
							Description: descriptions["domain_errors"],
							ElementType: types.StringType,
						},
					},
				},
			},
			"config": schema.SingleNestedAttribute{
				Computed:    true,
				Description: descriptions["config"],
				Attributes: map[string]schema.Attribute{
					"backend": schema.ListNestedAttribute{
						Computed:    true,
						Description: descriptions["config_backend"],
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Computed:    true,
									Description: descriptions["config_backend_type"],
								},
								"origin_url": schema.StringAttribute{
									Computed:    true,
									Description: descriptions["config_backend_origin_url"],
								},
								"origin_request_headers": schema.ListAttribute{
									Computed:    true,
									Description: descriptions["config_backend_origin_request_headers"],
									ElementType: types.StringType,
								},
							},
						},
					},
					"regions": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
					}},
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
	err = mapFields(distributionResp.Distribution, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN distribution", fmt.Sprintf("Error processing API response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
