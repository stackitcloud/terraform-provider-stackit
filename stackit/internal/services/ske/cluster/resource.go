package ske

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	serviceenablementUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceenablement/utils"
	skeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceenablement"
	enablementWait "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/wait"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	skeWait "github.com/stackitcloud/stackit-sdk-go/services/ske/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
	"golang.org/x/mod/semver"
)

const (
	DefaultOSName                = "flatcar"
	DefaultCRI                   = "containerd"
	DefaultVolumeType            = "storage_premium_perf1"
	DefaultVolumeSizeGB    int64 = 20
	VersionStateSupported        = "supported"
	VersionStatePreview          = "preview"
	VersionStateDeprecated       = "deprecated"

	SKEUpdateDoc = "SKE automatically updates the cluster Kubernetes version if you have set `maintenance.enable_kubernetes_version_updates` to true or if there is a mandatory update, as described in [Updates for Kubernetes versions and Operating System versions in SKE](https://docs.stackit.cloud/stackit/en/version-updates-in-ske-10125631.html)."
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &clusterResource{}
	_ resource.ResourceWithConfigure   = &clusterResource{}
	_ resource.ResourceWithImportState = &clusterResource{}
	_ resource.ResourceWithModifyPlan  = &clusterResource{}
)

type skeClient interface {
	GetClusterExecute(ctx context.Context, projectId, region, clusterName string) (*ske.Cluster, error)
}

type Model struct {
	Id                    types.String `tfsdk:"id"` // needed by TF
	ProjectId             types.String `tfsdk:"project_id"`
	Name                  types.String `tfsdk:"name"`
	KubernetesVersionMin  types.String `tfsdk:"kubernetes_version_min"`
	KubernetesVersionUsed types.String `tfsdk:"kubernetes_version_used"`
	NodePools             types.List   `tfsdk:"node_pools"`
	Maintenance           types.Object `tfsdk:"maintenance"`
	Network               types.Object `tfsdk:"network"`
	Hibernations          types.List   `tfsdk:"hibernations"`
	Extensions            types.Object `tfsdk:"extensions"`
	EgressAddressRanges   types.List   `tfsdk:"egress_address_ranges"`
	PodAddressRanges      types.List   `tfsdk:"pod_address_ranges"`
	Region                types.String `tfsdk:"region"`
}

// Struct corresponding to Model.NodePools[i]
type nodePool struct {
	Name                  types.String `tfsdk:"name"`
	MachineType           types.String `tfsdk:"machine_type"`
	OSName                types.String `tfsdk:"os_name"`
	OSVersionMin          types.String `tfsdk:"os_version_min"`
	OSVersion             types.String `tfsdk:"os_version"`
	OSVersionUsed         types.String `tfsdk:"os_version_used"`
	Minimum               types.Int64  `tfsdk:"minimum"`
	Maximum               types.Int64  `tfsdk:"maximum"`
	MaxSurge              types.Int64  `tfsdk:"max_surge"`
	MaxUnavailable        types.Int64  `tfsdk:"max_unavailable"`
	VolumeType            types.String `tfsdk:"volume_type"`
	VolumeSize            types.Int64  `tfsdk:"volume_size"`
	Labels                types.Map    `tfsdk:"labels"`
	Taints                types.List   `tfsdk:"taints"`
	CRI                   types.String `tfsdk:"cri"`
	AvailabilityZones     types.List   `tfsdk:"availability_zones"`
	AllowSystemComponents types.Bool   `tfsdk:"allow_system_components"`
}

// Types corresponding to nodePool
var nodePoolTypes = map[string]attr.Type{
	"name":                    basetypes.StringType{},
	"machine_type":            basetypes.StringType{},
	"os_name":                 basetypes.StringType{},
	"os_version_min":          basetypes.StringType{},
	"os_version":              basetypes.StringType{},
	"os_version_used":         basetypes.StringType{},
	"minimum":                 basetypes.Int64Type{},
	"maximum":                 basetypes.Int64Type{},
	"max_surge":               basetypes.Int64Type{},
	"max_unavailable":         basetypes.Int64Type{},
	"volume_type":             basetypes.StringType{},
	"volume_size":             basetypes.Int64Type{},
	"labels":                  basetypes.MapType{ElemType: types.StringType},
	"taints":                  basetypes.ListType{ElemType: types.ObjectType{AttrTypes: taintTypes}},
	"cri":                     basetypes.StringType{},
	"availability_zones":      basetypes.ListType{ElemType: types.StringType},
	"allow_system_components": basetypes.BoolType{},
}

// Struct corresponding to nodePool.Taints[i]
type taint struct {
	Effect types.String `tfsdk:"effect"`
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
}

// Types corresponding to taint
var taintTypes = map[string]attr.Type{
	"effect": basetypes.StringType{},
	"key":    basetypes.StringType{},
	"value":  basetypes.StringType{},
}

// Struct corresponding to Model.maintenance
type maintenance struct {
	EnableKubernetesVersionUpdates   types.Bool   `tfsdk:"enable_kubernetes_version_updates"`
	EnableMachineImageVersionUpdates types.Bool   `tfsdk:"enable_machine_image_version_updates"`
	Start                            types.String `tfsdk:"start"`
	End                              types.String `tfsdk:"end"`
}

// Types corresponding to maintenance
var maintenanceTypes = map[string]attr.Type{
	"enable_kubernetes_version_updates":    basetypes.BoolType{},
	"enable_machine_image_version_updates": basetypes.BoolType{},
	"start":                                basetypes.StringType{},
	"end":                                  basetypes.StringType{},
}

// Struct corresponding to Model.Network
type network struct {
	ID types.String `tfsdk:"id"`
}

// Types corresponding to network
var networkTypes = map[string]attr.Type{
	"id": basetypes.StringType{},
}

// Struct corresponding to Model.Hibernations[i]
type hibernation struct {
	Start    types.String `tfsdk:"start"`
	End      types.String `tfsdk:"end"`
	Timezone types.String `tfsdk:"timezone"`
}

// Types corresponding to hibernation
var hibernationTypes = map[string]attr.Type{
	"start":    basetypes.StringType{},
	"end":      basetypes.StringType{},
	"timezone": basetypes.StringType{},
}

// Struct corresponding to Model.Extensions
type extensions struct {
	Argus         types.Object `tfsdk:"argus"`
	Observability types.Object `tfsdk:"observability"`
	ACL           types.Object `tfsdk:"acl"`
	DNS           types.Object `tfsdk:"dns"`
}

// Types corresponding to extensions
var extensionsTypes = map[string]attr.Type{
	"argus":         basetypes.ObjectType{AttrTypes: argusTypes},
	"observability": basetypes.ObjectType{AttrTypes: observabilityTypes},
	"acl":           basetypes.ObjectType{AttrTypes: aclTypes},
	"dns":           basetypes.ObjectType{AttrTypes: dnsTypes},
}

// Struct corresponding to extensions.ACL
type acl struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	AllowedCIDRs types.List `tfsdk:"allowed_cidrs"`
}

// Types corresponding to acl
var aclTypes = map[string]attr.Type{
	"enabled":       basetypes.BoolType{},
	"allowed_cidrs": basetypes.ListType{ElemType: types.StringType},
}

// Struct corresponding to extensions.Argus
type argus struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	ArgusInstanceId types.String `tfsdk:"argus_instance_id"`
}

// Types corresponding to argus
var argusTypes = map[string]attr.Type{
	"enabled":           basetypes.BoolType{},
	"argus_instance_id": basetypes.StringType{},
}

// Struct corresponding to extensions.Observability
type observability struct {
	Enabled    types.Bool   `tfsdk:"enabled"`
	InstanceId types.String `tfsdk:"instance_id"`
}

// Types corresponding to observability
var observabilityTypes = map[string]attr.Type{
	"enabled":     basetypes.BoolType{},
	"instance_id": basetypes.StringType{},
}

// Struct corresponding to extensions.DNS
type dns struct {
	Enabled types.Bool `tfsdk:"enabled"`
	Zones   types.List `tfsdk:"zones"`
}

// Types corresponding to DNS
var dnsTypes = map[string]attr.Type{
	"enabled": basetypes.BoolType{},
	"zones":   basetypes.ListType{ElemType: types.StringType},
}

// NewClusterResource is a helper function to simplify the provider implementation.
func NewClusterResource() resource.Resource {
	return &clusterResource{}
}

// clusterResource is the resource implementation.
type clusterResource struct {
	skeClient        *ske.APIClient
	enablementClient *serviceenablement.APIClient
	providerData     core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *clusterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Metadata returns the resource type name.
func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_cluster"
}

// Configure adds the provider configured client to the resource.
func (r *clusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	skeClient := skeUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	serviceEnablementClient := serviceenablementUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.skeClient = skeClient
	r.enablementClient = serviceEnablementClient
	tflog.Info(ctx, "SKE cluster clients configured")
}

