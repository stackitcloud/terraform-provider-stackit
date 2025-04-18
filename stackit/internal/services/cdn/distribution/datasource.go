package cdn

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
		"id":              "",
		"distribution_id": "",
		"project_id":      "",
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
	return
}

func mapFields(distribution *cdn.Distribution, model *Model) error {
	if distribution == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if distribution.ProjectId == nil {
		return fmt.Errorf("Project ID not present")
	}

	if distribution.Id == nil {
		return fmt.Errorf("CDN distribution ID not present")
	}

	id := strings.Join([]string{*distribution.ProjectId, *distribution.Id}, core.Separator)

	model.ID = types.StringValue(id)
	model.DistributionId = types.StringValue(*distribution.Id)
	model.ProjectId = types.StringValue(*distribution.ProjectId)
	model.Status = types.StringValue(*distribution.Status)
	model.CreatedAt = types.StringValue(distribution.CreatedAt.String())
	model.UpdatedAt = types.StringValue(distribution.UpdatedAt.String())

	errors := []attr.Value{}
	if distribution.Errors != nil {
		for _, e := range *distribution.Errors {
			errors = append(errors, types.StringValue(*e.En))
		}
	}
	modelErrors, diags := types.ListValue(types.StringType, errors)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Errors = modelErrors

	regions := []attr.Value{}
	modelRegions, diags := types.ListValue(types.StringType, regions)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	headers := []attr.Value{}
	if distribution.Config.Backend.HttpBackend.OriginRequestHeaders != nil {
		for _, h := range *distribution.Config.Backend.HttpBackend.OriginRequestHeaders {
			headers = append(headers, types.StringValue(h))
		}
	}
	originRequestHeaders, diags := types.ListValue(types.StringType, headers)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	// note that httpbackend is hardcoded here as long as it is the only available backend
	backend, diags := types.ObjectValue(backendTypes, map[string]attr.Value{
		"type":                 types.StringValue(*distribution.Config.Backend.HttpBackend.Type),
		"originUrl":            types.StringValue(*distribution.Config.Backend.HttpBackend.OriginUrl),
		"originRequestHeaders": originRequestHeaders,
	})
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	config, diags := types.ObjectValue(configTypes, map[string]attr.Value{
		"backend": backend,
		"regions": modelRegions,
	})
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Config = config

	domains := []attr.Value{}
	if distribution.Domains != nil {
		for _, d := range *distribution.Domains {
			errors := []attr.Value{}
			for _, e := range *d.Errors {
				errors = append(errors, types.StringValue(*e.En))
			}
			modelDomainErrors, diags := types.ListValue(types.StringType, errors)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			modelDomain, diags := types.ObjectValue(domainTypes, map[string]attr.Value{
				"name":   types.StringValue(*d.Name),
				"status": types.StringValue(string(*d.Status)),
				"type":   types.StringValue(*d.Type),
				"errors": modelDomainErrors,
			})
			if diags.HasError() {
				return core.DiagsToError(diags)
			}

			domains = append(domains, modelDomain)
		}
	}

	modelDomains, diags := types.ListValue(types.ObjectType{AttrTypes: domainTypes}, domains)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Domains = modelDomains
	return nil
}
