package provideroptions

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	workflowsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/workflows/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

var _ datasource.DataSource = &providerOptionsDataSource{}

type Model struct {
	Region   types.String `tfsdk:"region"`
	Versions types.List   `tfsdk:"versions"`
}

type versionModel struct {
	Version        types.String `tfsdk:"version"`
	State          types.String `tfsdk:"state"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
}

var versionTypes = map[string]attr.Type{
	"version":         basetypes.StringType{},
	"state":           basetypes.StringType{},
	"expiration_date": basetypes.StringType{},
}

type providerOptionsDataSource struct {
	client       *workflows.APIClient
	providerData core.ProviderData
}

func NewWorkflowsProviderOptionsDataSource() datasource.DataSource {
	return &providerOptionsDataSource{}
}

// Metadata returns the data source type name.
func (d *providerOptionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflows_provider_options"
}

// Configure adds the provider configured client to the data source.
func (d *providerOptionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.providerData = providerData

	features.CheckExperimentEnabled(ctx, &d.providerData, features.WorkflowsExperiment, "stackit_workflows_provider_options", core.Datasource, &resp.Diagnostics)
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
func (d *providerOptionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := fmt.Sprintf("Lists Workflows versions supported by the Workflows API in a region. %s", core.DatasourceRegionFallbackDocstring)
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.WorkflowsExperiment, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Description: "STACKIT region to query. If not defined, the provider region is used.",
				Optional:    true,
				Computed:    true,
			},
			"versions": schema.ListNestedAttribute{
				Description: "Supported Workflows versions.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": schema.StringAttribute{
							Description: "Version identifier (e.g. `workflows-3.0-airflow-3.1`).",
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: "Lifecycle state of the version.",
							Computed:    true,
						},
						"expiration_date": schema.StringAttribute{
							Description: "RFC 3339 timestamp at which the version expires, or null if there is no scheduled expiry.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read reads the data source and writes its result to Terraform state.
func (d *providerOptionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	region := d.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "region", region)

	options, err := d.client.DefaultAPI.GetProviderOptions(ctx, region).Execute()
	if err != nil {
		tfutils.LogError(ctx, &resp.Diagnostics, err, "Error reading Workflows provider options", fmt.Sprintf("Region %q", region), nil)
		return
	}
	ctx = core.LogResponse(ctx)

	model.Region = types.StringValue(region)
	if err := mapVersions(ctx, options, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Workflows provider options", fmt.Sprintf("Processing response: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Info(ctx, "Workflows provider options read", map[string]any{"region": region, "count": len(options.GetVersions())})
}

func mapVersions(ctx context.Context, options *workflows.ProviderOptions, model *Model) error {
	objType := types.ObjectType{AttrTypes: versionTypes}
	if options == nil || options.Versions == nil {
		model.Versions = types.ListNull(objType)
		return nil
	}

	elements := make([]attr.Value, 0, len(options.Versions))
	for i := range options.Versions {
		v := options.Versions[i]
		exp := types.StringNull()
		if v.ExpirationDate != nil {
			exp = types.StringValue(v.ExpirationDate.Format(time.RFC3339))
		}
		obj, diags := types.ObjectValueFrom(ctx, versionTypes, versionModel{
			Version:        types.StringValue(v.Version),
			State:          types.StringValue(v.State),
			ExpirationDate: exp,
		})
		if diags.HasError() {
			return fmt.Errorf("%v", diags.Errors())
		}
		elements = append(elements, obj)
	}
	list, diags := types.ListValue(objType, elements)
	if diags.HasError() {
		return fmt.Errorf("%v", diags.Errors())
	}
	model.Versions = list
	return nil
}
