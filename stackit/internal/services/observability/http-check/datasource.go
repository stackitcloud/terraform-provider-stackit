package httpcheck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	observabilityUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &httpCheckDataSource{}
)

// NewHttpCheckDataSource creates a new instance of the httpCheckDataSource.
func NewHttpCheckDataSource() datasource.DataSource {
	return &httpCheckDataSource{}
}

// httpCheckDataSource is the datasource implementation.
type httpCheckDataSource struct {
	client *observability.APIClient
}

// Configure adds the provider configured client to the resource.
func (d *httpCheckDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_observability_http_check", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	d.client = observabilityUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability client configured")
}

// Metadata provides metadata for the alert group datasource.
func (d *httpCheckDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_http_check"
}

// Schema defines the schema for the alert group data source.
func (d *httpCheckDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Datasource for managing HTTP-checks in STACKIT Observability. It ships Telegraf HTTP response metrics to the observability instance, as documented here: `https://docs.influxdata.com/telegraf/v1/input-plugins/http_response/`",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"http_check_id": schema.StringAttribute{
				Description: descriptions["http_check_id"],
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: descriptions["url"],
				Computed:    true,
			},
		},
	}
}

func (d *httpCheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	httpCheckId := model.HttpCheckId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "http_check_id", httpCheckId)

	listHttpCheck, err := d.client.ListHttpChecks(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing http-checks", fmt.Sprintf("Listing API payload: %v", err))
		return
	}

	if listHttpCheck.HttpChecks == nil || len(*listHttpCheck.HttpChecks) == 0 {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing http-checks", "Response is empty")
		return
	}

	for _, httpCheck := range *listHttpCheck.HttpChecks {
		if httpCheck.Id != nil && *httpCheck.Id == httpCheckId {
			if err := mapFields(ctx, &httpCheck, &model); err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading http-check", "Unable to map http-check model")
				return
			}
			break
		}
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "http-check read")
}
