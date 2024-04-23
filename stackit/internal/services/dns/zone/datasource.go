package dns

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stackitcloud/stackit-sdk-go/services/dns/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &zoneDataSource{}
)

// NewZoneDataSource is a helper function to simplify the provider implementation.
func NewZoneDataSource() datasource.DataSource {
	return &zoneDataSource{}
}

// zoneDataSource is the data source implementation.
type zoneDataSource struct {
	client *dns.APIClient
}

// Metadata returns the data source type name.
func (d *zoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

func (d *zoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var apiClient *dns.APIClient
	var err error

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if providerData.DnsCustomEndpoint != "" {
		apiClient, err = dns.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.DnsCustomEndpoint),
		)
	} else {
		apiClient, err = dns.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	d.client = apiClient
	tflog.Info(ctx, "DNS zone client configured")
}

// Schema defines the schema for the data source.
func (d *zoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "DNS Zone resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`zone_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the dns zone is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"zone_id": schema.StringAttribute{
				Description: "The zone ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The user given name of the zone.",
				Computed:    true,
			},
			"dns_name": schema.StringAttribute{
				Description: "The zone name. E.g. `example.com`",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the zone.",
				Computed:    true,
			},
			"acl": schema.StringAttribute{
				Description: "The access control list.",
				Computed:    true,
			},
			"active": schema.BoolAttribute{
				Description: "",
				Computed:    true,
			},
			"contact_email": schema.StringAttribute{
				Description: "A contact e-mail for the zone.",
				Computed:    true,
			},
			"default_ttl": schema.Int64Attribute{
				Description: "Default time to live.",
				Computed:    true,
			},
			"expire_time": schema.Int64Attribute{
				Description: "Expire time.",
				Computed:    true,
			},
			"is_reverse_zone": schema.BoolAttribute{
				Description: "Specifies, if the zone is a reverse zone or not.",
				Computed:    true,
			},
			"negative_cache": schema.Int64Attribute{
				Description: "Negative caching.",
				Computed:    true,
			},
			"primary_name_server": schema.StringAttribute{
				Description: "Primary name server. FQDN.",
				Computed:    true,
			},
			"primaries": schema.ListAttribute{
				Description: `Primary name server for secondary zone.`,
				Computed:    true,
				ElementType: types.StringType,
			},
			"record_count": schema.Int64Attribute{
				Description: "Record count how many records are in the zone.",
				Computed:    true,
			},
			"refresh_time": schema.Int64Attribute{
				Description: "Refresh time.",
				Computed:    true,
			},
			"retry_time": schema.Int64Attribute{
				Description: "Retry time.",
				Computed:    true,
			},
			"serial_number": schema.Int64Attribute{
				Description: "Serial number.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Zone type.",
				Computed:    true,
			},
			"visibility": schema.StringAttribute{
				Description: "Visibility of the zone.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "Zone state.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *zoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	zoneId := model.ZoneId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "zone_id", zoneId)

	zoneResp, err := d.client.GetZone(ctx, projectId, zoneId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading zone", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if zoneResp != nil && zoneResp.Zone.State != nil && *zoneResp.Zone.State == wait.DeleteSuccess {
		resp.State.RemoveResource(ctx)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading zone", "Zone was deleted successfully")
		return
	}

	err = mapFields(ctx, zoneResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading zone", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "DNS zone read")
}
