package iplists

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

	lbiplists "github.com/stackitcloud/stackit-sdk-go/services/lbiplists/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	lbipListsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/lbiplists/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = (*ipListsServiceDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*ipListsServiceDataSource)(nil)
)

type DataSourceModel struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	Region      types.String `tfsdk:"region"`
	Name        types.String `tfsdk:"name"`
	Labels      types.Map    `tfsdk:"labels"`
	NumberOfIPs types.Int32  `tfsdk:"number_of_ips"`
	ContentHash types.String `tfsdk:"content_hash"`
}

type ipListsServiceDataSource struct {
	client       *lbiplists.APIClient
	providerData core.ProviderData
}

func NewIPListsServiceDataSource() datasource.DataSource {
	return &ipListsServiceDataSource{}
}

// Metadata implements [datasource.DataSource].
func (d *ipListsServiceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancer_ip_list"
}

// Configure implements [datasource.DataSourceWithConfigure].
func (d *ipListsServiceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &d.providerData, features.LoadbalancerIPList, "stackit_loadbalancer_ip_list", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := lbipListsUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Load Balancer IP Lists client configured")
}

// Schema implements [datasource.DataSource].
func (d *ipListsServiceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":          "Load Balancer IP Lists data source schema. " + core.DatasourceRegionFallbackDocstring,
		"id":            "Terraform's internal resource ID. It is structured as \"`project_id`,region,`name`\".",
		"project_id":    "STACKIT project ID to which the Load Balancer is associated.",
		"region":        "STACKIT region.",
		"labels":        "User-defined metadata as key-value pairs.",
		"name":          "Name of the Load Balancer IP List.",
		"number_of_ips": "The number of IP addresses in this IP list.",
		"content_hash":  "Unique SHA-256 hex fingerprint used to verify the stored IP list file content.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddExperimentDescription("Load Balancer IP Lists resource schema.", features.LoadbalancerIPList, core.Datasource),
		Description:         descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				Computed:    true,
				ElementType: types.StringType,
			},
			"number_of_ips": schema.Int32Attribute{
				Description: descriptions["number_of_ips"],
				Computed:    true,
			},
			"content_hash": schema.StringAttribute{
				Description: descriptions["content_hash"],
				Computed:    true,
			},
		},
	}
}

// Read implements [datasource.DataSource].
func (d *ipListsServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "name", name)

	ipList, err := d.client.DefaultAPI.GetIPList(ctx, projectId, region, name).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			utils.LogError(ctx, &resp.Diagnostics, err, "Error reading Load Balancer IP list", fmt.Sprintf("Calling API: %v", err), map[int]string{http.StatusNotFound: fmt.Sprintf("Load Balancer IP list %q not found", name)})
			return
		}
		utils.LogError(ctx, &resp.Diagnostics, err, "Error reading Load Balancer IP list", fmt.Sprintf("Calling API: %v", err), nil)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapDataSourceFields(ctx, ipList, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Load Balancer IP list", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Load Balancer IP list read")
}

func mapDataSourceFields(ctx context.Context, ipList *lbiplists.GetIPListResponse, model *DataSourceModel, region string) error {
	if ipList == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	name := ipList.Name
	if name == "" {
		if model.Name.ValueString() == "" {
			return fmt.Errorf("name not present")
		}
		name = model.Name.ValueString()
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, name)
	model.Name = types.StringValue(name)
	model.Region = types.StringValue(region)

	model.ContentHash = types.StringPointerValue(ipList.ContentHash)
	model.NumberOfIPs = types.Int32PointerValue(ipList.NumberOfIps)

	labels, err := utils.MapLabels(ctx, ipList.Labels, model.Labels)
	if err != nil {
		return fmt.Errorf("mapping labels: %w", err)
	}
	model.Labels = labels

	return nil
}
