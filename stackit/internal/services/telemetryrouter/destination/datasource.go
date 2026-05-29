package destination

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/telemetryrouter/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &telemetryRouterDestinationDataSource{}
)

func NewTelemetryRouterDestinationDataSource() datasource.DataSource {
	return &telemetryRouterDestinationDataSource{}
}

type DatasourceModel struct {
	ID             types.String `tfsdk:"id"` // Required by Terraform
	InstanceID     types.String `tfsdk:"instance_id"`
	DestinationID  types.String `tfsdk:"destination_id"`
	Region         types.String `tfsdk:"region"`
	ProjectID      types.String `tfsdk:"project_id"`
	DisplayName    types.String `tfsdk:"display_name"`
	Description    types.String `tfsdk:"description"`
	Config         types.Object `tfsdk:"config"`
	CreationTime   types.String `tfsdk:"creation_time"`
	CredentialType types.String `tfsdk:"credential_type"`
	Status         types.String `tfsdk:"status"`
}

// Struct corresponding to DatasourceModel.Config
type datasourceConfig struct {
	ConfigType    types.String `tfsdk:"config_type"`
	Filter        types.Object `tfsdk:"filter"`
	OpenTelemetry types.Object `tfsdk:"opentelemetry"`
	S3            types.Object `tfsdk:"s3"`
}

// Types corresponding to datasourceConfig
var datasourceConfigTypes = map[string]attr.Type{
	"config_type":   basetypes.StringType{},
	"filter":        basetypes.ObjectType{AttrTypes: datasourceFilterTypes},
	"opentelemetry": basetypes.ObjectType{AttrTypes: datasourceOpenTelemetryTypes},
	"s3":            basetypes.ObjectType{AttrTypes: datasourceS3Types},
}

// Struct corresponding to datasourceFilter
type datasourceFilter struct {
	Attributes types.List `tfsdk:"attributes"`
}

// Types corresponding to datasourceFilter
var datasourceFilterTypes = map[string]attr.Type{
	"attributes": basetypes.ListType{ElemType: types.ObjectType{AttrTypes: datasourceAttributeTypes}},
}

// Struct corresponding to a single attribute
type datasourceAttribute struct {
	Key     types.String `tfsdk:"key"`
	Level   types.String `tfsdk:"level"`
	Matcher types.String `tfsdk:"matcher"`
	Values  types.List   `tfsdk:"values"`
}

// Types corresponding to attributes
var datasourceAttributeTypes = map[string]attr.Type{
	"key":     basetypes.StringType{},
	"level":   basetypes.StringType{},
	"matcher": basetypes.StringType{},
	"values":  basetypes.ListType{ElemType: types.StringType},
}

// Struct corresponding to opentelemetry
type datasourceOpenTelemetry struct {
	Uri types.String `tfsdk:"uri"`
}

// Types corresponding to opentelemetry
var datasourceOpenTelemetryTypes = map[string]attr.Type{
	"uri": basetypes.StringType{},
}

// Struct corresponding to s3
type datasourceS3 struct {
	Bucket   types.String `tfsdk:"bucket"`
	Endpoint types.String `tfsdk:"endpoint"`
}

// Types corresponding to s3
var datasourceS3Types = map[string]attr.Type{
	"bucket":   basetypes.StringType{},
	"endpoint": basetypes.StringType{},
}

type telemetryRouterDestinationDataSource struct {
	client       *telemetryrouter.APIClient
	providerData core.ProviderData
}

func (d *telemetryRouterDestinationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_telemetryrouter_destination"
}

func (d *telemetryRouterDestinationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.providerData = providerData

	apiClient := utils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "TelemetryRouter client configured")
}

