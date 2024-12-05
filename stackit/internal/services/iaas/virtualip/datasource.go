package virtualip

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// virtualIPDataSourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var virtualIPDataSourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &virtualIPDataSource{}
)

// NewVirtualIPDataSource is a helper function to simplify the provider implementation.
func NewVirtualIPDataSource() datasource.DataSource {
	return &virtualIPDataSource{}
}

// networkDataSource is the data source implementation.
type virtualIPDataSource struct {
	client *iaasalpha.APIClient
}

// Metadata returns the data source type name.
func (d *virtualIPDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_ip"
}

func (d *virtualIPDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var apiClient *iaasalpha.APIClient
	var err error

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !virtualIPDataSourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_virtual_ip", "data source")
		if resp.Diagnostics.HasError() {
			return
		}
		virtualIPDataSourceBetaCheckDone = true
	}

	if providerData.IaaSCustomEndpoint != "" {
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaasalpha.NewAPIClient(
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
func (d *virtualIPDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Virtual IP data resource schema. Must have a `region` specified in the provider configuration.",
		MarkdownDescription: features.AddBetaDescription("Virtual IP data source schema. Must have a `region` specified in the provider configuration."),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source ID. It is structured as \"`project_id`,`network_id`,`virtual_ip_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT organization ID to which the network area is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_id": schema.StringAttribute{
				Description: "The network area ID to which the virtual IP is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"virtual_ip_id": schema.StringAttribute{
				Description: "The virtual IP ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of virtual IP.",
				Computed:    true,
			},
			"ip": schema.StringAttribute{
				Description: "The IP address of the virtual IP.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *virtualIPDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	virtualIpId := model.VirtualIpId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "virtual_ip_id", virtualIpId)

	virtualIpResp, err := d.client.GetVirtualIP(ctx, projectId, networkId, virtualIpId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading virtual IP", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, virtualIpResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading virtual IP", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Virtual IP read")
}
