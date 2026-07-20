package sqlserverflex

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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"

	sqlserverflexUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/sqlserverflex/utils"
	int32planmodifier2 "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/planmodifiers/int32planmodifier"
	listplanmodifier2 "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/planmodifiers/listplanmodifier"
	objectplanmodifier2 "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/planmodifiers/objectplanmodifier"
	stringplanmodifierCustom "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/planmodifiers/stringplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"
	"github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
	_ resource.ResourceWithModifyPlan  = &instanceResource{}
)

type Model struct {
	Id         types.String `tfsdk:"id"` // needed by TF
	InstanceId types.String `tfsdk:"instance_id"`
	ProjectId  types.String `tfsdk:"project_id"`
	Name       types.String `tfsdk:"name"`
	// Deprecated: ACL is deprecated and will be removed after January 2027
	ACL            types.List   `tfsdk:"acl"`
	BackupSchedule types.String `tfsdk:"backup_schedule"`
	Encryption     types.Object `tfsdk:"encryption"`
	Flavor         types.Object `tfsdk:"flavor"`
	FlavorId       types.String `tfsdk:"flavor_id"`
	Storage        types.Object `tfsdk:"storage"`
	Version        types.String `tfsdk:"version"`
	Replicas       types.Int32  `tfsdk:"replicas"`
	Edition        types.String `tfsdk:"edition"`
	// Deprecated: Options is deprecated and will be removed after January 2027
	Options       types.Object `tfsdk:"options"`
	RetentionDays types.Int32  `tfsdk:"retention_days"`
	Network       types.Object `tfsdk:"network"`
	Region        types.String `tfsdk:"region"`
}

// Struct corresponding to Model.Encryption
type encryptionModel struct {
	KekKeyId       types.String `tfsdk:"kek_key_id"`
	KekKeyRingId   types.String `tfsdk:"kek_keyring_id"`
	KekKeyVersion  types.String `tfsdk:"kek_key_version"`
	ServiceAccount types.String `tfsdk:"service_account"`
}

// types corresponding to encryptionModel
var encryptionTypes = map[string]attr.Type{
	"kek_key_id":      basetypes.StringType{},
	"kek_keyring_id":  basetypes.StringType{},
	"kek_key_version": basetypes.StringType{},
	"service_account": basetypes.StringType{},
}

// Struct corresponding to Model.Network
type networkModel struct {
	AccessScope types.String `tfsdk:"access_scope"`
	Acl         types.List   `tfsdk:"acl"`
}

// types corresponding to Network
var networkTypes = map[string]attr.Type{
	"access_scope": basetypes.StringType{},
	"acl":          basetypes.ListType{ElemType: types.StringType},
}

// Struct corresponding to Model.Flavor
type flavorModel struct {
	Id          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	CPU         types.Int64  `tfsdk:"cpu"`
	RAM         types.Int64  `tfsdk:"ram"`
}