func (d *telemetryRouterDestinationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("TelemetryRouter destination data source schema. %s", core.DatasourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
			},
			"destination_id": schema.StringAttribute{
				Description: schemaDescriptions["destination_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Computed:    true,
			},
			"config": schema.SingleNestedAttribute{
				Description: schemaDescriptions["config"],
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"config_type": schema.StringAttribute{
						Description: schemaDescriptions["config.config_type"],
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("OpenTelemetry", "S3"),
						},
					},
					"filter": schema.SingleNestedAttribute{
						Description: schemaDescriptions["config.filter"],
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"attributes": schema.ListNestedAttribute{
								Description: schemaDescriptions["config.filter.attributes"],
								Computed:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"key": schema.StringAttribute{
											Description: schemaDescriptions["config.filter.attributes.key"],
											Computed:    true,
										},
										"level": schema.StringAttribute{
											Description: schemaDescriptions["config.filter.attributes.level"],
											Computed:    true,
											Validators: []validator.String{
												stringvalidator.OneOf("resource", "scope", "logRecord"),
											},
										},
										"matcher": schema.StringAttribute{
											Description: schemaDescriptions["config.filter.attributes.matcher"],
											Computed:    true,
											Validators: []validator.String{
												stringvalidator.OneOf("=", "!="),
											},
										},
										"values": schema.ListAttribute{
											Description: schemaDescriptions["config.filter.attributes.values"],
											ElementType: types.StringType,
											Computed:    true,
										},
									},
								},
							},
						},
					},
					"opentelemetry": schema.SingleNestedAttribute{
						Description: schemaDescriptions["config.opentelemetry"],
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"uri": schema.StringAttribute{
								Description: schemaDescriptions["config.opentelemetry.uri"],
								Computed:    true,
							},
						},
					},
					"s3": schema.SingleNestedAttribute{
						Description: schemaDescriptions["config.s3"],
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"bucket": schema.StringAttribute{
								Description: schemaDescriptions["config.s3.bucket"],
								Computed:    true,
							},
							"endpoint": schema.StringAttribute{
								Description: schemaDescriptions["config.s3.endpoint"],
								Computed:    true,
							},
						},
					},
				},
			},
			"creation_time": schema.StringAttribute{
				Description: schemaDescriptions["creation_time"],
				Computed:    true,
			},
			"credential_type": schema.StringAttribute{
				Description: schemaDescriptions["credential_type"],
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
		},
	}
}

func (d *telemetryRouterDestinationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DatasourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()
	destinationID := model.DestinationID.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "destination_id", destinationID)

	destinationResponse, err := d.client.DefaultAPI.GetDestination(ctx, projectID, region, instanceID, destinationID).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		tfutils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Error reading TelemetryRouter destination",
			fmt.Sprintf("Destination with ID %q does not exist in project %q.", destinationID, projectID),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectID),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapDatasourceFields(ctx, destinationResponse, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryRouter destination", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter Destination read", map[string]interface{}{
		"instance_id":    instanceID,
		"destination_id": destinationID,
	})
}

