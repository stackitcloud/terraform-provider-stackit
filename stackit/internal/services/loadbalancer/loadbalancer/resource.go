package loadbalancer

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

type Model struct {
	Id              types.String `tfsdk:"id"` // needed by TF
	ProjectId       types.String `tfsdk:"project_id"`
	ExternalAddress types.String `tfsdk:"external_address"`
	Listeners       []Listener   `tfsdk:"listeners"`
	Name            types.String `tfsdk:"name"`
	Networks        []Network    `tfsdk:"networks"`
	Options         types.Object `tfsdk:"options"`
	PrivateAddress  types.String `tfsdk:"private_address"`
	TargetPools     []TargetPool `tfsdk:"target_pools"`
}

// Struct corresponding to each Model.Listener
type Listener struct {
	DisplayName types.String `tfsdk:"display_name"`
	Name        types.String `tfsdk:"name"`
	Port        types.Int64  `tfsdk:"port"`
	Protocol    types.String `tfsdk:"protocol"`
	TargetPool  types.String `tfsdk:"target_pool"`
}

// Struct corresponding to each Model.Network
type Network struct {
	NetworkId types.String `tfsdk:"network_id"`
	Role      types.String `tfsdk:"role"`
}

// Struct corresponding to Model.Options
type Options struct {
	ACL                types.List `tfsdk:"acl"`
	PrivateNetworkOnly types.Bool `tfsdk:"private_network_only"`
}

// Types corresponding to Options
var optionsTypes = map[string]attr.Type{
	"acl":                  basetypes.ListType{ElemType: basetypes.StringType{}},
	"private_network_only": basetypes.BoolType{},
}

// Struct corresponding to each Model.TargetPool
type TargetPool struct {
	ActiveHealthCheck types.Object `tfsdk:"active_health_check"`
	Name              types.String `tfsdk:"name"`
	TargetPort        types.Int64  `tfsdk:"target_port"`
	Targets           []Target     `tfsdk:"targets"`
}

// Struct corresponding to each Model.TargetPool.ActiveHealthCheck
type ActiveHealthCheck struct {
	HealthyThreshold   types.Int64  `tfsdk:"healthy_threshold"`
	Interval           types.String `tfsdk:"interval"`
	IntervalJitter     types.String `tfsdk:"interval_jitter"`
	Timeout            types.String `tfsdk:"timeout"`
	UnhealthyThreshold types.Int64  `tfsdk:"unhealthy_threshold"`
}

// Types corresponding to ActiveHealthCheck
var activeHealthCheckTypes = map[string]attr.Type{
	"healthy_threshold":   basetypes.Int64Type{},
	"interval":            basetypes.StringType{},
	"interval_jitter":     basetypes.StringType{},
	"timeout":             basetypes.StringType{},
	"unhealthy_threshold": basetypes.Int64Type{},
}

// Struct corresponding to each Model.TargetPool.Targets
type Target struct {
	DisplayName types.String `tfsdk:"display_name"`
	Ip          types.String `tfsdk:"ip"`
}

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	client *loadbalancer.APIClient
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancer"
}

