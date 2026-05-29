package destination

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/telemetryrouter/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &telemetryRouterDestinationResource{}
	_ resource.ResourceWithConfigure   = &telemetryRouterDestinationResource{}
	_ resource.ResourceWithImportState = &telemetryRouterDestinationResource{}
	_ resource.ResourceWithModifyPlan  = &telemetryRouterDestinationResource{}
)

//nolint:gosec // there is no G101: Potential hardcoded credentials
var schemaDescriptions = map[string]string{
	"id":                           "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`,`destionation_id`\".",
	"instance_id":                  "The TelemetryRouter instance ID",
	"destination_id":               "The TelemetryRouter destination ID",
	"region":                       "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"project_id":                   "STACKIT project ID associated with the TelemetryRouter instance",
	"display_name":                 "The displayed name of the TelemetryRouter destination",
	"description":                  "The description of the TelemetryRouter destination",
	"config":                       "The configuration of the TelemetryRouter destination",
	"config.filter":                "The TelemetryRouter destination's filter settings",
	"config.filter.attributes":     "The TelemetryRouter destination's filter attributes",
	"config.filter.attributes.key": "The TelemetryRouter destination's filter attribute key",
	"config.filter.attributes.level": fmt.Sprintf(
		"The TelemetryRouter destination's filter attribute level, possible values: %s",
		tfutils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(telemetryrouter.AllowedConfigFilterLevelEnumValues)...),
	),
	"config.filter.attributes.matcher": fmt.Sprintf(
		"The TelemetryRouter destination's filter attribute matcher, possible values: %s",
		tfutils.FormatPossibleValues("=", "!="),
	),
	"config.filter.attributes.values": "The TelemetryRouter destination's filter attribute values",
	"config.config_type": fmt.Sprintf(
		"The TelemetryRouter destinations's configuration type, possible values: %s",
		tfutils.FormatPossibleValues(
			string(telemetryrouter.DESTINATIONCONFIGTYPE_OPEN_TELEMETRY),
			string(telemetryrouter.DESTINATIONCONFIGTYPE_S3),
		),
	),
	"config.opentelemetry":                     "OpenTelemetry configuration",
	"config.opentelemetry.basic_auth":          "OpenTelemetry basic auth configuration",
	"config.opentelemetry.basic_auth.username": "OpenTelemetry basic auth username",
	"config.opentelemetry.basic_auth.password": "OpenTelemetry basic auth password",
	"config.opentelemetry.bearer_token":        "OpenTelemetry bearer token",
	"config.opentelemetry.uri":                 "OpenTelemetry destination URI",
	"config.s3":                                "S3 configuration",
	"config.s3.access_key":                     "S3 access key configuration",
	"config.s3.access_key.id":                  "S3 access key ID",
	"config.s3.access_key.secret":              "S3 access key secret",
	"config.s3.bucket":                         "S3 bucket name",
	"config.s3.endpoint":                       "S3 endpoint",
	"creation_time":                            "The date and time the creation of the TelemetryRouter destination was initiated",
	"credential_type":                          "The TelemetryRouter destination's credential type",
	"status": fmt.Sprintf(
		"The status of the TelemetryRouter destination, possible values: %s",
		tfutils.FormatPossibleValues("active", "deleting", "reconciling"),
	),
}

