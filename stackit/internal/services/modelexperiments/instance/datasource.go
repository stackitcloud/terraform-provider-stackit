package instance

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	modelexperimentsutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/utils"
)

var (
	_ datasource.DataSource              = &instanceDataSource{}
	_ datasource.DataSourceWithConfigure = &instanceDataSource{}
)

type InstanceDataSourceModel struct {
	Model
}

func NewInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

type instanceDataSource struct {
	client       modelexperiments.DefaultAPI
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (i *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_modelexperiments_instance"
}

// Configure enables provider-level data or clients to be set in the
// provider-defined DataSource type. It is separately executed for each
// ReadDataSource RPC.
func (i *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	tflog.Info(ctx, "Dremio instance client configured for data source")
}

// Schema defines the schema for the resource.
func (i *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: descriptions["url"],
				Computed:    true,
			},
			"deleted_experiment_retention": schema.StringAttribute{
				Description: descriptions["deleted_experiment_retention"],
				Computed:    true,
			},
			"bucket_name": schema.StringAttribute{
				Description: descriptions["bucket_name"],
				Computed:    true,
			},
		},
	}
}

// Read is called when the provider must read data source values in
// order to update state. Config values should be read from the
// ReadRequest and new state values set on the ReadResponse.
func (i *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// nolint:gocritic // function signature required by Terraform
	var model InstanceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := i.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	getInstanceResp, err := i.client.GetInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading AI Model Experiments instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapInstance(ctx, &getInstanceResp.Instance, &model.Model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading AI Model Experiments instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "AI Model Experiments instance read")
}
