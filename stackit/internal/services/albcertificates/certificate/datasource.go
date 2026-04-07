package certificate

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	certSdk "github.com/stackitcloud/stackit-sdk-go/services/certificates/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	certUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/albcertificates/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &certDataSource{}
)

// NewCertificatesDataSource is a helper function to simplify the provider implementation.
func NewCertificatesDataSource() datasource.DataSource {
	return &certDataSource{}
}

// certDataSource is the data source implementation.
type certDataSource struct {
	client       *certSdk.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *certDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_certificate"
}

// Configure adds the provider configured client to the data source.
func (r *certDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := certUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Certificate client configured")
}

// Schema defines the schema for the resource.
func (r *certDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "Certificates resource schema.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`cert_id`\".",
		"project_id":  "STACKIT project ID to which the certificate is associated.",
		"region":      "The resource region (e.g. eu01). If not defined, the provider region is used.",
		"cert_id":     "The ID of the certificate.",
		"name":        "Certificate name.",
		"private_key": "The PEM encoded private key part",
		"public_key":  "The PEM encoded public key part",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		MarkdownDescription: `ALB TLS Certificate data source schema. Must have a region specified in the provider configuration.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"cert_id": schema.StringAttribute{
				Description: descriptions["cert_id"],
				Required:    true,
			},
			"public_key": schema.StringAttribute{
				Description: descriptions["public_key"],
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *certDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	certId := model.CertID.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "cert_id", certId)

	certResp, err := r.client.DefaultAPI.GetCertificate(ctx, projectId, region, certId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading certificate",
			fmt.Sprintf("Certificate with ID %q does not exist in project %q.", certId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapDataFields(certResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading certificate", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Certificate read")
}
