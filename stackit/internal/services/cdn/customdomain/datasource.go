package cdn

import (
	"context"
	"fmt"

	cdnSdk "github.com/stackitcloud/stackit-sdk-go/services/cdn/v1api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	cdnUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &customDomainDataSource{}
	_ datasource.DataSourceWithConfigure = &customDomainDataSource{}
)

var certificateDataSourceTypes = map[string]attr.Type{
	"version": types.Int32Type,
}

type customDomainDataSource struct {
	client *cdnSdk.APIClient
}

func NewCustomDomainDataSource() datasource.DataSource {
	return &customDomainDataSource{}
}

type customDomainDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	DistributionId types.String `tfsdk:"distribution_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	Status         types.String `tfsdk:"status"`
	Errors         types.List   `tfsdk:"errors"`
	Certificate    types.Object `tfsdk:"certificate"`
}

func (d *customDomainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_cdn_custom_domain", core.Datasource)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := cdnUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
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
		MarkdownDescription: features.AddBetaDescription("CDN distribution data source schema.", core.Datasource),
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
			"certificate": schema.SingleNestedAttribute{
				Description: certificateSchemaDescriptions["main"],
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"version": schema.Int32Attribute{
						Description: certificateSchemaDescriptions["version"],
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *customDomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model customDomainDataSourceModel // Use the new data source model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	distributionId := model.DistributionId.ValueString()
	ctx = tflog.SetField(ctx, "distribution_id", distributionId)
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "name", name)

	customDomainResp, err := r.client.DefaultAPI.GetCustomDomain(ctx, projectId, distributionId, name).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading CDN custom domain",
			fmt.Sprintf("Unable to access CDN custom domain %q.", name),
			map[int]string{},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Call the new data source mapping function
	err = mapCustomDomainDataSourceFields(customDomainResp, &model, projectId, distributionId)
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

func mapCustomDomainDataSourceFields(customDomainResponse *cdnSdk.GetCustomDomainResponse, model *customDomainDataSourceModel, projectId, distributionId string) error {
	if customDomainResponse == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if customDomainResponse.CustomDomain.Name == "" {
		return fmt.Errorf("name is empty in response")
	}
	if customDomainResponse.CustomDomain.Status == "" {
		return fmt.Errorf("status is empty in response")
	}

	normalizedCert, err := normalizeCertificate(customDomainResponse.Certificate)
	if err != nil {
		return fmt.Errorf("Certificate error in normalizer: %w", err)
	}

	// If the certificate is managed, the certificate block in the state should be null.
	if normalizedCert.Type == "managed" {
		model.Certificate = types.ObjectNull(certificateDataSourceTypes)
	} else {
		// For custom certificates, we only care about the version.
		version := types.Int32Null()
		if normalizedCert.Version != nil {
			version = types.Int32Value(*normalizedCert.Version)
		}

		certificateObj, diags := types.ObjectValue(certificateDataSourceTypes, map[string]attr.Value{
			"version": version,
		})
		if diags.HasError() {
			return fmt.Errorf("failed to map certificate: %w", core.DiagsToError(diags))
		}
		model.Certificate = certificateObj
	}

	model.ID = types.StringValue(fmt.Sprintf("%s,%s,%s", projectId, distributionId, customDomainResponse.CustomDomain.Name))
	model.Status = types.StringValue(string(customDomainResponse.CustomDomain.Status))

	customDomainErrors := []attr.Value{}
	if customDomainResponse.CustomDomain.Errors != nil {
		for _, e := range customDomainResponse.CustomDomain.Errors {
			if e.En == "" {
				return fmt.Errorf("error description missing")
			}
			customDomainErrors = append(customDomainErrors, types.StringValue(e.En))
		}
	}
	modelErrors, diags := types.ListValue(types.StringType, customDomainErrors)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Errors = modelErrors

	// Also map the fields back to the model from the config
	model.ProjectId = types.StringValue(projectId)
	model.DistributionId = types.StringValue(distributionId)
	model.Name = types.StringValue(customDomainResponse.CustomDomain.Name)

	return nil
}
