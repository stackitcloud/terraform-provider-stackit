package dremio

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	dremioSdk "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi"

	dremioUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dremio/utils"
)

var (
	_ datasource.DataSource              = &instanceDataSource{}
	_ datasource.DataSourceWithConfigure = &instanceDataSource{}
)

type InstanceDataSourceModel struct {
	Model

	Authentication *AuthenticationDatasourceModel `tfsdk:"authentication"`
}

// AuthenticationModel maps the nested authentication block.
type AuthenticationDatasourceModel struct {
	// Required Fields
	Type types.String `tfsdk:"type"`

	// Optional Fields
	AzureAD *AzureADDatasourceModel `tfsdk:"azuread"`
	OAuth   *OAuthDatasourceModel   `tfsdk:"oauth"`
}

type AzureADDatasourceModel struct {
	// Required Fields
	AuthorityUrl types.String `tfsdk:"authority_url"`
	ClientId     types.String `tfsdk:"client_id"`

	RedirectUrl types.String `tfsdk:"redirect_url"`
}

type OAuthDatasourceModel struct {
	// Required Fields
	AuthorityUrl types.String    `tfsdk:"authority_url"`
	ClientId     types.String    `tfsdk:"client_id"`
	JwtClaims    *JwtClaimsModel `tfsdk:"jwt_claims"`

	// Optional Fields
	Scope      types.String         `tfsdk:"scope"`
	Parameters []AuthParameterModel `tfsdk:"parameters"`

	// Read-only Fields
	RedirectUrl types.String `tfsdk:"redirect_url"`
}

type instanceDataSource struct {
	client       *dremioSdk.APIClient
	providerData core.ProviderData
}

func NewInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

// Metadata returns the data source type name.
func (d *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dremio_instance"
}

// Configure enables provider-level data or clients to be set in the
// provider-defined DataSource type. It is separately executed for each
// ReadDataSource RPC.
func (d *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &providerData, features.DremioExperiment, "stackit_dremio_instance", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := dremioUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Dremio instance client configured for data source")
}

func mapDatasourceFields(instanceResp *dremioSdk.DremioResponse, model *InstanceDataSourceModel, region string) error {
	if instanceResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	err := mapModelFields(instanceResp, &model.Model, region)
	if err != nil {
		return fmt.Errorf("failed to map Model fields")
	}

	if model.Authentication == nil {
		model.Authentication = new(AuthenticationDatasourceModel)
	}
	err = mapDatasourceAuthentication(instanceResp, model.Authentication)
	if err != nil {
		return fmt.Errorf("failed to map Authentication fields")
	}

	return nil
}

func mapDatasourceAuthentication(instanceResp *dremioSdk.DremioResponse, auth *AuthenticationDatasourceModel) error {
	if instanceResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if auth == nil {
		return fmt.Errorf("auth input is nil")
	}

	auth.Type = types.StringValue(string(instanceResp.Authentication.Type))

	if instanceResp.Authentication.Azuread != nil {
		if auth.AzureAD == nil {
			auth.AzureAD = new(AzureADDatasourceModel)
		}

		azureADResp := instanceResp.Authentication.Azuread
		azureADModel := auth.AzureAD

		azureADModel.AuthorityUrl = types.StringValue(azureADResp.AuthorityUrl)
		azureADModel.ClientId = types.StringValue(azureADResp.ClientId)
		azureADModel.RedirectUrl = types.StringPointerValue(azureADResp.RedirectUrl)
	}

	if instanceResp.Authentication.Oauth != nil {
		if auth.OAuth == nil {
			auth.OAuth = new(OAuthDatasourceModel)
		}
		oauthResp := instanceResp.Authentication.Oauth
		oauthModel := auth.OAuth

		oauthModel.AuthorityUrl = types.StringValue(oauthResp.AuthorityUrl)
		oauthModel.ClientId = types.StringValue(oauthResp.ClientId)
		oauthModel.Scope = types.StringPointerValue(oauthResp.Scope)
		oauthModel.RedirectUrl = types.StringPointerValue(oauthResp.RedirectUrl)
		oauthModel.JwtClaims = &JwtClaimsModel{UserName: types.StringValue(oauthResp.JwtClaims.UserName)}

		if len(oauthResp.Parameters) > 0 {
			var params []AuthParameterModel
			for _, p := range oauthResp.Parameters {
				params = append(params, AuthParameterModel{
					Name:  types.StringValue(p.Name),
					Value: types.StringValue(p.Value),
				})
			}
			oauthModel.Parameters = params
		}
	}

	return nil
}

// Schema should return the schema for this data source.
func (d *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
				Optional:    true,
				Description: descriptions["region"],
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Computed:    true,
			},
			"endpoints": schema.SingleNestedAttribute{
				Description: descriptions["endpoints"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"arrow_flight": schema.StringAttribute{
						Description: descriptions["endpoints_arrow_flight"],
						Computed:    true,
					},
					"catalog": schema.StringAttribute{
						Description: descriptions["endpoints_catalog"],
						Computed:    true,
					},
					"ui": schema.StringAttribute{
						Description: descriptions["endpoints_ui"],
						Computed:    true,
					},
				},
			},
			"authentication": schema.SingleNestedAttribute{
				Description: descriptions["authentication"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: descriptions["authentication_type"],
						Computed:    true,
					},
					"azuread": schema.SingleNestedAttribute{
						Description: descriptions["azuread"],
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"authority_url": schema.StringAttribute{
								Description: descriptions["azuread_authority_url"],
								Computed:    true,
							},
							"client_id": schema.StringAttribute{
								Description: descriptions["azuread_client_id"],
								Computed:    true,
							},
							"redirect_url": schema.StringAttribute{
								Description: descriptions["azuread_redirect_url"],
								Computed:    true,
							},
						},
					},
					"oauth": schema.SingleNestedAttribute{
						Description: descriptions["oauth"],
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"authority_url": schema.StringAttribute{
								Description: descriptions["oauth_authority_url"],
								Computed:    true,
							},
							"client_id": schema.StringAttribute{
								Description: descriptions["oauth_client_id"],
								Computed:    true,
							},
							"scope": schema.StringAttribute{
								Description: descriptions["oauth_scope"],
								Optional:    true,
								Computed:    true,
							},
							"redirect_url": schema.StringAttribute{
								Description: descriptions["oauth_redirect_url"],
								Computed:    true,
							},
							"jwt_claims": schema.SingleNestedAttribute{
								Description: descriptions["oauth_jwt_claims"],
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"user_name": schema.StringAttribute{
										Description: descriptions["oauth_jwt_claims_user_name"],
										Computed:    true,
									},
								},
							},
							"parameters": schema.ListNestedAttribute{
								Description: descriptions["oauth_parameters"],
								Optional:    true,
								Computed:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"name": schema.StringAttribute{
											Description: descriptions["oauth_parameters_name"],
											Computed:    true,
										},
										"value": schema.StringAttribute{
											Description: descriptions["oauth_parameters_value"],
											Computed:    true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Read is called when the provider must read data source values in
// order to update state. Config values should be read from the
// ReadRequest and new state values set on the ReadResponse.
func (d *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// nolint:gocritic // function signature required by Terraform
	var model InstanceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	instanceResp, err := d.client.DefaultAPI.GetDremioInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Error reading Dremio instance",
			fmt.Sprintf("Dremio instance with ID %q does not exist in project %q and region %q", instanceId, projectId, region),
			map[int]string{
				http.StatusNotFound: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapDatasourceFields(instanceResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Dremio instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Dremio instance read")
}