// Schema defines the schema for the resource.
func (r *clusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main": "SKE Cluster Resource schema. Must have a `region` specified in the provider configuration.",
		"node_pools_plan_note": "When updating `node_pools` of a `stackit_ske_cluster`, the Terraform plan might appear incorrect as it matches the node pools by index rather than by name. " +
			"However, the SKE API correctly identifies node pools by name and applies the intended changes. Please review your changes carefully to ensure the correct configuration will be applied.",
		"max_surge":           "Maximum number of additional VMs that are created during an update.",
		"max_unavailable":     "Maximum number of VMs that that can be unavailable during an update.",
		"nodepool_validators": "If set (larger than 0), then it must be at least the amount of zones configured for the nodepool. The `max_surge` and `max_unavailable` fields cannot both be unset at the same time.",
		"region":              "The resource region. If not defined, the provider region is used.",
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("%s\n%s", descriptions["main"], descriptions["node_pools_plan_note"]),
		// Callout block: https://developer.hashicorp.com/terraform/registry/providers/docs#callouts
		MarkdownDescription: fmt.Sprintf("%s\n\n-> %s", descriptions["main"], descriptions["node_pools_plan_note"]),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`name`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the cluster is associated.",
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
				Description: "The cluster name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"kubernetes_version_min": schema.StringAttribute{
				Description: "The minimum Kubernetes version. This field will be used to set the minimum kubernetes version on creation/update of the cluster. If unset, the latest supported Kubernetes version will be used. " + SKEUpdateDoc + " To get the current kubernetes version being used for your cluster, use the read-only `kubernetes_version_used` field.",
				Optional:    true,
				Validators: []validator.String{
					validate.VersionNumber(),
				},
			},
			"kubernetes_version_used": schema.StringAttribute{
				Description: "Full Kubernetes version used. For example, if 1.22 was set in `kubernetes_version_min`, this value may result to 1.22.15. " + SKEUpdateDoc,
				Computed:    true,
			},
			"egress_address_ranges": schema.ListAttribute{
				Description: "The outgoing network ranges (in CIDR notation) of traffic originating from workload on the cluster.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"pod_address_ranges": schema.ListAttribute{
				Description: "The network ranges (in CIDR notation) used by pods of the cluster.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"node_pools": schema.ListNestedAttribute{
				Description: "One or more `node_pool` block as defined below.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Specifies the name of the node pool.",
							Required:    true,
						},
						"machine_type": schema.StringAttribute{
							Description: "The machine type.",
							Required:    true,
						},
						"availability_zones": schema.ListAttribute{
							Description: "Specify a list of availability zones. E.g. `eu01-m`",
							Required:    true,
							ElementType: types.StringType,
						},
						"allow_system_components": schema.BoolAttribute{
							Description: "Allow system components to run on this node pool.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
						"minimum": schema.Int64Attribute{
							Description: "Minimum number of nodes in the pool.",
							Required:    true,
						},
						"maximum": schema.Int64Attribute{
							Description: "Maximum number of nodes in the pool.",
							Required:    true,
						},
						"max_surge": schema.Int64Attribute{
							Description: fmt.Sprintf("%s %s", descriptions["max_surge"], descriptions["nodepool_validators"]),
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"max_unavailable": schema.Int64Attribute{
							Description: fmt.Sprintf("%s %s", descriptions["max_unavailable"], descriptions["nodepool_validators"]),
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"os_name": schema.StringAttribute{
							Description: "The name of the OS image. Defaults to `flatcar`.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(DefaultOSName),
						},
						"os_version_min": schema.StringAttribute{
							Description: "The minimum OS image version. This field will be used to set the minimum OS image version on creation/update of the cluster. If unset, the latest supported OS image version will be used. " + SKEUpdateDoc + " To get the current OS image version being used for the node pool, use the read-only `os_version_used` field.",
							Optional:    true,
							Validators: []validator.String{
								validate.VersionNumber(),
							},
						},
						"os_version": schema.StringAttribute{
							Description:        "This field is deprecated, use `os_version_min` to configure the version and `os_version_used` to get the currently used version instead.",
							DeprecationMessage: "Use `os_version_min` to configure the version and `os_version_used` to get the currently used version instead. Setting a specific OS image version will cause errors during minor OS upgrades due to forced updates.",
							Optional:           true,
						},
						"os_version_used": schema.StringAttribute{
							Description: "Full OS image version used. For example, if 3815.2 was set in `os_version_min`, this value may result to 3815.2.2. " + SKEUpdateDoc,
							Computed:    true,
						},
						"volume_type": schema.StringAttribute{
							Description: "Specifies the volume type. Defaults to `storage_premium_perf1`.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(DefaultVolumeType),
						},
						"volume_size": schema.Int64Attribute{
							Description: "The volume size in GB. Defaults to `20`",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(DefaultVolumeSizeGB),
						},
						"labels": schema.MapAttribute{
							Description: "Labels to add to each node.",
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.Map{
								mapplanmodifier.UseStateForUnknown(),
							},
						},
						"taints": schema.ListNestedAttribute{
							Description: "Specifies a taint list as defined below.",
							Optional:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"effect": schema.StringAttribute{
										Description: "The taint effect. E.g `PreferNoSchedule`.",
										Required:    true,
									},
									"key": schema.StringAttribute{
										Description: "Taint key to be applied to a node.",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.LengthAtLeast(1),
										},
									},
									"value": schema.StringAttribute{
										Description: "Taint value corresponding to the taint key.",
										Optional:    true,
										Computed:    true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
								},
							},
						},
						"cri": schema.StringAttribute{
							Description: "Specifies the container runtime. Defaults to `containerd`",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(DefaultCRI),
						},
					},
				},
			},
			"maintenance": schema.SingleNestedAttribute{
				Description: "A single maintenance block as defined below.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enable_kubernetes_version_updates": schema.BoolAttribute{
						Description: "Flag to enable/disable auto-updates of the Kubernetes version. Defaults to `true`. " + SKEUpdateDoc,
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"enable_machine_image_version_updates": schema.BoolAttribute{
						Description: "Flag to enable/disable auto-updates of the OS image version. Defaults to `true`. " + SKEUpdateDoc,
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"start": schema.StringAttribute{
						Description: "Time for maintenance window start. E.g. `01:23:45Z`, `05:00:00+02:00`.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^(((\d{2}:\d{2}:\d{2}(?:\.\d+)?))(Z|[\+-]\d{2}:\d{2})?)$`),
								"must be a full-time as defined by RFC3339, Section 5.6. E.g. `01:23:45Z`, `05:00:00+02:00`",
							),
						},
					},
					"end": schema.StringAttribute{
						Description: "Time for maintenance window end. E.g. `01:23:45Z`, `05:00:00+02:00`.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^(((\d{2}:\d{2}:\d{2}(?:\.\d+)?))(Z|[\+-]\d{2}:\d{2})?)$`),
								"must be a full-time as defined by RFC3339, Section 5.6. E.g. `01:23:45Z`, `05:00:00+02:00`",
							),
						},
					},
				},
			},
			"network": schema.SingleNestedAttribute{
				Description: "Network block as defined below.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "ID of the STACKIT Network Area (SNA) network into which the cluster will be deployed.",
						Optional:    true,
						Validators: []validator.String{
							validate.UUID(),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"hibernations": schema.ListNestedAttribute{
				Description: "One or more hibernation block as defined below.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							Description: "Start time of cluster hibernation in crontab syntax. E.g. `0 18 * * *` for starting everyday at 6pm.",
							Required:    true,
						},
						"end": schema.StringAttribute{
							Description: "End time of hibernation in crontab syntax. E.g. `0 8 * * *` for waking up the cluster at 8am.",
							Required:    true,
						},
						"timezone": schema.StringAttribute{
							Description: "Timezone name corresponding to a file in the IANA Time Zone database. i.e. `Europe/Berlin`.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"extensions": schema.SingleNestedAttribute{
				Description: "A single extensions block as defined below.",
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"argus": schema.SingleNestedAttribute{
						Description:        "A single argus block as defined below. This field is deprecated and will be removed 06 January 2026.",
						DeprecationMessage: "Use observability instead.",
						Optional:           true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Flag to enable/disable Argus extensions.",
								Required:    true,
							},
							"argus_instance_id": schema.StringAttribute{
								Description: "Argus instance ID to choose which Argus instance is used. Required when enabled is set to `true`.",
								Optional:    true,
							},
						},
					},
					"observability": schema.SingleNestedAttribute{
						Description: "A single observability block as defined below.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Flag to enable/disable Observability extensions.",
								Required:    true,
							},
							"instance_id": schema.StringAttribute{
								Description: "Observability instance ID to choose which Observability instance is used. Required when enabled is set to `true`.",
								Optional:    true,
							},
						},
					},
					"acl": schema.SingleNestedAttribute{
						Description: "Cluster access control configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Is ACL enabled?",
								Required:    true,
							},
							"allowed_cidrs": schema.ListAttribute{
								Description: "Specify a list of CIDRs to whitelist.",
								Required:    true,
								ElementType: types.StringType,
							},
						},
					},
					"dns": schema.SingleNestedAttribute{
						Description: "DNS extension configuration",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Flag to enable/disable DNS extensions",
								Required:    true,
							},
							"zones": schema.ListAttribute{
								Description: "Specify a list of domain filters for externalDNS (e.g., `foo.runs.onstackit.cloud`)",
								Optional:    true,
								Computed:    true,
								ElementType: types.StringType,
								PlanModifiers: []planmodifier.List{
									listplanmodifier.UseStateForUnknown(),
								},
								Validators: []validator.List{
									listvalidator.ValueStringsAre(validate.NoUUID()),
								},
							},
						},
					},
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

