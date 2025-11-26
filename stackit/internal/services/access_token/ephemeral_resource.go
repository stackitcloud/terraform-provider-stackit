package access_token

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/auth"
	"github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
)

var (
	_ ephemeral.EphemeralResource              = &accessTokenEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &accessTokenEphemeralResource{}
)

func NewAccessTokenEphemeralResource() ephemeral.EphemeralResource {
	return &accessTokenEphemeralResource{}
}

type accessTokenEphemeralResource struct {
	keyAuthConfig config.Configuration
}

func (e *accessTokenEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	ephemeralProviderData, ok := conversion.ParseEphemeralProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(
		ctx,
		&core.ProviderData{EnableBetaResources: ephemeralProviderData.EnableBetaResources},
		&resp.Diagnostics,
		"stackit_access_token", "ephemeral_resource",
	)
	if resp.Diagnostics.HasError() {
		return
	}

	e.keyAuthConfig = config.Configuration{
		ServiceAccountKey:     ephemeralProviderData.ServiceAccountKey,
		ServiceAccountKeyPath: ephemeralProviderData.ServiceAccountKeyPath,
		PrivateKeyPath:        ephemeralProviderData.PrivateKey,
		PrivateKey:            ephemeralProviderData.PrivateKeyPath,
		TokenCustomUrl:        ephemeralProviderData.TokenCustomEndpoint,
	}
}

type ephemeralTokenModel struct {
	AccessToken types.String `tfsdk:"access_token"`
}

func (e *accessTokenEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_token"
}

func (e *accessTokenEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: features.AddBetaDescription("Ephemeral resource that generates a short-lived STACKIT access token (JWT) using a service account key. "+
			"A new token is generated each time the resource is evaluated, and it remains consistent for the duration of a Terraform operation. "+
			"If a private key is not explicitly provided, the provider attempts to extract it from the service account key instead. "+
			"Token generation logic prioritizes environment variables first, followed by provider configuration. "+
			"Access tokens generated from service account keys expire after 60 minutes.", core.EphemeralResource),
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				Description: "JWT access token for STACKIT API authentication.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (e *accessTokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var model ephemeralTokenModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessToken, err := getAccessToken(&e.keyAuthConfig)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Access token generation failed", err.Error())
		return
	}

	model.AccessToken = types.StringValue(accessToken)
	resp.Diagnostics.Append(resp.Result.Set(ctx, model)...)
}

// getAccessToken initializes authentication using the provided config and returns an access token via the KeyFlow mechanism.
func getAccessToken(keyAuthConfig *config.Configuration) (string, error) {
	roundTripper, err := auth.KeyAuth(keyAuthConfig)
	if err != nil {
		return "", fmt.Errorf("failed to initialize authentication: %w", err)
	}

	// Type assert to access token functionality
	client, ok := roundTripper.(*clients.KeyFlow)
	if !ok {
		return "", fmt.Errorf("internal error: expected *clients.KeyFlow, but received a different implementation of http.RoundTripper")
	}

	// Retrieve the access token
	accessToken, err := client.GetAccessToken()
	if err != nil {
		return "", fmt.Errorf("error obtaining access token: %w", err)
	}

	return accessToken, nil
}
