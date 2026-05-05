package projectlock

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	sfsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sfs/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &projectlockDatasource{}
	_ datasource.DataSourceWithConfigure = &projectlockDatasource{}
)

// NewProjectLockDatasource is a helper function to simplify the provider implementation.
func NewProjectLockDatasource() datasource.DataSource {
	return &projectlockDatasource{}
}

// projectlockDatasource is the resource implementation.
type projectlockDatasource struct {
	client       *sfs.APIClient
	providerData core.ProviderData
}

// Configure adds the provider configured client to the resource.
func (r *projectlockDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := sfsUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "SFS client configured")
}

// Metadata returns the resource type name.
func (r *projectlockDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sfs_project_lock"
}

// Schema defines the schema for the resource.
func (r *projectlockDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":       "SFS project lock resource schema. Must have a `region` specified in the provider configuration. Always use only one project lock per project.",
		"id":         "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`\".",
		"project_id": "STACKIT Project ID to which the project lock is associated.",
		"region":     "The resource region. If not defined, the provider region is used.",
		"lock_id":    "ID of the lock.",
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
			"lock_id": schema.StringAttribute{
				Description: descriptions["lock_id"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectlockDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	projectResp, err := r.client.DefaultAPI.GetLock(ctx, region, projectId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading project lock",
			fmt.Sprintf("Project lock does not exist in project %q.", projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(projectResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project lock", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS project lock read")
}
