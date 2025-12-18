package snapshots

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource              = (*resourcePoolSnapshotDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*resourcePoolSnapshotDataSource)(nil)
)

// datasourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var datasourceBetaCheckDone bool

var snapshotModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"comment":                types.StringType,
		"created_at":             types.StringType,
		"resource_pool_id":       types.StringType,
		"snapshot_name":          types.StringType,
		"logical_size_gigabytes": types.Int64Type,
		"size_gigabytes":         types.Int64Type,
	},
}

type snapshotModel struct {
	Comment              types.String `tfsdk:"comment"`
	CreatedAt            types.String `tfsdk:"created_at"`
	ResourcePoolId       types.String `tfsdk:"resource_pool_id"`
	SnapshotName         types.String `tfsdk:"snapshot_name"`
	SizeGigabytes        types.Int64  `tfsdk:"size_gigabytes"`
	LogicalSizeGigabytes types.Int64  `tfsdk:"logical_size_gigabytes"`
}

type dataSourceModel struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	ResourcePoolId types.String `tfsdk:"resource_pool_id"`
	Region         types.String `tfsdk:"region"`
	Snapshots      types.List   `tfsdk:"snapshots"`
}

type resourcePoolSnapshotDataSource struct {
	client       *sfs.APIClient
	providerData core.ProviderData
}

func NewResourcePoolSnapshotDataSource() datasource.DataSource {
	return &resourcePoolSnapshotDataSource{}
}

// Configure implements datasource.DataSourceWithConfigure.
func (r *resourcePoolSnapshotDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !datasourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &r.providerData, &resp.Diagnostics, "stackit_sfs_resource_pool_snapshot", core.Datasource)
		if resp.Diagnostics.HasError() {
			return
		}
		datasourceBetaCheckDone = true
	}

	var apiClient *sfs.APIClient
	var err error
	if r.providerData.SfsCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "sfs_custom_endpoint", r.providerData.SfsCustomEndpoint)
		apiClient, err = sfs.NewAPIClient(
			config.WithCustomAuth(r.providerData.RoundTripper),
			config.WithEndpoint(r.providerData.SfsCustomEndpoint),
		)
	} else {
		apiClient, err = sfs.NewAPIClient(
			config.WithCustomAuth(r.providerData.RoundTripper),
			config.WithRegion(r.providerData.GetRegion()),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the datasource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "SFS client configured")
}

// Metadata implements datasource.DataSource.
func (r *resourcePoolSnapshotDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sfs_resource_pool_snapshot"
}

// Read implements datasource.DataSource.
func (r *resourcePoolSnapshotDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model dataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	resourcePoolId := model.ResourcePoolId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "resource_pool_id", resourcePoolId)
	ctx = tflog.SetField(ctx, "region", region)

	response, err := r.client.ListResourcePoolSnapshotsExecute(ctx, projectId, region, resourcePoolId)
	if err != nil {
		var openapiError *oapierror.GenericOpenAPIError
		if errors.As(err, &openapiError) {
			if openapiError.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading resource pool snapshot", fmt.Sprintf("Calling API: %v", err))
		return
	}
	// TODO: log traceId

	// Map response body to schema
	err = mapDataSourceFields(ctx, region, response.ResourcePoolSnapshots, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "error reading resource pool snapshot", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SFS resource pool snapshot read")
}

// Schema implements datasource.DataSource.
func (r *resourcePoolSnapshotDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Resource-pool datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription(description, core.Datasource),
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`resource_pool_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the resource pool snapshot is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"resource_pool_id": schema.StringAttribute{
				Description: "Resource pool ID",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "The resource region. Read-only attribute that reflects the provider region.",
			},
			"snapshots": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"comment": schema.StringAttribute{
							Computed:    true,
							Description: "(optional) A comment to add more information about a snapshot",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "creation date of the snapshot",
						},
						"resource_pool_id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of the Resource Pool of the Snapshot",
						},
						"snapshot_name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the Resource Pool Snapshot",
						},
						"size_gigabytes": schema.Int64Attribute{
							Computed:    true,
							Description: "Reflects the actual storage footprint in the backend at snapshot time (e.g. how much storage from the Resource Pool does it use)",
						},
						"logical_size_gigabytes": schema.Int64Attribute{
							Computed:    true,
							Description: "Represents the user-visible data size at the time of the snapshot (e.g. what’s in the snapshot)",
						},
					},
				},
				Computed:    true,
				Description: description,
			},
		},
	}
}

func mapDataSourceFields(ctx context.Context, region string, snapshots *[]sfs.ResourcePoolSnapshot, model *dataSourceModel) error {
	if snapshots == nil {
		return fmt.Errorf("resource pool snapshot empty in response")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	model.Region = types.StringValue(region)
	var resourcePoolId types.String
	if utils.IsUndefined(model.ResourcePoolId) {
		if snapshots == nil || len(*snapshots) == 0 {
			return fmt.Errorf("no resource pool id defined")
		}
		resourcePoolId = types.StringPointerValue((*snapshots)[0].ResourcePoolId)
	} else {
		resourcePoolId = model.ResourcePoolId
	}
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		resourcePoolId.ValueString(),
	)
	model.Snapshots = types.ListNull(snapshotModelType)
	var vals []attr.Value
	for _, snapshot := range *snapshots {
		elem := snapshotModel{
			ResourcePoolId:       types.StringPointerValue(snapshot.ResourcePoolId),
			SnapshotName:         types.StringPointerValue(snapshot.SnapshotName),
			SizeGigabytes:        types.Int64PointerValue(snapshot.SizeGigabytes),
			LogicalSizeGigabytes: types.Int64PointerValue(snapshot.LogicalSizeGigabytes),
		}
		if val := snapshot.Comment; val != nil {
			elem.Comment = types.StringPointerValue(val.Get())
		}
		if val := snapshot.CreatedAt; val != nil {
			elem.CreatedAt = types.StringValue(val.Format(time.RFC3339))
		}
		val, diags := types.ObjectValueFrom(ctx, snapshotModelType.AttrTypes, elem)
		if diags.HasError() {
			return fmt.Errorf("error while converting snapshot list: %v", diags.Errors())
		}
		vals = append(vals, val)
	}

	list, diags := types.ListValueFrom(ctx, snapshotModelType, vals)
	if diags.HasError() {
		return fmt.Errorf("cannot convert snapshot list: %v", diags.Errors())
	}
	model.Snapshots = list

	return nil
}