func mapDatasourceFields(ctx context.Context, destination *telemetryrouter.DestinationResponse, model *DatasourceModel) error {
	if destination == nil {
		return fmt.Errorf("destination is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	var destinationID string
	if model.DestinationID.ValueString() != "" {
		destinationID = model.DestinationID.ValueString()
	} else if destination.Id != "" {
		destinationID = destination.Id
	} else {
		return fmt.Errorf("destination id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), model.Region.ValueString(), model.InstanceID.ValueString(), destinationID)
	model.DestinationID = types.StringValue(destinationID)
	model.DisplayName = types.StringValue(destination.DisplayName)
	model.Description = types.StringPointerValue(destination.Description)
	model.CredentialType = types.StringValue(destination.CredentialType)
	model.CreationTime = types.StringValue(destination.CreationTime.Format(time.RFC3339))
	model.Status = types.StringValue(destination.Status)

	if err := mapDatasourceConfig(ctx, destination, model); err != nil {
		return fmt.Errorf("map config: %w", err)
	}

	return nil
}

func mapDatasourceConfig(ctx context.Context, destination *telemetryrouter.DestinationResponse, model *DatasourceModel) error {
	var conf datasourceConfig

	conf.ConfigType = types.StringValue(string(destination.Config.ConfigType))

	if err := mapDatasourceFilter(ctx, &destination.Config, &conf); err != nil {
		return err
	}

	if err := mapDatasourceOpenTelemetry(ctx, &destination.Config, &conf); err != nil {
		return err
	}

	if err := mapDatasourceS3(ctx, &destination.Config, &conf); err != nil {
		return err
	}

	configValue, diags := types.ObjectValueFrom(ctx, datasourceConfigTypes, conf)
	if diags.HasError() {
		return fmt.Errorf("mapping config: %w", core.DiagsToError(diags))
	}
	model.Config = configValue

	return nil
}

func mapDatasourceFilter(ctx context.Context, apiConf *telemetryrouter.DestinationConfig, conf *datasourceConfig) error {
	if apiConf.Filter == nil {
		conf.Filter = types.ObjectNull(datasourceFilterTypes)
		return nil
	}

	attrList := []attr.Value{}
	for _, currentAttr := range apiConf.Filter.Attributes {
		values, diags := types.ListValueFrom(ctx, types.StringType, currentAttr.Values)
		if diags.HasError() {
			return fmt.Errorf("mapping filter values: %w", core.DiagsToError(diags))
		}
		attrModel, diags := types.ObjectValueFrom(ctx, datasourceAttributeTypes, datasourceAttribute{
			Key:     types.StringValue(currentAttr.Key),
			Level:   types.StringValue(string(currentAttr.Level)),
			Matcher: types.StringValue(string(currentAttr.Matcher)),
			Values:  values,
		})
		if diags.HasError() {
			return fmt.Errorf("mapping filter config: %w", core.DiagsToError(diags))
		}
		attrList = append(attrList, attrModel)
	}

	var attrConfigs basetypes.ListValue
	var diags diag.Diagnostics
	if len(attrList) == 0 {
		attrConfigs = types.ListNull(types.ObjectType{AttrTypes: datasourceAttributeTypes})
	} else {
		attrConfigs, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: datasourceAttributeTypes}, attrList)
		if diags.HasError() {
			return fmt.Errorf("mapping attributes: %w", core.DiagsToError(diags))
		}
	}

	filterValue, diags := types.ObjectValueFrom(ctx, datasourceFilterTypes, datasourceFilter{
		Attributes: attrConfigs,
	})
	if diags.HasError() {
		return fmt.Errorf("mapping filter: %w", core.DiagsToError(diags))
	}
	conf.Filter = filterValue

	return nil
}

func mapDatasourceOpenTelemetry(ctx context.Context, apiConf *telemetryrouter.DestinationConfig, conf *datasourceConfig) error {
	if apiConf.OpenTelemetry == nil {
		conf.OpenTelemetry = types.ObjectNull(datasourceOpenTelemetryTypes)
		return nil
	}

	var ot datasourceOpenTelemetry
	ot.Uri = types.StringValue(apiConf.OpenTelemetry.Uri)

	otModel, diags := types.ObjectValueFrom(ctx, datasourceOpenTelemetryTypes, ot)
	if diags.HasError() {
		return fmt.Errorf("mapping open telemetry: %w", core.DiagsToError(diags))
	}

	conf.OpenTelemetry = otModel

	return nil
}

func mapDatasourceS3(ctx context.Context, apiConf *telemetryrouter.DestinationConfig, conf *datasourceConfig) error {
	if apiConf.S3 == nil {
		conf.S3 = types.ObjectNull(datasourceS3Types)
		return nil
	}

	var s3Struct datasourceS3
	s3Struct.Bucket = types.StringValue(apiConf.S3.Bucket)
	s3Struct.Endpoint = types.StringValue(apiConf.S3.Endpoint)

	s3Model, diags := types.ObjectValueFrom(ctx, datasourceS3Types, s3Struct)
	if diags.HasError() {
		return fmt.Errorf("mapping s3: %w", core.DiagsToError(diags))
	}

	conf.S3 = s3Model

	return nil
}
