package accesstoken

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/logs"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logs/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &logsAccessTokenDataSource{}
)

func NewLogsAccessTokenDataSource() datasource.DataSource {
	return &logsAccessTokenDataSource{}
}

type logsAccessTokenDataSource struct {
	client       *logs.APIClient
	providerData core.ProviderData
}

func (d *logsAccessTokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logs_access_token"
}

func (d *logsAccessTokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	tflog.Info(ctx, "Logs client configured")
}

func (d *logsAccessTokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Logs access token data source schema.", core.Datasource),
		Description:         fmt.Sprintf("Logs access token data source schema. %s", core.DatasourceRegionFallbackDocstring),
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
			"access_token": schema.StringAttribute{
				Description: schemaDescriptions["access_token"],
				Computed:    true,
				Sensitive:   true,
			},
			"creator": schema.StringAttribute{
				Description: schemaDescriptions["creator"],
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
			"expires": schema.BoolAttribute{
				Description: schemaDescriptions["expires"],
				Computed:    true,
			},
			"valid_until": schema.StringAttribute{
				Description: schemaDescriptions["valid_until"],
				Computed:    true,
			},
			"permissions": schema.ListAttribute{
				Description: schemaDescriptions["permissions"],
				ElementType: types.StringType,
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
			"lifetime": schema.Int64Attribute{
				Description: schemaDescriptions["lifetime"],
				Optional:    true,
			},
		},
	}
}

func (d *logsAccessTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
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

	accessTokenResponse, err := d.client.GetAccessToken(ctx, projectID, region, instanceID, accessTokenID).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Logs access token", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, accessTokenResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Logs access token", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Logs access token read", map[string]interface{}{
		"access_token_id": accessTokenID,
	})
}