// The argus extension is deprecated but can still be used until it is removed on 06 January 2026.
func (r *clusterResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var resourceModel Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &resourceModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// function is used in order to be able to write easier unit tests
	validateConfig(ctx, &resp.Diagnostics, &resourceModel)
}

func validateConfig(ctx context.Context, respDiags *diag.Diagnostics, model *Model) {
	// If no extensions are configured, return without error.
	if utils.IsUndefined(model.Extensions) {
		return
	}

	extensions := &extensions{}
	diags := model.Extensions.As(ctx, extensions, basetypes.ObjectAsOptions{})
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return
	}

	if !utils.IsUndefined(extensions.Argus) && !utils.IsUndefined(extensions.Observability) {
		core.LogAndAddError(ctx, respDiags, "Error configuring cluster", "You cannot provide both the `argus` and `observability` extension fields simultaneously. Please remove the deprecated `argus` field, and use `observability`.")
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	clusterName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", clusterName)
	ctx = tflog.SetField(ctx, "region", region)

	// If SKE functionality is not enabled, enable it
	err := r.enablementClient.EnableServiceRegional(ctx, region, projectId, utils.SKEServiceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating cluster", fmt.Sprintf("Calling API to enable SKE: %v", err))
		return
	}

	_, err = enablementWait.EnableServiceWaitHandler(ctx, r.enablementClient, region, projectId, utils.SKEServiceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating cluster", fmt.Sprintf("Wait for SKE enablement: %v", err))
		return
	}

	availableKubernetesVersions, availableMachines, err := r.loadAvailableVersions(ctx, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating cluster", fmt.Sprintf("Loading available Kubernetes and machine image versions: %v", err))
		return
	}

	r.createOrUpdateCluster(ctx, &resp.Diagnostics, &model, availableKubernetesVersions, availableMachines, nil, nil)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "SKE cluster created")
}

func sortK8sVersions(versions []ske.KubernetesVersion) {
	sort.Slice(versions, func(i, j int) bool {
		v1, v2 := (versions)[i].Version, (versions)[j].Version
		if v1 == nil {
			return false
		}
		if v2 == nil {
			return true
		}

		// we have to make copies of the input strings to add prefixes,
		// otherwise we would be changing the passed elements
		t1, t2 := *v1, *v2

		if !strings.HasPrefix(t1, "v") {
			t1 = "v" + t1
		}
		if !strings.HasPrefix(t2, "v") {
			t2 = "v" + t2
		}
		return semver.Compare(t1, t2) > 0
	})
}

// loadAvailableVersions loads the available k8s and machine versions from the API.
// The k8s versions are sorted  descending order, i.e. the latest versions (including previews)
// are listed first
func (r *clusterResource) loadAvailableVersions(ctx context.Context, region string) ([]ske.KubernetesVersion, []ske.MachineImage, error) {
	c := r.skeClient
	res, err := c.ListProviderOptions(ctx, region).Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("calling API: %w", err)
	}

	if res.KubernetesVersions == nil {
		return nil, nil, fmt.Errorf("API response has nil kubernetesVersions")
	}

	if res.MachineImages == nil {
		return nil, nil, fmt.Errorf("API response has nil machine images")
	}

	return *res.KubernetesVersions, *res.MachineImages, nil
}

// getCurrentVersions makes a call to get the details of a cluster and returns the current kubernetes version and a
// a map with the machine image for each nodepool, which can be used to check the current machine image versions.
// if the cluster doesn't exist or some error occurs, returns nil for both
func getCurrentVersions(ctx context.Context, c skeClient, m *Model) (kubernetesVersion *string, nodePoolMachineImages map[string]*ske.Image) {
	res, err := c.GetClusterExecute(ctx, m.ProjectId.ValueString(), m.Region.ValueString(), m.Name.ValueString())
	if err != nil || res == nil {
		return nil, nil
	}

	if res.Kubernetes != nil {
		kubernetesVersion = res.Kubernetes.Version
	}

	if res.Nodepools == nil {
		return kubernetesVersion, nil
	}

	nodePoolMachineImages = map[string]*ske.Image{}
	for _, nodePool := range *res.Nodepools {
		if nodePool.Name == nil || nodePool.Machine == nil || nodePool.Machine.Image == nil || nodePool.Machine.Image.Name == nil {
			continue
		}
		nodePoolMachineImages[*nodePool.Name] = nodePool.Machine.Image
	}

	return kubernetesVersion, nodePoolMachineImages
}

func (r *clusterResource) createOrUpdateCluster(ctx context.Context, diags *diag.Diagnostics, model *Model, availableKubernetesVersions []ske.KubernetesVersion, availableMachineVersions []ske.MachineImage, currentKubernetesVersion *string, currentMachineImages map[string]*ske.Image) {
	// cluster vars
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	region := model.Region.ValueString()
	kubernetes, hasDeprecatedVersion, err := toKubernetesPayload(model, availableKubernetesVersions, currentKubernetesVersion, diags)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Creating cluster config API payload: %v", err))
		return
	}
	if hasDeprecatedVersion {
		diags.AddWarning("Deprecated Kubernetes version", fmt.Sprintf("Version %s of Kubernetes is deprecated, please update it", *kubernetes.Version))
	}
	nodePools, deprecatedVersionsUsed, err := toNodepoolsPayload(ctx, model, availableMachineVersions, currentMachineImages)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Creating node pools API payload: %v", err))
		return
	}
	if len(deprecatedVersionsUsed) != 0 {
		diags.AddWarning("Deprecated node pools OS versions used", fmt.Sprintf("The following versions of machines are deprecated, please update them: [%s]", strings.Join(deprecatedVersionsUsed, ",")))
	}
	maintenance, err := toMaintenancePayload(ctx, model)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Creating maintenance API payload: %v", err))
		return
	}
	network, err := toNetworkPayload(ctx, model)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Creating network API payload: %v", err))
		return
	}
	hibernations, err := toHibernationsPayload(ctx, model)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Creating hibernations API payload: %v", err))
		return
	}
	extensions, err := toExtensionsPayload(ctx, model)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Creating extension API payload: %v", err))
		return
	}

	payload := ske.CreateOrUpdateClusterPayload{
		Extensions:  extensions,
		Hibernation: hibernations,
		Kubernetes:  kubernetes,
		Maintenance: maintenance,
		Network:     network,
		Nodepools:   &nodePools,
	}
	_, err = r.skeClient.CreateOrUpdateCluster(ctx, projectId, region, name).CreateOrUpdateClusterPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Calling API: %v", err))
		return
	}

	waitResp, err := skeWait.CreateOrUpdateClusterWaitHandler(ctx, r.skeClient, projectId, region, name).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Cluster creation waiting: %v", err))
		return
	}
	if waitResp.Status.Error != nil && waitResp.Status.Error.Message != nil && *waitResp.Status.Error.Code == ske.RUNTIMEERRORCODE_OBSERVABILITY_INSTANCE_NOT_FOUND {
		core.LogAndAddWarning(ctx, diags, "Warning during creating/updating cluster", fmt.Sprintf("Cluster is in Impaired state due to an invalid observability instance id, the cluster is usable but metrics won't be forwarded: %s", *waitResp.Status.Error.Message))
	}

	err = mapFields(ctx, waitResp, model, region)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
}

