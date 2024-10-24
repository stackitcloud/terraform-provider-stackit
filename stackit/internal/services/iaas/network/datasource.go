package network

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &networkDataSource{}
)

// NewNetworkDataSource is a helper function to simplify the provider implementation.
func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

// networkDataSource is the data source implementation.
type networkDataSource struct {
	client *iaas.APIClient
}

// Metadata returns the data source type name.
func (d *networkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (d *networkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the data source.
func (d *networkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`network_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the network is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_id": schema.StringAttribute{
				Description: "The network ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the network.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"nameservers": schema.ListAttribute{
				Description:        "This field is deprecated and will be removed after April 28th 2025, use `ipv4_nameservers` to configure the nameservers for the IPv4 networks.",
				DeprecationMessage: "Use `ipv4_nameservers` to configure the nameservers for the IPv4 networks.",
				Computed:           true,
				ElementType:        types.StringType,
			},
			"ipv4_gateway": schema.StringAttribute{
				Description: "The IPv4 gateway of a network. If not specified, the first IP of the network will be assigned as the gateway.",
				Computed:    true,
			},
			"ipv4_nameservers": schema.ListAttribute{
				Description: "The IPv4 prefix length of the network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipv4_prefix": schema.StringAttribute{
				Description: "The IPv4 prefix of the network (CIDR).",
				Computed:    true,
			},
			"ipv4_prefix_length": schema.Int64Attribute{
				Description: "The IPv4 prefix length of the network.",
				Computed:    true,
			},
			"prefixes": schema.ListAttribute{
				Description: "The prefixes of the network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"public_ip": schema.StringAttribute{
				Description: "The public IP of the network.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
			"routed": schema.BoolAttribute{
				Description: "Shows if the network is routed and therefore accessible from other networks.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *networkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	networkResp, err := d.client.GetNetwork(ctx, projectId, networkId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, networkResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network read")
}
