package resourcepool

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = (*resourcePoolDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*resourcePoolDataSource)(nil)
)

type dataSourceModel struct {
	Id                             types.String `tfsdk:"id"` // needed by TF
	ProjectId                      types.String `tfsdk:"project_id"`
	ResourcePoolId                 types.String `tfsdk:"resource_pool_id"`
	AvailabilityZone               types.String `tfsdk:"availability_zone"`
	IpAcl                          types.List   `tfsdk:"ip_acl"`
	Name                           types.String `tfsdk:"name"`
	PerformanceClass               types.String `tfsdk:"performance_class"`
	SizeGigabytes                  types.Int64  `tfsdk:"size_gigabytes"`
	SizeReducibleAt                types.String `tfsdk:"size_reducible_at"`
	PerformanceClassDowngradableAt types.String `tfsdk:"performance_class_downgradable_at"`
	Region                         types.String `tfsdk:"region"`
	SnapshotsAreVisible            types.Bool   `tfsdk:"snapshots_are_visible"`
}

type resourcePoolDataSource struct {
	client       *sfs.APIClient
	providerData core.ProviderData
}

func NewResourcePoolDataSource() datasource.DataSource {
	return &resourcePoolDataSource{}
}

// Configure implements datasource.DataSourceWithConfigure.
func (r *resourcePoolDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !datasourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_sfs_resource_pool", core.Datasource)
		if resp.Diagnostics.HasError() {
			return
		}
		datasourceBetaCheckDone = true
	}

	var apiClient *sfs.APIClient
	var err error
	if r.providerData.SfsCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "sfs_custom_endpoint", r.providerData.SfsCustomEndpoint)
		apiClient, err = sfs.NewAPIClient(
			config.WithCustomAuth(r.providerData.RoundTripper),
			config.WithEndpoint(r.providerData.SfsCustomEndpoint),
		)
	} else {
		apiClient, err = sfs.NewAPIClient(
			config.WithCustomAuth(r.providerData.RoundTripper),
			config.WithRegion(r.providerData.GetRegion()),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the datasource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "SFS client configured")
}

// Metadata implements datasource.DataSource.
func (r *resourcePoolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sfs_resource_pool"
}

// Read implements datasource.DataSource.
func (r *resourcePoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model dataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)
	ctx = tflog.SetField(ctx, "region", region)

	response, err := r.client.GetResourcePoolExecute(ctx, projectId, region, resourcePoolId)
	if err != nil {
		var openapiError *oapierror.GenericOpenAPIError
		if errors.As(err, &openapiError) {
			if openapiError.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading resource pool", fmt.Sprintf("Calling API: %v", err))
		return
	}
	// TODO: log traceId

	// Map response body to schema
	err = mapDataSourceFields(ctx, region, response.ResourcePool, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading resource pool", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS resource pool read")
}

// Schema implements datasource.DataSource.
func (r *resourcePoolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Resource-pool datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription(description, core.Datasource),
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`resource_pool_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the resource pool is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"resource_pool_id": schema.StringAttribute{
				Description: "Resourcepool ID",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"availability_zone": schema.StringAttribute{
				Computed:    true,
				Description: "Availability zone.",
			},
			"ip_acl": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         `List of IPs that can mount the resource pool in read-only; IPs must have a subnet mask (e.g. "172.16.0.0/24" for a range of IPs, or "172.16.0.250/32" for a specific IP).`,
				MarkdownDescription: markdownDescription,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(validate.CIDR()),
				},
			},
			"performance_class": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the performance class.",
			},
			"size_gigabytes": schema.Int64Attribute{
				CustomType:  nil,
				Computed:    true,
				Description: `Size of the resource pool (unit: gigabytes)`,
			},
			"name": schema.StringAttribute{
				Description: "Name of the resource pool.",
				Computed:    true,
			},
			"performance_class_downgradable_at": schema.StringAttribute{
				Computed:    true,
				Description: "Time when the performance class can be downgraded again.",
			},
			"size_reducible_at": schema.StringAttribute{
				Computed:    true,
				Description: "Time when the size can be reduced again.",
			},
			"snapshots_are_visible": schema.BoolAttribute{
				Computed:    true,
				Description: "If set to true, snapshots are visible and accessible to users. (default: false)",
			},
			"region": schema.StringAttribute{
				// the region cannot be found automatically, so it has to be passed
				Optional:    true,
				Description: "The resource region. Read-only attribute that reflects the provider region.",
			}},
	}
}

func mapDataSourceFields(ctx context.Context, region string, resourcePool *sfs.GetResourcePoolResponseResourcePool, model *dataSourceModel) error {
	if resourcePool == nil {
		return fmt.Errorf("resource pool empty in response")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.AvailabilityZone = types.StringPointerValue(resourcePool.AvailabilityZone)
	if resourcePool.Id == nil {
		return fmt.Errorf("resource pool id not present")
	}
	model.ResourcePoolId = types.StringPointerValue(resourcePool.Id)
	model.Region = types.StringValue(region)
	model.SnapshotsAreVisible = types.BoolPointerValue(resourcePool.SnapshotsAreVisible)
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		utils.Coalesce(model.ResourcePoolId, types.StringPointerValue(resourcePool.Id)).ValueString(),
	)

	if resourcePool.IpAcl != nil {
		var diags diag.Diagnostics
		model.IpAcl, diags = types.ListValueFrom(ctx, types.StringType, resourcePool.IpAcl)
		if diags.HasError() {
			return fmt.Errorf("failed to map ip acls: %w", core.DiagsToError(diags))
		}
	} else {
		model.IpAcl = types.ListNull(types.StringType)
	}

	model.Name = types.StringPointerValue(resourcePool.Name)
	if pc := resourcePool.PerformanceClass; pc != nil {
		model.PerformanceClass = types.StringPointerValue(pc.Name)
	}

	if resourcePool.Space != nil {
		model.SizeGigabytes = types.Int64PointerValue(resourcePool.Space.SizeGigabytes)
	}

	if t := resourcePool.PerformanceClassDowngradableAt; t != nil {
		model.PerformanceClassDowngradableAt = types.StringValue(t.Format(time.RFC3339))
	}

	if t := resourcePool.SizeReducibleAt; t != nil {
		model.SizeReducibleAt = types.StringValue(t.Format(time.RFC3339))
	}

	return nil
}