func toNodepoolsPayload(ctx context.Context, m *Model, availableMachineVersions []ske.MachineImage, currentMachineImages map[string]*ske.Image) ([]ske.Nodepool, []string, error) {
	nodePools := []nodePool{}
	diags := m.NodePools.ElementsAs(ctx, &nodePools, false)
	if diags.HasError() {
		return nil, nil, core.DiagsToError(diags)
	}

	cnps := []ske.Nodepool{}
	deprecatedVersionsUsed := []string{}
	for i := range nodePools {
		nodePool := nodePools[i]

		name := conversion.StringValueToPointer(nodePool.Name)
		if name == nil {
			return nil, nil, fmt.Errorf("found nil node pool name for node_pool[%d]", i)
		}

		// taints
		taintsModel := []taint{}
		diags := nodePool.Taints.ElementsAs(ctx, &taintsModel, false)
		if diags.HasError() {
			return nil, nil, core.DiagsToError(diags)
		}

		ts := []ske.Taint{}
		for _, v := range taintsModel {
			t := ske.Taint{
				Effect: ske.TaintGetEffectAttributeType(conversion.StringValueToPointer(v.Effect)),
				Key:    conversion.StringValueToPointer(v.Key),
				Value:  conversion.StringValueToPointer(v.Value),
			}
			ts = append(ts, t)
		}

		// labels
		var ls *map[string]string
		if nodePool.Labels.IsNull() {
			ls = nil
		} else {
			lsm := map[string]string{}
			for k, v := range nodePool.Labels.Elements() {
				nv, err := conversion.ToString(ctx, v)
				if err != nil {
					lsm[k] = ""
					continue
				}
				lsm[k] = nv
			}
			ls = &lsm
		}

		// zones
		zs := []string{}
		for _, v := range nodePool.AvailabilityZones.Elements() {
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			s, err := conversion.ToString(ctx, v)
			if err != nil {
				continue
			}
			zs = append(zs, s)
		}

		cn := ske.CRI{
			Name: ske.CRIGetNameAttributeType(conversion.StringValueToPointer(nodePool.CRI)),
		}

		providedVersionMin := conversion.StringValueToPointer(nodePool.OSVersionMin)
		if !nodePool.OSVersion.IsNull() {
			if providedVersionMin != nil {
				return nil, nil, fmt.Errorf("both `os_version` and `os_version_min` are set for for node_pool %q. Please use `os_version_min` only, `os_version` is deprecated", *name)
			}
			// os_version field deprecation
			// this if clause should be removed once os_version field is completely removed
			// os_version field value is used as minimum os version
			providedVersionMin = conversion.StringValueToPointer(nodePool.OSVersion)
		}

		machineOSName := conversion.StringValueToPointer(nodePool.OSName)
		if machineOSName == nil {
			return nil, nil, fmt.Errorf("found nil machine name for node_pool %q", *name)
		}

		currentMachineImage := currentMachineImages[*name]

		machineVersion, hasDeprecatedVersion, err := latestMatchingMachineVersion(availableMachineVersions, providedVersionMin, *machineOSName, currentMachineImage)
		if err != nil {
			return nil, nil, fmt.Errorf("getting latest matching machine image version: %w", err)
		}
		if hasDeprecatedVersion && machineVersion != nil {
			deprecatedVersionsUsed = append(deprecatedVersionsUsed, *machineVersion)
		}

		cnp := ske.Nodepool{
			Name:           name,
			Minimum:        conversion.Int64ValueToPointer(nodePool.Minimum),
			Maximum:        conversion.Int64ValueToPointer(nodePool.Maximum),
			MaxSurge:       conversion.Int64ValueToPointer(nodePool.MaxSurge),
			MaxUnavailable: conversion.Int64ValueToPointer(nodePool.MaxUnavailable),
			Machine: &ske.Machine{
				Type: conversion.StringValueToPointer(nodePool.MachineType),
				Image: &ske.Image{
					Name:    machineOSName,
					Version: machineVersion,
				},
			},
			Volume: &ske.Volume{
				Type: conversion.StringValueToPointer(nodePool.VolumeType),
				Size: conversion.Int64ValueToPointer(nodePool.VolumeSize),
			},
			Taints:                &ts,
			Cri:                   &cn,
			Labels:                ls,
			AvailabilityZones:     &zs,
			AllowSystemComponents: conversion.BoolValueToPointer(nodePool.AllowSystemComponents),
		}
		cnps = append(cnps, cnp)
	}

	if err := verifySystemComponentsInNodePools(cnps); err != nil {
		return nil, nil, err
	}

	return cnps, deprecatedVersionsUsed, nil
}

// verifySystemComponentsInNodePools checks if at least one node pool has the allow_system_components attribute set to true.
func verifySystemComponentsInNodePools(nodePools []ske.Nodepool) error {
	for _, nodePool := range nodePools {
		if nodePool.AllowSystemComponents != nil && *nodePool.AllowSystemComponents {
			return nil // A node pool allowing system components was found
		}
	}
	return fmt.Errorf("at least one node_pool must allow system components")
}

// latestMatchingMachineVersion determines the latest machine image version for the create/update payload.
// It considers the available versions for the specified OS (OSName), the minimum version configured by the user,
// and the current version in the cluster. The function's behavior is as follows:
//
// 1. If the minimum version is not set:
//   - Return the current version if it exists.
//   - Otherwise, return the latest available version for the specified OS.
//
// 2. If the minimum version is set:
//   - If the minimum version is a downgrade, use the current version instead.
//   - If a patch is not specified for the minimum version, return the latest patch for that minor version.
//
// 3. For the selected version, check its state and return it, indicating if it is deprecated or not.
func latestMatchingMachineVersion(availableImages []ske.MachineImage, versionMin *string, osName string, currentImage *ske.Image) (version *string, deprecated bool, err error) {
	deprecated = false

	if availableImages == nil {
		return nil, false, fmt.Errorf("nil available machine versions")
	}

	var availableMachineVersions []ske.MachineImageVersion
	for _, machine := range availableImages {
		if machine.Name != nil && *machine.Name == osName && machine.Versions != nil {
			availableMachineVersions = *machine.Versions
		}
	}

	if len(availableImages) == 0 {
		return nil, false, fmt.Errorf("there are no available machine versions for the provided machine image name %s", osName)
	}

	if versionMin == nil {
		// Different machine OSes have different versions.
		// If the current machine image is nil or the machine image name has been updated,
		// retrieve the latest supported version. Otherwise, use the current machine version.
		if currentImage == nil || currentImage.Name == nil || *currentImage.Name != osName {
			latestVersion, err := getLatestSupportedMachineVersion(availableMachineVersions)
			if err != nil {
				return nil, false, fmt.Errorf("get latest supported machine image version: %w", err)
			}
			return latestVersion, false, nil
		}
		versionMin = currentImage.Version
	} else if currentImage != nil && currentImage.Name != nil && *currentImage.Name == osName {
		// If the os_version_min is set but is lower than the current version used in the cluster,
		// retain the current version to avoid downgrading.
		minimumVersion := "v" + *versionMin
		currentVersion := "v" + *currentImage.Version

		if semver.Compare(minimumVersion, currentVersion) == -1 {
			versionMin = currentImage.Version
		}
	}

	var fullVersion bool
	versionExp := validate.FullVersionRegex
	versionRegex := regexp.MustCompile(versionExp)
	if versionRegex.MatchString(*versionMin) {
		fullVersion = true
	}

	providedVersionPrefixed := "v" + *versionMin

	if !semver.IsValid(providedVersionPrefixed) {
		return nil, false, fmt.Errorf("provided version is invalid")
	}

	var versionUsed *string
	var state *string
	var availableVersionsArray []string
	// Get the higher available version that matches the major, minor and patch version provided by the user
	for _, v := range availableMachineVersions {
		if v.State == nil || v.Version == nil {
			continue
		}
		availableVersionsArray = append(availableVersionsArray, *v.Version)
		vPreffixed := "v" + *v.Version

		if fullVersion {
			// [MAJOR].[MINOR].[PATCH] version provided, match available version
			if semver.Compare(vPreffixed, providedVersionPrefixed) == 0 {
				versionUsed = v.Version
				state = v.State
				break
			}
		} else {
			// [MAJOR].[MINOR] version provided, get the latest patch version
			if semver.MajorMinor(vPreffixed) == semver.MajorMinor(providedVersionPrefixed) &&
				(semver.Compare(vPreffixed, providedVersionPrefixed) == 1 || semver.Compare(vPreffixed, providedVersionPrefixed) == 0) &&
				(v.State != nil && *v.State != VersionStatePreview) {
				versionUsed = v.Version
				state = v.State
			}
		}
	}

	if versionUsed != nil {
		deprecated = strings.EqualFold(*state, VersionStateDeprecated)
	}

	// Throwing error if we could not match the version with the available versions
	if versionUsed == nil {
		return nil, false, fmt.Errorf("provided version is not one of the available machine image versions, available versions are: %s", strings.Join(availableVersionsArray, ","))
	}

	return versionUsed, deprecated, nil
}

func getLatestSupportedMachineVersion(versions []ske.MachineImageVersion) (*string, error) {
	foundMachineVersion := false
	var latestVersion *string
	for i := range versions {
		version := versions[i]
		if *version.State != VersionStateSupported {
			continue
		}
		if latestVersion != nil {
			oldSemVer := fmt.Sprintf("v%s", *latestVersion)
			newSemVer := fmt.Sprintf("v%s", *version.Version)
			if semver.Compare(newSemVer, oldSemVer) != 1 {
				continue
			}
		}

		foundMachineVersion = true
		latestVersion = version.Version
	}
	if !foundMachineVersion {
		return nil, fmt.Errorf("no supported machine version found")
	}
	return latestVersion, nil
}

