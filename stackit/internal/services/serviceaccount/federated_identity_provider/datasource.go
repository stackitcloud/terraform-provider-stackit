package federated_identity_provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	serviceaccountUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

var (
	_ datasource.DataSource = &serviceAccountFederatedIdentityProviderDatasource{}
)

func NewServiceAccountFederatedIdentityProviderDataSource() datasource.DataSource {
	return &serviceAccountFederatedIdentityProviderDatasource{}
}

type serviceAccountFederatedIdentityProviderDatasource struct {
	client *serviceaccount.APIClient
}

func (r *serviceAccountFederatedIdentityProviderDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_federated_identity_provider"
}

func (r *serviceAccountFederatedIdentityProviderDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: descriptions["id"],
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: descriptions["project_id"],
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"service_account_email": schema.StringAttribute{
				Required:    true,
				Description: descriptions["service_account_email"],
			},
			"federation_id": schema.StringAttribute{
				Description: descriptions["federation_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: descriptions["name"],
			},
			"issuer": schema.StringAttribute{
				Computed:    true,
				Description: descriptions["issuer"],
			},
			"assertions": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"item": schema.StringAttribute{
							Computed:    true,
							Description: descriptions["assertions.item"],
						},
						"operator": schema.StringAttribute{
							Computed:    true,
							Description: descriptions["assertions.operator"],
						},
						"value": schema.StringAttribute{
							Computed:    true,
							Description: descriptions["assertions.value"],
						},
					},
				},
				Computed:    true,
				Description: descriptions["assertions"],
			},
		},
	}
}

func (r *serviceAccountFederatedIdentityProviderDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := serviceaccountUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

func (r *serviceAccountFederatedIdentityProviderDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	federationId := model.FederationId.ValueString()

	apiResp, err := r.client.DefaultAPI.GetFederatedIdentityProvider(ctx, projectId, serviceAccountEmail, federationId).
		Execute()

	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		// due to security purposes, attempting to get access federation for a non-existent Service Account will return 403.
		if ok && oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusForbidden || oapiErr.StatusCode == http.StatusBadRequest {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading federated identity provider", fmt.Sprintf("Calling API: %v", err))
		return
	}

	if err := mapFields(ctx, apiResp, &model, projectId, serviceAccountEmail); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading federated identity provider", fmt.Sprintf("failed to map response to model: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
