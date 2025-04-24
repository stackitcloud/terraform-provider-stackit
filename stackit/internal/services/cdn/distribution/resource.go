package cdn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &distributionResource{}
	_ resource.ResourceWithConfigure   = &distributionResource{}
	_ resource.ResourceWithImportState = &distributionResource{}
)

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
	Backend backend   `tfsdk:"backend"` // The backend associated with the distribution
	Regions *[]string `tfsdk:"regions"` // The regions in which data will be cached
}

type backend struct {
	Type                 string             `tfsdk:"type"`                   // The type of the backend. Currently, only "http" backend is supported
	OriginURL            string             `tfsdk:"origin_url"`             // The origin URL of the backend
	OriginRequestHeaders *map[string]string `tfsdk:"origin_request_headers"` // Request headers that should be added by the CDN distribution to incoming requests
}

var configTypes = map[string]attr.Type{
	"backend": types.ObjectType{AttrTypes: backendTypes},
	"regions": types.ListType{ElemType: types.StringType},
}

var backendTypes = map[string]attr.Type{
	"type":                   types.StringType,
	"origin_url":             types.StringType,
	"origin_request_headers": types.MapType{ElemType: types.StringType},
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
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if r.providerData, ok = req.ProviderData.(core.ProviderData); !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}
	var apiClient *cdn.APIClient
	var err error
	if r.providerData.CdnCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "loadbalancer_custom_endpoint", r.providerData.LoadBalancerCustomEndpoint)
		apiClient, err = cdn.NewAPIClient(
			config.WithCustomAuth(r.providerData.RoundTripper),
			config.WithEndpoint(r.providerData.LoadBalancerCustomEndpoint),
		)
	} else {
		apiClient, err = cdn.NewAPIClient(
			config.WithCustomAuth(r.providerData.RoundTripper),
		)
	}
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "CDN client configured")
}

func (r *distributionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_distribution"
}

func (r *distributionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Computed:    true,
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
				Required:    true,
				Description: descriptions["config"],
				Attributes: map[string]schema.Attribute{
					"backend": schema.ListNestedAttribute{
						Required:    true,
						Description: descriptions["config_backend"],
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Required:    true,
									Description: descriptions["config_backend_type"],
								},
								"origin_url": schema.StringAttribute{
									Required:    true,
									Description: descriptions["config_backend_origin_url"],
								},
								"origin_request_headers": schema.ListAttribute{
									Optional:    true,
									Description: descriptions["config_backend_origin_request_headers"],
									ElementType: types.StringType,
								},
							},
						},
					},
					"regions": schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
					}},
			},
		},
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
	waitResp, err := wait.CreateDistributionPoolWaitHandler(ctx, r.client, projectId, *createResp.Distribution.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN distribution", fmt.Sprintf("Waiting for create: %v", err))
		return
	}

	err = mapFields(waitResp.Distribution, &model)
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

func toCreatePayload(ctx context.Context, model *Model) (*cdn.CreateDistributionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}
	config, err := convertConfig(ctx, model)
	if err != nil {
		return nil, err
	}
	payload := &cdn.CreateDistributionPayload{
		IntentId:             cdn.PtrString(uuid.NewString()),
		OriginUrl:            config.Backend.HttpBackend.OriginUrl,
		Regions:              config.Regions,
		OriginRequestHeaders: config.Backend.HttpBackend.OriginRequestHeaders,
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
	regions := []cdn.Region{}
	for _, r := range *configModel.Regions {
		regionEnum, err := cdn.NewRegionFromValue(r)
		if err != nil {
			return nil, err
		}
		regions = append(regions, *regionEnum)
	}

	originRequestHeaders := map[string]string{}
	for k, v := range *configModel.Backend.OriginRequestHeaders {
		originRequestHeaders[k] = v
	}
	return &cdn.Config{
		Backend: &cdn.ConfigBackend{
			HttpBackend: &cdn.HttpBackend{
				OriginRequestHeaders: &originRequestHeaders,
				OriginUrl:            &configModel.Backend.OriginURL,
				Type:                 &configModel.Backend.Type,
			},
		},
		Regions: &regions,
	}, nil
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
		if errors.As(err, oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN distribution", fmt.Sprintf("Calling API: %v", err))
		return
	}
	err = mapFields(cdnResp.Distribution, &model)
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

func (r *distributionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
	_, err := r.client.PatchDistribution(ctx, projectId, distributionId).PatchDistributionPayload(cdn.PatchDistributionPayload{
		Config: &cdn.ConfigPatch{
			Backend: &cdn.ConfigPatchBackend{
				HttpBackendPatch: &cdn.HttpBackendPatch{
					OriginRequestHeaders: configModel.Backend.OriginRequestHeaders,
					OriginUrl:            &configModel.Backend.OriginURL,
					Type:                 &configModel.Backend.Type,
				},
			},
			Regions: &regions,
		},
		IntentId: cdn.PtrString(uuid.NewString()),
	}).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", fmt.Sprintf("Patch distribution: %v", err))
	}
	waitResp, err := wait.UpdateDistributionWaitHandler(ctx, r.client, projectId, distributionId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Update CDN distribution", fmt.Sprintf("Waiting for update: %v", err))
		return
	}

	err = mapFields(waitResp.Distribution, &model)
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
	for _, r := range *distribution.Config.Regions {
		regions = append(regions, types.StringValue(string(r)))
	}
	modelRegions, diags := types.ListValue(types.StringType, regions)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	headers := map[string]attr.Value{}
	if distribution.Config.Backend.HttpBackend.OriginRequestHeaders != nil {
		for k, v := range *distribution.Config.Backend.HttpBackend.OriginRequestHeaders {
			headers[k] = types.StringValue(v)
		}
	}
	originRequestHeaders, diags := types.MapValue(types.StringType, headers)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	// note that httpbackend is hardcoded here as long as it is the only available backend
	backend, diags := types.ObjectValue(backendTypes, map[string]attr.Value{
		"type":                   types.StringValue(*distribution.Config.Backend.HttpBackend.Type),
		"origin_url":             types.StringValue(*distribution.Config.Backend.HttpBackend.OriginUrl),
		"origin_request_headers": originRequestHeaders,
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
			if d.Errors != nil {
				for _, e := range *d.Errors {
					errors = append(errors, types.StringValue(*e.En))
				}
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
