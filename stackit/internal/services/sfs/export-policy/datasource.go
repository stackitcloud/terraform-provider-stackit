package exportpolicy

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
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = (*exportPolicyDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*exportPolicyDataSource)(nil)
)

type exportPolicyDataSource struct {
	client       *sfs.APIClient
	providerData core.ProviderData
}

// Metadata implements datasource.DataSource.
func (d *exportPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sfs_export_policy"
}

// Configure implements datasource.DataSourceWithConfigure.
func (d *exportPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &d.providerData, &resp.Diagnostics, "stackit_sfs_export_policy", core.Datasource)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := sfsUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "SFS client configured")
}

func NewExportPolicyDataSource() datasource.DataSource {
	return &exportPolicyDataSource{}
}

// Read implements datasource.DataSource.
func (d *exportPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { //nolint:gocritic // defined by terraform api
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	exportPolicyId := model.ExportPolicyId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "policy_id", exportPolicyId)
	ctx = tflog.SetField(ctx, "region", region)

	ctx = core.InitProviderContext(ctx)

	// get export policy
	exportPolicyResp, err := d.client.GetShareExportPolicy(ctx, projectId, region, exportPolicyId).Execute()
	if err != nil {
		var openapiError *oapierror.GenericOpenAPIError
		if errors.As(err, &openapiError) {
			if openapiError.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading export policy", fmt.Sprintf("Calling API to get export policy: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// map export policy
	err = mapFields(ctx, exportPolicyResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading export policy", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "SFS export policy read")
}

// Schema implements datasource.DataSource.
func (d *exportPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "SFS export policy datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddBetaDescription(description, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`policy_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the export policy is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"policy_id": schema.StringAttribute{
				Description: "Export policy ID",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the export policy.",
				Computed:    true,
			},
			"rules": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "Description of the Rule",
						},
						"ip_acl": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
							Description: `IP access control list; IPs must have a subnet mask (e.g. "172.16.0.0/24" for a range of IPs, or "172.16.0.250/32" for a specific IP).`,
						},
						"order": schema.Int64Attribute{
							Description: "Order of the rule within a Share Export Policy. The order is used so that when a client IP matches multiple rules, the first rule is applied",
							Computed:    true,
						},
						"read_only": schema.BoolAttribute{
							Description: "Flag to indicate if client IPs matching this rule can only mount the share in read only mode",
							Computed:    true,
						},
						"set_uuid": schema.BoolAttribute{
							Description: "Flag to honor set UUID",
							Computed:    true,
						},
						"super_user": schema.BoolAttribute{
							Description: "Flag to indicate if client IPs matching this rule have root access on the Share",
							Computed:    true,
						},
					},
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				// the region cannot be found, so it has to be passed
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
			},
		},
	}
}
