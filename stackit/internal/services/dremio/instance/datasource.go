package dremio

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

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	dremioSdk "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi"

	dremioUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dremio/utils"
)

var (
	_ datasource.DataSource              = &instanceDataSource{}
	_ datasource.DataSourceWithConfigure = &instanceDataSource{}
)

type InstanceDataSourceModel struct {
	Model

	// Required Fields
	Authentication *DataSourceAuthenticationModel `tfsdk:"authentication"`
}

type DataSourceAuthenticationModel struct {
	Type         types.String         `tfsdk:"type"`
	AuthorityUrl types.String         `tfsdk:"authority_url"`
	ClientId     types.String         `tfsdk:"client_id"`
	JwtClaims    *JwtClaimsModel      `tfsdk:"jwt_claims"`
	Scope        types.String         `tfsdk:"scope"`
	Parameters   []AuthParameterModel `tfsdk:"parameters"`
	RedirectUrl  types.String         `tfsdk:"redirect_url"`
}

type instanceDataSource struct {
	client *dremioSdk.APIClient
}

func NewInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

// Metadata should return the full name of the data source, such as
// examplecloud_thing.
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

	apiClient := dremioUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Dremio instance client configured for data source")
}

// Schema should return the schema for this data source.
func (d *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                                "Manages a STACKIT Dremio instance.",
		"id":                                  "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`dremio_id`\".",
		"project_id":                          "STACKIT Project ID to which the resource is associated.",
		"instance_id":                         "The Dremio instance ID.",
		"region":                              "The STACKIT region name the resource is located in. If not defined, the provider region is used.",
		"display_name":                        "The display name is a short name chosen by the user to identify the resource.",
		"description":                         "The description is a longer text chosen by the user to provide more context for the resource.",
		"state":                               "The current state of the resource.",
		"error_message":                       "A message describing an actionable error the user can resolve. This field is empty if no such error exists.",
		"endpoints":                           "The available endpoints of the Dremio instance.",
		"endpoints_arrow_flight":              "The arrow flight endpoint of the Dremio instance.",
		"endpoints_catalog":                   "The Apache Iceberg endpoint of the Dremio instance.",
		"endpoints_ui":                        "The UI endpoint of the Dremio instance.",
		"authentication":                      "Dremio instance authentication settings. A change here triggers a Dremio restart and will incur downtime.",
		"authentication_type":                 "Type of authentication (local-only, azuread, oauth).",
		"authentication_authority_url":        "The Issuer location URI, where the OIDC provider configuration can be found.",
		"authentication_client_id":            "The client ID assigned by the Identity Provider.",
		"authentication_scope":                "A list of space-separated scopes. The `openid` scope is always required; other scopes can vary by provider.",
		"authentication_redirect_url":         "The URL where the Dremio instance is hosted. The URL must match the redirect URL set in the Identity Provider.",
		"authentication_jwt_claims":           "Maps fields from the JWT token to fields Dremio requires.",
		"authentication_jwt_claims_user_name": "Mapped user name claim (e.g. email).",
		"authentication_parameters":           "Any additional parameters the Identity Provider requires.",
		"authentication_parameters_name":      "Parameter name.",
		"authentication_parameters_value":     "Parameter value.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
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
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Required:    true,
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Computed:    true,
				Optional:    true,
			},
			"state": schema.StringAttribute{
				Description: descriptions["state"],
				Computed:    true,
			},
			"error_message": schema.StringAttribute{
				Description: descriptions["error_message"],
				Computed:    true,
				Optional:    true,
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
					"authority_url": schema.StringAttribute{
						Description: descriptions["oauth_authority_url"],
						Computed:    true,
						Optional:    true,
					},
					"client_id": schema.StringAttribute{
						Description: descriptions["oauth_client_id"],
						Computed:    true,
						Optional:    true,
					},
					"scope": schema.StringAttribute{
						Description: descriptions["oauth_scope"],
						Computed:    true,
						Optional:    true,
					},
					"redirect_url": schema.StringAttribute{
						Description: descriptions["oauth_redirect_url"],
						Computed:    true,
						Optional:    true,
					},
					"jwt_claims": schema.SingleNestedAttribute{
						Description: descriptions["oauth_jwt_claims"],
						Computed:    true,
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"user_name": schema.StringAttribute{
								Description: descriptions["oauth_jwt_claims_user_name"],
								Computed:    true,
							},
						},
					},
					"parameters": schema.ListNestedAttribute{
						Description: descriptions["oauth_parameters"],
						Computed:    true,
						Optional:    true,
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
	region := model.Region.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	instanceResp, err := d.client.DefaultAPI.GetDremioInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading runner", fmt.Sprintf("Dremio instance with ID %s not found in project %s and region %s", instanceId, projectId, region))
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Dremio instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapDataSourceFields(instanceResp, &model)
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

func mapDataSourceFields(instanceResp *dremioSdk.DremioResponse, model *InstanceDataSourceModel) error {
	if instanceResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	err := mapModelFields(instanceResp, &model.Model)
	if err != nil {
		return fmt.Errorf("failed to map Model fields")
	}
	err = mapDataSourceAuthentication(instanceResp, model)
	if err != nil {
		return fmt.Errorf("failed to map Authentication fields")
	}

	return nil
}

func mapDataSourceAuthentication(instanceResp *dremioSdk.DremioResponse, model *InstanceDataSourceModel) error {
	authResp := instanceResp.Authentication

	authModel := DataSourceAuthenticationModel{}

	authModel.Type = types.StringValue(authResp.Type)

	if instanceResp.Authentication.Type == "local-only" {
		// On local auth we don't need to map IDP fields
		return nil
	}

	if authResp.Type == "azuread" {
		azureADResp := authResp.Azuread
		authModel.AuthorityUrl = types.StringValue(azureADResp.AuthorityUrl)
		authModel.ClientId = types.StringValue(azureADResp.ClientId)
		authModel.RedirectUrl = types.StringPointerValue(azureADResp.RedirectUrl)
	}

	if authResp.Type == "oauth" {
		oauthResp := authResp.Oauth
		authModel.AuthorityUrl = types.StringValue(oauthResp.AuthorityUrl)
		authModel.ClientId = types.StringValue(oauthResp.ClientId)
		authModel.Scope = types.StringPointerValue(oauthResp.Scope)
		authModel.RedirectUrl = types.StringPointerValue(oauthResp.RedirectUrl)
		authModel.JwtClaims = &JwtClaimsModel{
			UserName: types.StringValue(oauthResp.JwtClaims.UserName),
		}

		if len(oauthResp.Parameters) > 0 {
			var params []AuthParameterModel
			for _, p := range oauthResp.Parameters {
				params = append(params, AuthParameterModel{
					Name:  types.StringValue(p.Name),
					Value: types.StringValue(p.Value),
				})
			}
			authModel.Parameters = params
		}
	}

	model.Authentication = &authModel

	return nil
}
