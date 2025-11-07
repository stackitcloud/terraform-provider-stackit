package cdn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	cdnUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &distributionResource{}
	_ resource.ResourceWithConfigure   = &distributionResource{}
	_ resource.ResourceWithImportState = &distributionResource{}
)

var schemaDescriptions = map[string]string{
	"id":                                    "Terraform's internal resource identifier. It is structured as \"`project_id`,`distribution_id`\".",
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
	"config_backend_type":                   "The configured backend type. ",
	"config_optimizer":                      "Configuration for the Image Optimizer. This is a paid feature that automatically optimizes images to reduce their file size for faster delivery, leading to improved website performance and a better user experience.",
	"config_backend_origin_url":             "The configured backend type for the distribution",
	"config_backend_origin_request_headers": "The configured origin request headers for the backend",
	"config_blocked_countries":              "The configured countries where distribution of content is blocked",
	"domain_name":                           "The name of the domain",
	"domain_status":                         "The status of the domain",
	"domain_type":                           "The type of the domain. Each distribution has one domain of type \"managed\", and domains of type \"custom\" may be additionally created by the user",
	"domain_errors":                         "List of domain errors",
}

type Model struct {
	ID             types.String `tfsdk:"id"`              // Required by Terraform
	DistributionId types.String `tfsdk:"distribution_id"` // DistributionID associated with the cdn distribution
	ProjectId      types.String `tfsdk:"project_id"`      // ProjectId associated with the cdn distribution
	Status         types.String `tfsdk:"status"`          // The status of the cdn distribution
	CreatedAt      types.String `tfsdk:"created_at"`      // When the distribution was created
	UpdatedAt      types.String `tfsdk:"updated_at"`      // When the distribution was last updated
	Errors         types.List   `tfsdk:"errors"`          // Any errors that the distribution has
	Domains        types.List   `tfsdk:"domains"`         // The domains associated with the distribution
	Config         types.Object `tfsdk:"config"`          // the configuration of the distribution
}

type distributionConfig struct {
	Backend          backend      `tfsdk:"backend"`           // The backend associated with the distribution
	Regions          *[]string    `tfsdk:"regions"`           // The regions in which data will be cached
	BlockedCountries *[]string    `tfsdk:"blocked_countries"` // The countries for which content will be blocked
	Optimizer        types.Object `tfsdk:"optimizer"`         // The optimizer configuration
}

type optimizerConfig struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type backend struct {
	Type                 string                `tfsdk:"type"`                   // The type of the backend. Currently, only "http" backend is supported
	OriginURL            string                `tfsdk:"origin_url"`             // The origin URL of the backend
	OriginRequestHeaders *map[string]string    `tfsdk:"origin_request_headers"` // Request headers that should be added by the CDN distribution to incoming requests
	Geofencing           *map[string][]*string `tfsdk:"geofencing"`             // The geofencing is an object mapping multiple alternative origins to country codes.
}

var configTypes = map[string]attr.Type{
	"backend":           types.ObjectType{AttrTypes: backendTypes},
	"regions":           types.ListType{ElemType: types.StringType},
	"blocked_countries": types.ListType{ElemType: types.StringType},
	"optimizer": types.ObjectType{
		AttrTypes: optimizerTypes,
	},
}

var optimizerTypes = map[string]attr.Type{
	"enabled": types.BoolType,
}

var geofencingTypes = types.MapType{ElemType: types.ListType{
	ElemType: types.StringType,
}}

var backendTypes = map[string]attr.Type{
	"type":                   types.StringType,
	"origin_url":             types.StringType,
	"origin_request_headers": types.MapType{ElemType: types.StringType},
	"geofencing":             geofencingTypes,
}

var domainTypes = map[string]attr.Type{
	"name":   types.StringType,
	"status": types.StringType,
	"type":   types.StringType,
	"errors": types.ListType{ElemType: types.StringType},
}

type distributionResource struct {
	client       *cdn.APIClient
	providerData core.ProviderData
}

func NewDistributionResource() resource.Resource {
	return &distributionResource{}
}

func (r *distributionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_cdn_distribution", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := cdnUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "CDN client configured")
}

func (r *distributionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_distribution"
}

