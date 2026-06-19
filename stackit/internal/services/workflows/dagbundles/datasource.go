package dagbundles

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	workflowsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/workflows/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var _ datasource.DataSource = &dagBundlesDataSource{}

type Model struct {
	ID         types.String `tfsdk:"id"`
	ProjectID  types.String `tfsdk:"project_id"`
	Region     types.String `tfsdk:"region"`
	InstanceID types.String `tfsdk:"instance_id"`
	DagBundles types.List   `tfsdk:"dag_bundles"`
}

type bundleSummary struct {
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
	RefreshInterval types.Int32  `tfsdk:"refresh_interval"`
	URL             types.String `tfsdk:"url"`
	Branch          types.String `tfsdk:"branch"`
	Subdir          types.String `tfsdk:"subdir"`
	BucketName      types.String `tfsdk:"bucket_name"`
	Endpoint        types.String `tfsdk:"endpoint"`
	Prefix          types.String `tfsdk:"prefix"`
}

var bundleSummaryTypes = map[string]attr.Type{
	"name":             basetypes.StringType{},
	"type":             basetypes.StringType{},
	"refresh_interval": basetypes.Int32Type{},
	"url":              basetypes.StringType{},
	"branch":           basetypes.StringType{},
	"subdir":           basetypes.StringType{},
	"bucket_name":      basetypes.StringType{},
	"endpoint":         basetypes.StringType{},
	"prefix":           basetypes.StringType{},
}

type dagBundlesDataSource struct {
	client       *workflows.APIClient
	providerData core.ProviderData
}

func NewWorkflowsDagBundlesDataSource() datasource.DataSource {
	return &dagBundlesDataSource{}
}

// Metadata returns the data source type name.
func (d *dagBundlesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflows_dag_bundles"
}

// Configure adds the provider configured client to the data source.
func (d *dagBundlesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.providerData = providerData

	features.CheckExperimentEnabled(ctx, &d.providerData, features.WorkflowsExperiment, "stackit_workflows_dag_bundles", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := workflowsUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
}

// Schema defines the schema for the data source.
func (d *dagBundlesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := fmt.Sprintf("Lists all DAG bundles attached to a Workflows instance. %s", core.DatasourceRegionFallbackDocstring)
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.WorkflowsExperiment, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Description: "Terraform's internal data-source ID. It is structured as \"`project_id`,`region`,`instance_id`\".", Computed: true},
			"project_id": schema.StringAttribute{Description: "STACKIT project ID.", Required: true, Validators: []validator.String{validate.UUID(), validate.NoSeparator()}},
			"region":     schema.StringAttribute{Description: "STACKIT region name.", Optional: true, Computed: true},
			"instance_id": schema.StringAttribute{
				Description: "Workflows instance ID.",
				Required:    true,
				Validators:  []validator.String{validate.UUID(), validate.NoSeparator()},
			},
			"dag_bundles": schema.ListNestedAttribute{
				Description: "DAG bundles attached to the instance.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":             schema.StringAttribute{Description: "Bundle name.", Computed: true},
						"type":             schema.StringAttribute{Description: "Bundle source type (`git` or `s3`).", Computed: true},
						"refresh_interval": schema.Int32Attribute{Description: "Refresh interval (seconds).", Computed: true},
						"url":              schema.StringAttribute{Description: "Git repository URL (git bundles only).", Computed: true},
						"branch":           schema.StringAttribute{Description: "Git branch (git bundles only).", Computed: true},
						"subdir":           schema.StringAttribute{Description: "Subdirectory inside the repository (git bundles only).", Computed: true},
						"bucket_name":      schema.StringAttribute{Description: "Bucket name (s3 bundles only).", Computed: true},
						"endpoint":         schema.StringAttribute{Description: "S3 endpoint (s3 bundles only).", Computed: true},
						"prefix":           schema.StringAttribute{Description: "Key prefix (s3 bundles only).", Computed: true},
					},
				},
			},
		},
	}
}

// Read reads the data source and writes its result to Terraform state.
func (d *dagBundlesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)

	listResp, err := d.client.DefaultAPI.ListDagBundles(ctx, projectID, region, instanceID).Execute()
	if err != nil {
		tfutils.LogError(ctx, &resp.Diagnostics, err, "Error listing Workflows DAG bundles", fmt.Sprintf("Instance %q", instanceID), nil)
		return
	}
	ctx = core.LogResponse(ctx)

	model.Region = types.StringValue(region)
	model.ID = types.StringValue(fmt.Sprintf("%s,%s,%s", projectID, region, instanceID))
	objType := types.ObjectType{AttrTypes: bundleSummaryTypes}
	elements := make([]attr.Value, 0, len(listResp.DagBundles))
	for i := range listResp.DagBundles {
		bs := bundleSummary{}
		switch {
		case listResp.DagBundles[i].GitDagBundleResponse != nil:
			g := listResp.DagBundles[i].GitDagBundleResponse
			bs.Type = types.StringValue(workflowsUtils.BundleTypeGit)
			bs.Name = types.StringValue(g.Name)
			bs.URL = types.StringValue(g.Url)
			bs.Branch = types.StringValue(g.Branch)
			bs.Subdir = types.StringPointerValue(g.Subdir)
			bs.RefreshInterval = types.Int32PointerValue(g.RefreshInterval)
		case listResp.DagBundles[i].S3DagBundleResponse != nil:
			s := listResp.DagBundles[i].S3DagBundleResponse
			bs.Type = types.StringValue(workflowsUtils.BundleTypeS3)
			bs.Name = types.StringValue(s.Name)
			bs.BucketName = types.StringValue(s.BucketName)
			bs.Endpoint = types.StringPointerValue(s.Endpoint)
			bs.Prefix = types.StringPointerValue(s.Prefix)
			bs.RefreshInterval = types.Int32PointerValue(s.RefreshInterval)
		default:
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing Workflows DAG bundles", fmt.Sprintf("Unknown bundle variant at index %d", i))
			return
		}
		obj, diags := types.ObjectValueFrom(ctx, bundleSummaryTypes, bs)
		if diags.HasError() {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing Workflows DAG bundles", fmt.Sprintf("Mapping bundle at index %d: %v", i, diags.Errors()))
			return
		}
		elements = append(elements, obj)
	}
	list, diags := types.ListValue(objType, elements)
	if diags.HasError() {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing Workflows DAG bundles", fmt.Sprintf("Building list: %v", diags.Errors()))
		return
	}
	model.DagBundles = list

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Info(ctx, "Workflows DAG bundles listed", map[string]any{"count": len(elements)})
}
