package objectstorage

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	objectstorageUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSourceWithConfigure = &defaultRetentionDataSource{}
)

func NewDefaultRetentionDataSource() datasource.DataSource {
	return &defaultRetentionDataSource{}
}

type defaultRetentionDataSource struct {
	client       *objectstorage.APIClient
	providerData core.ProviderData
}

// Configure implements [datasource.DataSourceWithConfigure].
func (r *defaultRetentionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	tflog.Info(ctx, "Application Load Balancer client configured")
}

// Schema implements [datasource.DataSource].
func (d *defaultRetentionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) { // nolint:gocritic
	descriptions := map[string]string{
		"main":        "ObjectStorage default-retention resource schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`bucket_name`\".",
		"bucket_name": "The associated bucket's name. It must be DNS conform.",
		"project_id":  "STACKIT Project ID to which the default-retention is associated.",
		"region":      "The resource region. If not defined, the provider region is used.",
		"days":        "The number retention period in days.",
		"mode":        "The retention mode for default retention on a bucket.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"bucket_name": schema.StringAttribute{
				Description: descriptions["bucket_name"],
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
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
			},
			"days": schema.Int32Attribute{
				Required:    true,
				Description: descriptions["days"],
			},
			"mode": schema.StringAttribute{
				Required:    true,
				Description: descriptions["mode"],
				Validators: []validator.String{
					stringvalidator.OneOf(sdkUtils.EnumSliceToStringSlice(objectstorage.AllowedRetentionModeEnumValues)...),
					validate.NoSeparator(),
				},
			},
		},
	}
}

// Metadata implements [datasource.DataSource].
func (d *defaultRetentionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_default_retention"
}

// Read implements [datasource.DataSource].
func (d *defaultRetentionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic
	var model model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	bucketName := model.BucketName.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bucket_name", bucketName)
	ctx = tflog.SetField(ctx, "region", region)

	// Read default-retention
	result, err := d.client.DefaultAPI.GetDefaultRetention(ctx, projectId, region, bucketName).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading default-retention", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(result, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading default-retention", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage default-retention read")
}