func (r *distributionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	backendOptions := []string{"http"}
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("CDN distribution data source schema.", core.Resource),
		Description:         "CDN distribution data source schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
			},
			"distribution_id": schema.StringAttribute{
				Description: schemaDescriptions["distribution_id"],
				Computed:    true,
				Validators:  []validator.String{validate.UUID()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				Optional:    false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
				Required:    true,
				Description: schemaDescriptions["config"],
				Attributes: map[string]schema.Attribute{
					"optimizer": schema.SingleNestedAttribute{
						Description: schemaDescriptions["config_optimizer"],
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional: true,
								Computed: true,
							},
						},
						Validators: []validator.Object{
							objectvalidator.AlsoRequires(path.MatchRelative().AtName("enabled")),
						},
					},
					"backend": schema.SingleNestedAttribute{
						Required:    true,
						Description: schemaDescriptions["config_backend"],
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Required:    true,
								Description: schemaDescriptions["config_backend_type"] + utils.FormatPossibleValues(backendOptions...),
								Validators:  []validator.String{stringvalidator.OneOf(backendOptions...)},
							},
							"origin_url": schema.StringAttribute{
								Required:    true,
								Description: schemaDescriptions["config_backend_origin_url"],
							},
							"origin_request_headers": schema.MapAttribute{
								Optional:    true,
								Description: schemaDescriptions["config_backend_origin_request_headers"],
								ElementType: types.StringType,
							},
							"geofencing": schema.MapAttribute{
								Description: "A map of URLs to a list of countries where content is allowed.",
								Optional:    true,
								ElementType: types.ListType{
									ElemType: types.StringType,
								},
								Validators: []validator.Map{
									mapvalidator.SizeAtLeast(1),
								},
							},
						},
					},
					"regions": schema.ListAttribute{
						Required:    true,
						Description: schemaDescriptions["config_regions"],
						ElementType: types.StringType,
					},
					"blocked_countries": schema.ListAttribute{
						Optional:    true,
						Description: schemaDescriptions["config_blocked_countries"],
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (r *distributionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !utils.IsUndefined(model.Config) {
		var config distributionConfig
		if !model.Config.IsNull() {
			diags := model.Config.As(ctx, &config, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return
			}
			if geofencing := config.Backend.Geofencing; geofencing != nil {
				for url, region := range *geofencing {
					if region == nil {
						core.LogAndAddError(ctx, &resp.Diagnostics, "Invalid geofencing config", fmt.Sprintf("The list of countries for URL %q must not be null.", url))
						continue
					}
					if len(region) == 0 {
						core.LogAndAddError(ctx, &resp.Diagnostics, "Invalid geofencing config", fmt.Sprintf("The list of countries for URL %q must not be empty.", url))
						continue
					}

					for i, countryPtr := range region {
						if countryPtr == nil {
							core.LogAndAddError(ctx, &resp.Diagnostics, "Invalid geofencing config", fmt.Sprintf("Found a null value in the country list for URL %q at index %d.", url, i))
							break
						}
					}
				}
			}
		}
	}
}

func (r *distributionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN distribution", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.CreateDistribution(ctx, projectId).CreateDistributionPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN distribution", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.CreateDistributionPoolWaitHandler(ctx, r.client, projectId, *createResp.Distribution.Id).SetTimeout(5 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN distribution", fmt.Sprintf("Waiting for create: %v", err))
		return
	}

	err = mapFields(ctx, waitResp.Distribution, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN distribution", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "CDN distribution created")
}

func (r *distributionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	distributionId := model.DistributionId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "distribution_id", distributionId)

	cdnResp, err := r.client.GetDistribution(ctx, projectId, distributionId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		// n.b. err is caught here if of type *oapierror.GenericOpenAPIError, which the stackit SDK client returns
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN distribution", fmt.Sprintf("Calling API: %v", err))
		return
	}
	err = mapFields(ctx, cdnResp.Distribution, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN ditribution", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "CDN distribution read")
}

func (r *distributionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	distributionId := model.DistributionId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "distribution_id", distributionId)

	configModel := distributionConfig{}
	diags = model.Config.As(ctx, &configModel, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    false,
		UnhandledUnknownAsEmpty: false,
	})
	if diags.HasError() {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", "Error mapping config")
		return
	}

	regions := []cdn.Region{}
	for _, r := range *configModel.Regions {
		regionEnum, err := cdn.NewRegionFromValue(r)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", fmt.Sprintf("Map regions: %v", err))
			return
		}
		regions = append(regions, *regionEnum)
	}

	// blockedCountries
	// Use a pointer to a slice to distinguish between an empty list (unblock all) and nil (no change).
	var blockedCountries *[]string
	if configModel.BlockedCountries != nil {
		// Use a temporary slice
		tempBlockedCountries := []string{}

		for _, blockedCountry := range *configModel.BlockedCountries {
			validatedBlockedCountry, err := validateCountryCode(blockedCountry)
			if err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", fmt.Sprintf("Blocked countries: %v", err))
				return
			}
			tempBlockedCountries = append(tempBlockedCountries, validatedBlockedCountry)
		}

		// Point to the populated slice
		blockedCountries = &tempBlockedCountries
	}

	geofencingPatch := map[string][]string{}
	if configModel.Backend.Geofencing != nil {
		gf := make(map[string][]string)
		for url, countries := range *configModel.Backend.Geofencing {
			countryStrings := make([]string, len(countries))
			for i, countryPtr := range countries {
				if countryPtr == nil {
					core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", fmt.Sprintf("Geofencing url %q has a null value", url))
					return
				}
				countryStrings[i] = *countryPtr
			}
			gf[url] = countryStrings
		}
		geofencingPatch = gf
	}

	configPatch := &cdn.ConfigPatch{
		Backend: &cdn.ConfigPatchBackend{
			HttpBackendPatch: &cdn.HttpBackendPatch{
				OriginRequestHeaders: configModel.Backend.OriginRequestHeaders,
				OriginUrl:            &configModel.Backend.OriginURL,
				Type:                 &configModel.Backend.Type,
				Geofencing:           &geofencingPatch, // Use the converted variable
			},
		},
		Regions:          &regions,
		BlockedCountries: blockedCountries,
	}

	if !utils.IsUndefined(configModel.Optimizer) {
		var optimizerModel optimizerConfig

		diags = configModel.Optimizer.As(ctx, &optimizerModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", "Error mapping optimizer config")
			return
		}

		optimizer := cdn.NewOptimizerPatch()
		if !utils.IsUndefined(optimizerModel.Enabled) {
			optimizer.SetEnabled(optimizerModel.Enabled.ValueBool())
		}
		configPatch.Optimizer = optimizer
	}

	_, err := r.client.PatchDistribution(ctx, projectId, distributionId).PatchDistributionPayload(cdn.PatchDistributionPayload{
		Config:   configPatch,
		IntentId: cdn.PtrString(uuid.NewString()),
	}).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", fmt.Sprintf("Patch distribution: %v", err))
		return
	}

	waitResp, err := wait.UpdateDistributionWaitHandler(ctx, r.client, projectId, distributionId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", fmt.Sprintf("Waiting for update: %v", err))
		return
	}

	err = mapFields(ctx, waitResp.Distribution, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "CDN distribution updated")
}

