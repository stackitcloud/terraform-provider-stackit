package instance

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	ufw "github.com/stackitcloud/stackit-sdk-go/services/ufw/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	ufwUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ufw/utils"
)

var (
	_ datasource.DataSource              = &instanceDataSource{}
	_ datasource.DataSourceWithConfigure = &instanceDataSource{}
)

func NewInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

type instanceDataSource struct {
	client       ufw.DefaultAPI
	providerData core.ProviderData
}

type DataSourceModel struct {
	Id         types.String `tfsdk:"id"`
	RuleId     types.String `tfsdk:"rule_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	Region     types.String `tfsdk:"region"`
	InstanceId types.String `tfsdk:"instance_id"`
	Product    types.String `tfsdk:"product"`
	SourceIP   types.String `tfsdk:"source_ip"`
	Type       types.String `tfsdk:"type"`
}

func (d *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ufw_instance"
}

func (d *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.providerData = providerData

	apiClient := ufwUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if apiClient == nil {
		return
	}

	d.client = apiClient.DefaultAPI
	tflog.Info(ctx, "UFW instance datasource client configured")
}

func (d *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "UFW Instance (Rule) datasource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform internal resource identifier in format 'project_id,region,rule_id'.",
				Computed:    true,
			},
			"rule_id": schema.StringAttribute{
				Description: "The rule UUID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT Project ID associated with the rule.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				Optional:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: "The target service instance ID.",
				Computed:    true,
			},
			"product": schema.StringAttribute{
				Description: "The source service product (e.g. 'edge-cloud').",
				Computed:    true,
			},
			"source_ip": schema.StringAttribute{
				Description: "The source IP (CIDR) to which the rule applies.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the rule (e.g., 'ACL').",
				Computed:    true,
			},
		},
	}
}

func (d *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic
	var model DataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	ctx, cancel := context.WithTimeout(ctx, core.DefaultOperationTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	ruleId := model.RuleId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "rule_id", ruleId)

	ruleData, err := d.client.GetRule(ctx, projectId, region, ruleId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading UFW instance datasource", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapDataSourceFields(ruleData, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading UFW instance datasource", fmt.Sprintf("Mapping fields: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "UFW instance datasource read")
}

func mapDataSourceFields(ruleResp *ufw.RuleResponse, model *DataSourceModel, region string) error {
	if ruleResp == nil {
		return fmt.Errorf("response payload is nil")
	}

	model.Region = types.StringValue(region)
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		model.RuleId.ValueString(),
	)
	model.InstanceId = types.StringValue(ruleResp.InstanceId)
	model.Product = types.StringValue(ruleResp.Product)
	model.SourceIP = types.StringValue(ruleResp.SourceIP)
	model.Type = types.StringValue(ruleResp.Type)

	return nil
}
