package token

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/modelserving"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &tokenDataSource{}
)

// NewTokenDataSource is a helper function to simplify the provider implementation.
func NewTokenDataSource() datasource.DataSource {
	return &tokenDataSource{}
}

// tokenDataSource is the data source implementation.
type tokenDataSource struct {
	client *modelserving.APIClient
}

// Metadata returns the data source type name.
func (d *tokenDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_model_serving_token"
}

// Configure adds the provider configured client to the data source.
func (d *tokenDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error configuring API client",
			fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData),
		)
		return
	}

	var apiClient *modelserving.APIClient
	var err error
	if providerData.DnsCustomEndpoint != "" {
		apiClient, err = modelserving.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.DnsCustomEndpoint),
		)
	} else {
		apiClient, err = modelserving.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error configuring API client",
			fmt.Sprintf(
				"Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration",
				err,
			),
		)
		return
	}

	d.client = apiClient

	tflog.Info(ctx, "Model-Serving auth token client configured")
}

// Schema defines the schema for the data source.
func (d *tokenDataSource) Schema(
	_ context.Context,
	_ datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Model Serving Auth Token datasource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`token_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the model serving auth token is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "STACKIT region to which the model serving auth token is associated.",
				Required:    true,
			},
			"token_id": schema.StringAttribute{
				Description: "The model serving auth token ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the model serving auth token.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the model serving auth token.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "State of the model serving auth token.",
				Computed:    true,
			},
			"content": schema.StringAttribute{
				Description: "Content of the model serving auth token.",
				Computed:    true,
			},
			"validUntil": schema.StringAttribute{
				Description: "The time until the model serving auth token is valid.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *tokenDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest, //nolint:gocritic // function signature required by Terraform
	resp *datasource.ReadResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model Model

	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	tokenId := model.TokenId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "token_id", tokenId)
	ctx = tflog.SetField(ctx, "region", region)

	getTokenResp, err := d.client.GetToken(ctx, region, projectId, tokenId).Execute()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading model serving auth token",
			fmt.Sprintf("Calling API: %v", err),
		)
		return
	}

	if getTokenResp != nil && getTokenResp.Token.State != nil &&
		*getTokenResp.Token.State == inactiveState {
		resp.State.RemoveResource(ctx)
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading model serving auth token",
			"Model serving auth token has expired",
		)
		return
	}

	err = mapGetResponse(getTokenResp, &model)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading model serving auth token",
			fmt.Sprintf("Processing API payload: %v", err),
		)
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Model-Serving auth token read")
}
