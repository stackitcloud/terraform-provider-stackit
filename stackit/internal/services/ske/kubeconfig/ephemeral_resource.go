package ske

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	skeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

const (
	defaultKubeconfigExpiration = 1800
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ ephemeral.EphemeralResource              = &kubeconfigEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &kubeconfigEphemeralResource{}
)

// NewKubeconfigEphemeralResource is a helper function to simplify the provider implementation.
func NewKubeconfigEphemeralResource() ephemeral.EphemeralResource {
	return &kubeconfigEphemeralResource{}
}

// kubeconfigEphemeralResource is the ephemeral resource implementation.
type kubeconfigEphemeralResource struct {
	client       *ske.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (e *kubeconfigEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_kubeconfig"
}

// Configure adds the provider configured client to the resource.
func (e *kubeconfigEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	ephemeralProviderData, ok := conversion.ParseEphemeralProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	e.providerData = ephemeralProviderData.ProviderData
	e.client = skeUtils.ConfigureClient(ctx, &e.providerData, &resp.Diagnostics)

	tflog.Info(ctx, "SKE kubeconfig client configured")
}

// ephemeralModel is the model for the ephemeral resource.
type ephemeralModel struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	ProjectId   types.String `tfsdk:"project_id"`
	Expiration  types.Int64  `tfsdk:"expiration"`
	Region      types.String `tfsdk:"region"`
	Kubeconfig  types.String `tfsdk:"kube_config"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
}

// Schema defines the schema for the ephemeral resource.
func (e *kubeconfigEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	description := "Ephemeral resource that generates a short-lived SKE kubeconfig. " +
		"A new kubeconfig is generated each time the resource is evaluated, and it remains consistent for the duration of a Terraform operation."

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Description: "Name of the SKE cluster.",
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the cluster is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"expiration": schema.Int64Attribute{
				Description: "Expiration time of the kubeconfig in seconds. Must be between `600` (10m) and `14400` (4h). " +
					"Defaults to `1800` (30m) for optimal security during Terraform operations, which is more restrictive than the API default of `3600` (1h).",
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(600),
					int64validator.AtMost(14400),
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
			},
			"kube_config": schema.StringAttribute{
				Description: "Raw short-lived admin kubeconfig.",
				Computed:    true,
				Sensitive:   true,
			},
			"expires_at": schema.StringAttribute{
				Description: "Timestamp when the kubeconfig expires.",
				Computed:    true,
			},
		},
	}
}

// Open creates the kubeconfig and sets the result.
func (e *kubeconfigEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var model ephemeralModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	clusterName := model.ClusterName.ValueString()
	region := e.providerData.GetRegionWithOverride(model.Region)

	// Kubeconfig only needs to be valid for the duration of the Terraform operation.
	// Defaulted to 1800s (30m) for better security than the API default (3600s).
	expiration := conversion.Int64ValueToPointer(model.Expiration)
	if expiration == nil {
		expiration = new(int64)
		*expiration = defaultKubeconfigExpiration
	}

	kubeconfigResp, err := getKubeconfig(ctx, e.client, projectId, region, clusterName, expiration)

	ctx = core.LogResponse(ctx)

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating kubeconfig", fmt.Sprintf("Calling SKE API: %v", err))
		return
	}

	if kubeconfigResp == nil || kubeconfigResp.Kubeconfig == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating kubeconfig", "API returned an empty response")
		return
	}

	model.Kubeconfig = types.StringPointerValue(kubeconfigResp.Kubeconfig)
	model.ExpiresAt = types.StringValue(kubeconfigResp.ExpirationTimestamp.Format(time.RFC3339))
	model.Region = types.StringValue(region)

	resp.Diagnostics.Append(resp.Result.Set(ctx, model)...)
	tflog.Info(ctx, "SKE kubeconfig opened")
}

// getKubeconfig initializes the API call to generate a new kubeconfig
func getKubeconfig(ctx context.Context, client *ske.APIClient, projectId, region, clusterName string, expiration *int64) (*ske.Kubeconfig, error) {
	var expirationStringPtr *string
	if expiration != nil {
		expirationStringPtr = new(string)
		*expirationStringPtr = strconv.FormatInt(*expiration, 10)
	}

	payload := ske.CreateKubeconfigPayload{
		ExpirationSeconds: expirationStringPtr,
	}

	return client.DefaultAPI.CreateKubeconfig(ctx, projectId, region, clusterName).CreateKubeconfigPayload(payload).Execute()
}
