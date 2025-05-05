package cdn

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &customDomainDataSource{}
	_ datasource.DataSourceWithConfigure = &customDomainDataSource{}
)

type customDomainDataSource struct {
	client *cdn.APIClient
}

func NewCustomDomainDataSource() datasource.DataSource {
	return &customDomainDataSource{}
}

func (d *customDomainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}
	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_cdn_custom_domain", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	var apiClient *cdn.APIClient
	var err error
	if providerData.CdnCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "cdn_custom_endpoint", providerData.CdnCustomEndpoint)
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
	tflog.Info(ctx, "CDN client configured")
}

func (r *customDomainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_custom_domain"
}

func (r *customDomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("CDN distribution data source schema."),
		Description:         "CDN distribution data source schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: customDomainSchemaDescriptions["id"],
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: customDomainSchemaDescriptions["name"],
				Required:    true,
			},
			"distribution_id": schema.StringAttribute{
				Description: customDomainSchemaDescriptions["distribution_id"],
				Required:    true,
				Validators:  []validator.String{validate.UUID()},
			},
			"project_id": schema.StringAttribute{
				Description: customDomainSchemaDescriptions["project_id"],
				Required:    true,
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: customDomainSchemaDescriptions["status"],
			},
			"errors": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: customDomainSchemaDescriptions["errors"],
			},
		},
	}
}

func (r *customDomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model CustomDomainModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	distributionId := model.DistributionId.ValueString()
	ctx = tflog.SetField(ctx, "distribution_id", distributionId)
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "name", name)

	customDomainResp, err := r.client.GetCustomDomain(ctx, projectId, distributionId, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		// n.b. err is caught here if of type *oapierror.GenericOpenAPIError, which the stackit SDK client returns
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN custom domain", fmt.Sprintf("Calling API: %v", err))
		return
	}
	err = mapCustomDomainFields(customDomainResp.CustomDomain, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN custom domain", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "CDN custom domain read")
}
