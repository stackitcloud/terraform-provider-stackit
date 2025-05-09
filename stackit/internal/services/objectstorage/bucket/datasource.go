package objectstorage

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	objectstorageUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &bucketDataSource{}
)

// NewBucketDataSource is a helper function to simplify the provider implementation.
func NewBucketDataSource() datasource.DataSource {
	return &bucketDataSource{}
}

// bucketDataSource is the data source implementation.
type bucketDataSource struct {
	client       *objectstorage.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *bucketDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_bucket"
}

// Configure adds the provider configured client to the data source.
func (r *bucketDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := objectstorageUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "ObjectStorage bucket client configured")
}

// Schema defines the schema for the data source.
func (r *bucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                     "ObjectStorage bucket data source schema. Must have a `region` specified in the provider configuration.",
		"id":                       "Terraform's internal data source identifier. It is structured as \"`project_id`,`name`\".",
		"name":                     "The bucket name. It must be DNS conform.",
		"project_id":               "STACKIT Project ID to which the bucket is associated.",
		"url_path_style":           "URL in path style.",
		"url_virtual_hosted_style": "URL in virtual hosted style.",
		"region":                   "The resource region. If not defined, the provider region is used.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
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
			"url_path_style": schema.StringAttribute{
				Computed: true,
			},
			"url_virtual_hosted_style": schema.StringAttribute{
				Computed: true,
			},
			"region": schema.StringAttribute{
				// the region cannot be found automatically, so it has to be passed
				Optional:    true,
				Description: descriptions["region"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *bucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	bucketName := model.Name.ValueString()
	var region string
	if utils.IsUndefined(model.Region) {
		region = r.providerData.GetRegion()
	} else {
		region = model.Region.ValueString()
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", bucketName)
	ctx = tflog.SetField(ctx, "region", region)

	bucketResp, err := r.client.GetBucket(ctx, projectId, region, bucketName).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading bucket",
			fmt.Sprintf("Bucket with name %q does not exist in project %q.", bucketName, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(bucketResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading bucket", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage bucket read")
}