func toHibernationsPayload(ctx context.Context, m *Model) (*ske.Hibernation, error) {
	hibernation := []hibernation{}
	diags := m.Hibernations.ElementsAs(ctx, &hibernation, false)
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	if len(hibernation) == 0 {
		return nil, nil
	}

	scs := []ske.HibernationSchedule{}
	for _, h := range hibernation {
		sc := ske.HibernationSchedule{
			Start: conversion.StringValueToPointer(h.Start),
			End:   conversion.StringValueToPointer(h.End),
		}
		if !h.Timezone.IsNull() && !h.Timezone.IsUnknown() {
			tz := h.Timezone.ValueString()
			sc.Timezone = &tz
		}
		scs = append(scs, sc)
	}

	return &ske.Hibernation{
		Schedules: &scs,
	}, nil
}

func toExtensionsPayload(ctx context.Context, m *Model) (*ske.Extension, error) {
	if m.Extensions.IsNull() || m.Extensions.IsUnknown() {
		return nil, nil
	}
	ex := extensions{}
	diags := m.Extensions.As(ctx, &ex, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting extensions object: %v", diags.Errors())
	}

	var skeAcl *ske.ACL
	if !(ex.ACL.IsNull() || ex.ACL.IsUnknown()) {
		acl := acl{}
		diags = ex.ACL.As(ctx, &acl, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting extensions.acl object: %v", diags.Errors())
		}
		aclEnabled := conversion.BoolValueToPointer(acl.Enabled)

		cidrs := []string{}
		diags = acl.AllowedCIDRs.ElementsAs(ctx, &cidrs, true)
		if diags.HasError() {
			return nil, fmt.Errorf("converting extensions.acl.cidrs object: %v", diags.Errors())
		}
		skeAcl = &ske.ACL{
			Enabled:      aclEnabled,
			AllowedCidrs: &cidrs,
		}
	}

	var skeObservability *ske.Observability
	if !utils.IsUndefined(ex.Observability) {
		observability := observability{}
		diags = ex.Observability.As(ctx, &observability, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting extensions.observability object: %v", diags.Errors())
		}
		observabilityEnabled := conversion.BoolValueToPointer(observability.Enabled)
		observabilityInstanceId := conversion.StringValueToPointer(observability.InstanceId)
		skeObservability = &ske.Observability{
			Enabled:    observabilityEnabled,
			InstanceId: observabilityInstanceId,
		}
	} else if !utils.IsUndefined(ex.Argus) { // Fallback to deprecated argus
		argus := argus{}
		diags = ex.Argus.As(ctx, &argus, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting extensions.argus object: %v", diags.Errors())
		}
		argusEnabled := conversion.BoolValueToPointer(argus.Enabled)
		argusInstanceId := conversion.StringValueToPointer(argus.ArgusInstanceId)
		skeObservability = &ske.Observability{
			Enabled:    argusEnabled,
			InstanceId: argusInstanceId,
		}
	}

	var skeDNS *ske.DNS
	if !(ex.DNS.IsNull() || ex.DNS.IsUnknown()) {
		dns := dns{}
		diags = ex.DNS.As(ctx, &dns, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("converting extensions.dns object: %v", diags.Errors())
		}
		dnsEnabled := conversion.BoolValueToPointer(dns.Enabled)

		zones := []string{}
		diags = dns.Zones.ElementsAs(ctx, &zones, true)
		if diags.HasError() {
			return nil, fmt.Errorf("converting extensions.dns.zones object: %v", diags.Errors())
		}
		skeDNS = &ske.DNS{
			Enabled: dnsEnabled,
			Zones:   &zones,
		}
	}

	return &ske.Extension{
		Acl:           skeAcl,
		Observability: skeObservability,
		Dns:           skeDNS,
	}, nil
}

func parseMaintenanceWindowTime(t string) (time.Time, error) {
	v, err := time.Parse("15:04:05-07:00", t)
	if err != nil {
		v, err = time.Parse("15:04:05Z", t)
	}
	return v, err
}

func toMaintenancePayload(ctx context.Context, m *Model) (*ske.Maintenance, error) {
	if m.Maintenance.IsNull() || m.Maintenance.IsUnknown() {
		return nil, nil
	}

	maintenance := maintenance{}
	diags := m.Maintenance.As(ctx, &maintenance, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting maintenance object: %v", diags.Errors())
	}

	var timeWindowStart *time.Time
	if !(maintenance.Start.IsNull() || maintenance.Start.IsUnknown()) {
		tempTime, err := parseMaintenanceWindowTime(maintenance.Start.ValueString())
		if err != nil {
			return nil, fmt.Errorf("converting maintenance object: %w", err)
		}
		timeWindowStart = sdkUtils.Ptr(tempTime)
	}

	var timeWindowEnd *time.Time
	if !(maintenance.End.IsNull() || maintenance.End.IsUnknown()) {
		tempTime, err := parseMaintenanceWindowTime(maintenance.End.ValueString())
		if err != nil {
			return nil, fmt.Errorf("converting maintenance object: %w", err)
		}
		timeWindowEnd = sdkUtils.Ptr(tempTime)
	}

	return &ske.Maintenance{
		AutoUpdate: &ske.MaintenanceAutoUpdate{
			KubernetesVersion:   conversion.BoolValueToPointer(maintenance.EnableKubernetesVersionUpdates),
			MachineImageVersion: conversion.BoolValueToPointer(maintenance.EnableMachineImageVersionUpdates),
		},
		TimeWindow: &ske.TimeWindow{
			Start: timeWindowStart,
			End:   timeWindowEnd,
		},
	}, nil
}

func toNetworkPayload(ctx context.Context, m *Model) (*ske.Network, error) {
	if m.Network.IsNull() || m.Network.IsUnknown() {
		return nil, nil
	}

	network := network{}
	diags := m.Network.As(ctx, &network, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting network object: %v", diags.Errors())
	}

	return &ske.Network{
		Id: conversion.StringValueToPointer(network.ID),
	}, nil
}

func mapFields(ctx context.Context, cl *ske.Cluster, m *Model, region string) error {
	if cl == nil {
		return fmt.Errorf("response input is nil")
	}
	if m == nil {
		return fmt.Errorf("model input is nil")
	}

	var name string
	if m.Name.ValueString() != "" {
		name = m.Name.ValueString()
	} else if cl.Name != nil {
		name = *cl.Name
	} else {
		return fmt.Errorf("name not present")
	}
	m.Name = types.StringValue(name)

	m.Id = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), region, name)
	m.Region = types.StringValue(region)

	if cl.Kubernetes != nil {
		m.KubernetesVersionUsed = types.StringPointerValue(cl.Kubernetes.Version)
	}

	m.EgressAddressRanges = types.ListNull(types.StringType)
	if cl.Status != nil {
		var diags diag.Diagnostics
		m.EgressAddressRanges, diags = types.ListValueFrom(ctx, types.StringType, cl.Status.EgressAddressRanges)
		if diags.HasError() {
			return fmt.Errorf("map egressAddressRanges: %w", core.DiagsToError(diags))
		}
	}

	m.PodAddressRanges = types.ListNull(types.StringType)
	if cl.Status != nil {
		var diags diag.Diagnostics
		m.PodAddressRanges, diags = types.ListValueFrom(ctx, types.StringType, cl.Status.PodAddressRanges)
		if diags.HasError() {
			return fmt.Errorf("map podAddressRanges: %w", core.DiagsToError(diags))
		}
	}

	err := mapNodePools(ctx, cl, m)
	if err != nil {
		return fmt.Errorf("map node_pools: %w", err)
	}
	err = mapMaintenance(ctx, cl, m)
	if err != nil {
		return fmt.Errorf("map maintenance: %w", err)
	}
	err = mapNetwork(cl, m)
	if err != nil {
		return fmt.Errorf("map network: %w", err)
	}
	err = mapHibernations(cl, m)
	if err != nil {
		return fmt.Errorf("map hibernations: %w", err)
	}
	err = mapExtensions(ctx, cl, m)
	if err != nil {
		return fmt.Errorf("map extensions: %w", err)
	}
	return nil
}

