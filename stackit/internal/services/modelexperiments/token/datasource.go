package token

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	modelexperimentsutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

var (
	_ datasource.DataSource              = &instanceTokenDataSource{}
	_ datasource.DataSourceWithConfigure = &instanceTokenDataSource{}
)

type InstanceTokenDataSourceModel struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	Region      types.String `tfsdk:"region"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	InstanceId  types.String `tfsdk:"instance_id"`
	TokenId     types.String `tfsdk:"token_id"`
	Labels      types.Map    `tfsdk:"labels"`
	ValidUntil  types.String `tfsdk:"valid_until"`
}

func NewInstanceTokenDataSource() datasource.DataSource {
	return &instanceTokenDataSource{}
}

type instanceTokenDataSource struct {
	client       modelexperiments.DefaultAPI
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (i *instanceTokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_modelexperiments_token"
}

// Configure enables provider-level data or clients to be set in the
// provider-defined DataSource type. It is separately executed for each
// ReadDataSource RPC.
func (i *instanceTokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	i.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := modelexperimentsutils.ConfigureClient(ctx, &i.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	i.client = apiClient.DefaultAPI
	tflog.Info(ctx, "Model Experiments instance token client configured for data source")
}

// Schema defines the schema for the resource.
func (i *instanceTokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: descriptions["main_datasource"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
			},
			"token_id": schema.StringAttribute{
				Description: descriptions["token_id"],
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				Computed:    true,
				ElementType: types.StringType,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Computed:    true,
			},
			"valid_until": schema.StringAttribute{
				Description: descriptions["valid_until"],
				Computed:    true,
			},
		},
	}
}

// Read is called when the provider must read data source values in
// order to update state. Config values should be read from the
// ReadRequest and new state values set on the ReadResponse.
func (i *instanceTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model InstanceTokenDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := i.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	tokenId := model.TokenId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)

	getInstanceTokenResp, err := i.client.GetInstanceToken(ctx, projectId, region, tokenId, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading AI Model Experiments instance token", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapDataSourceFields(ctx, &getInstanceTokenResp.Token, &model, region, instanceId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating AI Model Experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "AI Model Experiments instance token read")
}

func mapDataSourceFields(ctx context.Context, token *modelexperiments.TokenMetadata, model *InstanceTokenDataSourceModel, region string, instanceId string) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if token.Id == "" {
		return fmt.Errorf("token id not present")
	}

	mapValue, err := utils.MapLabels(ctx, token.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instanceId, token.Id)
	model.TokenId = types.StringValue(token.Id)
	model.Name = types.StringValue(token.Name)
	model.Description = types.StringPointerValue(token.Description)
	model.ValidUntil = types.StringValue(token.ValidUntil.Format(time.RFC3339))
	model.Labels = mapValue

	return nil
}