type Model struct {
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

// Struct corresponding to Model.Config
type config struct {
	ConfigType    types.String `tfsdk:"config_type"`
	Filter        types.Object `tfsdk:"filter"`
	OpenTelemetry types.Object `tfsdk:"opentelemetry"`
	S3            types.Object `tfsdk:"s3"`
}

// Types corresponding to config
var configTypes = map[string]attr.Type{
	"config_type":   basetypes.StringType{},
	"filter":        basetypes.ObjectType{AttrTypes: filterTypes},
	"opentelemetry": basetypes.ObjectType{AttrTypes: openTelemetryTypes},
	"s3":            basetypes.ObjectType{AttrTypes: s3Types},
}

// Struct corresponding to filter
type filter struct {
	Attributes types.List `tfsdk:"attributes"`
}

// Types corresponding to filter
var filterTypes = map[string]attr.Type{
	"attributes": basetypes.ListType{ElemType: types.ObjectType{AttrTypes: attributeTypes}},
}

// Struct corresponding to a single attribute
type attribute struct {
	Key     types.String `tfsdk:"key"`
	Level   types.String `tfsdk:"level"`
	Matcher types.String `tfsdk:"matcher"`
	Values  types.List   `tfsdk:"values"`
}

// Types corresponding to attributes
var attributeTypes = map[string]attr.Type{
	"key":     basetypes.StringType{},
	"level":   basetypes.StringType{},
	"matcher": basetypes.StringType{},
	"values":  basetypes.ListType{ElemType: types.StringType},
}

// Struct corresponding to opentelemetry
type openTelemetry struct {
	BasicAuth   types.Object `tfsdk:"basic_auth"`
	BearerToken types.String `tfsdk:"bearer_token"`
	Uri         types.String `tfsdk:"uri"`
}

// Types corresponding to opentelemetry
var openTelemetryTypes = map[string]attr.Type{
	"basic_auth":   basetypes.ObjectType{AttrTypes: basicAuthTypes},
	"bearer_token": basetypes.StringType{},
	"uri":          basetypes.StringType{},
}

// Struct corresponding to s3
type s3 struct {
	AccessKey types.Object `tfsdk:"access_key"`
	Bucket    types.String `tfsdk:"bucket"`
	Endpoint  types.String `tfsdk:"endpoint"`
}

// Types corresponding to s3
var s3Types = map[string]attr.Type{
	"access_key": basetypes.ObjectType{AttrTypes: accessKeyTypes},
	"bucket":     basetypes.StringType{},
	"endpoint":   basetypes.StringType{},
}

// Struct corresponding to basicAuth
type basicAuth struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

// Types corresponding to filter
var basicAuthTypes = map[string]attr.Type{
	"username": basetypes.StringType{},
	"password": basetypes.StringType{},
}

// Struct corresponding to accessKey
type accessKey struct {
	ID     types.String `tfsdk:"id"`
	Secret types.String `tfsdk:"secret"`
}

// Types corresponding to filter
var accessKeyTypes = map[string]attr.Type{
	"id":     basetypes.StringType{},
	"secret": basetypes.StringType{},
}

type telemetryRouterDestinationResource struct {
	client       *telemetryrouter.APIClient
	providerData core.ProviderData
}

func NewTelemetryRouterDestinationResource() resource.Resource {
	return &telemetryRouterDestinationResource{}
}

func (r *telemetryRouterDestinationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	r.providerData = providerData
	tflog.Info(ctx, "TelemetryRouter client configured")
}

func (r *telemetryRouterDestinationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tfutils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *telemetryRouterDestinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_telemetryrouter_destination"
}

func (r *telemetryRouterDestinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("TelemetryRouter destination resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"destination_id": schema.StringAttribute{
				Description: schemaDescriptions["destination_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Required:    true,
			},
			"config": schema.SingleNestedAttribute{
				Description: schemaDescriptions["config"],
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"config_type": schema.StringAttribute{
						Description: schemaDescriptions["config.config_type"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(telemetryrouter.DESTINATIONCONFIGTYPE_OPEN_TELEMETRY),
								string(telemetryrouter.DESTINATIONCONFIGTYPE_S3),
							),
						},
					},
					"filter": schema.SingleNestedAttribute{
						Description: schemaDescriptions["config.filter"],
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"attributes": schema.ListNestedAttribute{
								Description: schemaDescriptions["config.filter.attributes"],
								Required:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"key": schema.StringAttribute{
											Description: schemaDescriptions["config.filter.attributes.key"],
											Required:    true,
										},
										"level": schema.StringAttribute{
											Description: schemaDescriptions["config.filter.attributes.level"],
											Required:    true,
											Validators: []validator.String{
												stringvalidator.OneOf(sdkUtils.EnumSliceToStringSlice(telemetryrouter.AllowedConfigFilterLevelEnumValues)...),
											},
										},
										"matcher": schema.StringAttribute{
											Description: schemaDescriptions["config.filter.attributes.matcher"],
											Required:    true,
											Validators: []validator.String{
												stringvalidator.OneOf(sdkUtils.EnumSliceToStringSlice(telemetryrouter.AllowedConfigFilterMatcherEnumValues)...),
											},
										},
										"values": schema.ListAttribute{
											Description: schemaDescriptions["config.filter.attributes.values"],
											ElementType: types.StringType,
											Required:    true,
										},
									},
								},
							},
						},
					},
					"opentelemetry": schema.SingleNestedAttribute{
						Description: schemaDescriptions["config.opentelemetry"],
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"basic_auth": schema.SingleNestedAttribute{
								Description: schemaDescriptions["config.opentelemetry.basic_auth"],
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"username": schema.StringAttribute{
										Description: schemaDescriptions["config.opentelemetry.basic_auth.username"],
										Required:    true,
									},
									"password": schema.StringAttribute{
										Description: schemaDescriptions["config.opentelemetry.basic_auth.username"],
										Required:    true,
										Sensitive:   true,
									},
								},
							},
							"bearer_token": schema.StringAttribute{
								Description: schemaDescriptions["config.opentelemetry.bearer_token"],
								Optional:    true,
								Sensitive:   true,
							},
							"uri": schema.StringAttribute{
								Description: schemaDescriptions["config.opentelemetry.uri"],
								Required:    true,
							},
						},
					},
					"s3": schema.SingleNestedAttribute{
						Description: schemaDescriptions["config.s3"],
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"access_key": schema.SingleNestedAttribute{
								Description: schemaDescriptions["config.s3.access_key"],
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Description: schemaDescriptions["config.s3.access_key.id"],
										Required:    true,
									},
									"secret": schema.StringAttribute{
										Description: schemaDescriptions["config.s3.access_key.secret"],
										Required:    true,
										Sensitive:   true,
									},
								},
							},
							"bucket": schema.StringAttribute{
								Description: schemaDescriptions["config.s3.bucket"],
								Required:    true,
							},
							"endpoint": schema.StringAttribute{
								Description: schemaDescriptions["config.s3.endpoint"],
								Required:    true,
							},
						},
					},
				},
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Optional:    true,
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