func mapNodePools(ctx context.Context, cl *ske.Cluster, model *Model) error {
	modelNodePoolOSVersion := map[string]basetypes.StringValue{}
	modelNodePoolOSVersionMin := map[string]basetypes.StringValue{}

	modelNodePools := []nodePool{}
	if !model.NodePools.IsNull() && !model.NodePools.IsUnknown() {
		diags := model.NodePools.ElementsAs(ctx, &modelNodePools, false)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
	}

	for i := range modelNodePools {
		name := conversion.StringValueToPointer(modelNodePools[i].Name)
		if name != nil {
			modelNodePoolOSVersion[*name] = modelNodePools[i].OSVersion
			modelNodePoolOSVersionMin[*name] = modelNodePools[i].OSVersionMin
		}
	}

	if cl.Nodepools == nil {
		model.NodePools = types.ListNull(types.ObjectType{AttrTypes: nodePoolTypes})
		return nil
	}

	nodePools := []attr.Value{}
	for i, nodePoolResp := range *cl.Nodepools {
		nodePool := map[string]attr.Value{
			"name":                    types.StringPointerValue(nodePoolResp.Name),
			"machine_type":            types.StringPointerValue(nodePoolResp.Machine.Type),
			"os_name":                 types.StringNull(),
			"os_version_min":          modelNodePoolOSVersionMin[*nodePoolResp.Name],
			"os_version":              modelNodePoolOSVersion[*nodePoolResp.Name],
			"minimum":                 types.Int64PointerValue(nodePoolResp.Minimum),
			"maximum":                 types.Int64PointerValue(nodePoolResp.Maximum),
			"max_surge":               types.Int64PointerValue(nodePoolResp.MaxSurge),
			"max_unavailable":         types.Int64PointerValue(nodePoolResp.MaxUnavailable),
			"volume_type":             types.StringNull(),
			"volume_size":             types.Int64PointerValue(nodePoolResp.Volume.Size),
			"labels":                  types.MapNull(types.StringType),
			"cri":                     types.StringNull(),
			"availability_zones":      types.ListNull(types.StringType),
			"allow_system_components": types.BoolPointerValue(nodePoolResp.AllowSystemComponents),
		}

		if nodePoolResp.Machine != nil && nodePoolResp.Machine.Image != nil {
			nodePool["os_name"] = types.StringPointerValue(nodePoolResp.Machine.Image.Name)
			nodePool["os_version_used"] = types.StringPointerValue(nodePoolResp.Machine.Image.Version)
		}

		if nodePoolResp.Volume != nil {
			nodePool["volume_type"] = types.StringPointerValue(nodePoolResp.Volume.Type)
		}

		if nodePoolResp.Cri != nil {
			nodePool["cri"] = types.StringValue(string(nodePoolResp.Cri.GetName()))
		}

		taintsInModel := false
		if i < len(modelNodePools) && !modelNodePools[i].Taints.IsNull() && !modelNodePools[i].Taints.IsUnknown() {
			taintsInModel = true
		}
		err := mapTaints(nodePoolResp.Taints, nodePool, taintsInModel)
		if err != nil {
			return fmt.Errorf("mapping index %d, field taints: %w", i, err)
		}

		if nodePoolResp.Labels != nil {
			elems := map[string]attr.Value{}
			for k, v := range *nodePoolResp.Labels {
				elems[k] = types.StringValue(v)
			}
			elemsTF, diags := types.MapValue(types.StringType, elems)
			if diags.HasError() {
				return fmt.Errorf("mapping index %d, field labels: %w", i, core.DiagsToError(diags))
			}
			nodePool["labels"] = elemsTF
		}

		if nodePoolResp.AvailabilityZones != nil {
			elemsTF, diags := types.ListValueFrom(ctx, types.StringType, *nodePoolResp.AvailabilityZones)
			if diags.HasError() {
				return fmt.Errorf("mapping index %d, field availability_zones: %w", i, core.DiagsToError(diags))
			}
			nodePool["availability_zones"] = elemsTF
		}

		nodePoolTF, diags := basetypes.NewObjectValue(nodePoolTypes, nodePool)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}
		nodePools = append(nodePools, nodePoolTF)
	}
	nodePoolsTF, diags := basetypes.NewListValue(types.ObjectType{AttrTypes: nodePoolTypes}, nodePools)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.NodePools = nodePoolsTF
	return nil
}

func mapTaints(t *[]ske.Taint, nodePool map[string]attr.Value, existInModel bool) error {
	if t == nil || len(*t) == 0 {
		if existInModel {
			taintsTF, diags := types.ListValue(types.ObjectType{AttrTypes: taintTypes}, []attr.Value{})
			if diags.HasError() {
				return fmt.Errorf("create empty taints list: %w", core.DiagsToError(diags))
			}
			nodePool["taints"] = taintsTF
			return nil
		}
		nodePool["taints"] = types.ListNull(types.ObjectType{AttrTypes: taintTypes})
		return nil
	}

	taints := []attr.Value{}

	for i, taintResp := range *t {
		taint := map[string]attr.Value{
			"effect": types.StringValue(string(taintResp.GetEffect())),
			"key":    types.StringPointerValue(taintResp.Key),
			"value":  types.StringPointerValue(taintResp.Value),
		}
		taintTF, diags := basetypes.NewObjectValue(taintTypes, taint)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}
		taints = append(taints, taintTF)
	}

	taintsTF, diags := basetypes.NewListValue(types.ObjectType{AttrTypes: taintTypes}, taints)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	nodePool["taints"] = taintsTF
	return nil
}

func mapHibernations(cl *ske.Cluster, m *Model) error {
	if cl.Hibernation == nil {
		if !m.Hibernations.IsNull() {
			emptyHibernations, diags := basetypes.NewListValue(basetypes.ObjectType{AttrTypes: hibernationTypes}, []attr.Value{})
			if diags.HasError() {
				return fmt.Errorf("hibernations is an empty list, converting to terraform empty list: %w", core.DiagsToError(diags))
			}
			m.Hibernations = emptyHibernations
			return nil
		}
		m.Hibernations = basetypes.NewListNull(basetypes.ObjectType{AttrTypes: hibernationTypes})
		return nil
	}

	if cl.Hibernation.Schedules == nil {
		emptyHibernations, diags := basetypes.NewListValue(basetypes.ObjectType{AttrTypes: hibernationTypes}, []attr.Value{})
		if diags.HasError() {
			return fmt.Errorf("hibernations is an empty list, converting to terraform empty list: %w", core.DiagsToError(diags))
		}
		m.Hibernations = emptyHibernations
		return nil
	}

	hibernations := []attr.Value{}
	for i, hibernationResp := range *cl.Hibernation.Schedules {
		hibernation := map[string]attr.Value{
			"start":    types.StringPointerValue(hibernationResp.Start),
			"end":      types.StringPointerValue(hibernationResp.End),
			"timezone": types.StringPointerValue(hibernationResp.Timezone),
		}
		hibernationTF, diags := basetypes.NewObjectValue(hibernationTypes, hibernation)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}
		hibernations = append(hibernations, hibernationTF)
	}

	hibernationsTF, diags := basetypes.NewListValue(types.ObjectType{AttrTypes: hibernationTypes}, hibernations)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	m.Hibernations = hibernationsTF
	return nil
}

func mapMaintenance(ctx context.Context, cl *ske.Cluster, m *Model) error {
	// Aligned with SKE team that a flattened data structure is fine, because no extensions are planned.
	if cl.Maintenance == nil {
		m.Maintenance = types.ObjectNull(maintenanceTypes)
		return nil
	}
	ekvu := types.BoolNull()
	if cl.Maintenance.AutoUpdate.KubernetesVersion != nil {
		ekvu = types.BoolValue(*cl.Maintenance.AutoUpdate.KubernetesVersion)
	}
	emvu := types.BoolNull()
	if cl.Maintenance.AutoUpdate.KubernetesVersion != nil {
		emvu = types.BoolValue(*cl.Maintenance.AutoUpdate.MachineImageVersion)
	}
	startTime, endTime, err := getMaintenanceTimes(ctx, cl, m)
	if err != nil {
		return fmt.Errorf("getting maintenance times: %w", err)
	}
	maintenanceValues := map[string]attr.Value{
		"enable_kubernetes_version_updates":    ekvu,
		"enable_machine_image_version_updates": emvu,
		"start":                                types.StringValue(startTime),
		"end":                                  types.StringValue(endTime),
	}
	maintenanceObject, diags := types.ObjectValue(maintenanceTypes, maintenanceValues)
	if diags.HasError() {
		return fmt.Errorf("create maintenance object: %w", core.DiagsToError(diags))
	}
	m.Maintenance = maintenanceObject
	return nil
}

func mapNetwork(cl *ske.Cluster, m *Model) error {
	if cl.Network == nil {
		m.Network = types.ObjectNull(networkTypes)
		return nil
	}

	// If the network field is not provided, the SKE API returns an empty object.
	// If we parse that object into the terraform model, it will produce an inconsistent result after apply error

	emptyNetwork := &ske.Network{}
	if *cl.Network == *emptyNetwork && m.Network.IsNull() {
		if m.Network.Attributes() == nil {
			m.Network = types.ObjectNull(networkTypes)
		}
		return nil
	}

	id := types.StringNull()
	if cl.Network.Id != nil {
		id = types.StringValue(*cl.Network.Id)
	}
	networkValues := map[string]attr.Value{
		"id": id,
	}
	networkObject, diags := types.ObjectValue(networkTypes, networkValues)
	if diags.HasError() {
		return fmt.Errorf("create network object: %w", core.DiagsToError(diags))
	}
	m.Network = networkObject
	return nil
}

