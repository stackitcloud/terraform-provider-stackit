package enablement

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"

	serviceenablementUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceenablement/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &serviceEnablementDatasource{}
	_ datasource.DataSourceWithConfigure = &serviceEnablementDatasource{}
)

type DatasourceModel struct {
	ResourceModel
	Enabled types.Bool `tfsdk:"enabled"`
}

func NewServiceEnablementDataSource() datasource.DataSource {
	return &serviceEnablementDatasource{}
}

type serviceEnablementDatasource struct {
	providerData core.ProviderData
	client       serviceenablement.DefaultAPI
}

func (d *serviceEnablementDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	d.providerData = providerData
	d.client = clients.ServiceEnablementV2Client

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_service_enablement", core.Datasource)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "service enablement client configured")
}

func (d *serviceEnablementDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_enablement"
}

func (d *serviceEnablementDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := features.AddBetaDescription("Service enablement datasource schema.", core.Datasource)

	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal datasource identifier. It is structured as \"`project_id`,`region`,`service_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			// IMPORTANT: see note above at definition of 'allowedServiceIdsByRegion' variable
			"service_id": schema.StringAttribute{
				Description: fmt.Sprintf("The identifier of the STACKIT service. %s", utils.FormatPossibleValues(allowedServiceIds()...)),
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedServiceIds()...),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				// the region cannot be found, so it has to be passed
				Optional: true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether or not the service is enabled in that region.",
				Computed:    true,
			},
		},
	}
}

func (d *serviceEnablementDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	serviceId := model.ServiceId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", projectId)
	ctx = tflog.SetField(ctx, "service_id", serviceId)

	ctx = core.InitProviderContext(ctx)

	// Validation of the combination of service id and region has to be done here in Read for the datasource
	// since there is nothing like 'ModifyPlan' for datasources. Both the provider data (region overwrite)
	// and the datasource configuration are needed to do the validation.
	errorDiag := serviceenablementUtils.ValidateServiceIdAndRegionCombination(allowedServiceIdsByRegion, serviceId, region)
	if errorDiag != nil {
		resp.Diagnostics.Append(errorDiag)
		return
	}

	serviceEnablementResp, err := d.client.GetServiceStatusRegional(ctx, region, projectId, serviceId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service enablement", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapDatasourceFields(&model, serviceEnablementResp, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service enablement", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "service enablement read")
}

func mapDatasourceFields(model *DatasourceModel, resp *serviceenablement.ServiceStatus, region string) error {
	if model == nil {
		return fmt.Errorf("nil model")
	} else if resp == nil {
		return fmt.Errorf("nil response")
	} else if resp.State == nil {
		return fmt.Errorf("nil state")
	}

	err := mapFields(&model.ResourceModel, resp, region)
	if err != nil {
		return err
	}

	enabledStates := []serviceenablement.ServiceStatusState{
		serviceenablement.SERVICESTATUSSTATE_ENABLED,
		serviceenablement.SERVICESTATUSSTATE_ENABLING,
	}

	model.Enabled = types.BoolValue(slices.Contains(enabledStates, *resp.State))

	return nil
}
