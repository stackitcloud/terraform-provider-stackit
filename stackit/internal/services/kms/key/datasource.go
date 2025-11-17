package kms

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ datasource.DataSource = &keyDataSource{}
)

func NewKeyDataSource() datasource.DataSource {
	return &keyDataSource{}
}

type keyDataSource struct {
	client       *kms.APIClient
	providerData core.ProviderData
}

func (k *keyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_key"
}

func (k *keyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	k.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	k.client = kmsUtils.ConfigureClient(ctx, &k.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "KMS client configured")
}

func (k *keyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("KMS Key datasource schema. %s", core.DatasourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"access_scope": schema.StringAttribute{
				Description: fmt.Sprintf("The access scope of the key. Default is `%s`. %s", string(kms.ACCESSSCOPE_PUBLIC), utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedAccessScopeEnumValues)...)),
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"algorithm": schema.StringAttribute{
				Description: fmt.Sprintf("The encryption algorithm that the key will use to encrypt data. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedAlgorithmEnumValues)...)),
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "A user chosen description to distinguish multiple keys",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "The display name to distinguish multiple keys",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`keyring_id`,`key_id`\".",
				Computed:    true,
			},
			"import_only": schema.BoolAttribute{
				Description: "States whether versions can be created or only imported.",
				Computed:    true,
			},
			"key_id": schema.StringAttribute{
				Description: "The ID of the key",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"keyring_id": schema.StringAttribute{
				Description: "The ID of the associated key ring",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"protection": schema.StringAttribute{
				Description: fmt.Sprintf("The underlying system that is responsible for protecting the key material. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedProtectionEnumValues)...)),
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"purpose": schema.StringAttribute{
				Description: fmt.Sprintf("The purpose for which the key will be used. %s", utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(kms.AllowedPurposeEnumValues)...)),
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the key is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
			},
		},
	}
}

func (k *keyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := k.providerData.GetRegionWithOverride(model.Region)
	keyId := model.KeyId.ValueString()

	ctx = tflog.SetField(ctx, "keyring_id", keyRingId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "key_id", keyId)

	keyResponse, err := k.client.GetKey(ctx, projectId, region, keyRingId, keyId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading key",
			fmt.Sprintf("Key with ID %q does not exist in project %q.", keyId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapFields(keyResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key read")
}
