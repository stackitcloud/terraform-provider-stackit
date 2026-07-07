package sqlserverflex

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	sqlserverflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3beta2api"
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
	resp.TypeName = req.ProviderTypeName + "_sqlserverflex_instance"
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
		"main":                 "SQLServer Flex instance data source schema. Must have a `region` specified in the provider configuration.",
		"id":                   "Terraform's internal data source. ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
		"instance_id":          "ID of the SQLServer Flex instance.",
		"project_id":           "STACKIT project ID to which the instance is associated.",
		"name":                 "Instance name.",
		"acl":                  "The Access Control List (ACL) for the SQLServer Flex instance.",
		"backup_schedule":      `The backup schedule. Should follow the cron scheduling system format (e.g. "0 0 * * *").`,
		"options":              "Custom parameters for the SQLServer Flex instance.",
		"flavor_id":            "The flavor ID of the SQLServer Flex instance.",
		"network":              "The network configuration of the instance.",
		"network.access_scope": "The network access scope of the instance. This feature is in private preview. Supplying this object is only permitted for enabled accounts. If your account does not have access, the request will be rejected.",
		"network.acl":          "List of IPV4 cidr.",
		"retention_days":       "The days for how long the backup files should be stored before cleaned up. 30 to 90",
		"edition":              "Edition of the MSSQL server instance",
		"region":               "The resource region. If not defined, the provider region is used.",
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
			"acl": schema.ListAttribute{
				Description: descriptions["acl"],
				ElementType: types.StringType,
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
			"flavor_id": schema.StringAttribute{
				Description: descriptions["flavor_id"],
				Computed:    true,
			},
			"network": schema.SingleNestedAttribute{
				Description: descriptions["network"],
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"access_scope": schema.StringAttribute{
						Description: descriptions["network.access_scope"],
						Optional:    true,
					},
					"acl": schema.ListAttribute{
						Description: descriptions["network.acl"],
						ElementType: types.StringType,
						Computed:    true,
					},
				},
			},
			"replicas": schema.Int32Attribute{
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
			"edition": schema.StringAttribute{
				Description: descriptions["edition"],
				Computed:    true,
			},
			"options": schema.SingleNestedAttribute{
				Description: descriptions["options"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"edition": schema.StringAttribute{
						Computed: true,
					},
					"retention_days": schema.Int32Attribute{
						Computed: true,
					},
				},
			},
			"retention_days": schema.Int32Attribute{
				Description: descriptions["retention_days"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				// the region cannot be found, so it has to be passed
				Optional:    true,
				Description: descriptions["region"],
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
	instanceResp, err := r.client.DefaultAPI.GetInstance(ctx, projectId, region, instanceId).Execute()
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

	var options = &optionsModel{}
	if !(model.Options.IsNull() || model.Options.IsUnknown()) {
		diags = model.Options.As(ctx, options, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	flavorResp, err := getFlavor(ctx, r.client.DefaultAPI, projectId, region, instanceResp.FlavorId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Finding flavor: %v", err))
		return
	}
	flavor := &flavorModel{
		Id:          types.StringValue(flavorResp.Id),
		Description: types.StringValue(flavorResp.Description),
		CPU:         types.Int64Value(flavorResp.Cpu),
		RAM:         types.Int64Value(flavorResp.Memory),
	}

	err = mapFields(ctx, instanceResp, &model, flavor, region)
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
