package cdn

import (
	"context"
	"fmt"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	cdnUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Define Backend Types specifically for Data Source (NO SECRETS)
var dataSourceBackendTypes = map[string]attr.Type{
	"type":                   types.StringType,
	"origin_url":             types.StringType,
	"origin_request_headers": types.MapType{ElemType: types.StringType},
	"geofencing":             geofencingTypes, // Shared from resource.go
	"bucket_url":             types.StringType,
	"region":                 types.StringType,
}

var dataSourceConfigTypes = map[string]attr.Type{
	"backend":           types.ObjectType{AttrTypes: dataSourceBackendTypes},
	"regions":           types.ListType{ElemType: types.StringType},
	"blocked_countries": types.ListType{ElemType: types.StringType},
	"optimizer": types.ObjectType{
		AttrTypes: optimizerTypes, // Shared from resource.go
	},
}

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
	backendOptions := []string{"http", "bucket"}
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
							"bucket_url": schema.StringAttribute{
								Computed:    true,
								Description: schemaDescriptions["config_backend_bucket_url"],
							},
							"region": schema.StringAttribute{
								Computed:    true,
								Description: schemaDescriptions["config_backend_region"],
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

	ctx = core.InitProviderContext(ctx)

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

	ctx = core.LogResponse(ctx)

	// Use specific Data Source mapping function
	err = mapDataSourceFields(ctx, distributionResp.Distribution, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN distribution", fmt.Sprintf("Error processing API response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

// mapDataSourceFields is a specialized version of mapFields for the Data Source.
// It uses dataSourceConfigTypes (excludes bucket access and secrets) and skips state restoration logic.
func mapDataSourceFields(ctx context.Context, distribution *cdn.Distribution, model *Model) error {
	if distribution == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	// Basic fields mapping (same as resource)
	if distribution.ProjectId == nil || distribution.Id == nil || distribution.CreatedAt == nil || distribution.UpdatedAt == nil || distribution.Status == nil {
		return fmt.Errorf("missing required fields in response")
	}

	model.ID = utils.BuildInternalTerraformId(*distribution.ProjectId, *distribution.Id)
	model.DistributionId = types.StringValue(*distribution.Id)
	model.ProjectId = types.StringValue(*distribution.ProjectId)
	model.Status = types.StringValue(string(distribution.GetStatus()))
	model.CreatedAt = types.StringValue(distribution.CreatedAt.String())
	model.UpdatedAt = types.StringValue(distribution.UpdatedAt.String())

	// Distribution Errors
	distributionErrors := []attr.Value{}
	if distribution.Errors != nil {
		for _, e := range *distribution.Errors {
			distributionErrors = append(distributionErrors, types.StringValue(*e.En))
		}
	}
	modelErrors, diags := types.ListValue(types.StringType, distributionErrors)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Errors = modelErrors

	// Regions
	regions := []attr.Value{}
	for _, r := range *distribution.Config.Regions {
		regions = append(regions, types.StringValue(string(r)))
	}
	modelRegions, diags := types.ListValue(types.StringType, regions)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	// Blocked Countries
	var blockedCountries []attr.Value
	if distribution.Config != nil && distribution.Config.BlockedCountries != nil {
		for _, c := range *distribution.Config.BlockedCountries {
			blockedCountries = append(blockedCountries, types.StringValue(string(c)))
		}
	}
	modelBlockedCountries, diags := types.ListValue(types.StringType, blockedCountries)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	// Prepare Backend Values
	var backendValues map[string]attr.Value
	originRequestHeaders := types.MapNull(types.StringType)
	geofencingVal := types.MapNull(geofencingTypes.ElemType)

	// If HTTP Backend is present
	if distribution.Config.Backend.HttpBackend != nil {
		// Headers
		if origHeaders := distribution.Config.Backend.HttpBackend.OriginRequestHeaders; origHeaders != nil && len(*origHeaders) > 0 {
			headers := map[string]attr.Value{}
			for k, v := range *origHeaders {
				headers[k] = types.StringValue(v)
			}
			mappedHeaders, diags := types.MapValue(types.StringType, headers)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			originRequestHeaders = mappedHeaders
		}

		// Geofencing
		if geofencingAPI := distribution.Config.Backend.HttpBackend.Geofencing; geofencingAPI != nil && len(*geofencingAPI) > 0 {
			geofencingMapElems := make(map[string]attr.Value)
			for url, countries := range *geofencingAPI {
				listVal, diags := types.ListValueFrom(ctx, types.StringType, countries)
				if diags.HasError() {
					return core.DiagsToError(diags)
				}
				geofencingMapElems[url] = listVal
			}
			mappedGeofencing, diags := types.MapValue(geofencingTypes.ElemType, geofencingMapElems)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			geofencingVal = mappedGeofencing
		}

		backendValues = map[string]attr.Value{
			"type":                   types.StringValue("http"),
			"origin_url":             types.StringValue(*distribution.Config.Backend.HttpBackend.OriginUrl),
			"origin_request_headers": originRequestHeaders,
			"geofencing":             geofencingVal,
			"bucket_url":             types.StringNull(),
			"region":                 types.StringNull(),
		}
	} else if distribution.Config.Backend.BucketBackend != nil {
		// For Data Source, we strictly return what API gives us. No secret restoration.
		backendValues = map[string]attr.Value{
			"type":                   types.StringValue("bucket"),
			"bucket_url":             types.StringValue(*distribution.Config.Backend.BucketBackend.BucketUrl),
			"region":                 types.StringValue(*distribution.Config.Backend.BucketBackend.Region),
			"origin_url":             types.StringNull(),
			"origin_request_headers": types.MapNull(types.StringType),
			"geofencing":             types.MapNull(geofencingTypes.ElemType),
		}
	}

	// Use dataSourceBackendTypes (NO SECRETS)
	backend, diags := types.ObjectValue(dataSourceBackendTypes, backendValues)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	// Optimizer
	optimizerVal := types.ObjectNull(optimizerTypes)
	if o := distribution.Config.Optimizer; o != nil {
		if enabled, ok := o.GetEnabledOk(); ok {
			var diags diag.Diagnostics
			optimizerVal, diags = types.ObjectValue(optimizerTypes, map[string]attr.Value{
				"enabled": types.BoolValue(enabled),
			})
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}
	}

	// Use dataSourceConfigTypes
	cfg, diags := types.ObjectValue(dataSourceConfigTypes, map[string]attr.Value{
		"backend":           backend,
		"regions":           modelRegions,
		"blocked_countries": modelBlockedCountries,
		"optimizer":         optimizerVal,
	})
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Config = cfg

	// Domains
	domains := []attr.Value{}
	if distribution.Domains != nil {
		for _, d := range *distribution.Domains {
			domainErrors := []attr.Value{}
			if d.Errors != nil {
				for _, e := range *d.Errors {
					domainErrors = append(domainErrors, types.StringValue(*e.En))
				}
			}
			modelDomainErrors, diags := types.ListValue(types.StringType, domainErrors)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			modelDomain, diags := types.ObjectValue(domainTypes, map[string]attr.Value{
				"name":   types.StringValue(*d.Name),
				"status": types.StringValue(string(*d.Status)),
				"type":   types.StringValue(string(*d.Type)),
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
