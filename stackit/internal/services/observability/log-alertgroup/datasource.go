package logalertgroup

import (
	"context"
	"errors"
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
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &logAlertGroupDataSource{}
)

// NewLogAlertGroupDataSource creates a new instance of the alertGroupDataSource.
func NewLogAlertGroupDataSource() datasource.DataSource {
	return &logAlertGroupDataSource{}
}

// alertGroupDataSource is the datasource implementation.
type logAlertGroupDataSource struct {
	client *observability.APIClient
}

// Configure adds the provider configured client to the resource.
func (l *logAlertGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *observability.APIClient
	var err error
	if providerData.ObservabilityCustomEndpoint != "" {
		apiClient, err = observability.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ObservabilityCustomEndpoint),
		)
	} else {
		apiClient, err = observability.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.GetRegion()),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}
	l.client = apiClient
	tflog.Info(ctx, "Observability log alert group client configured")
}

// Metadata provides metadata for the log alert group datasource.
func (l *logAlertGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_logalertgroup"
}

// Schema defines the schema for the log alert group data source.
func (l *logAlertGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Observability log alert group datasource schema. Used to create alerts based on logs (Loki). Must have a `region` specified in the provider configuration.",
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
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"interval": schema.StringAttribute{
				Description: descriptions["interval"],
				Computed:    true,
				Validators: []validator.String{
					validate.ValidDurationString(),
				},
			},
			"rules": schema.ListNestedAttribute{
				Description: descriptions["rules"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"alert": schema.StringAttribute{
							Description: descriptions["alert"],
							Computed:    true,
						},
						"expression": schema.StringAttribute{
							Description: descriptions["expression"],
							Computed:    true,
						},
						"for": schema.StringAttribute{
							Description: descriptions["for"],
							Computed:    true,
						},
						"labels": schema.MapAttribute{
							Description: descriptions["labels"],
							ElementType: types.StringType,
							Computed:    true,
						},
						"annotations": schema.MapAttribute{
							Description: descriptions["annotations"],
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (l *logAlertGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	alertGroupName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "log_alert_group_name", alertGroupName)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	readAlertGroupResp, err := l.client.GetLogsAlertgroup(ctx, alertGroupName, instanceId, projectId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading log alert group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, readAlertGroupResp.Data, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading log alert group", fmt.Sprintf("Error processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
