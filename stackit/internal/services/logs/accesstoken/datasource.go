package accesstoken

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/logs"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/logs/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = &logsAccessTokenDataSource{}
	_ datasource.DataSourceWithConfigure = &logsAccessTokenDataSource{}
)

type DataSourceModel struct {
	ID            types.String `tfsdk:"id"` // Required by Terraform
	AccessTokenID types.String `tfsdk:"access_token_id"`
	InstanceID    types.String `tfsdk:"instance_id"`
	Region        types.String `tfsdk:"region"`
	ProjectID     types.String `tfsdk:"project_id"`
	Creator       types.String `tfsdk:"creator"`
	Description   types.String `tfsdk:"description"`
	DisplayName   types.String `tfsdk:"display_name"`
	Expires       types.Bool   `tfsdk:"expires"`
	ValidUntil    types.String `tfsdk:"valid_until"`
	Permissions   types.List   `tfsdk:"permissions"`
	Status        types.String `tfsdk:"status"`
}

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
		Description: fmt.Sprintf("Logs access token data source schema. %s", core.DatasourceRegionFallbackDocstring),
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
		},
	}
}

func (d *logsAccessTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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

	accessTokenResponse, err := d.client.GetAccessToken(ctx, projectID, region, instanceID, accessTokenID).Execute()
	if err != nil {
		tfutils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading Logs access token",
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

func mapDataSourceFields(ctx context.Context, accessToken *logs.AccessToken, model *DataSourceModel) error {
	if accessToken == nil {
		return fmt.Errorf("access token is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	var accessTokenID string
	if model.AccessTokenID.ValueString() != "" {
		accessTokenID = model.AccessTokenID.ValueString()
	} else if accessToken.Id != nil {
		accessTokenID = *accessToken.Id
	} else {
		return fmt.Errorf("access token id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), model.Region.ValueString(), model.InstanceID.ValueString(), accessTokenID)
	model.AccessTokenID = types.StringValue(accessTokenID)
	model.Region = types.StringValue(model.Region.ValueString())
	model.Creator = types.StringPointerValue(accessToken.Creator)
	model.Description = types.StringPointerValue(accessToken.Description)
	model.DisplayName = types.StringPointerValue(accessToken.DisplayName)
	model.Expires = types.BoolPointerValue(accessToken.Expires)
	model.Status = types.StringValue(string(*accessToken.Status))

	model.ValidUntil = types.StringNull()
	if accessToken.ValidUntil != nil {
		model.ValidUntil = types.StringValue(accessToken.ValidUntil.Format(time.RFC3339))
	}

	permissionList := types.ListNull(types.StringType)
	var diags diag.Diagnostics
	if accessToken.Permissions != nil && len(*accessToken.Permissions) > 0 {
		permissionList, diags = types.ListValueFrom(ctx, types.StringType, accessToken.Permissions)
		if diags.HasError() {
			return fmt.Errorf("mapping permissions: %w", core.DiagsToError(diags))
		}
	}
	model.Permissions = permissionList

	return nil
}
