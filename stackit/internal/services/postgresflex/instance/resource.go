package postgresflex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	postgresflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/postgresflex/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	listplanmodifier2 "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/planmodifiers/listplanmodifier"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3beta1api"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3beta1api/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
	_ resource.ResourceWithModifyPlan  = &instanceResource{}
)

const (
	NODE_TYPE_SINGLE  = "Single"
	NODE_TYPE_REPLICA = "Replica"

	NODE_TYPE_SINGLE_VALUE  int32 = 1
	NODE_TYPE_REPLICA_VALUE int32 = 3
)

type Model struct {
	Id         types.String `tfsdk:"id"` // needed by TF
	InstanceId types.String `tfsdk:"instance_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	Name       types.String `tfsdk:"name"`
	// Deprecated: ACL is deprecated and will be removed after February 2027.
	ACL            types.List   `tfsdk:"acl"`
	BackupSchedule types.String `tfsdk:"backup_schedule"`
	ConnectionInfo types.Object `tfsdk:"connection_info"`
	Flavor         types.Object `tfsdk:"flavor"`
	FlavorId       types.String `tfsdk:"flavor_id"`
	Replicas       types.Int32  `tfsdk:"replicas"`
	Storage        types.Object `tfsdk:"storage"`
	Encryption     types.Object `tfsdk:"encryption"`
	Network        types.Object `tfsdk:"network"`
	RetentionDays  types.Int32  `tfsdk:"retention_days"`
	Version        types.String `tfsdk:"version"`
	Region         types.String `tfsdk:"region"`
}

// Struct corresponding to Model.Flavor
type flavorModel struct {
	Id          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	CPU         types.Int64  `tfsdk:"cpu"`
	RAM         types.Int64  `tfsdk:"ram"`
	NodeType    types.String `tfsdk:"node_type"`
}

// Types corresponding to flavorModel
var flavorTypes = map[string]attr.Type{
	"id":          basetypes.StringType{},
	"description": basetypes.StringType{},
	"cpu":         basetypes.Int64Type{},
	"ram":         basetypes.Int64Type{},
	"node_type":   basetypes.StringType{},
}

// Struct corresponding to Model.Storage
type storageModel struct {
	Class types.String `tfsdk:"class"`
	Size  types.Int64  `tfsdk:"size"`
}

// Types corresponding to storageModel
var storageTypes = map[string]attr.Type{
	"class": basetypes.StringType{},
	"size":  basetypes.Int64Type{},
}

// Struct corresponding to Model.Network
type networkModel struct {
	Acl             types.List   `tfsdk:"acl"`
	AccessScope     types.String `tfsdk:"access_scope"`
	InstanceAddress types.String `tfsdk:"instance_address"`
	RouterAddress   types.String `tfsdk:"router_address"`
}

// Types corresponding to networkModel
var networkTypes = map[string]attr.Type{
	"acl":              basetypes.ListType{ElemType: types.StringType},
	"access_scope":     basetypes.StringType{},
	"instance_address": basetypes.StringType{},
	"router_address":   basetypes.StringType{},
}

// Struct corresponding to Model.Encryption
type encryptionModel struct {
	KekKeyId       types.String `tfsdk:"kek_key_id"`
	KekKeyRingId   types.String `tfsdk:"kek_key_ring_id"`
	KekKeyVersion  types.String `tfsdk:"kek_key_version"`
	ServiceAccount types.String `tfsdk:"service_account"`
}

// Types corresponding to encryptionModel
var encryptionTypes = map[string]attr.Type{
	"kek_key_id":      basetypes.StringType{},
	"kek_key_ring_id": basetypes.StringType{},
	"kek_key_version": basetypes.StringType{},
	"service_account": basetypes.StringType{},
}

// Types corresponding to connectionInfoModel
var connectionInfoTypes = map[string]attr.Type{
	"write": basetypes.ObjectType{AttrTypes: connectionInfoWriteTypes},
}

// Types corresponding to connectionInfoModel
var connectionInfoWriteTypes = map[string]attr.Type{
	"host": basetypes.StringType{},
	"port": basetypes.Int32Type{},
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client       *postgresflex.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	// skip initial empty configuration to avoid follow-up errors
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

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	handleV3Migration(ctx, &planModel, &configModel, resp)

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func handleV3Migration(_ context.Context, planModel, configModel *Model, resp *resource.ModifyPlanResponse) {
	if configModel == nil {
		resp.Diagnostics.AddError("handling v3 migration", "configModel is nil. This is an error in the provider. Please report it https://github.com/stackitcloud/terraform-provider-stackit/issues")
		return
	}
	if planModel == nil {
		resp.Diagnostics.AddError("handling v3 migration", "planModel is nil. This is an error in the provider. Please report it https://github.com/stackitcloud/terraform-provider-stackit/issues")
		return
	}

	// retention_days
	if configModel.RetentionDays.IsNull() || configModel.RetentionDays.IsUnknown() {
		if planModel.RetentionDays.IsNull() || planModel.RetentionDays.IsUnknown() {
			planModel.RetentionDays = types.Int32Value(32)
		}
		resp.Diagnostics.AddAttributeWarning(path.Root("retention_days"),
			"retention_days will be required in future", "retention_days will be a required field after February 2027. Set a value to prevent breaking changes. Fallback to 32 days during deprecation period.")
	}

	// backup_schedule
	if !(planModel.BackupSchedule.IsNull() || planModel.BackupSchedule.IsUnknown()) {
		backupSchedule := planModel.BackupSchedule.ValueString()
		backupScheduleSimplified := utils.SimplifyCronString(backupSchedule)
		if backupSchedule != backupScheduleSimplified {
			resp.Diagnostics.AddAttributeWarning(path.Root("backup_schedule"),
				"backup_schedule is not valid defined", fmt.Sprintf("backup_schedule is not correctly defined and will result in an error after February 2027. Set it to the value %q to prevent errors in future releases.", backupScheduleSimplified))
		}
	}
}

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresflex_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := postgresflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Postgres Flex instance client configured")
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	willBeRequired := " Will be required in the future. Set a value to prevent breaking changes."
	descriptions := map[string]string{
		"main":                       "Postgres Flex instance resource schema. Must have a `region` specified in the provider configuration.",
		"id":                         "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
		"instance_id":                "ID of the PostgresFlex instance.",
		"project_id":                 "STACKIT project ID to which the instance is associated.",
		"name":                       "Instance name.",
		"acl":                        "The Access Control List (ACL) for the PostgresFlex instance.",
		"region":                     "The resource region. If not defined, the provider region is used.",
		"backup_schedule":            "The schedule for on what time and how often the database backup will be created. Must be a valid cron expression using numeric minute and hour values, e.g: '0 2 * * *'.",
		"connection_info":            "The connection info for the PostgresFlex instance.",
		"connection_info.write":      "The DNS name and port in the instance overview.",
		"connection_info.write.host": "The host of the instance.",
		"connection_info.write.port": "The port of the instance.",
		"replicas":                   "How many replicas the instance should have. Valid values are 1 for single mode or 3 for replication.",
		"flavor_id":                  "The flavor ID of the PostgreSQL Flex instance.",
		"encryption.kek_key_id":      "The ID of the Key within the STACKIT-KMS to use for the encryption.",
		"encryption.kek_key_ring_id": "The ID of the keyring where the key is located within the STACKTI-KMS.",
		"encryption.kek_key_version": "Version of the key within the STACKIT-KMS to use for the encryption.",
		"encryption.service_account": "Service-Account linked to the Key within the STACKIT-KMS.",
		"storage_class":              "The storage class. You can list available storage classes using the [STACKIT CLI](https://github.com/stackitcloud/stackit-cli):\n```bash\nstackit postgresflex options --storages --flavor-id FLAVOR_ID\n```",
		"network":                    "The network configuration of the instance." + willBeRequired,
		"network.access_scope":       "The network access scope of the instance. This feature is in private preview. Supplying this object is only permitted for enabled accounts. If your account does not have access, the request will be rejected. " + utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(postgresflex.AllowedInstanceNetworkAccessScopeEnumValues)...),
		"network.acl":                "List of IPV4 cidr." + willBeRequired,
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$"),
						"must start with a letter, must have lower case letters, numbers or hyphens, and no hyphen at the end",
					),
				},
			},
			"acl": schema.ListAttribute{
				Description:        descriptions["acl"],
				DeprecationMessage: "acl is deprecated and will be removed after February 2027. Use instead `network.acl`.",
				ElementType:        types.StringType,
				Optional:           true,
				Computed:           true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier2.UseStateForUnknownIf(listplanmodifier2.ListUnchanged(path.Root("network").AtName("acl")), "sets `UseStateForUnknown` only if `network.acl` has not changed"),
				},
				Validators: []validator.List{
					listvalidator.ExactlyOneOf(
						path.Root("acl").Expression(),
						path.Root("network").AtName("acl").Expression(),
					),
				},
			},
			"backup_schedule": schema.StringAttribute{
				Description:   descriptions["backup_schedule"],
				Required:      true,
				PlanModifiers: []planmodifier.String{},
			},
			"connection_info": schema.SingleNestedAttribute{
				Description: descriptions["connection_info"],
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseNonNullStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"write": schema.SingleNestedAttribute{
						Description: descriptions["connection_info.write"],
						Computed:    true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseNonNullStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"host": schema.StringAttribute{
								Description: descriptions["connection_info.write.host"],
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseNonNullStateForUnknown(),
								},
							},
							"port": schema.Int32Attribute{
								Description: descriptions["connection_info.write.port"],
								Computed:    true,
								PlanModifiers: []planmodifier.Int32{
									int32planmodifier.UseNonNullStateForUnknown(),
								},
							},
						},
					},
				},
			},
			"flavor_id": schema.StringAttribute{
				Description: descriptions["flavor_id"],
				Computed:    true,
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.Root("flavor_id").Expression(),
						path.Root("flavor").Expression(),
					),
					stringvalidator.All(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseNonNullStateForUnknown(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							UseStateForUnknownIfFlavorUnchanged(req),
						},
					},
					"description": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							UseStateForUnknownIfFlavorUnchanged(req),
						},
					},
					"cpu": schema.Int64Attribute{
						Required: true,
					},
					"ram": schema.Int64Attribute{
						Required: true,
					},
					"node_type": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							UseStateForUnknownIfFlavorUnchanged(req),
						},
					},
				},
				Validators: []validator.Object{
					objectvalidator.ExactlyOneOf(
						path.Root("flavor_id").Expression(),
						path.Root("flavor").Expression(),
					),
				},
			},
			"encryption": schema.SingleNestedAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"kek_key_id": schema.StringAttribute{
						Description: descriptions["encryption.kek_key_id"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"kek_key_ring_id": schema.StringAttribute{
						Description: descriptions["encryption.kek_key_ring_id"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"kek_key_version": schema.StringAttribute{
						Description: descriptions["encryption.kek_key_version"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"service_account": schema.StringAttribute{
						Description: descriptions["encryption.service_account"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"replicas": schema.Int32Attribute{
				Description: descriptions["replicas"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int32{
					int32validator.OneOf(1, 3),
					int32validator.AlsoRequires(path.Root("flavor").Expression()),
					int32validator.ConflictsWith(path.Root("flavor_id").Expression()),
				},
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseNonNullStateForUnknown(),
				},
			},
			"retention_days": schema.Int32Attribute{
				Description: descriptions["retention_days"],
				Optional:    true,
				Computed:    true,
			},
			"network": schema.SingleNestedAttribute{
				Description: descriptions["network"],
				Computed:    true,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"access_scope": schema.StringAttribute{
						Description: descriptions["network.access_scope"],
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"acl": schema.ListAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier2.UseStateForUnknownIf(listplanmodifier2.ListUnchanged(path.Root("acl")), "sets `UseStateForUnknown` only if `acl` has not changed"),
						},
						Validators: []validator.List{
							listvalidator.ExactlyOneOf(
								path.Root("acl").Expression(),
								path.Root("network").AtName("acl").Expression(),
							),
							listvalidator.SizeAtLeast(1),
						},
					},
					"instance_address": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"router_address": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"storage": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"class": schema.StringAttribute{
						Description: descriptions["storage_class"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"size": schema.Int64Attribute{
						Required: true,
					},
				},
			},
			"version": schema.StringAttribute{
				Required: true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	var acl []string
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client.DefaultAPI, &model, flavor)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var network = &networkModel{}
	if !(model.Network.IsNull() || model.Network.IsUnknown()) {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var encryption *encryptionModel
	if !(model.Encryption.IsNull() || model.Encryption.IsUnknown()) {
		encryption = &encryptionModel{}
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, acl, flavor, storage, network, encryption)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new instance
	createResp, err := r.client.DefaultAPI.CreateInstance(ctx, projectId, region).CreateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	if createResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", "Got empty response")
		return
	}

	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"region":      region,
		"instance_id": createResp.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectId, region, createResp.Id).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	if instanceId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	instanceResp, err := r.client.DefaultAPI.GetInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		// Read the flavor here from the API, because during an import the flavor should be set
		flavorResp, err := getFlavor(ctx, r.client.DefaultAPI, projectId, region, instanceResp.FlavorId)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Finding flavor: %v", err))
			return
		}
		flavor = &flavorModel{
			Id:          types.StringValue(flavorResp.Id),
			Description: types.StringValue(flavorResp.Description),
			CPU:         types.Int64Value(flavorResp.Cpu),
			RAM:         types.Int64Value(flavorResp.Memory),
			NodeType:    types.StringValue(flavorResp.NodeType),
		}
	}

	// Map response body to schema
	err = mapFields(ctx, instanceResp, &model, flavor, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	var acl []string
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client.DefaultAPI, &model, flavor)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var network = &networkModel{}
	if !(model.Network.IsNull() || model.Network.IsUnknown()) {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model, acl, flavor, storage, network)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing instance
	err = r.client.DefaultAPI.PartialUpdateInstance(ctx, projectId, region, instanceId).PartialUpdateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.PartialUpdateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgresflex instance updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing instance
	err := r.client.DefaultAPI.DeleteInstance(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId).SetTimeout(45 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "Postgres Flex instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"region":      idParts[1],
		"instance_id": idParts[2],
	})
	tflog.Info(ctx, "Postgres Flex instance state imported")
}

func mapFields(ctx context.Context, resp *postgresflex.GetInstanceResponse, model *Model, flavor *flavorModel, region string) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if resp.Id != "" {
		instanceId = resp.Id
	} else {
		return fmt.Errorf("instance id not present")
	}

	var aclList basetypes.ListValue
	var diags diag.Diagnostics
	if resp.Network.Acl == nil {
		aclList = types.ListNull(types.StringType)
	} else {
		modelACL, err := utils.ListValueToStringSlice(model.ACL)
		if err != nil {
			return err
		}

		reconciledACL := utils.ReconcileStringSlices(modelACL, resp.Network.Acl)

		aclList, diags = types.ListValueFrom(ctx, types.StringType, reconciledACL)
		if diags.HasError() {
			return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
		}
	}

	networkValues := map[string]attr.Value{
		"acl":              aclList,
		"access_scope":     types.StringPointerValue((*string)(resp.Network.AccessScope)),
		"instance_address": types.StringPointerValue(resp.Network.InstanceAddress),
		"router_address":   types.StringPointerValue(resp.Network.RouterAddress),
	}
	networkObject, diags := types.ObjectValue(networkTypes, networkValues)
	if diags.HasError() {
		return fmt.Errorf("mapping network: %w", core.DiagsToError(diags))
	}

	flavorValues := map[string]attr.Value{
		"id":          flavor.Id,
		"description": flavor.Description,
		"cpu":         flavor.CPU,
		"ram":         flavor.RAM,
		"node_type":   flavor.NodeType,
	}
	flavorObject, diags := types.ObjectValue(flavorTypes, flavorValues)
	if diags.HasError() {
		return fmt.Errorf("mapping flavor: %w", core.DiagsToError(diags))
	}

	storageValues := map[string]attr.Value{
		"class": types.StringPointerValue(resp.Storage.Class),
		"size":  types.Int64PointerValue(resp.Storage.Size),
	}
	storageObject, diags := types.ObjectValue(storageTypes, storageValues)
	if diags.HasError() {
		return fmt.Errorf("mapping storage: %w", core.DiagsToError(diags))
	}

	encryptionObject := types.ObjectNull(encryptionTypes)
	if resp.Encryption != nil {
		encryptionValues := map[string]attr.Value{
			"kek_key_id":      types.StringValue(resp.Encryption.KekKeyId),
			"kek_key_ring_id": types.StringValue(resp.Encryption.KekKeyRingId),
			"kek_key_version": types.StringValue(resp.Encryption.KekKeyVersion),
			"service_account": types.StringValue(resp.Encryption.ServiceAccount),
		}
		encryptionObject, diags = types.ObjectValue(encryptionTypes, encryptionValues)
		if diags.HasError() {
			return fmt.Errorf("mapping encryption: %w", core.DiagsToError(diags))
		}
	}

	// If the API returned "0 0 * * *" but user defined "00 00 * * *" in its config,
	// we keep the user's "00 00 * * *" in the state to satisfy Terraform.
	backupScheduleApiResp := types.StringValue(resp.BackupSchedule)
	if utils.SimplifyCronString(model.BackupSchedule.ValueString()) != utils.SimplifyCronString(backupScheduleApiResp.ValueString()) {
		// If the API actually changed it to something else, use the API value
		model.BackupSchedule = types.StringValue(resp.BackupSchedule)
	}

	// connection info
	writeObject, diags := types.ObjectValue(connectionInfoWriteTypes, map[string]attr.Value{
		"host": types.StringValue(resp.ConnectionInfo.Write.Host),
		"port": types.Int32Value(resp.ConnectionInfo.Write.Port),
	})
	if diags.HasError() {
		return fmt.Errorf("mapping connection info write: %w", core.DiagsToError(diags))
	}

	connectionObject, diags := types.ObjectValue(connectionInfoTypes, map[string]attr.Value{
		"write": writeObject,
	})
	if diags.HasError() {
		return fmt.Errorf("mapping connection info: %w", core.DiagsToError(diags))
	}

	if model.Replicas.IsNull() || model.Replicas.IsUnknown() {
		var replica *int32
		switch flavor.NodeType.ValueString() {
		case NODE_TYPE_SINGLE:
			replica = new(NODE_TYPE_SINGLE_VALUE)
		case NODE_TYPE_REPLICA:
			replica = new(NODE_TYPE_REPLICA_VALUE)
		}
		model.Replicas = types.Int32PointerValue(replica)
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instanceId)
	model.InstanceId = types.StringValue(instanceId)
	model.Name = types.StringValue(resp.Name)
	model.ACL = aclList
	model.Flavor = flavorObject
	model.FlavorId = types.StringValue(resp.FlavorId)
	model.Encryption = encryptionObject
	model.RetentionDays = types.Int32PointerValue(resp.RetentionDays.Get())
	model.Storage = storageObject
	model.Version = types.StringValue(resp.Version)
	model.Region = types.StringValue(region)
	model.Network = networkObject
	model.ConnectionInfo = connectionObject
	return nil
}

func toCreatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel, network *networkModel, encryption *encryptionModel) (*postgresflex.CreateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	var flavorId string
	if flavor != nil && !(flavor.Id.IsNull() || flavor.Id.IsUnknown()) {
		flavorId = flavor.Id.ValueString()
	} else if !(model.FlavorId.IsNull() || model.FlavorId.IsUnknown()) {
		flavorId = model.FlavorId.ValueString()
	} else {
		return nil, fmt.Errorf("flavor id is missing")
	}

	networkPayload := postgresflex.InstanceNetworkCreate{}
	if acl != nil {
		networkPayload.Acl = acl
	} else if network != nil && !(network.Acl.IsNull() || network.Acl.IsUnknown()) {
		var err error
		networkPayload.Acl, err = conversion.StringListToSlice(network.Acl)
		if err != nil {
			return nil, err
		}
		if !(network.AccessScope.IsUnknown() || network.AccessScope.IsNull()) {
			networkPayload.AccessScope = (*postgresflex.InstanceNetworkAccessScope)(network.AccessScope.ValueStringPointer())
		}
	} else {
		return nil, fmt.Errorf("no acl defined")
	}

	var encryptionPayload *postgresflex.InstanceEncryption
	if encryption != nil {
		encryptionPayload = &postgresflex.InstanceEncryption{
			KekKeyId:             encryption.KekKeyId.ValueString(),
			KekKeyRingId:         encryption.KekKeyRingId.ValueString(),
			KekKeyVersion:        encryption.KekKeyVersion.ValueString(),
			ServiceAccount:       encryption.ServiceAccount.ValueString(),
			AdditionalProperties: nil,
		}
	}

	var retentionDays postgresflex.NullableInt32
	if !(model.RetentionDays.IsNull() || model.RetentionDays.IsUnknown()) {
		retentionDays = *postgresflex.NewNullableInt32(conversion.Int32ValueToPointer(model.RetentionDays))
	}

	var backupSchedule string
	if !(model.BackupSchedule.IsNull() || model.BackupSchedule.IsUnknown()) {
		backupSchedule = utils.SimplifyCronString(model.BackupSchedule.ValueString())
	}

	return &postgresflex.CreateInstancePayload{
		BackupSchedule: backupSchedule,
		FlavorId:       flavorId,
		Name:           model.Name.ValueString(),
		Network:        networkPayload,
		Encryption:     encryptionPayload,
		RetentionDays:  retentionDays,
		Storage: postgresflex.StorageCreate{
			Class: conversion.StringValueToPointer(storage.Class),
			Size:  storage.Size.ValueInt64(),
		},
		Version: model.Version.ValueString(),
	}, nil
}

func toUpdatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel, network *networkModel) (*postgresflex.PartialUpdateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	var flavorId *string
	if flavor != nil && !(flavor.Id.IsNull() || flavor.Id.IsUnknown()) {
		flavorId = flavor.Id.ValueStringPointer()
	} else if !(model.FlavorId.IsNull() || model.FlavorId.IsUnknown()) {
		flavorId = model.FlavorId.ValueStringPointer()
	} else {
		return nil, fmt.Errorf("flavor id is missing")
	}

	networkPayload := &postgresflex.InstanceNetworkOpt{}
	if acl != nil {
		networkPayload.Acl = acl
	} else if network != nil && !(network.Acl.IsNull() || network.Acl.IsUnknown()) {
		var err error
		networkPayload.Acl, err = conversion.StringListToSlice(network.Acl)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no acl defined")
	}

	var backupSchedule *string
	if !(model.BackupSchedule.IsNull() || model.BackupSchedule.IsUnknown()) {
		backupSchedule = new(utils.SimplifyCronString(model.BackupSchedule.ValueString()))
	}

	return &postgresflex.PartialUpdateInstancePayload{
		BackupSchedule: backupSchedule,
		FlavorId:       flavorId,
		Name:           conversion.StringValueToPointer(model.Name),
		Network:        networkPayload,
		Storage: &postgresflex.StorageUpdate{
			Size: conversion.Int64ValueToPointer(storage.Size),
		},
		RetentionDays: conversion.Int32ValueToPointer(model.RetentionDays),
		Version:       conversion.StringValueToPointer(model.Version),
	}, nil
}

type postgresFlexClient interface {
	ListFlavors(ctx context.Context, projectId, region string) postgresflex.ApiListFlavorsRequest
	ListFlavorsExecute(r postgresflex.ApiListFlavorsRequest) (*postgresflex.ListFlavorsResponse, error)
}

func loadFlavorId(ctx context.Context, client postgresFlexClient, model *Model, flavor *flavorModel) error {
	if model == nil {
		return fmt.Errorf("nil model")
	}
	if flavor == nil {
		return fmt.Errorf("nil flavor")
	}
	cpu := conversion.Int64ValueToPointer(flavor.CPU)
	if cpu == nil {
		return fmt.Errorf("nil CPU")
	}
	ram := conversion.Int64ValueToPointer(flavor.RAM)
	if ram == nil {
		return fmt.Errorf("nil RAM")
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	req := client.ListFlavors(ctx, projectId, region)
	res, err := client.ListFlavorsExecute(req)
	if err != nil {
		return fmt.Errorf("listing postgresflex flavors: %w", err)
	}

	avl := ""
	if res.Flavors == nil {
		return fmt.Errorf("finding flavors for project %s", projectId)
	}
	if model.Replicas.IsNull() || model.Replicas.IsUnknown() {
		return fmt.Errorf("no replicas defined")
	}
	var nodeType string
	switch model.Replicas.ValueInt32() {
	case NODE_TYPE_SINGLE_VALUE:
		nodeType = NODE_TYPE_SINGLE
	case NODE_TYPE_REPLICA_VALUE:
		nodeType = NODE_TYPE_REPLICA
	default:
		return fmt.Errorf("unknown replica count. only 1 and 3 are supported")
	}
	for _, f := range res.Flavors {
		if f.Id == "" || f.Cpu == 0 || f.Memory == 0 {
			continue
		}
		if f.Cpu == *cpu && f.Memory == *ram && f.NodeType == nodeType {
			flavor.Id = types.StringValue(f.Id)
			flavor.Description = types.StringValue(f.Description)
			flavor.NodeType = types.StringValue(f.NodeType)
			break
		}
		avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM, %s node type", avl, f.Cpu, f.Memory, f.NodeType)
	}
	if flavor.Id.ValueString() == "" {
		return fmt.Errorf("couldn't find flavor, available specs are:%s", avl)
	}

	return nil
}

func getFlavor(ctx context.Context, client postgresFlexClient, projectId, region, flavorId string) (*postgresflex.ListFlavors, error) {
	req := client.ListFlavors(ctx, projectId, region)
	flavorsResp, err := client.ListFlavorsExecute(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list flavors: %w", err)
	}
	for _, flavor := range flavorsResp.Flavors {
		if flavor.Id == flavorId {
			return &flavor, nil
		}
	}
	return nil, fmt.Errorf("flavor with ID %q not found in project %q", flavorId, projectId)
}
