package runner

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	intakeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/intake/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/stackitcloud/stackit-sdk-go/services/intake"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource = &runnerDataSource{}
)

// NewRunnerDataSource is a helper function to simplify the provider implementation
func NewRunnerDataSource() datasource.DataSource {
	return &runnerDataSource{}
}

type runnerDataSource struct {
	client *intake.APIClient
}

func (r *runnerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intake_runner"
}

// Configure adds the provider configured client to the data source
func (r *runnerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := intakeUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Intake runner client configured for data source")
}

// Schema defines the schema for the data source
func (r *runnerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                  "Datasource for STACKIT Intake Runner.",
		"id":                    "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`runner_id`\".",
		"project_id":            "STACKIT Project ID to which the runner is associated.",
		"runner_id":             "The runner ID.",
		"name":                  "The name of the runner.",
		"description":           "The description of the runner.",
		"labels":                "User-defined labels.",
		"max_message_size_kib":  "The maximum message size in KiB.",
		"max_messages_per_hour": "The maximum number of messages per hour.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
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
			"runner_id": schema.StringAttribute{
				Description: descriptions["runner_id"],
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
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				ElementType: types.StringType,
				Computed:    true,
			},
			"max_message_size_kib": schema.Int64Attribute{
				Description: descriptions["max_message_size_kib"],
				Computed:    true,
			},
			"max_messages_per_hour": schema.Int64Attribute{
				Description: descriptions["max_messages_per_hour"],
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *runnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	runnerId := model.RunnerId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "runner_id", runnerId)

	runnerResp, err := r.client.GetIntakeRunner(ctx, projectId, region, runnerId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading runner", fmt.Sprintf("Runner with ID %s not found in project %s and region %s", runnerId, projectId, region))
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading runner", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(runnerResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading runner", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Intake runner read")
}
