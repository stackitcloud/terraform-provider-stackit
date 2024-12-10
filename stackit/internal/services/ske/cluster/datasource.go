package ske

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &clusterDataSource{}
)

// NewClusterDataSource is a helper function to simplify the provider implementation.
func NewClusterDataSource() datasource.DataSource {
	return &clusterDataSource{}
}

// clusterDataSource is the data source implementation.
type clusterDataSource struct {
	client *ske.APIClient
}

// Metadata returns the data source type name.
func (r *clusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_cluster"
}

// Configure adds the provider configured client to the data source.
func (r *clusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "SKE client configured")
}
func (r *clusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "SKE Cluster data source schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source. ID. It is structured as \"`project_id`,`name`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the cluster is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The cluster name.",
				Required:    true,
			},
			"kubernetes_version_min": schema.StringAttribute{
				Description: `The minimum Kubernetes version, this field is always nil. ` + SKEUpdateDoc + " To get the current kubernetes version being used for your cluster, use the `kubernetes_version_used` field.",
				Computed:    true,
			},
			"kubernetes_version": schema.StringAttribute{
				Description:        "Kubernetes version. This field is deprecated, use `kubernetes_version_used` instead",
				Computed:           true,
				DeprecationMessage: "This field is always nil, use `kubernetes_version_used` to get the cluster kubernetes version. This field would cause errors when the cluster got a kubernetes version minor upgrade, either triggered by automatic or forceful updates.",
			},
			"kubernetes_version_used": schema.StringAttribute{
				Description: "Full Kubernetes version used. For example, if `1.22` was selected, this value may result to `1.22.15`",
				Computed:    true,
			},
			"allow_privileged_containers": schema.BoolAttribute{
				Description:        "DEPRECATED as of Kubernetes 1.25+\n Flag to specify if privileged mode for containers is enabled or not.\nThis should be used with care since it also disables a couple of other features like the use of some volume type (e.g. PVCs).",
				DeprecationMessage: "Please remove this flag from your configuration when using Kubernetes version 1.25+.",
				Computed:           true,
			},

			"node_pools": schema.ListNestedAttribute{
				Description: "One or more `node_pool` block as defined below.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Specifies the name of the node pool.",
							Computed:    true,
						},
						"machine_type": schema.StringAttribute{
							Description: "The machine type.",
							Computed:    true,
						},
						"os_name": schema.StringAttribute{
							Description: "The name of the OS image.",
							Computed:    true,
						},
						"os_version_min": schema.StringAttribute{
							Description: "The minimum OS image version, this field is always nil. " + SKEUpdateDoc + " To get the current OS image version being used for the node pool, use the read-only `os_version_used` field.",
							Computed:    true,
						},
						"os_version": schema.StringAttribute{
							Description: "The OS image version.",
							Computed:    true,
						},
						"os_version_used": schema.StringAttribute{
							Description: "Full OS image version used. For example, if 3815.2 was set in `os_version_min`, this value may result to 3815.2.2. " + SKEUpdateDoc,
							Computed:    true,
						},
						"minimum": schema.Int64Attribute{
							Description: "Minimum number of nodes in the pool.",
							Computed:    true,
						},

						"maximum": schema.Int64Attribute{
							Description: "Maximum number of nodes in the pool.",
							Computed:    true,
						},

						"max_surge": schema.Int64Attribute{
							Description: "The maximum number of nodes upgraded simultaneously.",
							Computed:    true,
						},
						"max_unavailable": schema.Int64Attribute{
							Description: "The maximum number of nodes unavailable during upgraded.",
							Computed:    true,
						},
						"volume_type": schema.StringAttribute{
							Description: "Specifies the volume type.",
							Computed:    true,
						},
						"volume_size": schema.Int64Attribute{
							Description: "The volume size in GB.",
							Computed:    true,
						},
						"labels": schema.MapAttribute{
							Description: "Labels to add to each node.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"taints": schema.ListNestedAttribute{
							Description: "Specifies a taint list as defined below.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"effect": schema.StringAttribute{
										Description: "The taint effect.",
										Computed:    true,
									},
									"key": schema.StringAttribute{
										Description: "Taint key to be applied to a node.",
										Computed:    true,
									},
									"value": schema.StringAttribute{
										Description: "Taint value corresponding to the taint key.",
										Computed:    true,
									},
								},
							},
						},
						"cri": schema.StringAttribute{
							Description: "Specifies the container runtime.",
							Computed:    true,
						},
						"availability_zones": schema.ListAttribute{
							Description: "Specify a list of availability zones.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"allow_system_components": schema.BoolAttribute{
							Description: "Allow system components to run on this node pool.",
							Computed:    true,
						},
					},
				},
			},
			"maintenance": schema.SingleNestedAttribute{
				Description: "A single maintenance block as defined below",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"enable_kubernetes_version_updates": schema.BoolAttribute{
						Description: "Flag to enable/disable auto-updates of the Kubernetes version.",
						Computed:    true,
					},
					"enable_machine_image_version_updates": schema.BoolAttribute{
						Description: "Flag to enable/disable auto-updates of the OS image version.",
						Computed:    true,
					},
					"start": schema.StringAttribute{
						Description: "Date time for maintenance window start.",
						Computed:    true,
					},
					"end": schema.StringAttribute{
						Description: "Date time for maintenance window end.",
						Computed:    true,
					},
				},
			},

			"network": schema.SingleNestedAttribute{
				Description: "Network block as defined below.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "ID of the STACKIT Network Area (SNA) network into which the cluster will be deployed.",
						Computed:    true,
						Validators: []validator.String{
							validate.UUID(),
						},
					},
				},
			},

			"hibernations": schema.ListNestedAttribute{
				Description: "One or more hibernation block as defined below.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							Description: "Start time of cluster hibernation in crontab syntax.",
							Computed:    true,
						},
						"end": schema.StringAttribute{
							Description: "End time of hibernation, in crontab syntax.",
							Computed:    true,
						},
						"timezone": schema.StringAttribute{
							Description: "Timezone name corresponding to a file in the IANA Time Zone database.",
							Computed:    true,
						},
					},
				},
			},

			"extensions": schema.SingleNestedAttribute{
				Description: "A single extensions block as defined below",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"argus": schema.SingleNestedAttribute{
						Description: "A single argus block as defined below",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Flag to enable/disable argus extensions.",
								Computed:    true,
							},
							"argus_instance_id": schema.StringAttribute{
								Description: "Instance ID of argus",
								Computed:    true,
							},
						},
					},
					"acl": schema.SingleNestedAttribute{
						Description: "Cluster access control configuration",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Is ACL enabled?",
								Computed:    true,
							},
							"allowed_cidrs": schema.ListAttribute{
								Description: "Specify a list of CIDRs to whitelist",
								Computed:    true,
								ElementType: types.StringType,
							},
						},
					},
					"dns": schema.SingleNestedAttribute{
						Description: "DNS extension configuration",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Flag to enable/disable DNS extensions",
								Computed:    true,
							},
							"zones": schema.ListAttribute{
								Description: "Specify a list of domain filters for externalDNS (e.g., `foo.runs.onstackit.cloud`)",
								Computed:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
			},
			"kube_config": schema.StringAttribute{
				Description:        "Kube config file used for connecting to the cluster. This field will be empty for clusters with Kubernetes v1.27+, or if you have obtained the kubeconfig or performed credentials rotation using the new process, either through the Portal or the SKE API. Use the stackit_ske_kubeconfig resource instead. For more information, see How to rotate SKE credentials (https://docs.stackit.cloud/stackit/en/how-to-rotate-ske-credentials-200016334.html).",
				Sensitive:          true,
				Computed:           true,
				DeprecationMessage: "This field will be empty for clusters with Kubernetes v1.27+, or if you have obtained the kubeconfig or performed credentials rotation using the new process, either through the Portal or the SKE API. Use the stackit_ske_kubeconfig resource instead. For more information, see How to rotate SKE credentials (https://docs.stackit.cloud/stackit/en/how-to-rotate-ske-credentials-200016334.html).",
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *clusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var state Model
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := state.ProjectId.ValueString()
	name := state.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)
	clusterResp, err := r.client.GetCluster(ctx, projectId, name).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading cluster", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, clusterResp, &state)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading cluster", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "SKE cluster read")
}