// Configure adds the provider configured client to the resource.
func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *loadbalancer.APIClient
	var err error
	if providerData.LoadBalancerCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "loadbalancer_custom_endpoint", providerData.LoadBalancerCustomEndpoint)
		apiClient, err = loadbalancer.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.LoadBalancerCustomEndpoint),
		)
	} else {
		apiClient, err = loadbalancer.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithServiceAccountEmail(providerData.ServiceAccountEmail),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Load Balancer client configured")
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                 "Load Balancer project resource schema.",
		"id":                   "Terraform's internal resource ID. It is structured as \"`project_id`\",\"`name`\".",
		"project_id":           "STACKIT project ID to which the Load Balancer is associated.",
		"external_address":     "External Load Balancer IP address where this Load Balancer is exposed.",
		"listeners":            "List of all listeners which will accept traffic. Limited to 20.",
		"listeners.name":       "Will be used to reference a listener and will replace display name in the future.",
		"port":                 "Port number where we listen for traffic.",
		"protocol":             "Protocol is the highest network protocol we understand to load balance.",
		"target_pool":          "Reference target pool by target pool name.",
		"name":                 "Load balancer name.",
		"networks":             "List of networks that listeners and targets reside in.",
		"network_id":           "Openstack network ID.",
		"role":                 "The role defines how the load balancer is using the network.",
		"options":              "Defines any optional functionality you want to have enabled on your load balancer.",
		"acl":                  "Load Balancer is accessible only from an IP address in this range.",
		"private_network_only": "If true, Load Balancer is accessible only via a private network IP address.",
		"private_address":      "Transient private Load Balancer IP address. It can change any time.",
		"target_pools":         "List of all target pools which will be used in the Load Balancer. Limited to 20.",
		"healthy_threshold":    "Healthy threshold of the health checking.",
		"interval":             "Interval duration of health checking in seconds.",
		"interval_jitter":      "Interval duration threshold of the health checking in seconds.",
		"timeout":              "Active health checking timeout duration in seconds.",
		"unhealthy_threshold":  "Unhealthy threshold of the health checking.",
		"target_pools.name":    "Target pool name.",
		"target_port":          "Identical port number where each target listens for traffic.",
		"targets":              "List of all targets which will be used in the pool. Limited to 250.",
		"targets.display_name": "Target display name",
		"ip":                   "Target IP",
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
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the dns record set is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"external_address": schema.StringAttribute{
				Description: descriptions["external_address"],
				Optional:    true,
				Computed:    true,
			},
			"listeners": schema.ListNestedAttribute{
				Description: descriptions["listeners"],
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"display_name": schema.StringAttribute{
							Description: descriptions["listeners.display_name"],
							Optional:    true,
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: descriptions["listeners.display_name"],
							Computed:    true,
						},
						"port": schema.Int64Attribute{
							Description: descriptions["port"],
							Optional:    true,
							Computed:    true,
						},
						"protocol": schema.StringAttribute{
							Description: descriptions["protocol"],
							Optional:    true,
							Computed:    true,
						},
						"target_pool": schema.StringAttribute{
							Description: descriptions["target_pool"],
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"networks": schema.ListNestedAttribute{
				Description: descriptions["networks"],
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{
							Description: descriptions["network_id"],
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								validate.UUID(),
								validate.NoSeparator(),
							},
						},
						"role": schema.StringAttribute{
							Description: descriptions["role"],
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"options": schema.SingleNestedAttribute{
				Description: descriptions["options"],
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"acl": schema.SetAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Validators: []validator.Set{
							setvalidator.ValueStringsAre(
								validate.CIDR(),
							),
						},
					},
					"private_network_only": schema.BoolAttribute{
						Description: descriptions["private_network_only"],
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"private_address": schema.StringAttribute{
				Description: descriptions["private_address"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"target_pools": schema.ListNestedAttribute{
				Description: descriptions["target_pools"],
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"active_health_check": schema.SingleNestedAttribute{
							Description: descriptions["active_health_check"],
							Optional:    true,
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"healthy_threshold": schema.Int64Attribute{
									Description: descriptions["healthy_threshold"],
									Optional:    true,
									Computed:    true,
								},
								"interval": schema.StringAttribute{
									Description: descriptions["interval"],
									Optional:    true,
									Computed:    true,
								},
								"interval_jitter": schema.StringAttribute{
									Description: descriptions["interval_jitter"],
									Optional:    true,
									Computed:    true,
								},
								"timeout": schema.StringAttribute{
									Description: descriptions["timeout"],
									Optional:    true,
									Computed:    true,
								},
								"unhealthy_threshold": schema.Int64Attribute{
									Description: descriptions["unhealthy_threshold"],
									Optional:    true,
									Computed:    true,
								},
							},
						},
						"name": schema.StringAttribute{
							Description: descriptions["target_pools.name"],
							Optional:    true,
							Computed:    true,
						},
						"target_port": schema.StringAttribute{
							Description: descriptions["target_port"],
							Optional:    true,
							Computed:    true,
						},
						"targets": schema.ListNestedAttribute{
							Description: descriptions["targets"],
							Optional:    true,
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"display_name": schema.StringAttribute{
										Description: descriptions["targets.display_name"],
										Optional:    true,
										Computed:    true,
									},
									"ip": schema.StringAttribute{
										Description: descriptions["ip"],
										Optional:    true,
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Generate API request body from model
	_, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform

}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: container_id
func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

}

func toCreatePayload(ctx context.Context, model *Model) (*loadbalancer.CreateLoadBalancerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	listeners := toListenersPayload(model)
	networks := toNetworksPayload(model)
	options, err := toOptionsPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting options: %v", err)
	}
	targetPools, err := toTargetPoolsPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting target pools: %v", err)
	}

	return &loadbalancer.CreateLoadBalancerPayload{
		ExternalAddress: model.ExternalAddress.ValueStringPointer(),
		Listeners:       listeners,
		Name:            model.Name.ValueStringPointer(),
		Networks:        networks,
		Options:         options,
		TargetPools:     targetPools,
	}, nil
}

func toListenersPayload(model *Model) *[]loadbalancer.Listener {
	if model.Listeners == nil {
		return nil
	}

	listeners := []loadbalancer.Listener{}
	for _, listener := range model.Listeners {
		listeners = append(listeners, loadbalancer.Listener{
			DisplayName: listener.DisplayName.ValueStringPointer(),
			Port:        listener.Port.ValueInt64Pointer(),
			Protocol:    listener.Protocol.ValueStringPointer(),
			TargetPool:  listener.TargetPool.ValueStringPointer(),
		})
	}

	return &listeners
}

func toNetworksPayload(model *Model) *[]loadbalancer.Network {
	if model.Networks == nil {
		return nil
	}

	networks := []loadbalancer.Network{}
	for _, network := range model.Networks {
		networks = append(networks, loadbalancer.Network{
			NetworkId: network.NetworkId.ValueStringPointer(),
			Role:      network.Role.ValueStringPointer(),
		})
	}

	return &networks
}

func toOptionsPayload(ctx context.Context, model *Model) (*loadbalancer.LoadBalancerOptions, error) {
	var optionsModel = &Options{}
	if !(model.Options.IsNull() || model.Options.IsUnknown()) {
		diags := model.Options.As(ctx, optionsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("%v", diags.Errors())
		}
	}

	accessControl := &loadbalancer.LoadbalancerOptionAccessControl{}
	if !(optionsModel.ACL.IsNull() || optionsModel.ACL.IsUnknown()) {
		var acl []string
		diags := optionsModel.ACL.ElementsAs(ctx, &acl, false)
		if diags.HasError() {
			return nil, fmt.Errorf("converting acl: %v", diags.Errors())
		}
		accessControl.AllowedSourceRanges = &acl
	}

	options := &loadbalancer.LoadBalancerOptions{
		AccessControl:      accessControl,
		PrivateNetworkOnly: optionsModel.PrivateNetworkOnly.ValueBoolPointer(),
	}

	return options, nil
}

func toTargetPoolsPayload(ctx context.Context, model *Model) (*[]loadbalancer.TargetPool, error) {
	if model.TargetPools == nil {
		return nil, nil
	}

	var targetPools []loadbalancer.TargetPool
	for _, targetPool := range model.TargetPools {
		var activeHealthCheck *loadbalancer.ActiveHealthCheck
		if !(targetPool.ActiveHealthCheck.IsNull() || targetPool.ActiveHealthCheck.IsUnknown()) {
			var activeHealthCheckModel ActiveHealthCheck
			diags := targetPool.ActiveHealthCheck.As(ctx, &activeHealthCheckModel, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, fmt.Errorf("converting active health check: %v", diags.Errors())
			}

			activeHealthCheck = &loadbalancer.ActiveHealthCheck{
				HealthyThreshold:   activeHealthCheckModel.HealthyThreshold.ValueInt64Pointer(),
				Interval:           activeHealthCheckModel.Interval.ValueStringPointer(),
				IntervalJitter:     activeHealthCheckModel.IntervalJitter.ValueStringPointer(),
				Timeout:            activeHealthCheckModel.Timeout.ValueStringPointer(),
				UnhealthyThreshold: activeHealthCheckModel.UnhealthyThreshold.ValueInt64Pointer(),
			}
		}

		var targets []loadbalancer.Target
		for _, target := range targetPool.Targets {
			targets = append(targets, loadbalancer.Target{
				DisplayName: target.DisplayName.ValueStringPointer(),
				Ip:          target.Ip.ValueStringPointer(),
			})
		}

		targetPools = append(targetPools, loadbalancer.TargetPool{
			ActiveHealthCheck: activeHealthCheck,
			Name:              targetPool.Name.ValueStringPointer(),
			TargetPort:        targetPool.TargetPort.ValueInt64Pointer(),
			Targets:           &targets,
		})
	}

	return &targetPools, nil
}