func (r *telemetryRouterDestinationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var resourceModel Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &resourceModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// function is used in order to be able to write easier unit tests
	validateConfig(ctx, &resp.Diagnostics, &resourceModel)
}

func validateConfig(ctx context.Context, respDiags *diag.Diagnostics, model *Model) {
	conf := &config{}
	diags := model.Config.As(ctx, conf, basetypes.ObjectAsOptions{})
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return
	}

	if tfutils.IsUndefined(conf.ConfigType) {
		core.LogAndAddError(ctx, respDiags, "Error configuring destination", "config.config_type cannot be empty")
	}

	switch conf.ConfigType.ValueString() {
	case string(telemetryrouter.DESTINATIONCONFIGTYPE_OPEN_TELEMETRY):
		if !tfutils.IsUndefined(conf.S3) {
			core.LogAndAddError(
				ctx,
				respDiags,
				"Error configuring destination",
				"S3 configuration is not supported for OpenTelemetry destination",
			)
		}
		if tfutils.IsUndefined(conf.OpenTelemetry) {
			core.LogAndAddError(
				ctx,
				respDiags,
				"Error configuring destination",
				"OpenTelemetry configuration is required",
			)
		}

		ot := &openTelemetry{}
		diags = conf.OpenTelemetry.As(ctx, ot, basetypes.ObjectAsOptions{})
		respDiags.Append(diags...)
		if respDiags.HasError() {
			return
		}

		if !tfutils.IsUndefined(ot.BasicAuth) && !tfutils.IsUndefined(ot.BearerToken) {
			core.LogAndAddError(
				ctx,
				respDiags,
				"Error configuring destination",
				"Basic Auth and Bearer Token can't be used at the same time with OpenTelemetry destination",
			)
		}
	case string(telemetryrouter.DESTINATIONCONFIGTYPE_S3):
		if !tfutils.IsUndefined(conf.OpenTelemetry) {
			core.LogAndAddError(
				ctx,
				respDiags,
				"Error configuring destination",
				"OpenTelemetry configuration is not supported for S3 destination",
			)
		}
		if tfutils.IsUndefined(conf.S3) {
			core.LogAndAddError(
				ctx,
				respDiags,
				"Error configuring destination",
				"S3 configuration is required",
			)
		}
	default:
		core.LogAndAddError(ctx, respDiags, "Error configuring destination", "unknown config.config_type")
	}
}

