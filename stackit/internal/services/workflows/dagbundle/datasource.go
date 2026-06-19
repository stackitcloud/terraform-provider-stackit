package dagbundle

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

var _ datasource.DataSource = &dagBundleDataSource{}

func NewWorkflowsDagBundleDataSource() datasource.DataSource {
	return &dagBundleDataSource{}
}

type dagBundleDataSource struct {
	client       *workflows.APIClient
	providerData core.ProviderData
}

func (d *dagBundleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflows_dag_bundle"
}

func (d *dagBundleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.providerData = providerData

	features.CheckExperimentEnabled(ctx, &d.providerData, features.WorkflowsExperiment, "stackit_workflows_dag_bundle", core.Datasource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := workflowsUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
}

func (d *dagBundleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := fmt.Sprintf("Workflows DAG bundle data source. %s", core.DatasourceRegionFallbackDocstring)
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.WorkflowsExperiment, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Description: schemaDescriptions["id"], Computed: true},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				Validators:  []validator.String{validate.UUID(), validate.NoSeparator()},
			},
			"region": schema.StringAttribute{Description: schemaDescriptions["region"], Optional: true, Computed: true},
			"instance_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Required:    true,
				Validators:  []validator.String{validate.UUID(), validate.NoSeparator()},
			},
			"name": schema.StringAttribute{Description: schemaDescriptions["name"], Required: true},
			"git": schema.SingleNestedAttribute{
				Description: "Git-backed DAG bundle source. Populated when the bundle is git-typed.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"url":              schema.StringAttribute{Description: schemaDescriptions["git.url"], Computed: true},
					"branch":           schema.StringAttribute{Description: schemaDescriptions["git.branch"], Computed: true},
					"subdir":           schema.StringAttribute{Description: schemaDescriptions["git.subdir"], Computed: true},
					"refresh_interval": schema.Int32Attribute{Description: schemaDescriptions["git.refresh_interval"], Computed: true},
					"auth": schema.SingleNestedAttribute{
						Description: schemaDescriptions["git.auth"],
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"type":     schema.StringAttribute{Description: schemaDescriptions["git.auth.type"], Computed: true},
							"username": schema.StringAttribute{Description: schemaDescriptions["git.auth.username"], Computed: true},
							"password": schema.StringAttribute{Description: "Git password or PAT. Never returned by the API; always null when read.", Computed: true, Sensitive: true},
						},
					},
				},
			},
			"s3": schema.SingleNestedAttribute{
				Description: "S3-backed DAG bundle source. Populated when the bundle is s3-typed.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"bucket_name":      schema.StringAttribute{Description: schemaDescriptions["s3.bucket_name"], Computed: true},
					"endpoint":         schema.StringAttribute{Description: schemaDescriptions["s3.endpoint"], Computed: true},
					"prefix":           schema.StringAttribute{Description: schemaDescriptions["s3.prefix"], Computed: true},
					"refresh_interval": schema.Int32Attribute{Description: schemaDescriptions["s3.refresh_interval"], Computed: true},
					"auth": schema.SingleNestedAttribute{
						Description: schemaDescriptions["s3.auth"],
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"type":              schema.StringAttribute{Description: schemaDescriptions["s3.auth.type"], Computed: true},
							"access_key_id":     schema.StringAttribute{Description: schemaDescriptions["s3.auth.access_key_id"], Computed: true},
							"secret_access_key": schema.StringAttribute{Description: "S3 secret access key. Never returned by the API; always null when read.", Computed: true, Sensitive: true},
						},
					},
				},
			},
		},
	}
}

func (d *dagBundleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "bundle_name", name)

	bundle, err := d.client.DefaultAPI.GetDagBundle(ctx, projectID, region, instanceID, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		tfutils.LogError(
			ctx, &resp.Diagnostics, err,
			"Error reading Workflows DAG bundle",
			fmt.Sprintf("Bundle %q does not exist on instance %q.", name, instanceID),
			map[int]string{http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectID)},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = core.LogResponse(ctx)

	if err := mapFields(ctx, bundle, &model, region); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Workflows DAG bundle", fmt.Sprintf("Processing response: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Debug(ctx, "Workflows DAG bundle read")
}
