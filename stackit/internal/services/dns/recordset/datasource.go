package dns

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
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stackitcloud/stackit-sdk-go/services/dns/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &recordSetDataSource{}
)

// NewRecordSetDataSource NewZoneDataSource is a helper function to simplify the provider implementation.
func NewRecordSetDataSource() datasource.DataSource {
	return &recordSetDataSource{}
}

// recordSetDataSource is the data source implementation.
type recordSetDataSource struct {
	client *dns.APIClient
}

// Metadata returns the data source type name.
func (d *recordSetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record_set"
}

// Configure adds the provider configured client to the data source.
func (d *recordSetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *dns.APIClient
	var err error
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
	tflog.Info(ctx, "DNS record set client configured")
}

// Schema defines the schema for the data source.
func (d *recordSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "DNS Record Set Resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`zone_id`,`record_set_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the dns record set is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"zone_id": schema.StringAttribute{
				Description: "The zone ID to which is dns record set is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"record_set_id": schema.StringAttribute{
				Description: "The rr set id.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the record which should be a valid domain according to rfc1035 Section 2.3.4. E.g. `example.com`",
				Computed:    true,
			},
			"fqdn": schema.StringAttribute{
				Description: "Fully qualified domain name (FQDN) of the record set.",
				Computed:    true,
			},
			"records": schema.ListAttribute{
				Description: "Records.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ttl": schema.Int64Attribute{
				Description: "Time to live. E.g. 3600",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The record set type. E.g. `A` or `CNAME`",
				Computed:    true,
			},
			"active": schema.BoolAttribute{
				Description: "Specifies if the record set is active or not.",
				Computed:    true,
			},
			"comment": schema.StringAttribute{
				Description: "Comment.",
				Computed:    true,
			},
			"error": schema.StringAttribute{
				Description: "Error shows error in case create/update/delete failed.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "Record set state.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *recordSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	zoneId := model.ZoneId.ValueString()
	recordSetId := model.RecordSetId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "zone_id", zoneId)
	ctx = tflog.SetField(ctx, "record_set_id", recordSetId)
	recordSetResp, err := d.client.GetRecordSet(ctx, projectId, zoneId, recordSetId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading record set",
			fmt.Sprintf("The record set %q or zone %q does not exist in project %q.", recordSetId, zoneId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	if recordSetResp != nil && recordSetResp.Rrset.State != nil && *recordSetResp.Rrset.State == wait.DeleteSuccess {
		resp.State.RemoveResource(ctx)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading record set", "Record set was deleted successfully")
		return
	}

	err = mapFields(ctx, recordSetResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading record set", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "DNS record set read")
}
