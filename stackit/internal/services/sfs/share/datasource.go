package share

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
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	sfsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = (*shareDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*shareDataSource)(nil)
)

type dataSourceModel struct {
	Id                      types.String `tfsdk:"id"` // needed by TF
	ProjectId               types.String `tfsdk:"project_id"`
	ResourcePoolId          types.String `tfsdk:"resource_pool_id"`
	ShareId                 types.String `tfsdk:"share_id"`
	Name                    types.String `tfsdk:"name"`
	MountPath               types.String `tfsdk:"mount_path"`
	SpaceHardLimitGigabytes types.Int64  `tfsdk:"space_hard_limit_gigabytes"`
	ExportPolicyName        types.String `tfsdk:"export_policy"`
	Region                  types.String `tfsdk:"region"`
}
type shareDataSource struct {
	client       *sfs.APIClient
	providerData core.ProviderData
}

func NewShareDataSource() datasource.DataSource {
	return &shareDataSource{}
}

// Configure implements datasource.DataSourceWithConfigure.
func (r *shareDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !datasourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_sfs_share", core.Datasource)
		if resp.Diagnostics.HasError() {
			return
		}
		datasourceBetaCheckDone = true
	}

	apiClient := sfsUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "SFS client configured")
}

// Metadata implements datasource.DataSource.
func (r *shareDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sfs_share"
}

// Read implements datasource.DataSource.
func (r *shareDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model dataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	shareId := model.ShareId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)
	ctx = tflog.SetField(ctx, "share_id", shareId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	response, err := r.client.GetShareExecute(ctx, projectId, region, resourcePoolId, shareId)
	if err != nil {
		var openapiError *oapierror.GenericOpenAPIError
		if errors.As(err, &openapiError) {
			if openapiError.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading share", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapDataSourceFields(ctx, region, response.Share, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading share", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS share read")
}

// Schema implements datasource.DataSource.
func (r *shareDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "NFS-Share datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription(description, core.Datasource),
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`share_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the share is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"resource_pool_id": schema.StringAttribute{
				Description: "The ID of the resource pool for the NFS share.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"share_id": schema.StringAttribute{
				Description: "share ID",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"export_policy": schema.StringAttribute{
				Description: `Name of the Share Export Policy to use in the Share.
Note that if this is not set, the Share can only be mounted in read only by 
clients with IPs matching the IP ACL of the Resource Pool hosting this Share. 
You can also assign a Share Export Policy after creating the Share`,
				Computed: true,
			},
			"space_hard_limit_gigabytes": schema.Int64Attribute{
				Computed: true,
				Description: `Space hard limit for the Share. 
				If zero, the Share will have access to the full space of the Resource Pool it lives in.
				(unit: gigabytes)`,
			},
			"mount_path": schema.StringAttribute{
				Computed:    true,
				Description: "Mount path of the Share, used to mount the Share",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the Share",
			},
			"region": schema.StringAttribute{
				// the region cannot be found automatically, so it has to be passed
				Optional:    true,
				Description: "The resource region. Read-only attribute that reflects the provider region.",
			},
		},
	}
}

func mapDataSourceFields(_ context.Context, region string, share *sfs.GetShareResponseShare, model *dataSourceModel) error {
	if share == nil {
		return fmt.Errorf("share empty in response")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if share.Id == nil {
		return fmt.Errorf("share id not present")
	}
	model.ShareId = types.StringPointerValue(share.Id)
	model.Region = types.StringValue(region)

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		model.ResourcePoolId.ValueString(),
		utils.Coalesce(model.ShareId, types.StringPointerValue(share.Id)).ValueString(),
	)
	model.Name = types.StringPointerValue(share.Name)
	if policy := share.ExportPolicy.Get(); policy != nil {
		model.ExportPolicyName = types.StringPointerValue(policy.Name)
	}

	model.SpaceHardLimitGigabytes = types.Int64PointerValue(share.SpaceHardLimitGigabytes)
	if share.HasExportPolicy() {
		model.ExportPolicyName = types.StringPointerValue(share.ExportPolicy.Get().Name)
	}

	model.MountPath = types.StringPointerValue(share.MountPath)

	return nil
}