func (r *distributionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	distributionId := model.DistributionId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "distribution_id", distributionId)

	_, err := r.client.DeleteDistribution(ctx, projectId, distributionId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Delete CDN distribution", fmt.Sprintf("Delete distribution: %v", err))
	}
	_, err = wait.DeleteDistributionWaitHandler(ctx, r.client, projectId, distributionId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Delete CDN distribution", fmt.Sprintf("Waiting for deletion: %v", err))
		return
	}
	tflog.Info(ctx, "CDN distribution deleted")
}

func (r *distributionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing CDN distribution", fmt.Sprintf("Expected import identifier on the format: [project_id]%q[distribution_id], got %q", core.Separator, req.ID))
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("distribution_id"), idParts[1])...)
	tflog.Info(ctx, "CDN distribution state imported")
}

func mapFields(ctx context.Context, distribution *cdn.Distribution, model *Model) error {
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

	if distribution.CreatedAt == nil {
		return fmt.Errorf("CreatedAt missing in response")
	}

	if distribution.UpdatedAt == nil {
		return fmt.Errorf("UpdatedAt missing in response")
	}

	if distribution.Status == nil {
		return fmt.Errorf("Status missing in response")
	}

	model.ID = utils.BuildInternalTerraformId(*distribution.ProjectId, *distribution.Id)
	model.DistributionId = types.StringValue(*distribution.Id)
	model.ProjectId = types.StringValue(*distribution.ProjectId)
	model.Status = types.StringValue(string(distribution.GetStatus()))
	model.CreatedAt = types.StringValue(distribution.CreatedAt.String())
	model.UpdatedAt = types.StringValue(distribution.UpdatedAt.String())

	// distributionErrors
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

	// regions
	regions := []attr.Value{}
	for _, r := range *distribution.Config.Regions {
		regions = append(regions, types.StringValue(string(r)))
	}
	modelRegions, diags := types.ListValue(types.StringType, regions)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	// blockedCountries
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

	// originRequestHeaders
	originRequestHeaders := types.MapNull(types.StringType)
	if origHeaders := distribution.Config.Backend.HttpBackend.OriginRequestHeaders; origHeaders != nil && len(*origHeaders) > 0 {
		headers := map[string]attr.Value{}
		for k, v := range *origHeaders {
			headers[k] = types.StringValue(v)
		}
		mappedHeaders, diags := types.MapValue(types.StringType, headers)
		originRequestHeaders = mappedHeaders
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
	}

	// geofencing
	var oldConfig distributionConfig
	oldGeofencingMap := make(map[string][]*string)
	if !model.Config.IsNull() {
		diags = model.Config.As(ctx, &oldConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
		if oldConfig.Backend.Geofencing != nil {
			oldGeofencingMap = *oldConfig.Backend.Geofencing
		}
	}

	reconciledGeofencingData := make(map[string][]string)
	if geofencingAPI := distribution.Config.Backend.HttpBackend.Geofencing; geofencingAPI != nil && len(*geofencingAPI) > 0 {
		newGeofencingMap := *geofencingAPI
		for url, newCountries := range newGeofencingMap {
			oldCountriesPtrs := oldGeofencingMap[url]

			oldCountries := utils.ConvertPointerSliceToStringSlice(oldCountriesPtrs)

			reconciledCountries := utils.ReconcileStringSlices(oldCountries, newCountries)
			reconciledGeofencingData[url] = reconciledCountries
		}
	}

	geofencingVal := types.MapNull(geofencingTypes.ElemType)
	if len(reconciledGeofencingData) > 0 {
		geofencingMapElems := make(map[string]attr.Value)
		for url, countries := range reconciledGeofencingData {
			listVal, diags := types.ListValueFrom(ctx, types.StringType, countries)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			geofencingMapElems[url] = listVal
		}

		var mappedGeofencing basetypes.MapValue
		mappedGeofencing, diags = types.MapValue(geofencingTypes.ElemType, geofencingMapElems)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
		geofencingVal = mappedGeofencing
	}

	// note that httpbackend is hardcoded here as long as it is the only available backend
	backend, diags := types.ObjectValue(backendTypes, map[string]attr.Value{
		"type":                   types.StringValue(*distribution.Config.Backend.HttpBackend.Type),
		"origin_url":             types.StringValue(*distribution.Config.Backend.HttpBackend.OriginUrl),
		"origin_request_headers": originRequestHeaders,
		"geofencing":             geofencingVal,
	})
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	optimizerVal := types.ObjectNull(optimizerTypes)
	if o := distribution.Config.Optimizer; o != nil {
		optimizerEnabled, ok := o.GetEnabledOk()
		if ok {
			var diags diag.Diagnostics
			optimizerVal, diags = types.ObjectValue(optimizerTypes, map[string]attr.Value{
				"enabled": types.BoolValue(optimizerEnabled),
			})
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
		}
	}
	cfg, diags := types.ObjectValue(configTypes, map[string]attr.Value{
		"backend":           backend,
		"regions":           modelRegions,
		"blocked_countries": modelBlockedCountries,
		"optimizer":         optimizerVal,
	})
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Config = cfg

	domains := []attr.Value{}
	if distribution.Domains != nil {
		for _, d := range *distribution.Domains {
			domainErrors := []attr.Value{}
			if d.Errors != nil {
				for _, e := range *d.Errors {
					if e.En == nil {
						return fmt.Errorf("error description missing")
					}
					domainErrors = append(domainErrors, types.StringValue(*e.En))
				}
			}
			modelDomainErrors, diags := types.ListValue(types.StringType, domainErrors)
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			if d.Name == nil || d.Status == nil || d.Type == nil {
				return fmt.Errorf("domain entry incomplete")
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

func toCreatePayload(ctx context.Context, model *Model) (*cdn.CreateDistributionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}
	cfg, err := convertConfig(ctx, model)
	if err != nil {
		return nil, err
	}
	var optimizer *cdn.Optimizer
	if cfg.Optimizer != nil {
		optimizer = cdn.NewOptimizer(cfg.Optimizer.GetEnabled())
	}

	payload := &cdn.CreateDistributionPayload{
		IntentId:             cdn.PtrString(uuid.NewString()),
		OriginUrl:            cfg.Backend.HttpBackend.OriginUrl,
		Regions:              cfg.Regions,
		BlockedCountries:     cfg.BlockedCountries,
		OriginRequestHeaders: cfg.Backend.HttpBackend.OriginRequestHeaders,
		Geofencing:           cfg.Backend.HttpBackend.Geofencing,
		Optimizer:            optimizer,
	}

	return payload, nil
}

func convertConfig(ctx context.Context, model *Model) (*cdn.Config, error) {
	if model == nil {
		return nil, errors.New("model cannot be nil")
	}
	if model.Config.IsNull() || model.Config.IsUnknown() {
		return nil, errors.New("config cannot be nil or unknown")
	}
	configModel := distributionConfig{}
	diags := model.Config.As(ctx, &configModel, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    false,
		UnhandledUnknownAsEmpty: false,
	})
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	// regions
	regions := []cdn.Region{}
	for _, r := range *configModel.Regions {
		regionEnum, err := cdn.NewRegionFromValue(r)
		if err != nil {
			return nil, err
		}
		regions = append(regions, *regionEnum)
	}

	// blockedCountries
	var blockedCountries []string
	if configModel.BlockedCountries != nil {
		for _, blockedCountry := range *configModel.BlockedCountries {
			validatedBlockedCountry, err := validateCountryCode(blockedCountry)
			if err != nil {
				return nil, err
			}
			blockedCountries = append(blockedCountries, validatedBlockedCountry)
		}
	}

	// geofencing
	geofencing := map[string][]string{}
	if configModel.Backend.Geofencing != nil {
		for endpoint, countryCodes := range *configModel.Backend.Geofencing {
			geofencingCountry := make([]string, len(countryCodes))
			for i, countryCodePtr := range countryCodes {
				if countryCodePtr == nil {
					return nil, fmt.Errorf("geofencing url %q has a null value", endpoint)
				}
				validatedCountry, err := validateCountryCode(*countryCodePtr)
				if err != nil {
					return nil, err
				}
				geofencingCountry[i] = validatedCountry
			}
			geofencing[endpoint] = geofencingCountry
		}
	}

	// originRequestHeaders
	originRequestHeaders := map[string]string{}
	if configModel.Backend.OriginRequestHeaders != nil {
		for k, v := range *configModel.Backend.OriginRequestHeaders {
			originRequestHeaders[k] = v
		}
	}

	cdnConfig := &cdn.Config{
		Backend: &cdn.ConfigBackend{
			HttpBackend: &cdn.HttpBackend{
				OriginRequestHeaders: &originRequestHeaders,
				OriginUrl:            &configModel.Backend.OriginURL,
				Type:                 &configModel.Backend.Type,
				Geofencing:           &geofencing,
			},
		},
		Regions:          &regions,
		BlockedCountries: &blockedCountries,
	}

	if !utils.IsUndefined(configModel.Optimizer) {
		var optimizerModel optimizerConfig
		diags := configModel.Optimizer.As(ctx, &optimizerModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, core.DiagsToError(diags)
		}

		if !utils.IsUndefined(optimizerModel.Enabled) {
			cdnConfig.Optimizer = cdn.NewOptimizer(optimizerModel.Enabled.ValueBool())
		}
	}

	return cdnConfig, nil
}

// validateCountryCode checks for a valid country user input. This is just a quick check
// since the API already does a more thorough check.
func validateCountryCode(country string) (string, error) {
	if len(country) != 2 {
		return "", errors.New("country code must be exactly 2 characters long")
	}

	upperCountry := strings.ToUpper(country)

	// Check if both characters are alphabetical letters within the ASCII range A-Z.
	// Yes, we could use the unicode package, but we are only targeting ASCII letters specifically, so
	// let's omit this dependency.
	char1 := upperCountry[0]
	char2 := upperCountry[1]

	if !((char1 >= 'A' && char1 <= 'Z') && (char2 >= 'A' && char2 <= 'Z')) {
		return "", fmt.Errorf("country code '%s' must consist of two alphabetical letters (A-Z or a-z)", country)
	}

	return upperCountry, nil
}
