package dremio

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	dremioSdk "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi"

	dremioUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dremio/utils"
)

var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

type UserDataSourceModel struct {
	Model
}

type userDataSource struct {
	client       *dremioSdk.APIClient
	providerData core.ProviderData
}

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// Metadata returns the data source type name.
func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dremio_user"
}

// Configure enables provider-level data or clients to be set in the
// provider-defined DataSource type. It is separately executed for each
// ReadDataSource RPC.
func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &providerData, features.DremioExperiment, "stackit_dremio_user", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := dremioUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Dremio user client configured for data source")
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: features.AddExperimentDescription(descriptions["main"], features.DremioExperiment, core.Datasource),
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
			"user_id": schema.StringAttribute{
				Description: descriptions["user_id"],
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Computed:    true,
			},
			"error_message": schema.StringAttribute{
				Description: descriptions["error_message"],
				Optional:    true,
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: descriptions["email"],
				Computed:    true,
			},
			"first_name": schema.StringAttribute{
				Description: descriptions["first_name"],
				Computed:    true,
			},
			"last_name": schema.StringAttribute{
				Description: descriptions["last_name"],
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: descriptions["state"],
				Computed:    true,
			},
		},
	}
}

// Read is called when the provider must read data source values in
// order to update state. Config values should be read from the
// ReadRequest and new state values set on the ReadResponse.
func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// nolint:gocritic // function signature required by Terraform
	var model UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	userId := model.UserId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "user_id", userId)

	userResp, err := d.client.DefaultAPI.GetDremioUser(ctx, projectId, region, instanceId, userId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Error reading Dremio user",
			fmt.Sprintf("Dremio user with ID %q does not exist in project %q and region %q in instance %q", userId, projectId, region, instanceId),
			map[int]string{
				http.StatusNotFound: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(userResp, &model.Model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Dremio user", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Dremio user read")
}
