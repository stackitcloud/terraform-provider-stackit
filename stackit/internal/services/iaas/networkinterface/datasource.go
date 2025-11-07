package networkinterface

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &networkInterfaceDataSource{}
)

// NewNetworkDataSource is a helper function to simplify the provider implementation.
func NewNetworkInterfaceDataSource() datasource.DataSource {
	return &networkInterfaceDataSource{}
}

// networkInterfaceDataSource is the data source implementation.
type networkInterfaceDataSource struct {
	client *iaas.APIClient
}

// Metadata returns the data source type name.
func (d *networkInterfaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_interface"
}

func (d *networkInterfaceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the data source.
func (d *networkInterfaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	typeOptions := []string{"server", "metadata", "gateway"}
	description := "Network interface datasource schema. Must have a `region` specified in the provider configuration."

	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source ID. It is structured as \"`project_id`,`network_id`,`network_interface_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the network interface is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_id": schema.StringAttribute{
				Description: "The network ID to which the network interface is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_interface_id": schema.StringAttribute{
				Description: "The network interface ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the network interface.",
				Computed:    true,
			},
			"allowed_addresses": schema.ListAttribute{
				Description: "The list of CIDR (Classless Inter-Domain Routing) notations.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"device": schema.StringAttribute{
				Description: "The device UUID of the network interface.",
				Computed:    true,
			},
			"ipv4": schema.StringAttribute{
				Description: "The IPv4 address.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a network interface.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"mac": schema.StringAttribute{
				Description: "The MAC address of network interface.",
				Computed:    true,
			},
			"security": schema.BoolAttribute{
				Description: "The Network Interface Security. If set to false, then no security groups will apply to this network interface.",
				Computed:    true,
			},
			"security_group_ids": schema.ListAttribute{
				Description: "The list of security group UUIDs. If security is set to false, setting this field will lead to an error.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"type": schema.StringAttribute{
				Description: "Type of network interface. Some of the possible values are: " + utils.FormatPossibleValues(typeOptions...),
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *networkInterfaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	networkInterfaceId := model.NetworkInterfaceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "network_interface_id", networkInterfaceId)

	networkInterfaceResp, err := d.client.GetNic(ctx, projectId, networkId, networkInterfaceId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading network interface",
			fmt.Sprintf("Network interface with ID %q or network with ID %q does not exist in project %q.", networkInterfaceId, networkId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapFields(ctx, networkInterfaceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network interface", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network interface read")
}
