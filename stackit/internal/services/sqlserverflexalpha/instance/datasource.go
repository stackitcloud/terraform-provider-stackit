package sqlserverflex

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	sqlserverflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflexalpha/utils"

	sqlserverflex "github.com/stackitcloud/terraform-provider-stackit/pkg/sqlserverflexalpha"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &instanceDataSource{}
)

// NewInstanceDataSource is a helper function to simplify the provider implementation.
func NewInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

// instanceDataSource is the data source implementation.
type instanceDataSource struct {
	client       *sqlserverflex.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflexalpha_instance"
}

// Configure adds the provider configured client to the data source.
func (r *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := sqlserverflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "SQLServer Flex instance client configured")
}

// Schema defines the schema for the data source.
func (r *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":            "SQLServer Flex instance data source schema. Must have a `region` specified in the provider configuration.",
		"id":              "Terraform's internal data source. ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
		"instance_id":     "ID of the SQLServer Flex instance.",
		"project_id":      "STACKIT project ID to which the instance is associated.",
		"name":            "Instance name.",
		"acl":             "The Access Control List (ACL) for the SQLServer Flex instance.",
		"backup_schedule": `The backup schedule. Should follow the cron scheduling system format (e.g. "0 0 * * *").`,
		"options":         "Custom parameters for the SQLServer Flex instance.",
		"region":          "The resource region. If not defined, the provider region is used.",
		// TODO
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
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
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"backup_schedule": schema.StringAttribute{
				Description: descriptions["backup_schedule"],
				Computed:    true,
			},
			"flavor": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"description": schema.StringAttribute{
						Computed: true,
					},
					"cpu": schema.Int64Attribute{
						Computed: true,
					},
					"ram": schema.Int64Attribute{
						Computed: true,
					},
				},
			},
			"replicas": schema.Int64Attribute{
				Computed: true,
			},
			"storage": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"class": schema.StringAttribute{
						Computed: true,
					},
					"size": schema.Int64Attribute{
						Computed: true,
					},
				},
			},
			"version": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"edition": schema.StringAttribute{
				Computed: true,
			},
			"retention_days": schema.Int64Attribute{
				Computed: true,
			},
			"region": schema.StringAttribute{
				// the region cannot be found, so it has to be passed
				Optional:    true,
				Description: descriptions["region"],
			},
			"encryption": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"key_id": schema.StringAttribute{
						Description: descriptions["key_id"],
						Computed:    true,
					},
					"key_version": schema.StringAttribute{
						Description: descriptions["key_version"],
						Computed:    true,
					},
					"keyring_id": schema.StringAttribute{
						Description: descriptions["keyring_id"],
						Computed:    true,
					},
					"service_account": schema.StringAttribute{
						Description: descriptions["service_account"],
						Computed:    true,
					},
				},
				Description: descriptions["encryption"],
			},
			"network": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"access_scope": schema.StringAttribute{
						Description: descriptions["access_scope"],
						Computed:    true,
					},
					"instance_address": schema.StringAttribute{
						Description: descriptions["instance_address"],
						Computed:    true,
					},
					"router_address": schema.StringAttribute{
						Description: descriptions["router_address"],
						Computed:    true,
					},
					"acl": schema.ListAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
						Computed:    true,
					},
				},
				Description: descriptions["network"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)
	instanceResp, err := r.client.GetInstanceRequest(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading instance",
			fmt.Sprintf("Instance with ID %q does not exist in project %q.", instanceId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	var flavor = &flavorModel{}
	if model.Flavor.IsNull() || model.Flavor.IsUnknown() {
		flavor.Id = types.StringValue(*instanceResp.FlavorId)
		if flavor.Id.IsNull() || flavor.Id.IsUnknown() || flavor.Id.String() == "" {
			panic("WTF FlavorId can not be null or empty string")
		}
		err = getFlavorModelById(ctx, r.client, &model, flavor)
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), err.Error())
			return
		}
		if flavor.CPU.IsNull() || flavor.CPU.IsUnknown() || flavor.CPU.String() == "" {
			panic("WTF FlavorId can not be null or empty string")
		}
	}

	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var encryption = &encryptionModel{}
	if !(model.Encryption.IsNull() || model.Encryption.IsUnknown()) {
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var network = &networkModel{}
	if !(model.Network.IsNull() || model.Network.IsUnknown()) {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	err = mapFields(ctx, instanceResp, &model, flavor, storage, encryption, network, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SQLServer Flex instance read")
}
