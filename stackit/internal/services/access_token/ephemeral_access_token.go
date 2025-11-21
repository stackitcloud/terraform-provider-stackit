package access_token

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ ephemeral.EphemeralResource              = &accessTokenEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &accessTokenEphemeralResource{}
)

func NewAccessTokenEphemeralResource() ephemeral.EphemeralResource {
	return &accessTokenEphemeralResource{}
}

type accessTokenEphemeralResource struct{}

func (e *accessTokenEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		tflog.Info(ctx, "provider data is nil (not okay)")
	}

	tflog.Info(ctx, fmt.Sprintf("providerdata %s", req.ProviderData))
}

type ephemeralTokenModel struct {
	AccessToken types.String `tfsdk:"access_token"`
}

func (e *accessTokenEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_token"
}

func (e *accessTokenEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "",
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				Description: "",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (e *accessTokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var model ephemeralTokenModel

	generatedAccessToken := uuid.NewString()

	ctx = tflog.SetField(ctx, "access_token", generatedAccessToken)

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Access token generated")

	model.AccessToken = types.StringValue(generatedAccessToken)

	resp.Diagnostics.Append(resp.Result.Set(ctx, model)...)
}