func (r *telemetryRouterDestinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	instanceId := model.InstanceID.ValueString()
	projectId := model.ProjectID.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	payload, err := toCreatePayload(ctx, resp.Diagnostics, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter destination", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	regionId := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "region", regionId)
	createResp, err := r.client.DefaultAPI.CreateDestination(ctx, projectId, regionId, instanceId).CreateDestinationPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter destination", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if createResp == nil || createResp.Id == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter destination", "Create API response: Incomplete response (id missing)")
		return
	}

	destinationId := createResp.Id
	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":     projectId,
		"region":         region,
		"instance_id":    instanceId,
		"destination_id": destinationId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := wait.CreateDestinationWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId, createResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter destination", fmt.Sprintf("Waiting for TelemetryRouter destination to become active: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryRouter destination", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter destination created")
}

func (r *telemetryRouterDestinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := model.Region.ValueString()
	instanceID := model.InstanceID.ValueString()
	destinationID := model.DestinationID.ValueString()

	if destinationID == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "destination_id", destinationID)

	instanceResponse, err := r.client.DefaultAPI.GetDestination(ctx, projectID, region, instanceID, destinationID).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryRouter destination", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, instanceResponse, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryRouter destination", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter destination read", map[string]any{
		"instance_id":    instanceID,
		"destination_id": destinationID,
	})
}

func (r *telemetryRouterDestinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := model.Region.ValueString()
	instanceID := model.InstanceID.ValueString()
	destinationID := model.DestinationID.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "destination_id", destinationID)

	payload, err := toUpdatePayload(ctx, resp.Diagnostics, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter destination", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	updateResp, err := r.client.DefaultAPI.UpdateDestination(ctx, projectID, region, instanceID, destinationID).UpdateDestinationPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter destination", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.UpdateDestinationWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID, updateResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter destination", fmt.Sprintf("Waiting for TelemetryRouter destination to become active: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryRouter destination", fmt.Sprintf("Processing response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryRouter destination updated", map[string]any{
		"instance_id":    instanceID,
		"destination_id": destinationID,
	})
}

func (r *telemetryRouterDestinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := model.Region.ValueString()
	instanceID := model.InstanceID.ValueString()
	destinationID := model.DestinationID.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "destination_id", destinationID)

	err := r.client.DefaultAPI.DeleteDestination(ctx, projectID, region, instanceID, destinationID).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryRouter destination", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteDestinationWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID, destinationID).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryRouter destination", fmt.Sprintf("Waiting for TelemetryRouter destination to become deleted: %v", err))
		return
	}

	tflog.Info(ctx, "TelemetryRouter destination deleted")
}

func (r *telemetryRouterDestinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing TelemetryRouter destination", fmt.Sprintf("Invalid import ID %q: expected format is `project_id`,`region`,`instance_id`,`destination_id`", req.ID))
		return
	}

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":     idParts[0],
		"region":         idParts[1],
		"instance_id":    idParts[2],
		"destination_id": idParts[3],
	})
	tflog.Info(ctx, "TelemetryRouter Destination state imported")
}

