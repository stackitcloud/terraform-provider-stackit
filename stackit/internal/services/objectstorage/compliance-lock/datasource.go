package compliancelock

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	objectstorageUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &compliancelockDataSource{}
)

// NewComplianceLockDataSource is a helper function to simplify the provider implementation.
func NewComplianceLockDataSource() datasource.DataSource {
	return &compliancelockDataSource{}
}

// compliancelockDataSource is the data source implementation.
type compliancelockDataSource struct {
	client       *objectstorage.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (d *compliancelockDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_compliance_lock"
}

// Configure adds the provider configured client to the data source.
func (d *compliancelockDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := objectstorageUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "ObjectStorage compliance lock client configured")
}

// Schema defines the schema for the resource.
func (d *compliancelockDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":               "ObjectStorage compliance lock resource schema. Must have a `region` specified in the provider configuration.",
		"id":                 "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`\".",
		"project_id":         "STACKIT Project ID to which the compliance lock is associated.",
		"region":             "The resource region. If not defined, the provider region is used.",
		"max_retention_days": "Maximum retention period in days.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
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
			"max_retention_days": schema.Int64Attribute{
				Description: descriptions["max_retention_days"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// the region cannot be found automatically, so it has to be passed
				Description: descriptions["region"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *compliancelockDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	complianceResp, err := d.client.GetComplianceLock(ctx, projectId, region).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading compliance lock",
			fmt.Sprintf("Compliance lock does not exist in project %q.", projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(complianceResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading compliance lock", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage compliance lock read")
}
