package access_token

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

// #nosec G101 tokenUrl is a public endpoint, not a hardcoded credential
const tokenUrl = "https://service-account.api.stackit.cloud/token"

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
}

func (e *accessTokenEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	e.serviceAccountKey = providerData.ServiceAccountKey
	e.serviceAccountKeyPath = providerData.ServiceAccountKeyPath
	e.privateKey = providerData.PrivateKey
	e.privateKeyPath = providerData.PrivateKeyPath
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

	serviceAccountKey, diags := loadServiceAccountKey(ctx, e.serviceAccountKey, e.serviceAccountKeyPath)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	privateKey, diags := resolvePrivateKey(ctx, e.privateKey, e.privateKeyPath, serviceAccountKey)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := initKeyFlowClient(ctx, serviceAccountKey, privateKey)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessToken, err := client.GetAccessToken()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Access token generation failed", fmt.Sprintf("Error generating access token: %v", err))
		return
	}

	ctx = tflog.SetField(ctx, "access_token", accessToken)
	model.AccessToken = types.StringValue(accessToken)
	resp.Diagnostics.Append(resp.Result.Set(ctx, model)...)
}

// loadServiceAccountKey loads the service account key based on env vars, or fallback to provider config.
func loadServiceAccountKey(ctx context.Context, cfgValue, cfgPath string) (*clients.ServiceAccountKeyResponse, diag.Diagnostics) {
	var diags diag.Diagnostics

	env := os.Getenv("STACKIT_SERVICE_ACCOUNT_KEY")
	envPath := os.Getenv("STACKIT_SERVICE_ACCOUNT_KEY_PATH")

	var data []byte
	switch {
	case env != "":
		data = []byte(env)
	case envPath != "":
		b, err := os.ReadFile(envPath)
		if err != nil {
			core.LogAndAddError(ctx, &diags, "Failed to read service account key file (env path)", fmt.Sprintf("Error reading key file: %v", err))
			return nil, diags
		}
		data = b
	case cfgValue != "":
		data = []byte(cfgValue)
	case cfgPath != "":
		b, err := os.ReadFile(cfgPath)
		if err != nil {
			core.LogAndAddError(ctx, &diags, "Failed to read service account key file (provider path)", fmt.Sprintf("Error reading key file: %v", err))
			return nil, diags
		}
		data = b
	default:
		core.LogAndAddError(ctx, &diags, "Missing service account key", "Neither STACKIT_SERVICE_ACCOUNT_KEY, STACKIT_SERVICE_ACCOUNT_KEY_PATH, provider value, nor path were provided.")
		return nil, diags
	}

	var key clients.ServiceAccountKeyResponse
	if err := json.Unmarshal(data, &key); err != nil {
		core.LogAndAddError(ctx, &diags, "Failed to parse service account key", fmt.Sprintf("Unmarshal error: %v", err))
		return nil, diags
	}

	return &key, diags
}

// resolvePrivateKey determines the private key value using env, conf, fallbacks.
func resolvePrivateKey(ctx context.Context, cfgValue, cfgPath string, key *clients.ServiceAccountKeyResponse) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	env := os.Getenv("STACKIT_PRIVATE_KEY")
	envPath := os.Getenv("STACKIT_PRIVATE_KEY_PATH")

	switch {
	case env != "":
		return env, diags
	case envPath != "":
		content, err := os.ReadFile(envPath)
		if err != nil {
			core.LogAndAddError(ctx, &diags, "Failed to read private key file (env path)", fmt.Sprintf("Error: %v", err))
			return "", diags
		}
		return string(content), diags
	case cfgValue != "":
		return cfgValue, diags
	case cfgPath != "":
		content, err := os.ReadFile(cfgPath)
		if err != nil {
			core.LogAndAddError(ctx, &diags, "Failed to read private key file (provider path)", fmt.Sprintf("Error: %v", err))
			return "", diags
		}
		return string(content), diags
	case key.Credentials != nil && key.Credentials.PrivateKey != nil:
		return *key.Credentials.PrivateKey, diags
	default:
		core.LogAndAddError(ctx, &diags, "Missing private key", "No private key set via env, provider, or service account credentials.")
		return "", diags
	}
}

// initKeyFlowClient configures and initializes a new KeyFlow client using the key and private key.
func initKeyFlowClient(ctx context.Context, key *clients.ServiceAccountKeyResponse, privateKey string) (*clients.KeyFlow, diag.Diagnostics) {
	var diags diag.Diagnostics

	client := &clients.KeyFlow{}
	cfg := &clients.KeyFlowConfig{
		ServiceAccountKey: key,
		PrivateKey:        privateKey,
		TokenUrl:          tokenUrl,
	}

	if err := client.Init(cfg); err != nil {
		core.LogAndAddError(ctx, &diags, "Failed to initialize KeyFlow", fmt.Sprintf("KeyFlow client init error: %v", err))
		return nil, diags
	}

	return client, diags
}
