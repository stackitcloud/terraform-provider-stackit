package artifactories

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	registry "github.com/stackitcloud/stackit-sdk-go/services/registry/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &artifactoriesDataSource{}
)

type DataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	ProjectId     types.String `tfsdk:"project_id"`
	Region        types.String `tfsdk:"region"`
	Artifactories types.List   `tfsdk:"artifactories"`
}

type artifactoryItemModel struct {
	ArtifactoryId types.String `tfsdk:"artifactory_id"`
	Name          types.String `tfsdk:"name"`
	CreateTime    types.String `tfsdk:"create_time"`
	State         types.String `tfsdk:"state"`
	Url           types.String `tfsdk:"url"`
	Proxy         types.Object `tfsdk:"proxy"`
}

type proxyItemModel struct {
	Description types.String `tfsdk:"description"`

	RegistryType types.String `tfsdk:"registry_type"`

	Url types.String `tfsdk:"url"`
}

var proxyTypes = map[string]attr.Type{

	"description": types.StringType,

	"registry_type": types.StringType,

	"url": types.StringType,
}

var artifactoryItemTypes = map[string]attr.Type{

	"artifactory_id": types.StringType,

	"name": types.StringType,

	"create_time": types.StringType,

	"state": types.StringType,

	"url": types.StringType,

	"proxy": types.ObjectType{AttrTypes: proxyTypes},
}

var descriptions = map[string]string{

	"id": "Terraform's internal data source ID, structured as `project_id`,`region`.",

	"project_id": "STACKIT project ID to which the container registry artifactories are associated.",

	"region": "STACKIT region where the container registry artifactories are located. If not defined, the provider default_region is used.",

	"artifactories": "List of container registry artifactories.",

	"artifactory_id": "The unique identifier of the container registry artifactory.",

	"name": "A user-chosen name to distinguish multiple STACKIT artifactories.",

	"create_time": "The date and time the creation of the STACKIT artifactory was triggered.",

	"state": "The current state of the STACKIT artifactory.",

	"url": "The base URL of the STACKIT artifactory.",

	"proxy": "Proxy configuration if this artifactory is created as a proxy-cache for an upstream registry.",

	"proxy.registry_type": "Which replication adapter to use for this registry.",

	"proxy.url": "Upstream registry URL to proxy.",

	"proxy.description": "Description of the upstream registry.",
}

func NewArtifactoriesDataSource() datasource.DataSource {
	return &artifactoriesDataSource{}
}

type artifactoriesDataSource struct {
	client       registry.DefaultAPI
	providerData core.ProviderData
}

func (d *artifactoriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_containerregistry_artifactories"
}

func (d *artifactoriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.providerData = providerData
	d.client = clients.ContainerRegistryV1Client
	tflog.Info(ctx, "Container Registry client configured")
}

func (d *artifactoriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Container registry artifactories list data source schema.",
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
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				Computed:    true,
			},
			"artifactories": schema.ListNestedAttribute{
				Description: descriptions["artifactories"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"artifactory_id": schema.StringAttribute{
							Description: descriptions["artifactory_id"],
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: descriptions["name"],
							Computed:    true,
						},
						"create_time": schema.StringAttribute{
							Description: descriptions["create_time"],
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: descriptions["state"],
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: descriptions["url"],
							Computed:    true,
						},
						"proxy": schema.SingleNestedAttribute{
							Description: descriptions["proxy"],
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"registry_type": schema.StringAttribute{
									Description: descriptions["proxy.registry_type"],
									Computed:    true,
								},
								"url": schema.StringAttribute{
									Description: descriptions["proxy.url"],
									Computed:    true,
								},
								"description": schema.StringAttribute{
									Description: descriptions["proxy.description"],
									Computed:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *artifactoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
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

	var allArtifactories []registry.Artifactory
	apiReq := d.client.ListArtifactories(ctx, projectId, region)
	for {
		listResp, err := apiReq.Execute()
		if err != nil {
			utils.LogError(
				ctx,
				&resp.Diagnostics,
				err,
				"Error reading artifactories",
				fmt.Sprintf("Calling API: %v", err),
				map[int]string{
					http.StatusNotFound: fmt.Sprintf("Project %q or region %q not found", projectId, region),
				},
			)
			return
		}

		if listResp == nil {
			break
		}

		allArtifactories = append(allArtifactories, listResp.Artifactories...)

		if listResp.NextPageToken == nil || *listResp.NextPageToken == "" {
			break
		}

		apiReq = apiReq.NextPageToken(*listResp.NextPageToken)
	}

	ctx = core.LogResponse(ctx)

	err := mapFields(ctx, allArtifactories, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading artifactories", fmt.Sprintf("Mapping API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Container Registry artifactories read")
}

func mapFields(ctx context.Context, artifactories []registry.Artifactory, model *DataSourceModel, region string) error {
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region)
	model.Region = types.StringValue(region)

	items := make([]attr.Value, 0, len(artifactories))
	for _, art := range artifactories {
		item := artifactoryItemModel{
			ArtifactoryId: types.StringValue(art.Id),
			Name:          types.StringValue(art.Name),
			CreateTime:    types.StringValue(art.CreateTime.Format(time.RFC3339)),
			State:         types.StringValue(string(art.State)),
			Url:           types.StringValue(art.Url),
		}

		if art.Proxy == nil {
			item.Proxy = types.ObjectNull(proxyTypes)
		} else {
			p := proxyItemModel{
				RegistryType: types.StringValue(art.Proxy.RegistryType),
				Url:          types.StringValue(art.Proxy.Url),
				Description:  types.StringPointerValue(art.Proxy.Description),
			}

			proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, p)
			if diags.HasError() {
				return fmt.Errorf("mapping proxy: %w", core.DiagsToError(diags))
			}
			item.Proxy = proxyObj
		}

		itemObj, diags := types.ObjectValueFrom(ctx, artifactoryItemTypes, item)
		if diags.HasError() {
			return fmt.Errorf("mapping artifactory item: %w", core.DiagsToError(diags))
		}

		items = append(items, itemObj)
	}

	itemsList, diags := types.ListValue(types.ObjectType{AttrTypes: artifactoryItemTypes}, items)
	if diags.HasError() {
		return fmt.Errorf("mapping artifactories list: %w", core.DiagsToError(diags))
	}
	model.Artifactories = itemsList

	return nil
}
