package artifactory

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
	_ datasource.DataSource = &artifactoryDataSource{}
)

type DataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	ProjectId     types.String `tfsdk:"project_id"`
	Region        types.String `tfsdk:"region"`
	ArtifactoryId types.String `tfsdk:"artifactory_id"`
	Name          types.String `tfsdk:"name"`
	CreateTime    types.String `tfsdk:"create_time"`
	State         types.String `tfsdk:"state"`
	Url           types.String `tfsdk:"url"`
	Proxy         types.Object `tfsdk:"proxy"`
}

type dsProxyModel struct {
	Description  types.String `tfsdk:"description"`
	RegistryType types.String `tfsdk:"registry_type"`
	Url          types.String `tfsdk:"url"`
}

var dsProxyTypes = map[string]attr.Type{
	"description":   types.StringType,
	"registry_type": types.StringType,
	"url":           types.StringType,
}

func NewArtifactoryDataSource() datasource.DataSource {
	return &artifactoryDataSource{}
}

type artifactoryDataSource struct {
	client       registry.DefaultAPI
	providerData core.ProviderData
}

func (d *artifactoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_containerregistry_artifactory"
}

func (d *artifactoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.providerData = providerData
	d.client = clients.ContainerRegistryV1Client
	tflog.Info(ctx, "Container Registry client configured")
}

func (d *artifactoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Container registry artifactory data source schema.",
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
			"artifactory_id": schema.StringAttribute{
				Description: descriptions["artifactory_id"],
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
				},
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
	}
}

func (d *artifactoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	artifactoryId := model.ArtifactoryId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "artifactory_id", artifactoryId)

	artifactory, err := d.client.GetArtifactory(ctx, projectId, region, artifactoryId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Error reading artifactory",
			fmt.Sprintf("Calling API: %v", err),
			map[int]string{
				http.StatusNotFound: fmt.Sprintf("Artifactory %q not found", artifactoryId),
			},
		)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapDataSourceFields(ctx, artifactory, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading artifactory", fmt.Sprintf("Mapping API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Container Registry artifactory read")
}

func mapDataSourceFields(ctx context.Context, artifactory *registry.Artifactory, model *DataSourceModel, region string) error {
	if artifactory == nil {
		return fmt.Errorf("artifactory is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, artifactory.Id)
	model.Region = types.StringValue(region)
	model.ArtifactoryId = types.StringValue(artifactory.Id)
	model.Name = types.StringValue(artifactory.Name)
	model.CreateTime = types.StringValue(artifactory.CreateTime.Format(time.RFC3339))
	model.State = types.StringValue(string(artifactory.State))
	model.Url = types.StringValue(artifactory.Url)

	if artifactory.Proxy == nil {
		model.Proxy = types.ObjectNull(dsProxyTypes)
	} else {
		p := dsProxyModel{
			RegistryType: types.StringValue(artifactory.Proxy.RegistryType),
			Url:          types.StringValue(artifactory.Proxy.Url),
			Description:  types.StringPointerValue(artifactory.Proxy.Description),
		}

		proxyObj, diags := types.ObjectValueFrom(ctx, dsProxyTypes, p)
		if diags.HasError() {
			return fmt.Errorf("mapping proxy: %w", core.DiagsToError(diags))
		}
		model.Proxy = proxyObj
	}

	return nil
}
