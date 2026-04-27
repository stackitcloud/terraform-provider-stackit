package accesstoken

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/telemetryrouter/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &telemetryRouterAccessTokenDataSource{}
	_ datasource.DataSourceWithConfigure = &telemetryRouterAccessTokenDataSource{}
)

type DataSourceModel struct {
	ID             types.String `tfsdk:"id"` // Required by Terraform
	AccessTokenID  types.String `tfsdk:"access_token_id"`
	InstanceID     types.String `tfsdk:"instance_id"`
	Region         types.String `tfsdk:"region"`
	ProjectID      types.String `tfsdk:"project_id"`
	CreatorID      types.String `tfsdk:"creator_id"`
	Description    types.String `tfsdk:"description"`
	DisplayName    types.String `tfsdk:"display_name"`
	ExpirationTime types.String `tfsdk:"expiration_time"`
	Status         types.String `tfsdk:"status"`
}

func NewTelemetryRouterAccessTokenDataSource() datasource.DataSource {
	return &telemetryRouterAccessTokenDataSource{}
}

type telemetryRouterAccessTokenDataSource struct {
	client       *telemetryrouter.APIClient
	providerData core.ProviderData
}

func (d *telemetryRouterAccessTokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_telemetryrouter_access_token"
}

func (d *telemetryRouterAccessTokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.providerData = providerData

	apiClient := utils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "TelemetryRouter client configured")
}

func (d *telemetryRouterAccessTokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("TelemetryRouter access token data source schema. %s", core.DatasourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
			},
			"access_token_id": schema.StringAttribute{
				Description: schemaDescriptions["access_token_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				// the region cannot be found, so it has to be passed
				Optional: true,
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"creator_id": schema.StringAttribute{
				Description: schemaDescriptions["creator_id"],
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Computed:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"expiration_time": schema.StringAttribute{
				Description: schemaDescriptions["expiration_time"],
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
		},
	}
}

func (d *telemetryRouterAccessTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()
	accessTokenID := model.AccessTokenID.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "access_token_id", accessTokenID)

	accessTokenResponse, err := d.client.DefaultAPI.GetAccessToken(ctx, projectID, region, instanceID, accessTokenID).Execute()
	if err != nil {
		tfutils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading TelemetryRouter access token",
			fmt.Sprintf("Access token with ID %q does not exist in project %q.", accessTokenID, projectID),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectID),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapDataSourceFields(ctx, accessTokenResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryRouter access token", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter access token read", map[string]interface{}{
		"access_token_id": accessTokenID,
	})
}

func mapDataSourceFields(ctx context.Context, accessToken *telemetryrouter.GetAccessTokenResponse, model *DataSourceModel) error {
	if accessToken == nil {
		return fmt.Errorf("access token is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	var accessTokenID string
	if model.AccessTokenID.ValueString() != "" {
		accessTokenID = model.AccessTokenID.ValueString()
	} else if accessToken.Id != "" {
		accessTokenID = accessToken.Id
	} else {
		return fmt.Errorf("access token id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), model.Region.ValueString(), model.InstanceID.ValueString(), accessTokenID)
	model.AccessTokenID = types.StringValue(accessTokenID)
	model.Region = types.StringValue(model.Region.ValueString())
	model.CreatorID = types.StringValue(accessToken.CreatorId)
	if accessToken.Description != nil && *accessToken.Description != "" {
		model.Description = types.StringPointerValue(accessToken.Description)
	}
	model.DisplayName = types.StringValue(accessToken.DisplayName)
	model.Status = types.StringValue(accessToken.Status)

	model.ExpirationTime = types.StringNull()
	if accessToken.HasExpirationTime() && accessToken.ExpirationTime.Get() != nil {
		model.ExpirationTime = types.StringValue(accessToken.ExpirationTime.Get().Format(time.RFC3339))
	}

	return nil
}