func toCreatePayload(ctx context.Context, diags diag.Diagnostics, model *Model) (*telemetryrouter.CreateDestinationPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	payload := &telemetryrouter.CreateDestinationPayload{
		DisplayName: model.DisplayName.ValueString(),
		Description: conversion.StringValueToPointer(model.Description),
	}

	conf, err := toConfig(ctx, diags, model)
	if err != nil {
		return nil, err
	}
	payload.Config = *conf

	return payload, nil
}

func toUpdatePayload(ctx context.Context, diags diag.Diagnostics, model *Model) (*telemetryrouter.UpdateDestinationPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	payload := &telemetryrouter.UpdateDestinationPayload{
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
		Description: conversion.StringValueToPointer(model.Description),
	}

	conf, err := toConfig(ctx, diags, model)
	if err != nil {
		return nil, err
	}
	payload.Config = conf

	return payload, nil
}

func toConfig(ctx context.Context, diags diag.Diagnostics, model *Model) (*telemetryrouter.DestinationConfig, error) {
	result := telemetryrouter.DestinationConfig{}
	var conf config
	diags.Append(model.Config.As(ctx, &conf, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, fmt.Errorf("converting config object: %v", diags.Errors())
	}

	result.ConfigType = telemetryrouter.DestinationConfigType(conf.ConfigType.ValueString())
	switch result.ConfigType {
	case telemetryrouter.DESTINATIONCONFIGTYPE_OPEN_TELEMETRY:
		destinationConfigOpenTelemetry, err := toOpenTelemetry(ctx, diags, &conf)
		if err != nil {
			return nil, err
		}
		result.OpenTelemetry = destinationConfigOpenTelemetry
	case telemetryrouter.DESTINATIONCONFIGTYPE_S3:
		destinationConfigS3, err := toS3(ctx, diags, &conf)
		if err != nil {
			return nil, err
		}
		result.S3 = destinationConfigS3
	}

	configFilter, err := toConfigFilter(ctx, diags, &conf)
	if err != nil {
		return nil, err
	}

	result.Filter = configFilter

	return &result, nil
}

func toConfigFilter(ctx context.Context, diags diag.Diagnostics, conf *config) (*telemetryrouter.ConfigFilter, error) {
	if !conf.Filter.IsNull() && !conf.Filter.IsUnknown() {
		var fltr filter
		diags.Append(conf.Filter.As(ctx, &fltr, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, fmt.Errorf("converting filter object: %v", diags.Errors())
		}

		var attributes []attribute
		diags.Append(fltr.Attributes.ElementsAs(ctx, &attributes, false)...)
		if diags.HasError() {
			return nil, fmt.Errorf("converting attributes list: %v", diags.Errors())
		}

		configFilterAttributes := make([]telemetryrouter.ConfigFilterAttributes, 0, len(attributes))
		for _, item := range attributes {
			var values []string
			valuesDiags := item.Values.ElementsAs(ctx, &values, false)
			diags.Append(valuesDiags...)
			if !valuesDiags.HasError() {
				configFilterAttributes = append(configFilterAttributes, telemetryrouter.ConfigFilterAttributes{
					Key:     item.Key.ValueString(),
					Level:   telemetryrouter.ConfigFilterLevel(item.Level.ValueString()),
					Matcher: telemetryrouter.ConfigFilterMatcher(item.Matcher.ValueString()),
					Values:  values,
				})
			}
		}
		if len(configFilterAttributes) > 0 {
			return telemetryrouter.NewConfigFilter(
				configFilterAttributes,
			), nil
		}
	}

	return nil, nil
}

func toOpenTelemetry(ctx context.Context, diags diag.Diagnostics, conf *config) (*telemetryrouter.DestinationConfigOpenTelemetry, error) {
	if !conf.OpenTelemetry.IsNull() && !conf.OpenTelemetry.IsUnknown() {
		var ot openTelemetry
		var result telemetryrouter.DestinationConfigOpenTelemetry
		diags.Append(conf.OpenTelemetry.As(ctx, &ot, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, fmt.Errorf("converting opentelemetry object: %v", diags.Errors())
		}

		if !tfutils.IsUndefined(ot.BasicAuth) {
			var basicAuthVal basicAuth
			diags.Append(ot.BasicAuth.As(ctx, &basicAuthVal, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return nil, fmt.Errorf("converting basic_auth object: %v", diags.Errors())
			}
			result.BasicAuth = telemetryrouter.NewDestinationConfigOpenTelemetryBasicAuth(
				basicAuthVal.Password.ValueString(),
				basicAuthVal.Username.ValueString(),
			)
		}
		result.BearerToken = ot.BearerToken.ValueStringPointer()
		result.Uri = ot.Uri.ValueString()

		return &result, nil
	}

	return nil, nil
}

func toS3(ctx context.Context, diags diag.Diagnostics, conf *config) (*telemetryrouter.DestinationConfigS3, error) {
	if !conf.S3.IsNull() && !conf.S3.IsUnknown() {
		var s3Inst s3
		var result telemetryrouter.DestinationConfigS3
		diags.Append(conf.S3.As(ctx, &s3Inst, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, fmt.Errorf("converting s3 object: %v", diags.Errors())
		}

		if !tfutils.IsUndefined(s3Inst.AccessKey) {
			var accKey accessKey
			diags.Append(s3Inst.AccessKey.As(ctx, &accKey, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return nil, fmt.Errorf("converting access_key object: %v", diags.Errors())
			}
			result.AccessKey = telemetryrouter.NewDestinationConfigS3AccessKey(
				accKey.ID.ValueString(),
				accKey.Secret.ValueString(),
			)
		}
		result.Bucket = s3Inst.Bucket.ValueString()
		result.Endpoint = s3Inst.Endpoint.ValueString()

		return &result, nil
	}

	return nil, nil
}

func mapFields(ctx context.Context, destination *telemetryrouter.DestinationResponse, model *Model, region string) error {
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

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), region, model.InstanceID.ValueString(), destinationID)
	model.Region = types.StringValue(region)
	model.DestinationID = types.StringValue(destinationID)
	model.DisplayName = types.StringValue(destination.DisplayName)
	model.Description = types.StringPointerValue(destination.Description)
	model.CredentialType = types.StringValue(destination.CredentialType)
	model.CreationTime = types.StringValue(destination.CreationTime.Format(time.RFC3339))
	model.Status = types.StringValue(destination.Status)

	if err := mapConfig(ctx, destination, model); err != nil {
		return fmt.Errorf("map config: %w", err)
	}

	return nil
}

func mapConfig(ctx context.Context, destination *telemetryrouter.DestinationResponse, model *Model) error {
	var conf config
	confIsEmpty := true
	if !model.Config.IsNull() {
		diags := model.Config.As(ctx, &conf, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("converting config object: %v", diags.Errors())
		}
		confIsEmpty = false
	}

	conf.ConfigType = types.StringValue(string(destination.Config.ConfigType))

	if err := mapFilter(ctx, &destination.Config, &conf); err != nil {
		return err
	}

	if err := mapOpenTelemetry(ctx, &destination.Config, &conf); err != nil {
		return err
	}

	if err := mapS3(ctx, &destination.Config, &conf); err != nil {
		return err
	}

	if confIsEmpty {
		confModel, diags := types.ObjectValueFrom(ctx, configTypes, conf)
		if diags.HasError() {
			return fmt.Errorf("mapping config: %w", core.DiagsToError(diags))
		}

		model.Config = confModel
	}

	return nil
}

func mapFilter(ctx context.Context, apiConf *telemetryrouter.DestinationConfig, conf *config) error {
	if apiConf.Filter == nil {
		conf.Filter = types.ObjectNull(filterTypes)
		return nil
	}

	attrList := []attr.Value{}
	for _, currentAttr := range apiConf.Filter.Attributes {
		values, diags := types.ListValueFrom(ctx, types.StringType, currentAttr.Values)
		if diags.HasError() {
			return fmt.Errorf("mapping filter values: %w", core.DiagsToError(diags))
		}
		attrModel, diags := types.ObjectValueFrom(ctx, attributeTypes, attribute{
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
		attrConfigs = types.ListNull(types.ObjectType{AttrTypes: attributeTypes})
	} else {
		attrConfigs, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, attrList)
		if diags.HasError() {
			return fmt.Errorf("mapping attributes: %w", core.DiagsToError(diags))
		}
	}

	filterValue, diags := types.ObjectValueFrom(ctx, filterTypes, filter{
		Attributes: attrConfigs,
	})
	if diags.HasError() {
		return fmt.Errorf("mapping filter: %w", core.DiagsToError(diags))
	}
	conf.Filter = filterValue

	return nil
}

func mapOpenTelemetry(ctx context.Context, apiConf *telemetryrouter.DestinationConfig, conf *config) error {
	if apiConf.OpenTelemetry == nil {
		conf.OpenTelemetry = types.ObjectNull(openTelemetryTypes)
		return nil
	}

	var ot openTelemetry
	diags := conf.OpenTelemetry.As(ctx, &ot, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
	})
	if diags.HasError() {
		return fmt.Errorf("converting opentelemetry object: %v", diags.Errors())
	}
	ot.Uri = types.StringValue(apiConf.OpenTelemetry.Uri)
	if tfutils.IsUndefined(ot.BearerToken) {
		ot.BearerToken = types.StringNull()
	}

	if apiConf.OpenTelemetry.BasicAuth == nil {
		ot.BasicAuth = types.ObjectNull(basicAuthTypes)
	} else {
		var ba basicAuth
		diags = ot.BasicAuth.As(ctx, &ba, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})
		if diags.HasError() {
			return fmt.Errorf("converting basic auth object: %v", diags.Errors())
		}
		ba.Username = types.StringValue(apiConf.OpenTelemetry.BasicAuth.Username)
		if tfutils.IsUndefined(ba.Password) {
			ba.Password = types.StringValue("")
		}
		ot.BasicAuth, diags = types.ObjectValueFrom(ctx, basicAuthTypes, ba)
		if diags.HasError() {
			return fmt.Errorf("mapping basic auth: %w", core.DiagsToError(diags))
		}
	}

	otModel, diags := types.ObjectValueFrom(ctx, openTelemetryTypes, ot)
	if diags.HasError() {
		return fmt.Errorf("mapping open telemetry: %w", core.DiagsToError(diags))
	}

	conf.OpenTelemetry = otModel

	return nil
}

func mapS3(ctx context.Context, apiConf *telemetryrouter.DestinationConfig, conf *config) error {
	if apiConf.S3 == nil {
		conf.S3 = types.ObjectNull(s3Types)
		return nil
	}

	var s3Struct s3
	diags := conf.S3.As(ctx, &s3Struct, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
	})
	if diags.HasError() {
		return fmt.Errorf("converting s3 object: %v", diags.Errors())
	}
	s3Struct.Bucket = types.StringValue(apiConf.S3.Bucket)
	s3Struct.Endpoint = types.StringValue(apiConf.S3.Endpoint)

	if apiConf.S3.AccessKey == nil {
		s3Struct.AccessKey = types.ObjectNull(accessKeyTypes)
	} else {
		var ak accessKey
		diags = s3Struct.AccessKey.As(ctx, &ak, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("converting access key object: %v", diags.Errors())
		}
		ak.ID = types.StringValue(apiConf.S3.AccessKey.Id)
		s3Struct.AccessKey, diags = types.ObjectValueFrom(ctx, accessKeyTypes, ak)
		if diags.HasError() {
			return fmt.Errorf("mapping access key: %w", core.DiagsToError(diags))
		}
	}

	s3Model, diags := types.ObjectValueFrom(ctx, s3Types, s3Struct)
	if diags.HasError() {
		return fmt.Errorf("mapping s3: %w", core.DiagsToError(diags))
	}

	conf.S3 = s3Model

	return nil
}
