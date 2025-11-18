package kms

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	kmsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/kms/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &wrappingKeyDataSource{}
)

func NewWrappingKeyDataSource() datasource.DataSource {
	return &wrappingKeyDataSource{}
}

type wrappingKeyDataSource struct {
	client       *kms.APIClient
	providerData core.ProviderData
}

func (w *wrappingKeyDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kms_wrapping_key"
}

func (w *wrappingKeyDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	var ok bool
	w.providerData, ok = conversion.ParseProviderData(ctx, request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	w.client = kmsUtils.ConfigureClient(ctx, &w.providerData, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "KMS client configured")
}

func (w *wrappingKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "KMS wrapping key datasource schema.",
		Attributes: map[string]schema.Attribute{
			"access_scope": schema.StringAttribute{
				Description: fmt.Sprintf("The access scope of the key. Default is `%s`. %s", string(kms.ACCESSSCOPE_PUBLIC), utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedAccessScopeEnumValues)...)),
				Computed:    true,
			},
			"algorithm": schema.StringAttribute{
				Description: fmt.Sprintf("The wrapping algorithm used to wrap the key to import. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedWrappingAlgorithmEnumValues)...)),
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "A user chosen description to distinguish multiple wrapping keys.",
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "The display name to distinguish multiple wrapping keys.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`keyring_id`,`wrapping_key_id`\".",
				Computed:    true,
			},
			"keyring_id": schema.StringAttribute{
				Description: "The ID of the associated keyring",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"protection": schema.StringAttribute{
				Description: fmt.Sprintf("The underlying system that is responsible for protecting the key material. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedProtectionEnumValues)...)),
				Computed:    true,
			},
			"purpose": schema.StringAttribute{
				Description: fmt.Sprintf("The purpose for which the key will be used. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedWrappingPurposeEnumValues)...)),
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the keyring is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
			},
			"wrapping_key_id": schema.StringAttribute{
				Description: "The ID of the wrapping key",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"public_key": schema.StringAttribute{
				Description: "The public key of the wrapping key.",
				Computed:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "The date and time the wrapping key will expire.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The date and time the creation of the wrapping key was triggered.",
				Computed:    true,
			},
		},
	}
}

func (w *wrappingKeyDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := request.Config.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := w.providerData.GetRegionWithOverride(model.Region)
	wrappingKeyId := model.WrappingKeyId.ValueString()

	ctx = tflog.SetField(ctx, "keyring_id", keyRingId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "wrapping_key_id", wrappingKeyId)

	wrappingKeyResponse, err := w.client.GetWrappingKey(ctx, projectId, region, keyRingId, wrappingKeyId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&response.Diagnostics,
			err,
			"Reading wrapping key",
			fmt.Sprintf("Wrapping key with ID %q does not exist in project %q.", wrappingKeyId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		response.State.RemoveResource(ctx)
		return
	}

	err = mapFields(wrappingKeyResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading wrapping key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Wrapping key read")
}
