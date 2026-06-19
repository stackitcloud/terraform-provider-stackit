package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	workflowsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/workflows/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var _ datasource.DataSource = &instanceDataSource{}

func NewWorkflowsInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

type instanceDataSource struct {
	client       *workflows.APIClient
	providerData core.ProviderData
}

// datasourceModel is the resource Model plus the embedded dag_bundles read-only
// list. The bundles list lives only on the datasource because individual
// bundles are managed by the stackit_workflows_dag_bundle resource — exposing
// the list on the parent resource would create a two-writer state model.
type datasourceModel struct {
	ID                       types.String `tfsdk:"id"`
	InstanceID               types.String `tfsdk:"instance_id"`
	Region                   types.String `tfsdk:"region"`
	ProjectID                types.String `tfsdk:"project_id"`
	DisplayName              types.String `tfsdk:"display_name"`
	Description              types.String `tfsdk:"description"`
	Version                  types.String `tfsdk:"version"`
	EnableStackitExampleDags types.Bool   `tfsdk:"enable_stackit_example_dags"`
	EnableAirflowExampleDags types.Bool   `tfsdk:"enable_airflow_example_dags"`
	ObservabilityID          types.String `tfsdk:"observability_id"`
	Network                  types.Object `tfsdk:"network"`
	IdentityProvider         types.Object `tfsdk:"identity_provider"`
	Endpoints                types.Object `tfsdk:"endpoints"`
	DagBundles               types.List   `tfsdk:"dag_bundles"`
	Status                   types.String `tfsdk:"status"`
	StatusMessage            types.String `tfsdk:"status_message"`
	CreatedAt                types.String `tfsdk:"created_at"`
}

type dagBundleSummary struct {
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

var dagBundleSummaryTypes = map[string]attr.Type{
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

func (d *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflows_instance"
}

func (d *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.providerData = providerData

	features.CheckExperimentEnabled(ctx, &d.providerData, features.WorkflowsExperiment, "stackit_workflows_instance", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := workflowsUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
}

func (d *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := fmt.Sprintf("Workflows instance data source schema. %s", core.DatasourceRegionFallbackDocstring)
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.WorkflowsExperiment, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Optional:    true,
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: schemaDescriptions["version"],
				Computed:    true,
			},
			"enable_stackit_example_dags": schema.BoolAttribute{
				Description: schemaDescriptions["enable_stackit_example_dags"],
				Computed:    true,
			},
			"enable_airflow_example_dags": schema.BoolAttribute{
				Description: schemaDescriptions["enable_airflow_example_dags"],
				Computed:    true,
			},
			"observability_id": schema.StringAttribute{
				Description: schemaDescriptions["observability_id"],
				Computed:    true,
			},
			"network": schema.SingleNestedAttribute{
				Description: schemaDescriptions["network"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: schemaDescriptions["network.id"],
						Computed:    true,
					},
				},
			},
			"identity_provider": schema.SingleNestedAttribute{
				Description: schemaDescriptions["identity_provider"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type":               schema.StringAttribute{Description: schemaDescriptions["identity_provider.type"], Computed: true},
					"name":               schema.StringAttribute{Description: schemaDescriptions["identity_provider.name"], Computed: true},
					"client_id":          schema.StringAttribute{Description: schemaDescriptions["identity_provider.client_id"], Computed: true},
					"client_secret":      schema.StringAttribute{Description: "OAuth2 client secret. Never returned by the API; always null when read.", Computed: true, Sensitive: true},
					"scope":              schema.StringAttribute{Description: schemaDescriptions["identity_provider.scope"], Computed: true},
					"discovery_endpoint": schema.StringAttribute{Description: schemaDescriptions["identity_provider.discovery_endpoint"], Computed: true},
					"api_audience":       schema.SetAttribute{Description: schemaDescriptions["identity_provider.api_audience"], Computed: true, ElementType: types.StringType},
					"resource":           schema.StringAttribute{Description: schemaDescriptions["identity_provider.resource"], Computed: true},
					"roles_claim":        schema.StringAttribute{Description: schemaDescriptions["identity_provider.roles_claim"], Computed: true},
				},
			},
			"endpoints": schema.SingleNestedAttribute{
				Description: schemaDescriptions["endpoints"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"url":          schema.StringAttribute{Description: schemaDescriptions["endpoints.url"], Computed: true},
					"redirect_url": schema.StringAttribute{Description: schemaDescriptions["endpoints.redirect_url"], Computed: true},
				},
			},
			"dag_bundles": schema.ListNestedAttribute{
				Description: "DAG bundles attached to this instance. Manage individual bundles via `stackit_workflows_dag_bundle`.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":             schema.StringAttribute{Description: "Bundle name.", Computed: true},
						"type":             schema.StringAttribute{Description: "Bundle type: `git` or `s3`.", Computed: true},
						"refresh_interval": schema.Int32Attribute{Description: "Bundle refresh interval in seconds.", Computed: true},
						"url":              schema.StringAttribute{Description: "Git repository URL (git bundles only).", Computed: true},
						"branch":           schema.StringAttribute{Description: "Git branch (git bundles only).", Computed: true},
						"subdir":           schema.StringAttribute{Description: "Subdirectory inside the Git repository (git bundles only).", Computed: true},
						"bucket_name":      schema.StringAttribute{Description: "S3 bucket name (s3 bundles only).", Computed: true},
						"endpoint":         schema.StringAttribute{Description: "S3 endpoint (s3 bundles only).", Computed: true},
						"prefix":           schema.StringAttribute{Description: "S3 key prefix (s3 bundles only).", Computed: true},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
			"status_message": schema.StringAttribute{
				Description: schemaDescriptions["status_message"],
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: schemaDescriptions["created_at"],
				Computed:    true,
			},
		},
	}
}

