package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// serverDataSourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var serverDataSourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &serverDataSource{}
)

type DataSourceModel struct {
	Id                types.String `tfsdk:"id"` // needed by TF
	ProjectId         types.String `tfsdk:"project_id"`
	ServerId          types.String `tfsdk:"server_id"`
	MachineType       types.String `tfsdk:"machine_type"`
	Name              types.String `tfsdk:"name"`
	AvailabilityZone  types.String `tfsdk:"availability_zone"`
	BootVolume        types.Object `tfsdk:"boot_volume"`
	ImageId           types.String `tfsdk:"image_id"`
	NetworkInterfaces types.List   `tfsdk:"network_interfaces"`
	KeypairName       types.String `tfsdk:"keypair_name"`
	Labels            types.Map    `tfsdk:"labels"`
	AffinityGroup     types.String `tfsdk:"affinity_group"`
	UserData          types.String `tfsdk:"user_data"`
	CreatedAt         types.String `tfsdk:"created_at"`
	LaunchedAt        types.String `tfsdk:"launched_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

// NewServerDataSource is a helper function to simplify the provider implementation.
func NewServerDataSource() datasource.DataSource {
	return &serverDataSource{}
}

// serverDataSource is the data source implementation.
type serverDataSource struct {
	client *iaas.APIClient
}

// Metadata returns the data source type name.
func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var apiClient *iaas.APIClient
	var err error

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !serverDataSourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_server", "data source")
		if resp.Diagnostics.HasError() {
			return
		}
		serverDataSourceBetaCheckDone = true
	}

	if providerData.IaaSCustomEndpoint != "" {
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	d.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the datasource.
func (r *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Server datasource schema. Must have a `region` specified in the provider configuration."),
		Description:         "Server datasource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`server_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the server is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"server_id": schema.StringAttribute{
				Description: "The server ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the server.",
				Computed:    true,
			},
			"machine_type": schema.StringAttribute{
				MarkdownDescription: "Name of the type of the machine for the server. Possible values are documented in [Virtual machine flavors](https://docs.stackit.cloud/stackit/en/virtual-machine-flavors-75137231.html)",
				Computed:            true,
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zone of the server.",
				Computed:    true,
			},
			"boot_volume": schema.SingleNestedAttribute{
				Description: "The boot volume for the server",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"performance_class": schema.StringAttribute{
						Description: "The performance class of the server.",
						Computed:    true,
					},
					"size": schema.Int64Attribute{
						Description: "The size of the boot volume in GB.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "The type of the source. " + utils.SupportedValuesDocumentation(supportedSourceTypes),
						Computed:    true,
					},
					"id": schema.StringAttribute{
						Description: "The ID of the source, either image ID or volume ID",
						Computed:    true,
					},
				},
			},
			"image_id": schema.StringAttribute{
				Description: "The image ID to be used for an ephemeral disk on the server.",
				Computed:    true,
			},
			"network_interfaces": schema.ListAttribute{
				Description: "The IDs of network interfaces which should be attached to the server. Updating it will recreate the server.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"keypair_name": schema.StringAttribute{
				Description: "The name of the keypair used during server creation.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
			"affinity_group": schema.StringAttribute{
				Description: "The affinity group the server is assigned to.",
				Computed:    true,
			},
			"user_data": schema.StringAttribute{
				Description: "User data that is passed via cloud-init to the server.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Date-time when the server was created",
				Computed:    true,
			},
			"launched_at": schema.StringAttribute{
				Description: "Date-time when the server was launched",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date-time when the server was updated",
				Computed:    true,
			},
		},
	}
}

// // Read refreshes the Terraform state with the latest data.
func (r *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)

	serverReq := r.client.GetServer(ctx, projectId, serverId)
	serverReq = serverReq.Details(true)
	serverResp, err := serverReq.Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapDataSourceFields(ctx, serverResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading server", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "server read")
}

func mapDataSourceFields(ctx context.Context, serverResp *iaas.Server, model *DataSourceModel) error {
	if serverResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var serverId string
	if model.ServerId.ValueString() != "" {
		serverId = model.ServerId.ValueString()
	} else if serverResp.Id != nil {
		serverId = *serverResp.Id
	} else {
		return fmt.Errorf("server id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		serverId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	labels, diags := types.MapValueFrom(ctx, types.StringType, map[string]interface{}{})
	if diags.HasError() {
		return fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
	}
	if serverResp.Labels != nil && len(*serverResp.Labels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *serverResp.Labels)
		if diags.HasError() {
			return fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else if model.Labels.IsNull() {
		labels = types.MapNull(types.StringType)
	}
	var createdAt basetypes.StringValue
	if serverResp.CreatedAt != nil {
		createdAtValue := *serverResp.CreatedAt
		createdAt = types.StringValue(createdAtValue.Format(time.RFC3339))
	}
	var updatedAt basetypes.StringValue
	if serverResp.UpdatedAt != nil {
		updatedAtValue := *serverResp.UpdatedAt
		updatedAt = types.StringValue(updatedAtValue.Format(time.RFC3339))
	}
	var launchedAt basetypes.StringValue
	if serverResp.LaunchedAt != nil {
		launchedAtValue := *serverResp.LaunchedAt
		launchedAt = types.StringValue(launchedAtValue.Format(time.RFC3339))
	}
	if serverResp.Nics != nil {
		var respNics []string
		for _, nic := range *serverResp.Nics {
			respNics = append(respNics, *nic.NicId)
		}
		nicTF, diags := types.ListValueFrom(ctx, types.StringType, respNics)
		if diags.HasError() {
			return fmt.Errorf("failed to map networkInterfaces: %w", core.DiagsToError(diags))
		}

		model.NetworkInterfaces = nicTF
	} else {
		model.NetworkInterfaces = types.ListNull(types.StringType)
	}

	model.AvailabilityZone = types.StringPointerValue(serverResp.AvailabilityZone)
	model.ServerId = types.StringValue(serverId)
	model.MachineType = types.StringPointerValue(serverResp.MachineType)

	model.Name = types.StringPointerValue(serverResp.Name)
	model.Labels = labels
	model.ImageId = types.StringPointerValue(serverResp.ImageId)
	model.KeypairName = types.StringPointerValue(serverResp.KeypairName)
	model.AffinityGroup = types.StringPointerValue(serverResp.AffinityGroup)
	model.CreatedAt = createdAt
	model.UpdatedAt = updatedAt
	model.LaunchedAt = launchedAt

	return nil
}
