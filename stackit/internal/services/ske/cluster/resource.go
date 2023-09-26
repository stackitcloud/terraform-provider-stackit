package ske

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
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
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &clusterResource{}
	_ resource.ResourceWithConfigure   = &clusterResource{}
	_ resource.ResourceWithImportState = &clusterResource{}
)

type Cluster struct {
	Id                        types.String  `tfsdk:"id"` // needed by TF
	ProjectId                 types.String  `tfsdk:"project_id"`
	Name                      types.String  `tfsdk:"name"`
	KubernetesVersion         types.String  `tfsdk:"kubernetes_version"`
	KubernetesVersionUsed     types.String  `tfsdk:"kubernetes_version_used"`
	AllowPrivilegedContainers types.Bool    `tfsdk:"allow_privileged_containers"`
	NodePools                 []NodePool    `tfsdk:"node_pools"`
	Maintenance               types.Object  `tfsdk:"maintenance"`
	Hibernations              []Hibernation `tfsdk:"hibernations"`
	Extensions                *Extensions   `tfsdk:"extensions"`
	KubeConfig                types.String  `tfsdk:"kube_config"`
}

type NodePool struct {
	Name              types.String `tfsdk:"name"`
	MachineType       types.String `tfsdk:"machine_type"`
	OSName            types.String `tfsdk:"os_name"`
	OSVersion         types.String `tfsdk:"os_version"`
	Minimum           types.Int64  `tfsdk:"minimum"`
	Maximum           types.Int64  `tfsdk:"maximum"`
	MaxSurge          types.Int64  `tfsdk:"max_surge"`
	MaxUnavailable    types.Int64  `tfsdk:"max_unavailable"`
	VolumeType        types.String `tfsdk:"volume_type"`
	VolumeSize        types.Int64  `tfsdk:"volume_size"`
	Labels            types.Map    `tfsdk:"labels"`
	Taints            []Taint      `tfsdk:"taints"`
	CRI               types.String `tfsdk:"cri"`
	AvailabilityZones types.List   `tfsdk:"availability_zones"`
}

type Taint struct {
	Effect types.String `tfsdk:"effect"`
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
}

type Maintenance struct {
	EnableKubernetesVersionUpdates   types.Bool   `tfsdk:"enable_kubernetes_version_updates"`
	EnableMachineImageVersionUpdates types.Bool   `tfsdk:"enable_machine_image_version_updates"`
	Start                            types.String `tfsdk:"start"`
	End                              types.String `tfsdk:"end"`
}

var maintenanceTypes = map[string]attr.Type{
	"enable_kubernetes_version_updates":    basetypes.BoolType{},
	"enable_machine_image_version_updates": basetypes.BoolType{},
	"start":                                basetypes.StringType{},
	"end":                                  basetypes.StringType{},
}

type Hibernation struct {
	Start    types.String `tfsdk:"start"`
	End      types.String `tfsdk:"end"`
	Timezone types.String `tfsdk:"timezone"`
}

type Extensions struct {
	Argus *ArgusExtension `tfsdk:"argus"`
	ACL   *ACL            `tfsdk:"acl"`
}

type ACL struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	AllowedCIDRs types.List `tfsdk:"allowed_cidrs"`
}

type ArgusExtension struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	ArgusInstanceId types.String `tfsdk:"argus_instance_id"`
}

// NewClusterResource is a helper function to simplify the provider implementation.
func NewClusterResource() resource.Resource {
	return &clusterResource{}
}

// clusterResource is the resource implementation.
type clusterResource struct {
	client *ske.APIClient
}

// Metadata returns the resource type name.
func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_cluster"
}

// Configure adds the provider configured client to the resource.
func (r *clusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *ske.APIClient
	var err error
	if providerData.SKECustomEndpoint != "" {
		apiClient, err = ske.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.SKECustomEndpoint),
		)
	} else {
		apiClient, err = ske.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "SKE cluster client configured")
}

