package networks

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &networksDataSource{}
)

// NetworkItem represents a single network in the list.
type NetworkItem struct {
	NetworkId types.String `tfsdk:"network_id"`
	Name      types.String `tfsdk:"name"`
}

// NetworksModel represents the model for the plural data source.
type NetworksModel struct {
	Id        types.String  `tfsdk:"id"`
	ProjectId types.String  `tfsdk:"project_id"`
	Region    types.String  `tfsdk:"region"`
	NameRegex types.String  `tfsdk:"name_regex"`
	Labels    types.Map     `tfsdk:"labels"`
	Items     []NetworkItem `tfsdk:"items"`
}

// NewNetworksDataSource creates a new instance of plural data source.
func NewNetworksDataSource() datasource.DataSource {
	return &networksDataSource{}
}

// networksDataSource is the data source implementation for querying multiple networks.
type networksDataSource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

func (d *networksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}

func (d *networksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "IaaS client configured")
}

func (d *networksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists STACKIT networks in a project, optionally filtered by name or labels.",
		Description:         "Lists STACKIT networks in a project, optionally filtered by name or labels.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the networks are associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				Optional:    true,
			},
			"name_regex": schema.StringAttribute{
				Description: "Optional regular expression to filter networks by name.",
				Optional:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Optional label selector to filter networks by labels.",
				ElementType: types.StringType,
				Optional:    true,
				Validators:  validate.LabelValidators(),
			},
			"items": schema.ListNestedAttribute{
				Description: "List of networks matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{
							Description: "The network ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the network.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *networksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic
	var model NetworksModel
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

	// Compile the regex if provided
	var compiledRegex *regexp.Regexp
	var err error
	if !model.NameRegex.IsNull() && model.NameRegex.ValueString() != "" {
		compiledRegex, err = regexp.Compile(model.NameRegex.ValueString())
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Invalid name_regex", err.Error())
			return
		}
	}

	// Fetch all networks for the given project and region, applying label selector if provided
	listReq := d.client.DefaultAPI.ListNetworks(ctx, projectId, region)
	if !model.Labels.IsNull() && !model.Labels.IsUnknown() {
		var labels map[string]string
		diags = model.Labels.ElementsAs(ctx, &labels, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		var pairs []string
		for k, v := range labels {
			if v == "" {
				pairs = append(pairs, k)
			} else {
				pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
			}
		}
		sort.Strings(pairs)
		labelSelector := strings.Join(pairs, ",")
		if labelSelector != "" {
			listReq = listReq.LabelSelector(labelSelector)
		}
	}

	networksResp, err := listReq.Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading networks",
			fmt.Sprintf("Networks could not be listed for project %q.", projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	if networksResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Reading networks", "API response is nil")
		return
	}
	if networksResp.Items == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Reading networks", "network items in the API response are nil")
		return
	}

	// Filter networks based on the compiled regex and populate the items list
	items := make([]NetworkItem, 0)
	for _, netResp := range networksResp.Items {
		if compiledRegex != nil && !compiledRegex.MatchString(netResp.Name) {
			continue
		}

		item := NetworkItem{
			NetworkId: types.StringValue(netResp.Id),
			Name:      types.StringValue(netResp.Name),
		}
		items = append(items, item)
	}

	model.Id = utils.BuildInternalTerraformId(projectId, region)
	model.Items = items

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Networks read")
}
