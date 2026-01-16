package postgresflexalpha

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	postgresflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	client       *postgresflexalpha.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *instanceDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_postgresflexalpha_instance"
}

// Configure adds the provider configured client to the data source.
func (r *instanceDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := postgresflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Postgres Flex instance client configured")
}

// Schema defines the schema for the data source.
func (r *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "Postgres Flex instance data source schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal data source. ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
		"instance_id": "ID of the PostgresFlex instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"name":        "Instance name.",
		"acl":         "The Access Control List (ACL) for the PostgresFlex instance.",
		"region":      "The resource region. If not defined, the provider region is used.",
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
				Computed: true,
			},
			"retention_days": schema.Int64Attribute{
				Computed: true,
			},
			"flavor_id": schema.StringAttribute{
				Computed: true,
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
					"acl": schema.ListAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
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
				},
				Description: descriptions["network"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) { // nolint:gocritic // function signature required by Terraform
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

	var storage = &storageModel{}
	if !model.Storage.IsNull() && !model.Storage.IsUnknown() {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var network = &networkModel{}
	if !model.Network.IsNull() && !model.Network.IsUnknown() {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var encryption = &encryptionModel{}
	if !model.Encryption.IsNull() && !model.Encryption.IsUnknown() {
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	err = mapFields(ctx, instanceResp, &model, storage, encryption, network, region)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading instance",
			fmt.Sprintf("Processing API payload: %v", err),
		)
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex instance read")
}