func getMaintenanceTimes(ctx context.Context, cl *ske.Cluster, m *Model) (startTime, endTime string, err error) {
	startTimeAPI := *cl.Maintenance.TimeWindow.Start
	endTimeAPI := *cl.Maintenance.TimeWindow.End

	if m.Maintenance.IsNull() || m.Maintenance.IsUnknown() {
		return startTimeAPI.Format("15:04:05Z07:00"), endTimeAPI.Format("15:04:05Z07:00"), nil
	}

	maintenance := &maintenance{}
	diags := m.Maintenance.As(ctx, maintenance, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return "", "", fmt.Errorf("converting maintenance object %w", core.DiagsToError(diags.Errors()))
	}

	startTime = startTimeAPI.Format("15:04:05Z07:00")
	if !(maintenance.Start.IsNull() || maintenance.Start.IsUnknown()) {
		startTimeTF, err := time.Parse("15:04:05Z07:00", maintenance.Start.ValueString())
		if err != nil {
			return "", "", fmt.Errorf("parsing start time '%s' from TF config as RFC time: %w", maintenance.Start.ValueString(), err)
		}
		// If the start times from the API and the TF model just differ in format, we keep the current TF model value
		if startTimeAPI.Format("15:04:05Z07:00") == startTimeTF.Format("15:04:05Z07:00") {
			startTime = maintenance.Start.ValueString()
		}
	}

	endTime = endTimeAPI.Format("15:04:05Z07:00")
	if !(maintenance.End.IsNull() || maintenance.End.IsUnknown()) {
		endTimeTF, err := time.Parse("15:04:05Z07:00", maintenance.End.ValueString())
		if err != nil {
			return "", "", fmt.Errorf("parsing end time '%s' from TF config as RFC time: %w", maintenance.End.ValueString(), err)
		}
		// If the end times from the API and the TF model just differ in format, we keep the current TF model value
		if endTimeAPI.Format("15:04:05Z07:00") == endTimeTF.Format("15:04:05Z07:00") {
			endTime = maintenance.End.ValueString()
		}
	}

	return startTime, endTime, nil
}

func checkDisabledExtensions(ctx context.Context, ex *extensions) (aclDisabled, observabilityDisabled, dnsDisabled bool, err error) {
	var diags diag.Diagnostics
	acl := acl{}
	if ex.ACL.IsNull() {
		acl.Enabled = types.BoolValue(false)
	} else {
		diags = ex.ACL.As(ctx, &acl, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return false, false, false, fmt.Errorf("converting extensions.acl object: %v", diags.Errors())
		}
	}

	observability := observability{}
	if ex.Argus.IsNull() && ex.Observability.IsNull() {
		observability.Enabled = types.BoolValue(false)
	} else if !ex.Argus.IsNull() {
		argus := argus{}
		diags = ex.Argus.As(ctx, &argus, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return false, false, false, fmt.Errorf("converting extensions.argus object: %v", diags.Errors())
		}
		observability.Enabled = argus.Enabled
		observability.InstanceId = argus.ArgusInstanceId
	} else {
		diags = ex.Observability.As(ctx, &observability, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return false, false, false, fmt.Errorf("converting extensions.observability object: %v", diags.Errors())
		}
	}

	dns := dns{}
	if ex.DNS.IsNull() {
		dns.Enabled = types.BoolValue(false)
	} else {
		diags = ex.DNS.As(ctx, &dns, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return false, false, false, fmt.Errorf("converting extensions.dns object: %v", diags.Errors())
		}
	}

	return !acl.Enabled.ValueBool(), !observability.Enabled.ValueBool(), !dns.Enabled.ValueBool(), nil
}

func mapExtensions(ctx context.Context, cl *ske.Cluster, m *Model) error {
	if cl.Extensions == nil {
		m.Extensions = types.ObjectNull(extensionsTypes)
		return nil
	}

	var diags diag.Diagnostics
	ex := extensions{}
	if !m.Extensions.IsNull() {
		diags := m.Extensions.As(ctx, &ex, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("converting extensions object: %v", diags.Errors())
		}
	}

	// If the user provides the extensions block with the enabled flags as false
	// the SKE API will return an empty extensions block, which throws an inconsistent
	// result after apply error. To prevent this error, if both flags are false,
	// we set the fields provided by the user in the terraform model

	// If the extensions field is not provided, the SKE API returns an empty object.
	// If we parse that object into the terraform model, it will produce an inconsistent result after apply
	// error

	aclDisabled, observabilityDisabled, dnsDisabled, err := checkDisabledExtensions(ctx, &ex)
	if err != nil {
		return fmt.Errorf("checking if extensions are disabled: %w", err)
	}
	disabledExtensions := false
	if aclDisabled && observabilityDisabled && dnsDisabled {
		disabledExtensions = true
	}

	emptyExtensions := &ske.Extension{}
	if *cl.Extensions == *emptyExtensions && (disabledExtensions || m.Extensions.IsNull()) {
		if m.Extensions.Attributes() == nil {
			m.Extensions = types.ObjectNull(extensionsTypes)
		}
		return nil
	}

	aclExtension := types.ObjectNull(aclTypes)
	if cl.Extensions.Acl != nil {
		enabled := types.BoolNull()
		if cl.Extensions.Acl.Enabled != nil {
			enabled = types.BoolValue(*cl.Extensions.Acl.Enabled)
		}

		cidrsList, diags := types.ListValueFrom(ctx, types.StringType, cl.Extensions.Acl.AllowedCidrs)
		if diags.HasError() {
			return fmt.Errorf("creating allowed_cidrs list: %w", core.DiagsToError(diags))
		}

		aclValues := map[string]attr.Value{
			"enabled":       enabled,
			"allowed_cidrs": cidrsList,
		}

		aclExtension, diags = types.ObjectValue(aclTypes, aclValues)
		if diags.HasError() {
			return fmt.Errorf("creating acl: %w", core.DiagsToError(diags))
		}
	} else if aclDisabled && !ex.ACL.IsNull() {
		aclExtension = ex.ACL
	}

	// Deprecated: argus won't be received from backend. Use observabilty instead.
	argusExtension := types.ObjectNull(argusTypes)
	observabilityExtension := types.ObjectNull(observabilityTypes)
	if cl.Extensions.Observability != nil {
		enabled := types.BoolNull()
		if cl.Extensions.Observability.Enabled != nil {
			enabled = types.BoolValue(*cl.Extensions.Observability.Enabled)
		}

		observabilityInstanceId := types.StringNull()
		if cl.Extensions.Observability.InstanceId != nil {
			observabilityInstanceId = types.StringValue(*cl.Extensions.Observability.InstanceId)
		}

		observabilityExtensionValues := map[string]attr.Value{
			"enabled":     enabled,
			"instance_id": observabilityInstanceId,
		}

		argusExtensionValues := map[string]attr.Value{
			"enabled":           enabled,
			"argus_instance_id": observabilityInstanceId,
		}

		observabilityExtension, diags = types.ObjectValue(observabilityTypes, observabilityExtensionValues)
		if diags.HasError() {
			return fmt.Errorf("creating observability extension: %w", core.DiagsToError(diags))
		}
		argusExtension, diags = types.ObjectValue(argusTypes, argusExtensionValues)
		if diags.HasError() {
			return fmt.Errorf("creating argus extension: %w", core.DiagsToError(diags))
		}
	} else if observabilityDisabled && !ex.Observability.IsNull() {
		observabilityExtension = ex.Observability

		// set deprecated argus extension
		observability := observability{}
		diags = ex.Observability.As(ctx, &observability, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("converting extensions.observability object: %v", diags.Errors())
		}
		argusExtensionValues := map[string]attr.Value{
			"enabled":           observability.Enabled,
			"argus_instance_id": observability.InstanceId,
		}
		argusExtension, diags = types.ObjectValue(argusTypes, argusExtensionValues)
		if diags.HasError() {
			return fmt.Errorf("creating argus extension: %w", core.DiagsToError(diags))
		}
	}

	dnsExtension := types.ObjectNull(dnsTypes)
	if cl.Extensions.Dns != nil {
		enabled := types.BoolNull()
		if cl.Extensions.Dns.Enabled != nil {
			enabled = types.BoolValue(*cl.Extensions.Dns.Enabled)
		}

		zonesList, diags := types.ListValueFrom(ctx, types.StringType, cl.Extensions.Dns.Zones)
		if diags.HasError() {
			return fmt.Errorf("creating zones list: %w", core.DiagsToError(diags))
		}

		dnsValues := map[string]attr.Value{
			"enabled": enabled,
			"zones":   zonesList,
		}

		dnsExtension, diags = types.ObjectValue(dnsTypes, dnsValues)
		if diags.HasError() {
			return fmt.Errorf("creating dns: %w", core.DiagsToError(diags))
		}
	} else if dnsDisabled && !ex.DNS.IsNull() {
		dnsExtension = ex.DNS
	}

	// Deprecation: Argus was renamed to observability. Depending on which attribute was used in the terraform config the
	// according one has to be set here.
	var extensionsValues map[string]attr.Value
	if utils.IsUndefined(ex.Argus) {
		extensionsValues = map[string]attr.Value{
			"acl":           aclExtension,
			"argus":         types.ObjectNull(argusTypes),
			"observability": observabilityExtension,
			"dns":           dnsExtension,
		}
	} else {
		extensionsValues = map[string]attr.Value{
			"acl":           aclExtension,
			"argus":         argusExtension,
			"observability": types.ObjectNull(observabilityTypes),
			"dns":           dnsExtension,
		}
	}

	extensions, diags := types.ObjectValue(extensionsTypes, extensionsValues)
	if diags.HasError() {
		return fmt.Errorf("creating extensions: %w", core.DiagsToError(diags))
	}
	m.Extensions = extensions
	return nil
}

