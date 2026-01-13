package certcheck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	observabilityUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &certCheckDataSource{}
)

// NewCertCheckDataSource creates a new instance of the certCheckDataSource.
func NewCertCheckDataSource() datasource.DataSource {
	return &certCheckDataSource{}
}

// certCheckDataSource is the datasource implementation.
type certCheckDataSource struct {
	client *observability.APIClient
}

// Configure adds the provider configured client to the resource.
func (d *certCheckDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_observability_cert_check", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	d.client = observabilityUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability client configured")
}

// Metadata provides metadata for the alert group datasource.
func (d *certCheckDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_cert_check"
}

// Schema defines the schema for the alert group data source.
func (d *certCheckDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Datasource for managing cert-checks in STACKIT Observability. It ships Telegraf X509-Cert metrics to the observability instance, as documented here: `https://docs.influxdata.com/telegraf/v1/input-plugins/x509_cert/`",
		Attributes: map[string]schema.Attribute{
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
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"cert_check_id": schema.StringAttribute{
				Description: descriptions["cert_check_id"],
				Required:    true,
			},
			"source": schema.StringAttribute{
				Description: descriptions["source"],
				Computed:    true,
			},
		},
	}
}

func (d *certCheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	certCheckId := model.CertCheckId.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "cert_check_id", certCheckId)

	listCertChecks, err := d.client.ListCertChecks(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing cert-checks", fmt.Sprintf("Listing API payload: %v", err))
		return
	}

	if listCertChecks.CertChecks == nil || len(*listCertChecks.CertChecks) == 0 {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing cert-checks", "Response is empty")
		return
	}

	for _, certCheck := range *listCertChecks.CertChecks {
		if certCheck.Id != nil && *certCheck.Id == certCheckId {
			if err := mapFields(ctx, &certCheck, &model); err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading cert-check", "Unable to map cert-check model")
				return
			}
			break
		}
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "cert-check read")
}
