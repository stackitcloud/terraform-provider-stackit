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
)

var (
	_ ephemeral.EphemeralResource              = &accessTokenEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &accessTokenEphemeralResource{}
)

func NewAccessTokenEphemeralResource() ephemeral.EphemeralResource {
	return &accessTokenEphemeralResource{}
}

type accessTokenEphemeralResource struct {
	serviceAccountKeyPath string
	serviceAccountKey     string
	privateKeyPath        string
	privateKey            string
	tokenCustomEndpoint   string
}

func (e *accessTokenEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	providerData, ok := conversion.ParseEphemeralProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	e.serviceAccountKey = providerData.ServiceAccountKey
	e.serviceAccountKeyPath = providerData.ServiceAccountKeyPath
	e.privateKey = providerData.PrivateKey
	e.privateKeyPath = providerData.PrivateKeyPath
	e.tokenCustomEndpoint = providerData.TokenCustomEndpoint
}

type ephemeralTokenModel struct {
	AccessToken types.String `tfsdk:"access_token"`
}

func (e *accessTokenEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_token"
}

func (e *accessTokenEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "STACKIT Access Token ephemeral resource schema.",
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

	cfg := config.Configuration{
		ServiceAccountKey:     e.serviceAccountKey,
		ServiceAccountKeyPath: e.serviceAccountKeyPath,
		PrivateKeyPath:        e.privateKeyPath,
		PrivateKey:            e.privateKey,
		TokenCustomUrl:        e.tokenCustomEndpoint,
	}

	rt, err := auth.KeyAuth(&cfg)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Access token generation failed", fmt.Sprintf("Failed to initialize authentication: %v", err))
		return
	}

	// Type assert to access token functionality
	client, ok := rt.(*clients.KeyFlow)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Access token generation failed", "Internal error: expected *clients.KeyFlow, but received a different implementation of http.RoundTripper")
		return
	}

	// Retrieve the access token
	accessToken, err := client.GetAccessToken()
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Access token retrieval failed",
			fmt.Sprintf("Error obtaining access token: %v", err),
		)
		return
	}

	model.AccessToken = types.StringValue(accessToken)
	resp.Diagnostics.Append(resp.Result.Set(ctx, model)...)
}
