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

func (k *keyDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kms_key"
}

func (k *keyDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	var ok bool
	k.providerData, ok = conversion.ParseProviderData(ctx, request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	apiClient := kmsUtils.ConfigureClient(ctx, &k.providerData, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	k.client = apiClient
	tflog.Info(ctx, "Key configured")
}

func (k *keyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":         "KMS Key resource schema. Must have a `region` specified in the provider configuration.",
		"backend":      "The backend that is used for KMS. Right now, only software is accepted.",
		"algorithm":    "The encryption algorithm that the key will use to encrypt data",
		"description":  "A user chosen description to distinguish multiple keys",
		"display_name": "The display name to distinguish multiple keys",
		"id":           "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
		"import_only":  "Specifies if the the key should be import_only",
		"key_ring_id":  "The ID of the associated key ring",
		"purpose":      "The purpose for which the key will be used",
		"project_id":   "STACKIT project ID to which the key ring is associated.",
		"region_id":    "The STACKIT region name the key ring is located in.",
	}

	response.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"algorithm": schema.StringAttribute{
				Description: descriptions["algorithm"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"backend": schema.StringAttribute{
				Description: descriptions["backend"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"import_only": schema.BoolAttribute{
				Description: descriptions["id"],
				Computed:    false,
				Required:    true,
			},
			"key_id": schema.StringAttribute{
				Description: descriptions["key_id"],
				Computed:    false,
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"key_ring_id": schema.StringAttribute{
				Description: descriptions["key_ring_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"purpose": schema.StringAttribute{
				Description: descriptions["purpose"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
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
		},
	}
}

func (k *keyDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := request.Config.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := k.providerData.GetRegionWithOverride(model.Region)
	keyId := model.KeyId.ValueString()

	ctx = tflog.SetField(ctx, "key_ring_id", keyRingId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "key_id", keyId)

	keyResponse, err := k.client.GetKey(ctx, projectId, region, keyRingId, keyId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&response.Diagnostics,
			err,
			"Reading key",
			fmt.Sprintf("Key with ID %q does not exist in project %q.", keyId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		response.State.RemoveResource(ctx)
		return
	}

	err = mapFields(keyResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = response.State.Set(ctx, model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key read")
}
