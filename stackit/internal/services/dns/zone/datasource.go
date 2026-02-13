package dns

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	dnsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/dns/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
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

// ConfigValidators validates the resource configuration
func (d *zoneDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("zone_id"),
			path.MatchRoot("dns_name"),
		),
	}
}

func (d *zoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := dnsUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
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
				Optional:    true,
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
				Description: "The zone name. E.g. `example.com` (must not end with a trailing dot).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(253),
					stringvalidator.RegexMatches(
						dnsNameNoTrailingDotRegex,
						"dns_name must not end with a trailing dot",
					),
				},
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

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	zoneId := model.ZoneId.ValueString()
	dnsName := model.DnsName.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "zone_id", zoneId)
	ctx = tflog.SetField(ctx, "dns_name", dnsName)

	var zoneResp *dns.ZoneResponse
	var err error

	if zoneId != "" {
		zoneResp, err = d.client.GetZone(ctx, projectId, zoneId).Execute()
		if err != nil {
			utils.LogError(
				ctx,
				&resp.Diagnostics,
				err,
				"Reading zone",
				fmt.Sprintf("Zone with ID %q does not exist in project %q.", zoneId, projectId),
				map[int]string{
					http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
				},
			)
			resp.State.RemoveResource(ctx)
			return
		}

		ctx = core.LogResponse(ctx)
	} else {
		listZoneResp, err := d.client.ListZones(ctx, projectId).
			DnsNameEq(dnsName).
			ActiveEq(true).
			Execute()
		if err != nil {
			utils.LogError(
				ctx,
				&resp.Diagnostics,
				err,
				"Reading zone",
				fmt.Sprintf("Zone with DNS name %q does not exist in project %q.", dnsName, projectId),
				map[int]string{
					http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
				},
			)
			resp.State.RemoveResource(ctx)
			return
		}

		ctx = core.LogResponse(ctx)

		if *listZoneResp.TotalItems != 1 {
			utils.LogError(
				ctx,
				&resp.Diagnostics,
				fmt.Errorf("zone with DNS name %q does not exist in project %q", dnsName, projectId),
				"Reading zone",
				fmt.Sprintf("Zone with DNS name %q does not exist in project %q.", dnsName, projectId),
				nil,
			)
			resp.State.RemoveResource(ctx)
			return
		}
		zones := *listZoneResp.Zones
		zoneResp = dns.NewZoneResponse(zones[0])
	}

	if zoneResp != nil && zoneResp.Zone.State != nil && *zoneResp.Zone.State == dns.ZONESTATE_DELETE_SUCCEEDED {
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
