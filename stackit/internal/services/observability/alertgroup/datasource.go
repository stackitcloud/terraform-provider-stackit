package alertgroup

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	observabilityUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &alertGroupDataSource{}
)

// NewAlertGroupDataSource creates a new instance of the alertGroupDataSource.
func NewAlertGroupDataSource() datasource.DataSource {
	return &alertGroupDataSource{}
}

// alertGroupDataSource is the datasource implementation.
type alertGroupDataSource struct {
	client *observability.APIClient
}

// Configure adds the provider configured client to the resource.
func (a *alertGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := observabilityUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	a.client = apiClient
	tflog.Info(ctx, "Observability alert group client configured")
}

// Metadata provides metadata for the alert group datasource.
func (a *alertGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_alertgroup"
}

// Schema defines the schema for the alert group data source.
func (a *alertGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Observability alert group datasource schema. Used to create alerts based on metrics (Thanos). Uses the `default_region` specified in the provider configuration as a fallback in case no `region` is defined on resource level.",
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

func (a *alertGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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
	ctx = tflog.SetField(ctx, "alert_group_name", alertGroupName)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	readAlertGroupResp, err := a.client.GetAlertgroup(ctx, alertGroupName, instanceId, projectId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading alert group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, readAlertGroupResp.Data, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading alert group", fmt.Sprintf("Error processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