func (d *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var dsm datasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &dsm)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := dsm.ProjectID.ValueString()
	region := d.providerData.GetRegionWithOverride(dsm.Region)
	instanceID := dsm.InstanceID.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)

	instance, err := d.client.DefaultAPI.GetInstance(ctx, projectID, region, instanceID).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		tfutils.LogError(
			ctx, &resp.Diagnostics, err,
			"Error reading Workflows instance",
			fmt.Sprintf("Instance with ID %q does not exist in project %q.", instanceID, projectID),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectID),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = core.LogResponse(ctx)

	m := Model{
		InstanceID:       dsm.InstanceID,
		Region:           dsm.Region,
		ProjectID:        dsm.ProjectID,
		IdentityProvider: dsm.IdentityProvider,
	}
	if err := mapFields(ctx, instance, &m, region); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Workflows instance", fmt.Sprintf("Processing response: %v", err))
		return
	}
	dagBundles, err := mapInstanceDagBundles(ctx, instance.DagBundles)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Workflows instance", fmt.Sprintf("Mapping dag_bundles: %v", err))
		return
	}

	dsm.ID = m.ID
	dsm.InstanceID = m.InstanceID
	dsm.Region = m.Region
	dsm.ProjectID = m.ProjectID
	dsm.DisplayName = m.DisplayName
	dsm.Description = m.Description
	dsm.Version = m.Version
	dsm.EnableStackitExampleDags = m.EnableStackitExampleDags
	dsm.EnableAirflowExampleDags = m.EnableAirflowExampleDags
	dsm.ObservabilityID = m.ObservabilityID
	dsm.Network = m.Network
	dsm.IdentityProvider = m.IdentityProvider
	dsm.Endpoints = m.Endpoints
	dsm.DagBundles = dagBundles
	dsm.Status = m.Status
	dsm.StatusMessage = m.StatusMessage
	dsm.CreatedAt = m.CreatedAt

	resp.Diagnostics.Append(resp.State.Set(ctx, dsm)...)
	tflog.Debug(ctx, "Workflows instance read", map[string]any{"instance_id": instanceID})
}

// mapInstanceDagBundles converts the embedded bundle list returned by the
// Instance GET into a Terraform List value. The Instance variant of DagBundle
// uses the request shape (GitDagBundle/S3DagBundle) rather than the response
// shape that the per-bundle endpoint returns.
func mapInstanceDagBundles(ctx context.Context, bundles []workflows.DagBundle) (types.List, error) {
	objType := types.ObjectType{AttrTypes: dagBundleSummaryTypes}
	if bundles == nil {
		return types.ListNull(objType), nil
	}
	elements := make([]attr.Value, 0, len(bundles))
	for i, b := range bundles {
		bs := dagBundleSummary{}
		switch {
		case b.GitDagBundle != nil:
			g := b.GitDagBundle
			bs.Type = types.StringValue(workflowsUtils.BundleTypeGit)
			bs.Name = types.StringValue(g.Name)
			bs.URL = types.StringValue(g.Url)
			bs.Branch = types.StringValue(g.Branch)
			bs.Subdir = types.StringPointerValue(g.Subdir)
			bs.RefreshInterval = types.Int32PointerValue(g.RefreshInterval)
		case b.S3DagBundle != nil:
			s := b.S3DagBundle
			bs.Type = types.StringValue(workflowsUtils.BundleTypeS3)
			bs.Name = types.StringValue(s.Name)
			bs.BucketName = types.StringValue(s.BucketName)
			bs.Endpoint = types.StringPointerValue(s.Endpoint)
			bs.Prefix = types.StringPointerValue(s.Prefix)
			bs.RefreshInterval = types.Int32PointerValue(s.RefreshInterval)
		default:
			return types.ListNull(objType), fmt.Errorf("unknown dag_bundle variant at index %d; upgrade the provider", i)
		}
		obj, diags := types.ObjectValueFrom(ctx, dagBundleSummaryTypes, bs)
		if diags.HasError() {
			return types.ListNull(objType), fmt.Errorf("mapping bundle at index %d: %w", i, core.DiagsToError(diags))
		}
		elements = append(elements, obj)
	}
	list, diags := types.ListValue(objType, elements)
	if diags.HasError() {
		return types.ListNull(objType), fmt.Errorf("building list: %w", core.DiagsToError(diags))
	}
	return list, nil
}