// Types corresponding to flavorModel
var flavorTypes = map[string]attr.Type{
	"id":          basetypes.StringType{},
	"description": basetypes.StringType{},
	"cpu":         basetypes.Int64Type{},
	"ram":         basetypes.Int64Type{},
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

// Struct corresponding to Model.Options
type optionsModel struct {
	Edition       types.String `tfsdk:"edition"`
	RetentionDays types.Int32  `tfsdk:"retention_days"`
}

// Types corresponding to optionsModel
var optionsTypes = map[string]attr.Type{
	"edition":        basetypes.StringType{},
	"retention_days": basetypes.Int32Type{},
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client       *sqlserverflex.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflex_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := sqlserverflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "SQLServer Flex instance client configured")
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

func handleV3Migration(ctx context.Context, planModel, configModel *Model, resp *resource.ModifyPlanResponse) {
	// backup_schedule
	if configModel.BackupSchedule.IsNull() || configModel.BackupSchedule.IsUnknown() {
		if planModel.BackupSchedule.IsNull() || planModel.BackupSchedule.IsUnknown() {
			planModel.BackupSchedule = types.StringValue("0 0 * * *")
		}
		resp.Diagnostics.AddAttributeWarning(path.Root("backup_schedule"),
			"backup_schedule will be required in future", "backup_schedule will be a required field after January 2027. Set a value to prevent breaking changes. Fallback to '0 0 * * *' during deprecation period.")
	}

	// storage
	if configModel.Storage.IsNull() || configModel.Storage.IsUnknown() {
		if planModel.Storage.IsNull() || planModel.Storage.IsUnknown() {
			planModel.Storage = types.ObjectValueMust(storageTypes, map[string]attr.Value{
				"class": types.StringValue("premium-perf12-stackit"),
				"size":  types.Int64Value(40),
			})
		}
		resp.Diagnostics.AddAttributeWarning(path.Root("storage"),
			"storage will be required in future", "storage will be a required field after January 2027. Set values to prevent breaking changes. Fallback to class 'premium-perf12-stackit' with a size of 40 gigabytes during deprecation period.")
	} else {
		var storageConfig = &storageModel{}
		resp.Diagnostics.Append(configModel.Storage.As(ctx, storageConfig, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		var storagePlan = &storageModel{}
		resp.Diagnostics.Append(configModel.Storage.As(ctx, storagePlan, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		// storage.class
		if storageConfig.Class.IsNull() || storageConfig.Class.ValueString() == "" {
			if storagePlan.Class.IsNull() || storagePlan.Class.ValueString() == "" {
				storagePlan.Class = types.StringValue("premium-perf12-stackit")
			}
			resp.Diagnostics.AddAttributeWarning(path.Root("storage.class"),
				"storage.class will be required in future", "storage.class will be a required field after January 2027. Set a value to prevent breaking changes. Fallback to 'premium-perf12-stackit' during deprecation period.")
		}

		// storage.size
		if storageConfig.Size.IsNull() {
			if storagePlan.Size.IsNull() {
				storagePlan.Size = types.Int64Value(40)
			}
			resp.Diagnostics.AddAttributeWarning(path.Root("storage.size"),
				"storage.size will be required in future", "storage.size will be a required field after January 2027. Set a value to prevent breaking changes. Fallback to 40 gigabytes during deprecation period.")
		}

		var diags diag.Diagnostics
		planModel.Storage, diags = types.ObjectValue(storageTypes, map[string]attr.Value{
			"class": storagePlan.Class,
			"size":  storagePlan.Size,
		})
		resp.Diagnostics.Append(diags...)
	}

	// version
	if configModel.Version.IsNull() || configModel.Version.IsUnknown() {
		if planModel.Version.IsNull() || planModel.Version.IsUnknown() {
			planModel.Version = types.StringValue(string(sqlserverflex.INSTANCEVERSION__2022))
		}
		resp.Diagnostics.AddAttributeWarning(path.Root("version"),
			"version will be required in future", "version will be a required field after January 2027. Set a value to prevent breaking changes. Fallback to '2022' during deprecation period.")
	}

	// acl
	if (configModel.ACL.IsNull() || configModel.ACL.IsUnknown()) && (configModel.Network.IsNull() || configModel.Network.IsUnknown()) {
		// Not setting default ACL and scope to the configModel, instead we send an empty array to the API, where they set the default value.
		resp.Diagnostics.AddAttributeWarning(path.Root("network").AtName("acl"),
			"network.acl will be required in future", "network.acl will be a required field after January 2027. Set values to prevent breaking changes.")
	}

	// retention_days
	var optionsConfig = &optionsModel{}
	if !(configModel.Options.IsNull() || configModel.Options.IsUnknown()) {
		resp.Diagnostics.Append(configModel.Options.As(ctx, optionsConfig, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var optionsPlan = &optionsModel{}
	if !(configModel.Options.IsNull() || configModel.Options.IsUnknown()) {
		resp.Diagnostics.Append(configModel.Options.As(ctx, optionsPlan, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if (optionsConfig.RetentionDays.IsNull() || optionsConfig.RetentionDays.IsUnknown()) && (configModel.RetentionDays.IsNull() || configModel.RetentionDays.IsUnknown()) {
		if (optionsPlan.RetentionDays.IsNull() || optionsPlan.RetentionDays.IsUnknown()) && (planModel.RetentionDays.IsNull() || planModel.RetentionDays.IsUnknown()) {
			planModel.RetentionDays = types.Int32Value(30)
		}
		resp.Diagnostics.AddAttributeWarning(path.Root("retention_days"),
			"retention_days will be required in future", "retention_days will be a required field after January 2027. Set a value to prevent breaking changes. Fallback to 30 days during deprecation period.")
	}
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	willBeRequired := " Will be required in the future. Set a value to prevent breaking changes."
	descriptions := map[string]string{
		"main":                 "SQLServer Flex instance resource schema. Must have a `region` specified in the provider configuration.",
		"id":                   "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
		"instance_id":          "ID of the SQLServer Flex instance.",
		"project_id":           "STACKIT project ID to which the instance is associated.",
		"name":                 "Instance name.",
		"acl":                  "The Access Control List (ACL) for the SQLServer Flex instance.",
		"backup_schedule":      `The backup schedule. Should follow the cron scheduling system format (e.g. "0 0 * * *")` + willBeRequired,
		"encryption":           "Parameter to define which key to use for storage encryption.",
		"kek_key_id":           "UUID of the key within the STACKIT-KMS to use for the encryption.",
		"kek_keyring_id":       "UUID of the keyring where the key is located within the STACKTI-KMS.",
		"kek_key_version":      "Version of the key within the STACKIT-KMS to use for the encryption.",
		"service_account":      "Service-Account linked to the Key within the STACKIT-KMS.",
		"options":              "Custom parameters for the SQLServer Flex instance.",
		"flavor_id":            "The flavor ID of the SQLServer Flex instance.",
		"network":              "The network configuration of the instance." + willBeRequired,
		"network.access_scope": "The network access scope of the instance. This feature is in private preview. Supplying this object is only permitted for enabled accounts. If your account does not have access, the request will be rejected.",
		"network.acl":          "List of IPV4 cidr." + willBeRequired,
		"retention_days":       "The days (30 to 90) for how long the backup files should be stored before cleaned up." + willBeRequired,
		"edition":              "Edition of the MSSQL server instance.",
		"region":               "The resource region. If not defined, the provider region is used.",
		"storage":              "The object containing information about the storage size and class." + willBeRequired,
		"storage.class":        "The storage class. You can list available storage classes using the [STACKIT CLI](https://github.com/stackitcloud/stackit-cli):\n```bash\nstackit beta sqlserverflex options --storages --flavor-id FLAVOR_ID\n```" + willBeRequired,
		"storage.size":         "The storage size in Gigabytes." + willBeRequired,
		"version":              "The sqlserver version used for the instance. " + utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(sqlserverflex.AllowedInstanceVersionEnumValues)...) + willBeRequired,
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
				DeprecationMessage: "acl is deprecated and will be removed after January 2027. Use instead `network.acl`.",
				ElementType:        types.StringType,
				Optional:           true,
				Computed:           true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier2.UseStateForUnknownIf(listplanmodifier2.ListChanged(path.Root("network").AtName("acl")), "sets `UseStateForUnknown` only if `network.acl` has not changed"),
				},
				Validators: []validator.List{
					listvalidator.ConflictsWith(
						path.Root("network").AtName("acl").Expression(),
					),
				},
			},
			"backup_schedule": schema.StringAttribute{
				Description: descriptions["backup_schedule"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifierCustom.CronNormalizationModifier{},
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"encryption": schema.SingleNestedAttribute{
				Description: descriptions["encryption"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"kek_key_id": schema.StringAttribute{
						Description: descriptions["kek_key_id"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.UUID(),
							validate.NoSeparator(),
						},
					},
					"kek_keyring_id": schema.StringAttribute{
						Description: descriptions["kek_keyring_id"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.UUID(),
							validate.NoSeparator(),
						},
					},
					"kek_key_version": schema.StringAttribute{
						Description: descriptions["kek_key_version"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"service_account": schema.StringAttribute{
						Description: descriptions["service_account"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"description": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"cpu": schema.Int64Attribute{
						Required: true,
					},
					"ram": schema.Int64Attribute{
						Required: true,
					},
				},
				Validators: []validator.Object{
					objectvalidator.ExactlyOneOf(
						path.Root("flavor_id").Expression(),
						path.Root("flavor").Expression(),
					),
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
				},
			},
			"network": schema.SingleNestedAttribute{
				Description: descriptions["network"],
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier2.UseStateForUnknownIf(objectplanmodifier2.ListChanged(path.Root("acl")), "sets `UseStateForUnknown` only if `acl` has not changed"),
				},
				Attributes: map[string]schema.Attribute{
					"access_scope": schema.StringAttribute{
						Description: "The network access scope of the instance. This feature is in private preview. Supplying this object is only permitted for enabled accounts. If your account does not have access, the request will be rejected. " + utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(sqlserverflex.AllowedInstanceNetworkAccessScopeEnumValues)...),
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"acl": schema.ListAttribute{
						Description: "List of IPV4 cidr.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier2.UseStateForUnknownIf(listplanmodifier2.ListChanged(path.Root("acl")), "sets `UseStateForUnknown` only if `acl` has not changed"),
						},
						Validators: []validator.List{
							listvalidator.ConflictsWith(
								path.Root("acl").Expression(),
							),
							listvalidator.SizeAtLeast(1),
						},
					},
				},
			},
			"replicas": schema.Int32Attribute{
				Computed: true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"storage": schema.SingleNestedAttribute{
				Description: descriptions["storage"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"class": schema.StringAttribute{
						Description: descriptions["storage.class"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"size": schema.Int64Attribute{
						Description: descriptions["storage.size"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"edition": schema.StringAttribute{
				Description: descriptions["edition"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"options": schema.SingleNestedAttribute{
				DeprecationMessage: "option is deprecated and will be removed after January 2027.",
				Optional:           true,
				Computed:           true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier2.UseStateForUnknownIf(objectplanmodifier2.Int32Changed(path.Root("retention_days")), "sets `UseStateForUnknown` only if `retention_days` has not changed"),
				},
				Attributes: map[string]schema.Attribute{
					"edition": schema.StringAttribute{
						DeprecationMessage: "edition is deprecated and will be removed after January 2027.",
						Computed:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"retention_days": schema.Int32Attribute{
						DeprecationMessage: "retention_days is deprecated and will be removed after January 2027. Use instead `retention_days` from root.",
						Optional:           true,
						Computed:           true,
						PlanModifiers: []planmodifier.Int32{
							int32planmodifier2.UseStateForUnknownIf(int32planmodifier2.Int32Changed(path.Root("retention_days")), "sets `UseStateForUnknown` only if `retention_days` has not changed"),
						},
						Validators: []validator.Int32{
							int32validator.ConflictsWith(
								path.Root("retention_days").Expression(),
							),
						},
					},
				},
			},
			"retention_days": schema.Int32Attribute{
				Description: descriptions["retention_days"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier2.UseStateForUnknownIf(int32planmodifier2.Int32Changed(path.Root("options").AtName("retention_days")), "sets `UseStateForUnknown` only if `options.retention_days` has not changed"),
				},
				Validators: []validator.Int32{
					int32validator.ConflictsWith(
						path.Root("options").AtName("retention_days").Expression(),
					),
				},
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

	var encryption *encryptionModel
	if !(model.Encryption.IsNull() || model.Encryption.IsUnknown()) {
		encryption = &encryptionModel{}
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
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

	var options = &optionsModel{}
	if !(model.Options.IsNull() || model.Options.IsUnknown()) {
		diags = model.Options.As(ctx, options, basetypes.ObjectAsOptions{})
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
	payload, err := toCreatePayload(&model, acl, encryption, flavor, storage, options, network)
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

	if createResp.Id == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", "Got empty instance id")
		return
	}

	instanceId := createResp.Id
	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectId,
		"region":      region,
		"instance_id": instanceId,
	})
	if resp.Diagnostics.HasError() {
		return
	}
	// The creation waiter sometimes returns an error from the API: "instance with id xxx has unexpected status Failure"
	// which can be avoided by sleeping before wait
	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId).SetSleepBeforeWait(30 * time.Second).WaitWithContext(ctx)
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

	select {
	case <-ctx.Done():
		return
	// After the instance creation, database might not be ready to accept connections immediately. That is why we add a sleep
	case <-time.After(120 * time.Second):
		// continue
	}

	tflog.Info(ctx, "SQLServer Flex instance created")
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

	// Get flavor
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
	tflog.Info(ctx, "SQLServer Flex instance read")
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

	var options = &optionsModel{}
	if !(model.Options.IsNull() || model.Options.IsUnknown()) {
		diags = model.Options.As(ctx, options, basetypes.ObjectAsOptions{})
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
	payload, err := toUpdatePayload(&model, acl, flavor, storage, options, network)
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

	waitResp, err := wait.UpdateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId).WaitWithContext(ctx)
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
	tflog.Info(ctx, "SQLServer Flex instance updated")
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

	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "SQLServer Flex instance deleted")
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
	tflog.Info(ctx, "SQLServer Flex instance state imported")
}

func mapFields(ctx context.Context, resp *sqlserverflex.GetInstanceResponse, model *Model, flavor *flavorModel, region string) error {
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
		respACL := resp.Network.Acl
		modelACL, err := utils.ListValueToStringSlice(model.ACL)
		if err != nil {
			return err
		}

		reconciledACL := utils.ReconcileStringSlices(modelACL, respACL)

		aclList, diags = types.ListValueFrom(ctx, types.StringType, reconciledACL)
		if diags.HasError() {
			return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
		}
	}

	networkValues := map[string]attr.Value{
		"acl":          aclList,
		"access_scope": types.StringPointerValue((*string)(resp.Network.AccessScope)),
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
	}
	flavorObject, diags := types.ObjectValue(flavorTypes, flavorValues)
	if diags.HasError() {
		return fmt.Errorf("creating flavor: %w", core.DiagsToError(diags))
	}

	storageValues := map[string]attr.Value{
		"class": types.StringPointerValue(resp.Storage.Class),
		"size":  types.Int64PointerValue(resp.Storage.Size),
	}
	storageObject, diags := types.ObjectValue(storageTypes, storageValues)
	if diags.HasError() {
		return fmt.Errorf("creating storage: %w", core.DiagsToError(diags))
	}

	optionsValues := map[string]attr.Value{
		"edition":        types.StringValue(string(resp.Edition)),
		"retention_days": types.Int32Value(resp.RetentionDays),
	}
	optionsObject, diags := types.ObjectValue(optionsTypes, optionsValues)
	if diags.HasError() {
		return fmt.Errorf("creating options: %w", core.DiagsToError(diags))
	}

	var encryptionValues map[string]attr.Value
	var encryptionObject types.Object
	if resp.Encryption != nil {
		encryptionValues = map[string]attr.Value{
			"kek_key_id":      types.StringValue(resp.Encryption.KekKeyId),
			"kek_keyring_id":  types.StringValue(resp.Encryption.KekKeyRingId),
			"kek_key_version": types.StringValue(resp.Encryption.KekKeyVersion),
			"service_account": types.StringValue(resp.Encryption.ServiceAccount),
		}
		encryptionObject, diags = types.ObjectValue(encryptionTypes, encryptionValues)
		if diags.HasError() {
			return fmt.Errorf("creating encryption: %w", core.DiagsToError(diags))
		}
	} else {
		encryptionObject = types.ObjectNull(encryptionTypes)
	}

	// If the API returned "0 0 * * *" but user defined "00 00 * * *" in its config,
	// we keep the user's "00 00 * * *" in the state to satisfy Terraform.
	backupScheduleApiResp := types.StringValue(resp.BackupSchedule)
	if utils.SimplifyCronString(model.BackupSchedule.ValueString()) != utils.SimplifyCronString(backupScheduleApiResp.ValueString()) {
		// If the API actually changed it to something else, use the API value
		model.BackupSchedule = types.StringValue(resp.BackupSchedule)
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instanceId)
	model.InstanceId = types.StringValue(instanceId)
	model.Name = types.StringValue(resp.Name)
	model.ACL = aclList
	model.Flavor = flavorObject
	model.FlavorId = types.StringValue(resp.FlavorId)
	model.Replicas = types.Int32Value(int32(resp.Replicas))
	model.Storage = storageObject
	model.Version = types.StringValue(string(resp.Version))
	model.Options = optionsObject
	model.Region = types.StringValue(region)
	model.RetentionDays = types.Int32Value(resp.RetentionDays)
	model.Edition = types.StringValue(string(resp.Edition))
	model.Network = networkObject
	model.Encryption = encryptionObject
	return nil
}

func toCreatePayload(model *Model, acl []string, encryption *encryptionModel, flavor *flavorModel, storage *storageModel, options *optionsModel, network *networkModel) (*sqlserverflex.CreateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	// Encryption
	var encryptionPayload *sqlserverflex.InstanceEncryption
	if encryption != nil {
		encryptionPayload = &sqlserverflex.InstanceEncryption{}
		if !encryption.KekKeyId.IsNull() || !encryption.KekKeyId.IsUnknown() {
			encryptionPayload.KekKeyId = encryption.KekKeyId.ValueString()
		}
		if !encryption.KekKeyRingId.IsNull() || !encryption.KekKeyRingId.IsUnknown() {
			encryptionPayload.KekKeyRingId = encryption.KekKeyRingId.ValueString()
		}
		if !encryption.KekKeyVersion.IsNull() || !encryption.KekKeyVersion.IsUnknown() {
			encryptionPayload.KekKeyVersion = encryption.KekKeyVersion.ValueString()
		}
		if !encryption.ServiceAccount.IsNull() || !encryption.ServiceAccount.IsUnknown() {
			encryptionPayload.ServiceAccount = encryption.ServiceAccount.ValueString()
		}
	}

	// Network
	networkPayload := sqlserverflex.CreateInstancePayloadNetwork{}
	if acl != nil {
		networkPayload.Acl = acl
	} else if network != nil && !(network.Acl.IsNull() || network.Acl.IsUnknown()) {
		var err error
		networkPayload.Acl, err = conversion.StringListToSlice(network.Acl)
		if err != nil {
			return nil, err
		}
		networkPayload.AccessScope = (*sqlserverflex.InstanceNetworkAccessScope)(network.AccessScope.ValueStringPointer())
	} else {
		// TODO: Return here an error after the deprecation period. During the deprecation period, we set here an empty ACL to catch the breaking change from v2 -> v3 api.
		networkPayload.Acl = []string{}
	}

	// Flavor
	var flavorId string
	if flavor != nil && !(flavor.Id.IsNull() || flavor.Id.IsUnknown()) {
		flavorId = flavor.Id.ValueString()
	} else if !model.FlavorId.IsNull() {
		flavorId = model.FlavorId.ValueString()
	} else {
		return nil, fmt.Errorf("flavor is missing")
	}

	// Storage
	storagePayload := sqlserverflex.StorageCreate{}
	if storage == nil {
		return nil, fmt.Errorf("storage configuration is missing")
	}
	storagePayload.Class = storage.Class.ValueString()
	storagePayload.Size = storage.Size.ValueInt64()

	// Retention days
	var retentionDays int32
	if options != nil && !options.RetentionDays.IsNull() {
		retentionDays = options.RetentionDays.ValueInt32()
	} else if !model.RetentionDays.IsNull() {
		retentionDays = model.RetentionDays.ValueInt32()
	} else {
		return nil, fmt.Errorf("retention days are missing")
	}

	return &sqlserverflex.CreateInstancePayload{
		BackupSchedule:       model.BackupSchedule.ValueString(),
		Encryption:           encryptionPayload,
		FlavorId:             flavorId,
		Labels:               nil,
		Name:                 model.Name.ValueString(),
		Network:              networkPayload,
		RetentionDays:        retentionDays,
		Storage:              storagePayload,
		Version:              sqlserverflex.InstanceVersion(model.Version.ValueString()),
		AdditionalProperties: nil,
	}, nil
}

func toUpdatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel, options *optionsModel, network *networkModel) (*sqlserverflex.PartialUpdateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	networkPayload := &sqlserverflex.PartialUpdateInstancePayloadNetwork{}
	if acl != nil {
		networkPayload.Acl = acl
	} else if network != nil && !(network.Acl.IsNull() || network.Acl.IsUnknown()) {
		var err error
		networkPayload.Acl, err = conversion.StringListToSlice(network.Acl)
		if err != nil {
			return nil, err
		}
	} else {
		// TODO: Return here an error after the deprecation period. During the deprecation period, we set here an empty ACL to catch the breaking change from v2 -> v3 api.
		networkPayload.Acl = []string{}
	}

	var flavorId *string
	if flavor != nil && !(flavor.Id.IsNull() || flavor.Id.IsUnknown()) {
		flavorId = flavor.Id.ValueStringPointer()
	} else if !(model.FlavorId.IsNull() || model.FlavorId.IsUnknown()) {
		flavorId = model.FlavorId.ValueStringPointer()
	} else {
		return nil, fmt.Errorf("flavor is missing")
	}

	var versionPayload *sqlserverflex.InstanceVersionOpt
	if version := conversion.StringValueToPointer(model.Version); version != nil {
		versionPayload = new(sqlserverflex.InstanceVersionOpt(*version))
	}

	storagePayload := &sqlserverflex.StorageUpdate{}
	if storage == nil || storage.Size.IsNull() {
		return nil, fmt.Errorf("storage configuration is missing")
	}
	storagePayload.Size = storage.Size.ValueInt64Pointer()

	// Retention days
	var retentionDays *int32
	if !(model.RetentionDays.IsNull() || model.RetentionDays.IsUnknown()) {
		retentionDays = model.RetentionDays.ValueInt32Pointer()
	} else if !(options.RetentionDays.IsNull() || options.RetentionDays.IsUnknown()) {
		retentionDays = options.RetentionDays.ValueInt32Pointer()
	}

	return &sqlserverflex.PartialUpdateInstancePayload{
		BackupSchedule:       conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:             flavorId,
		Labels:               nil,
		Name:                 conversion.StringValueToPointer(model.Name),
		Network:              networkPayload,
		RetentionDays:        retentionDays,
		Storage:              storagePayload,
		Version:              versionPayload,
		AdditionalProperties: nil,
	}, nil
}

type sqlserverflexClient interface {
	ListFlavors(ctx context.Context, projectId, region string) sqlserverflex.ApiListFlavorsRequest
	ListFlavorsExecute(r sqlserverflex.ApiListFlavorsRequest) (*sqlserverflex.ListFlavorsResponse, error)
}

func loadFlavorId(ctx context.Context, client sqlserverflexClient, model *Model, flavor *flavorModel) error {
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
		return fmt.Errorf("listing sqlserverflex flavors: %w", err)
	}

	avl := ""
	if res.Flavors == nil {
		return fmt.Errorf("finding flavors for project %s", projectId)
	}
	for _, f := range res.Flavors {
		if f.Id == "" || f.Cpu == 0 || f.Memory == 0 {
			continue
		}
		if f.Cpu == *cpu && f.Memory == *ram {
			flavor.Id = types.StringValue(f.Id)
			flavor.Description = types.StringValue(f.Description)
			break
		}
		avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM", avl, f.Cpu, f.Memory)
	}
	if flavor.Id.ValueString() == "" {
		return fmt.Errorf("couldn't find flavor, available specs are:%s", avl)
	}

	return nil
}

func getFlavor(ctx context.Context, client sqlserverflexClient, projectId, region, flavorId string) (*sqlserverflex.ListFlavors, error) {
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