// Schema defines the schema for the resource.
func (r *clusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "SKE Cluster Resource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`name`\".",
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
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,10}$`),
						"must start with a letter, must have lower case letters, numbers or hyphens, no hyphen at the end and less than 11 characters.",
					),
					validate.NoSeparator(),
				},
			},
			"kubernetes_version": schema.StringAttribute{
				Description: "Kubernetes version. Must only contain major and minor version (e.g. 1.22)",
				Required:    true,
				Validators: []validator.String{
					validate.MinorVersionNumber(),
				},
			},
			"kubernetes_version_used": schema.StringAttribute{
				Description: "Full Kubernetes version used. For example, if 1.22 was selected, this value may result to 1.22.15",
				Computed:    true,
			},
			"allow_privileged_containers": schema.BoolAttribute{
				Description: "Flag to specify if privileged mode for containers is enabled or not.\nThis should be used with care since it also disables a couple of other features like the use of some volume type (e.g. PVCs).\nDeprecated as of Kubernetes 1.25 and later",
				Optional:    true,
			},
			"node_pools": schema.ListNestedAttribute{
				Description: "One or more `node_pool` block as defined below.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(10),
				},
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
						"minimum": schema.Int64Attribute{
							Description: "Minimum number of nodes in the pool.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
								int64validator.AtMost(100),
							},
						},
						"maximum": schema.Int64Attribute{
							Description: "Maximum number of nodes in the pool.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
								int64validator.AtMost(100),
							},
						},
						"max_surge": schema.Int64Attribute{
							Description: "Maximum number of additional VMs that are created during an update.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
								int64validator.AtMost(10),
							},
						},
						"max_unavailable": schema.Int64Attribute{
							Description: "Maximum number of VMs that that can be unavailable during an update.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"os_name": schema.StringAttribute{
							Description: "The name of the OS image. E.g. `flatcar`.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(DefaultOSName),
						},
						"os_version": schema.StringAttribute{
							Description: "The OS image version.",
							Required:    true,
						},
						"volume_type": schema.StringAttribute{
							Description: "Specifies the volume type. E.g. `storage_premium_perf1`.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(DefaultVolumeType),
						},
						"volume_size": schema.Int64Attribute{
							Description: "The volume size in GB. E.g. `20`",
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
									},
								},
							},
						},
						"cri": schema.StringAttribute{
							Description: "Specifies the container runtime. E.g. `containerd`",
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
						Description: "Flag to enable/disable auto-updates of the Kubernetes version.",
						Required:    true,
					},
					"enable_machine_image_version_updates": schema.BoolAttribute{
						Description: "Flag to enable/disable auto-updates of the OS image version.",
						Required:    true,
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
						Description: "A single argus block as defined below.",
						Optional:    true,
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
				},
			},
			"kube_config": schema.StringAttribute{
				Description: "Kube config file used for connecting to the cluster",
				Sensitive:   true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *clusterResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model Cluster
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = checkAllowPrivilegedContainers(model.AllowPrivilegedContainers, model.KubernetesVersion)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func checkAllowPrivilegedContainers(allowPrivilegeContainers types.Bool, kubernetesVersion types.String) diag.Diagnostics {
	var diags diag.Diagnostics

	if kubernetesVersion.IsNull() {
		diags.AddError("'Kubernetes version' missing", "This field is required")
		return diags
	}
	comparison := semver.Compare(fmt.Sprintf("v%s", kubernetesVersion.ValueString()), "v1.25")
	if comparison < 0 {
		if allowPrivilegeContainers.IsNull() {
			diags.AddError("'Allow privilege containers' missing", "This field is required for Kubernetes prior to 1.25")
		}
	} else {
		if !allowPrivilegeContainers.IsNull() {
			diags.AddError("'Allow privilege containers' deprecated", "This field is deprecated as of Kubernetes 1.25 and later. Please remove this field")
		}
	}

	return diags
}

// Create creates the resource and sets the initial Terraform state.
func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Cluster
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	clusterName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", clusterName)

	availableVersions, err := r.loadAvaiableVersions(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating cluster", fmt.Sprintf("Loading available Kubernetes versions: %v", err))
		return
	}

	r.createOrUpdateCluster(ctx, &resp.Diagnostics, &model, availableVersions)
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

func (r *clusterResource) loadAvaiableVersions(ctx context.Context) ([]ske.KubernetesVersion, error) {
	c := r.client
	res, err := c.GetOptions(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("calling API: %w", err)
	}

	if res.KubernetesVersions == nil {
		return nil, fmt.Errorf("API response has nil kubernetesVersions")
	}

	return *res.KubernetesVersions, nil
}

func (r *clusterResource) createOrUpdateCluster(ctx context.Context, diags *diag.Diagnostics, model *Cluster, availableVersions []ske.KubernetesVersion) {
	// cluster vars
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	kubernetes, hasDeprecatedVersion, err := toKubernetesPayload(model, availableVersions)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Creating cluster config API payload: %v", err))
		return
	}
	if hasDeprecatedVersion {
		diags.AddWarning("Deprecated Kubernetes version", fmt.Sprintf("Version %s of Kubernetes is deprecated, please update it", *kubernetes.Version))
	}
	nodePools := toNodepoolsPayload(ctx, model)
	maintenance, err := toMaintenancePayload(ctx, model)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Creating maintenance API payload: %v", err))
		return
	}
	hibernations := toHibernationsPayload(model)
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
		Nodepools:   &nodePools,
	}
	_, err = r.client.CreateOrUpdateCluster(ctx, projectId, name).CreateOrUpdateClusterPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Calling API: %v", err))
		return
	}

	wr, err := ske.CreateOrUpdateClusterWaitHandler(ctx, r.client, projectId, name).SetTimeout(30 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Cluster creation waiting: %v", err))
		return
	}
	got, ok := wr.(*ske.ClusterResponse)
	if !ok {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Wait result conversion, got %+v", wr))
		return
	}
	if got.Status.Error != nil && got.Status.Error.Message != nil && *got.Status.Error.Code == ske.InvalidArgusInstanceErrorCode {
		core.LogAndAddWarning(ctx, diags, "Warning during creating/updating cluster", fmt.Sprintf("Cluster is in Impaired state due to an invalid argus instance id, the cluster is usable but metrics won't be forwarded: %s", *got.Status.Error.Message))
	}

	err = mapFields(ctx, got, model)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Handle credential
	err = r.getCredential(ctx, model)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error creating/updating cluster", fmt.Sprintf("Getting credential: %v", err))
		return
	}
}

func (r *clusterResource) getCredential(ctx context.Context, model *Cluster) error {
	c := r.client
	res, err := c.GetCredentials(ctx, model.ProjectId.ValueString(), model.Name.ValueString()).Execute()
	if err != nil {
		return fmt.Errorf("fetching cluster credentials: %w", err)
	}
	model.KubeConfig = types.StringPointerValue(res.Kubeconfig)
	return nil
}

func toNodepoolsPayload(ctx context.Context, m *Cluster) []ske.Nodepool {
	cnps := []ske.Nodepool{}
	for i := range m.NodePools {
		// taints
		ts := []ske.Taint{}
		nodePool := m.NodePools[i]
		for _, v := range nodePool.Taints {
			t := ske.Taint{
				Effect: v.Effect.ValueStringPointer(),
				Key:    v.Key.ValueStringPointer(),
				Value:  v.Value.ValueStringPointer(),
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
			s, err := conversion.ToString(context.TODO(), v)
			if err != nil {
				continue
			}
			zs = append(zs, s)
		}

		cn := ske.CRI{
			Name: nodePool.CRI.ValueStringPointer(),
		}
		cnp := ske.Nodepool{
			Name:           nodePool.Name.ValueStringPointer(),
			Minimum:        conversion.ToPtrInt32(nodePool.Minimum),
			Maximum:        conversion.ToPtrInt32(nodePool.Maximum),
			MaxSurge:       conversion.ToPtrInt32(nodePool.MaxSurge),
			MaxUnavailable: conversion.ToPtrInt32(nodePool.MaxUnavailable),
			Machine: &ske.Machine{
				Type: nodePool.MachineType.ValueStringPointer(),
				Image: &ske.Image{
					Name:    nodePool.OSName.ValueStringPointer(),
					Version: nodePool.OSVersion.ValueStringPointer(),
				},
			},
			Volume: &ske.Volume{
				Type: nodePool.VolumeType.ValueStringPointer(),
				Size: conversion.ToPtrInt32(nodePool.VolumeSize),
			},
			Taints:            &ts,
			Cri:               &cn,
			Labels:            ls,
			AvailabilityZones: &zs,
		}
		cnps = append(cnps, cnp)
	}
	return cnps
}

func toHibernationsPayload(m *Cluster) *ske.Hibernation {
	scs := []ske.HibernationSchedule{}
	for _, h := range m.Hibernations {
		sc := ske.HibernationSchedule{
			Start: h.Start.ValueStringPointer(),
			End:   h.End.ValueStringPointer(),
		}
		if !h.Timezone.IsNull() && !h.Timezone.IsUnknown() {
			tz := h.Timezone.ValueString()
			sc.Timezone = &tz
		}
		scs = append(scs, sc)
	}

	if len(scs) == 0 {
		return nil
	}

	return &ske.Hibernation{
		Schedules: &scs,
	}
}

func toExtensionsPayload(ctx context.Context, m *Cluster) (*ske.Extension, error) {
	if m.Extensions == nil {
		return nil, nil
	}
	ex := &ske.Extension{}
	if m.Extensions.Argus != nil {
		ex.Argus = &ske.Argus{
			Enabled:         m.Extensions.Argus.Enabled.ValueBoolPointer(),
			ArgusInstanceId: m.Extensions.Argus.ArgusInstanceId.ValueStringPointer(),
		}
	}
	if m.Extensions.ACL != nil {
		cidrs := []string{}
		diags := m.Extensions.ACL.AllowedCIDRs.ElementsAs(ctx, &cidrs, true)
		if diags.HasError() {
			return nil, fmt.Errorf("error in extension object converion %v", diags.Errors())
		}
		ex.Acl = &ske.ACL{
			Enabled:      m.Extensions.ACL.Enabled.ValueBoolPointer(),
			AllowedCidrs: &cidrs,
		}
	}
	return ex, nil
}

func toMaintenancePayload(ctx context.Context, m *Cluster) (*ske.Maintenance, error) {
	if m.Maintenance.IsNull() || m.Maintenance.IsUnknown() {
		return nil, nil
	}

	maintenance := Maintenance{}
	diags := m.Maintenance.As(ctx, &maintenance, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("error in maintenance object conversion %v", diags.Errors())
	}

	var timeWindowStart *string
	if !(maintenance.Start.IsNull() || maintenance.Start.IsUnknown()) {
		// API expects RFC3339 datetime
		timeWindowStart = utils.Ptr(
			fmt.Sprintf("0000-01-01T%s", maintenance.Start.ValueString()),
		)
	}

	var timeWindowEnd *string
	if !(maintenance.End.IsNull() || maintenance.End.IsUnknown()) {
		// API expects RFC3339 datetime
		timeWindowEnd = utils.Ptr(
			fmt.Sprintf("0000-01-01T%s", maintenance.End.ValueString()),
		)
	}

	return &ske.Maintenance{
		AutoUpdate: &ske.MaintenanceAutoUpdate{
			KubernetesVersion:   maintenance.EnableKubernetesVersionUpdates.ValueBoolPointer(),
			MachineImageVersion: maintenance.EnableMachineImageVersionUpdates.ValueBoolPointer(),
		},
		TimeWindow: &ske.TimeWindow{
			Start: timeWindowStart,
			End:   timeWindowEnd,
		},
	}, nil
}

func mapFields(ctx context.Context, cl *ske.ClusterResponse, m *Cluster) error {
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
	idParts := []string{
		m.ProjectId.ValueString(),
		name,
	}
	m.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	if cl.Kubernetes != nil {
		// The k8s version returned by the API includes the patch version, while we only support major and minor in the kubernetes_version field
		// This prevents inconsistent state by automatic updates to the patch version in the API
		versionPreffixed := "v" + *cl.Kubernetes.Version
		majorMinorVersionPreffixed := semver.MajorMinor(versionPreffixed)
		majorMinorVersion, _ := strings.CutPrefix(majorMinorVersionPreffixed, "v")
		m.KubernetesVersion = types.StringPointerValue(utils.Ptr(majorMinorVersion))
		m.KubernetesVersionUsed = types.StringPointerValue(cl.Kubernetes.Version)
		m.AllowPrivilegedContainers = types.BoolPointerValue(cl.Kubernetes.AllowPrivilegedContainers)
	}
	if cl.Nodepools == nil {
		m.NodePools = []NodePool{}
	} else {
		nodepools := *cl.Nodepools
		m.NodePools = []NodePool{}
		for i := range nodepools {
			np := nodepools[i]

			maimna := types.StringNull()
			maimver := types.StringNull()
			if np.Machine != nil && np.Machine.Image != nil {
				maimna = types.StringPointerValue(np.Machine.Image.Name)
				maimver = types.StringPointerValue(np.Machine.Image.Version)
			}
			vt := types.StringNull()
			if np.Volume != nil {
				vt = types.StringPointerValue(np.Volume.Type)
			}
			crin := types.StringNull()
			if np.Cri != nil {
				crin = types.StringPointerValue(np.Cri.Name)
			}
			n := NodePool{
				Name:              types.StringPointerValue(np.Name),
				MachineType:       types.StringPointerValue(np.Machine.Type),
				OSName:            maimna,
				OSVersion:         maimver,
				Minimum:           conversion.ToTypeInt64(np.Minimum),
				Maximum:           conversion.ToTypeInt64(np.Maximum),
				MaxSurge:          conversion.ToTypeInt64(np.MaxSurge),
				MaxUnavailable:    conversion.ToTypeInt64(np.MaxUnavailable),
				VolumeType:        vt,
				VolumeSize:        conversion.ToTypeInt64(np.Volume.Size),
				Labels:            types.MapNull(types.StringType),
				Taints:            nil,
				CRI:               crin,
				AvailabilityZones: types.ListNull(types.StringType),
			}
			if np.Labels != nil {
				elems := map[string]attr.Value{}
				for k, v := range *np.Labels {
					elems[k] = types.StringValue(v)
				}
				n.Labels = types.MapValueMust(types.StringType, elems)
			}
			if np.Taints != nil {
				for _, v := range *np.Taints {
					if n.Taints == nil {
						n.Taints = []Taint{}
					}
					n.Taints = append(n.Taints, Taint{
						Effect: types.StringPointerValue(v.Effect),
						Key:    types.StringPointerValue(v.Key),
						Value:  types.StringPointerValue(v.Value),
					})
				}
			}
			if np.AvailabilityZones == nil {
				n.AvailabilityZones = types.ListNull(types.StringType)
			} else {
				elems := []attr.Value{}
				for _, v := range *np.AvailabilityZones {
					elems = append(elems, types.StringValue(v))
				}
				n.AvailabilityZones = types.ListValueMust(types.StringType, elems)
			}
			m.NodePools = append(m.NodePools, n)
		}
	}

	err := mapMaintenance(ctx, cl, m)
	if err != nil {
		return err
	}
	mapHibernations(cl, m)
	mapExtensions(cl, m)
	return nil
}

func mapHibernations(cl *ske.ClusterResponse, m *Cluster) {
	if cl.Hibernation == nil || cl.Hibernation.Schedules == nil {
		return
	}

	m.Hibernations = []Hibernation{}
	for _, h := range *cl.Hibernation.Schedules {
		m.Hibernations = append(m.Hibernations, Hibernation{
			Start:    types.StringPointerValue(h.Start),
			End:      types.StringPointerValue(h.End),
			Timezone: types.StringPointerValue(h.Timezone),
		})
	}
}

func mapMaintenance(ctx context.Context, cl *ske.ClusterResponse, m *Cluster) error {
	// Aligned with SKE team that a flattened data structure is fine, because not extensions are planned.
	if cl.Maintenance == nil {
		m.Maintenance = types.ObjectNull(map[string]attr.Type{})
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
		return fmt.Errorf("failed to get maintenance times: %w", err)
	}
	maintenanceValues := map[string]attr.Value{
		"enable_kubernetes_version_updates":    ekvu,
		"enable_machine_image_version_updates": emvu,
		"start":                                types.StringValue(startTime),
		"end":                                  types.StringValue(endTime),
	}
	maintenanceObject, diags := types.ObjectValue(maintenanceTypes, maintenanceValues)
	if diags.HasError() {
		return fmt.Errorf("failed to create flavor: %w", core.DiagsToError(diags))
	}
	m.Maintenance = maintenanceObject
	return nil
}

func getMaintenanceTimes(ctx context.Context, cl *ske.ClusterResponse, m *Cluster) (startTime, endTime string, err error) {
	startTimeAPI, err := time.Parse(time.RFC3339, *cl.Maintenance.TimeWindow.Start)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse start time '%s' from API response as RFC3339 datetime: %w", *cl.Maintenance.TimeWindow.Start, err)
	}
	endTimeAPI, err := time.Parse(time.RFC3339, *cl.Maintenance.TimeWindow.End)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse end time '%s' from API response as RFC3339 datetime: %w", *cl.Maintenance.TimeWindow.End, err)
	}

	if m.Maintenance.IsNull() || m.Maintenance.IsUnknown() {
		return startTimeAPI.Format("15:04:05Z07:00"), endTimeAPI.Format("15:04:05Z07:00"), nil
	}

	maintenance := &Maintenance{}
	diags := m.Maintenance.As(ctx, maintenance, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return "", "", fmt.Errorf("error in maintenance object conversion %w", core.DiagsToError(diags.Errors()))
	}

	if maintenance.Start.IsNull() || maintenance.Start.IsUnknown() {
		startTime = startTimeAPI.Format("15:04:05Z07:00")
	} else {
		startTimeTF, err := time.Parse("15:04:05Z07:00", maintenance.Start.ValueString())
		if err != nil {
			return "", "", fmt.Errorf("failed to parse start time '%s' from TF config as RFC time: %w", maintenance.Start.ValueString(), err)
		}
		if startTimeAPI.Format("15:04:05Z07:00") != startTimeTF.Format("15:04:05Z07:00") {
			return "", "", fmt.Errorf("start time '%v' from API response doesn't match start time '%v' from TF config", *cl.Maintenance.TimeWindow.Start, maintenance.Start.ValueString())
		}
		startTime = maintenance.Start.ValueString()
	}

	if maintenance.End.IsNull() || maintenance.End.IsUnknown() {
		endTime = endTimeAPI.Format("15:04:05Z07:00")
	} else {
		endTimeTF, err := time.Parse("15:04:05Z07:00", maintenance.End.ValueString())
		if err != nil {
			return "", "", fmt.Errorf("failed to parse end time '%s' from TF config as RFC time: %w", maintenance.End.ValueString(), err)
		}
		if endTimeAPI.Format("15:04:05Z07:00") != endTimeTF.Format("15:04:05Z07:00") {
			return "", "", fmt.Errorf("end time '%v' from API response doesn't match end time '%v' from TF config", *cl.Maintenance.TimeWindow.End, maintenance.End.ValueString())
		}
		endTime = maintenance.End.ValueString()
	}

	return startTime, endTime, nil
}

func mapExtensions(cl *ske.ClusterResponse, m *Cluster) {
	if cl.Extensions == nil || (cl.Extensions.Argus == nil && cl.Extensions.Acl == nil) {
		return
	}
	if m.Extensions == nil {
		m.Extensions = &Extensions{}
	}
	if cl.Extensions.Argus != nil {
		m.Extensions.Argus = &ArgusExtension{
			Enabled:         types.BoolPointerValue(cl.Extensions.Argus.Enabled),
			ArgusInstanceId: types.StringPointerValue(cl.Extensions.Argus.ArgusInstanceId),
		}
	}

	if cl.Extensions.Acl != nil {
		cidr := []attr.Value{}
		if cl.Extensions.Acl.AllowedCidrs != nil {
			for _, v := range *cl.Extensions.Acl.AllowedCidrs {
				cidr = append(cidr, types.StringValue(v))
			}
		}
		m.Extensions.ACL = &ACL{
			Enabled:      types.BoolPointerValue(cl.Extensions.Acl.Enabled),
			AllowedCIDRs: types.ListValueMust(types.StringType, cidr),
		}
	}
}

func toKubernetesPayload(m *Cluster, availableVersions []ske.KubernetesVersion) (kubernetesPayload *ske.Kubernetes, hasDeprecatedVersion bool, err error) {
	versionUsed, hasDeprecatedVersion, err := latestMatchingVersion(availableVersions, m.KubernetesVersion.ValueStringPointer())
	if err != nil {
		return nil, false, fmt.Errorf("getting latest matching kubernetes version: %w", err)
	}

	k := &ske.Kubernetes{
		Version:                   versionUsed,
		AllowPrivilegedContainers: m.AllowPrivilegedContainers.ValueBoolPointer(),
	}
	return k, hasDeprecatedVersion, nil
}

func latestMatchingVersion(availableVersions []ske.KubernetesVersion, providedVersion *string) (version *string, deprecated bool, err error) {
	deprecated = false

	if availableVersions == nil {
		return nil, false, fmt.Errorf("nil available kubernetes versions")
	}

	if providedVersion == nil {
		return nil, false, fmt.Errorf("provided version is nil")
	}

	providedVersionPrefixed := "v" + *providedVersion

	if !semver.IsValid(providedVersionPrefixed) {
		return nil, false, fmt.Errorf("provided version is invalid")
	}

	var versionUsed *string
	// Get the higher available version that matches the major and minor version provided by the user
	for _, v := range availableVersions {
		if v.State == nil || v.Version == nil {
			continue
		}
		vPreffixed := "v" + *v.Version
		if semver.MajorMinor(vPreffixed) == semver.MajorMinor(providedVersionPrefixed) &&
			(semver.Compare(vPreffixed, providedVersionPrefixed) == 1 || semver.Compare(vPreffixed, providedVersionPrefixed) == 0) {
			versionUsed = v.Version

			if strings.EqualFold(*v.State, VersionStateDeprecated) {
				deprecated = true
			} else {
				deprecated = false
			}
		}
	}

	// Throwing error if we could not match the version with the available versions
	if versionUsed == nil {
		return nil, false, fmt.Errorf("provided version is not one of the available kubernetes versions")
	}

	return versionUsed, deprecated, nil
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var state Cluster
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := state.ProjectId.ValueString()
	name := state.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)

	clResp, err := r.client.GetCluster(ctx, projectId, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading cluster", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, clResp, &state)
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
	var model Cluster
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	clName := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", clName)

	availableVersions, err := r.loadAvaiableVersions(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating cluster", fmt.Sprintf("Loading available Kubernetes versions: %v", err))
		return
	}

	r.createOrUpdateCluster(ctx, &resp.Diagnostics, &model, availableVersions)
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
	var model Cluster
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)

	c := r.client
	_, err := c.DeleteCluster(ctx, projectId, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting cluster", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = ske.DeleteClusterWaitHandler(ctx, r.client, projectId, name).SetTimeout(15 * time.Minute).WaitWithContext(ctx)
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

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing cluster",
			fmt.Sprintf("Expected import identifier with format: [project_id],[name]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[1])...)
	tflog.Info(ctx, "SKE cluster state imported")
}
