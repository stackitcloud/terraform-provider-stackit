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
	_ datasource.DataSource = &keyRingDataSource{}
)

func NewKeyRingDataSource() datasource.DataSource {
	return &keyRingDataSource{}
}

type keyRingDataSource struct {
	client       *kms.APIClient
	providerData core.ProviderData
}

func (k *keyRingDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kms_key_ring"
}

func (k *keyRingDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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
	tflog.Info(ctx, "Key ring configured")
}

func (k *keyRingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":         "KMS Key Ring resource schema.",
		"description":  "A user chosen description to distinguish multiple key rings.",
		"display_name": "The display name to distinguish multiple key rings.",
		"key_ring_id":  "An auto generated unique id which identifies the key ring.",
		"id":           "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
		"project_id":   "STACKIT project ID to which the key ring is associated.",
		"region_id":    "The STACKIT region name the key ring is located in.",
	}

	response.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["description"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"key_ring_id": schema.StringAttribute{
				Description: descriptions["key_ring_id"],
				Computed:    false,
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
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
			"region": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
			},
		},
	}
}

func (k *keyRingDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model

	diags := request.Config.Get(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	keyRingId := model.KeyRingId.ValueString()
	region := k.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "key_ring_id", keyRingId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	keyRingResponse, err := k.client.GetKeyRing(ctx, projectId, region, keyRingId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&response.Diagnostics,
			err,
			"Reading key ring",
			fmt.Sprintf("Key ring with ID %q does not exist in project %q.", keyRingId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		response.State.RemoveResource(ctx)
		return
	}

	err = mapFields(keyRingResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &response.Diagnostics, "Error reading key ring", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = response.State.Set(ctx, &model)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key ring read")
}
