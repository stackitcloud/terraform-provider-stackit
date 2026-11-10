package replicationadapters

import (
	"context"
	"fmt"
	"net/http"

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
	_ datasource.DataSource = &replicationAdaptersDataSource{}
)

type DataSourceModel struct {
	Id                  types.String `tfsdk:"id"`
	ProjectId           types.String `tfsdk:"project_id"`
	Region              types.String `tfsdk:"region"`
	ReplicationAdapters types.List   `tfsdk:"replication_adapters"`
}

type replicationAdapterModel struct {
	CredentialType types.String `tfsdk:"credential_type"`
	DisplayName    types.String `tfsdk:"display_name"`
	RegistryType   types.String `tfsdk:"registry_type"`
}

var replicationAdapterTypes = map[string]attr.Type{
	"credential_type": types.StringType,
	"display_name":    types.StringType,
	"registry_type":   types.StringType,
}

var descriptions = map[string]string{ //nolint:gosec // linter false positive
	"id":                                   "Terraform's internal data source ID, structured as `project_id`,`region`.",
	"project_id":                           "STACKIT project ID to which the replication adapters are associated.",
	"region":                               "STACKIT region where the replication adapters are located. If not defined, the provider default_region is used.",
	"replication_adapters":                 "List of supported replication adapters.",
	"replication_adapters.credential_type": "User-hint for what type of credentials are expected for the replication adapter type.",
	"replication_adapters.display_name":    "Human-friendly name of the provider the adapter relates to.",
	"replication_adapters.registry_type":   "Identifies the replication adapter.",
}

func NewReplicationAdaptersDataSource() datasource.DataSource {
	return &replicationAdaptersDataSource{}
}

type replicationAdaptersDataSource struct {
	client       registry.DefaultAPI
	providerData core.ProviderData
}

func (d *replicationAdaptersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_containerregistry_replication_adapters"
}

func (d *replicationAdaptersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.providerData = providerData
	d.client = clients.ContainerRegistryV1Client
	tflog.Info(ctx, "Container Registry client configured")
}

func (d *replicationAdaptersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Container registry replication adapters data source schema.",
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
			"replication_adapters": schema.ListNestedAttribute{
				Description: descriptions["replication_adapters"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"credential_type": schema.StringAttribute{
							Description: descriptions["replication_adapters.credential_type"],
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: descriptions["replication_adapters.display_name"],
							Computed:    true,
						},
						"registry_type": schema.StringAttribute{
							Description: descriptions["replication_adapters.registry_type"],
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *replicationAdaptersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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

	listResp, err := d.client.ListReplicationAdapters(ctx, projectId, region).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Error reading replication adapters",
			fmt.Sprintf("Calling API: %v", err),
			map[int]string{
				http.StatusNotFound: fmt.Sprintf("Project %q or region %q not found", projectId, region),
			},
		)
		return
	}

	ctx = core.LogResponse(ctx)

	var adapters []registry.ReplicationAdapter
	if listResp != nil {
		adapters = listResp.ReplicationAdapters
	}

	err = mapFields(ctx, adapters, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading replication adapters", fmt.Sprintf("Mapping API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Container Registry replication adapters read")
}

func mapFields(ctx context.Context, adapters []registry.ReplicationAdapter, model *DataSourceModel, region string) error {
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region)
	model.Region = types.StringValue(region)

	items := make([]attr.Value, 0, len(adapters))
	for _, adapter := range adapters {
		item := replicationAdapterModel{
			CredentialType: types.StringValue(string(adapter.CredentialType)),
			DisplayName:    types.StringValue(adapter.DisplayName),
			RegistryType:   types.StringValue(adapter.RegistryType),
		}

		itemObj, diags := types.ObjectValueFrom(ctx, replicationAdapterTypes, item)
		if diags.HasError() {
			return fmt.Errorf("mapping replication adapter item: %w", core.DiagsToError(diags))
		}

		items = append(items, itemObj)
	}

	itemsList, diags := types.ListValue(types.ObjectType{AttrTypes: replicationAdapterTypes}, items)
	if diags.HasError() {
		return fmt.Errorf("mapping replication adapters list: %w", core.DiagsToError(diags))
	}
	model.ReplicationAdapters = itemsList

	return nil
}