func toKubernetesPayload(m *Model, availableVersions []ske.KubernetesVersion, currentKubernetesVersion *string, diags *diag.Diagnostics) (kubernetesPayload *ske.Kubernetes, hasDeprecatedVersion bool, err error) {
	providedVersionMin := m.KubernetesVersionMin.ValueStringPointer()
	versionUsed, hasDeprecatedVersion, err := latestMatchingKubernetesVersion(availableVersions, providedVersionMin, currentKubernetesVersion, diags)
	if err != nil {
		return nil, false, fmt.Errorf("getting latest matching kubernetes version: %w", err)
	}

	k := &ske.Kubernetes{
		Version: versionUsed,
	}
	return k, hasDeprecatedVersion, nil
}

func latestMatchingKubernetesVersion(availableVersions []ske.KubernetesVersion, kubernetesVersionMin, currentKubernetesVersion *string, diags *diag.Diagnostics) (version *string, deprecated bool, err error) {
	if availableVersions == nil {
		return nil, false, fmt.Errorf("nil available kubernetes versions")
	}

	if kubernetesVersionMin == nil {
		if currentKubernetesVersion == nil {
			latestVersion, err := getLatestSupportedKubernetesVersion(availableVersions)
			if err != nil {
				return nil, false, fmt.Errorf("get latest supported kubernetes version: %w", err)
			}
			return latestVersion, false, nil
		}
		kubernetesVersionMin = currentKubernetesVersion
	} else if currentKubernetesVersion != nil {
		// For an already existing cluster, if kubernetes_version_min is set to a lower version than what is being used in the cluster
		// return the currently used version
		kubernetesVersionUsed := *currentKubernetesVersion
		kubernetesVersionMinString := *kubernetesVersionMin

		minVersionPrefixed := "v" + kubernetesVersionMinString
		usedVersionPrefixed := "v" + kubernetesVersionUsed

		if semver.Compare(minVersionPrefixed, usedVersionPrefixed) == -1 {
			kubernetesVersionMin = currentKubernetesVersion
		}
	}

	versionRegex := regexp.MustCompile(validate.FullVersionRegex)
	fullVersion := versionRegex.MatchString(*kubernetesVersionMin)

	providedVersionPrefixed := "v" + *kubernetesVersionMin
	if !semver.IsValid(providedVersionPrefixed) {
		return nil, false, fmt.Errorf("provided version is invalid")
	}

	var (
		selectedVersion        *ske.KubernetesVersion
		availableVersionsArray []string
	)
	if fullVersion {
		availableVersionsArray, selectedVersion = selectFullVersion(availableVersions, providedVersionPrefixed)
	} else {
		availableVersionsArray, selectedVersion = selectMatchingVersion(availableVersions, providedVersionPrefixed)
	}

	deprecated = isDeprecated(selectedVersion)

	if isPreview(selectedVersion) {
		diags.AddWarning("preview version selected", fmt.Sprintf("only the preview version %q matched the selection criteria", *selectedVersion.Version))
	}

	// Throwing error if we could not match the version with the available versions
	if selectedVersion == nil {
		return nil, false, fmt.Errorf("provided version is not one of the available kubernetes versions, available versions are: %s", strings.Join(availableVersionsArray, ","))
	}

	return selectedVersion.Version, deprecated, nil
}

func selectFullVersion(availableVersions []ske.KubernetesVersion, kubernetesVersionMin string) (availableVersionsArray []string, selectedVersion *ske.KubernetesVersion) {
	for _, versionCandidate := range availableVersions {
		if versionCandidate.State == nil || versionCandidate.Version == nil {
			continue
		}
		availableVersionsArray = append(availableVersionsArray, *versionCandidate.Version)
		vPrefixed := "v" + *versionCandidate.Version

		// [MAJOR].[MINOR].[PATCH] version provided, match available version
		if semver.Compare(vPrefixed, kubernetesVersionMin) == 0 {
			selectedVersion = &versionCandidate
			break
		}
	}
	return availableVersionsArray, selectedVersion
}

func selectMatchingVersion(availableVersions []ske.KubernetesVersion, kubernetesVersionMin string) (availableVersionsArray []string, selectedVersion *ske.KubernetesVersion) {
	sortK8sVersions(availableVersions)
	for _, candidateVersion := range availableVersions {
		if candidateVersion.State == nil || candidateVersion.Version == nil {
			continue
		}
		availableVersionsArray = append(availableVersionsArray, *candidateVersion.Version)
		vPreffixed := "v" + *candidateVersion.Version

		// [MAJOR].[MINOR] version provided, get the latest non-preview patch version
		if semver.MajorMinor(vPreffixed) == semver.MajorMinor(kubernetesVersionMin) &&
			(semver.Compare(vPreffixed, kubernetesVersionMin) >= 0) &&
			(candidateVersion.State != nil) {
			// take the current version as a candidate, if we have no other version inspected before
			// OR the previously found version was a preview version
			if selectedVersion == nil || (isSupported(&candidateVersion) && isPreview(selectedVersion)) {
				selectedVersion = &candidateVersion
			}
			// all other cases are ignored
		}
	}
	return availableVersionsArray, selectedVersion
}

func isDeprecated(v *ske.KubernetesVersion) bool {
	if v == nil {
		return false
	}

	if v.State == nil {
		return false
	}

	return *v.State == VersionStateDeprecated
}

func isPreview(v *ske.KubernetesVersion) bool {
	if v == nil {
		return false
	}

	if v.State == nil {
		return false
	}

	return *v.State == VersionStatePreview
}

func isSupported(v *ske.KubernetesVersion) bool {
	if v == nil {
		return false
	}

	if v.State == nil {
		return false
	}

	return *v.State == VersionStateSupported
}

func getLatestSupportedKubernetesVersion(versions []ske.KubernetesVersion) (*string, error) {
	foundKubernetesVersion := false
	var latestVersion *string
	for i := range versions {
		version := versions[i]
		if *version.State != VersionStateSupported {
			continue
		}
		if latestVersion != nil {
			oldSemVer := fmt.Sprintf("v%s", *latestVersion)
			newSemVer := fmt.Sprintf("v%s", *version.Version)
			if semver.Compare(newSemVer, oldSemVer) != 1 {
				continue
			}
		}

		foundKubernetesVersion = true
		latestVersion = version.Version
	}
	if !foundKubernetesVersion {
		return nil, fmt.Errorf("no supported Kubernetes version found")
	}
	return latestVersion, nil
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var state Model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := state.ProjectId.ValueString()
	name := state.Name.ValueString()
	region := r.providerData.GetRegionWithOverride(state.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)
	ctx = tflog.SetField(ctx, "region", region)

	clResp, err := r.skeClient.GetCluster(ctx, projectId, region, name).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading cluster", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, clResp, &state, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading cluster", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SKE cluster read")
}

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	clName := model.Name.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", clName)
	ctx = tflog.SetField(ctx, "region", region)

	availableKubernetesVersions, availableMachines, err := r.loadAvailableVersions(ctx, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating cluster", fmt.Sprintf("Loading available Kubernetes and machine image versions: %v", err))
		return
	}

	currentKubernetesVersion, currentMachineImages := getCurrentVersions(ctx, r.skeClient, &model)

	r.createOrUpdateCluster(ctx, &resp.Diagnostics, &model, availableKubernetesVersions, availableMachines, currentKubernetesVersion, currentMachineImages)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SKE cluster updated")
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)
	ctx = tflog.SetField(ctx, "region", region)

	c := r.skeClient
	_, err := c.DeleteCluster(ctx, projectId, region, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting cluster", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = skeWait.DeleteClusterWaitHandler(ctx, r.skeClient, projectId, region, name).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting cluster", fmt.Sprintf("Cluster deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "SKE cluster deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,name
func (r *clusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing cluster",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[name]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[2])...)
	tflog.Info(ctx, "SKE cluster state imported")
}
