package loadbalancer

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &loadBalancerResource{}
	_ resource.ResourceWithConfigure   = &loadBalancerResource{}
	_ resource.ResourceWithImportState = &loadBalancerResource{}
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
	ACL                types.Set  `tfsdk:"acl"`
	PrivateNetworkOnly types.Bool `tfsdk:"private_network_only"`
}

// Types corresponding to Options
var optionsTypes = map[string]attr.Type{
	"acl":                  basetypes.SetType{ElemType: basetypes.StringType{}},
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

// NewLoadBalancerResource is a helper function to simplify the provider implementation.
func NewLoadBalancerResource() resource.Resource {
	return &loadBalancerResource{}
}

// loadBalancerResource is the resource implementation.
type loadBalancerResource struct {
	client *loadbalancer.APIClient
}

// Metadata returns the resource type name.
func (r *loadBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancer"
}

// Configure adds the provider configured client to the resource.
func (r *loadBalancerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			config.WithRegion(providerData.Region),
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
func (r *loadBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                 "Load Balancer resource schema.",
		"id":                   "Terraform's internal resource ID. It is structured as \"`project_id`\",\"`name`\".",
		"project_id":           "STACKIT project ID to which the Load Balancer is associated.",
		"external_address":     "External Load Balancer IP address where this Load Balancer is exposed.",
		"listeners":            "List of all listeners which will accept traffic. Limited to 20.",
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
		Description:         descriptions["main"],
		MarkdownDescription: "",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				},
			},
			"external_address": schema.StringAttribute{
				Description: descriptions["external_address"],
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"listeners": schema.ListNestedAttribute{
				Description: descriptions["listeners"],
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 20),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"display_name": schema.StringAttribute{
							Description: descriptions["listeners.display_name"],
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"port": schema.Int64Attribute{
							Description: descriptions["port"],
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.RequiresReplace(),
							},
						},
						"protocol": schema.StringAttribute{
							Description: descriptions["protocol"],
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf("PROTOCOL_UNSPECIFIED", "PROTOCOL_TCP", "PROTOCOL_UDP", "PROTOCOL_TCP_PROXY"),
							},
						},
						"target_pool": schema.StringAttribute{
							Description: descriptions["target_pool"],
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					validate.NoSeparator(),
				},
			},
			"networks": schema.ListNestedAttribute{
				Description: descriptions["networks"],
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{
							Description: descriptions["network_id"],
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								validate.UUID(),
								validate.NoSeparator(),
							},
						},
						"role": schema.StringAttribute{
							Description: descriptions["role"],
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf("ROLE_UNSPECIFIED", "ROLE_LISTENERS_AND_TARGETS", "ROLE_LISTENERS", "ROLE_TARGETS"),
							},
						},
					},
				},
			},
			"options": schema.SingleNestedAttribute{
				Description: descriptions["options"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"acl": schema.SetAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.RequiresReplace(),
						},
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
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
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
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 20),
				},
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
							Required:    true,
						},
						"target_port": schema.Int64Attribute{
							Description: descriptions["target_port"],
							Required:    true,
						},
						"targets": schema.ListNestedAttribute{
							Description: descriptions["targets"],
							Required:    true,
							Validators: []validator.List{
								listvalidator.SizeBetween(1, 250),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"display_name": schema.StringAttribute{
										Description: descriptions["targets.display_name"],
										Required:    true,
									},
									"ip": schema.StringAttribute{
										Description: descriptions["ip"],
										Required:    true,
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
func (r *loadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Get status of load balancer functionality
	statusResp, err := r.client.GetStatus(ctx, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error getting status of load balancer functionality", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// If load balancer functionality is not enabled, enable it
	if *statusResp.Status != wait.FunctionalityStatusReady {
		_, err = r.client.EnableLoadBalancing(ctx, projectId).XRequestID("").Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error enabling load balancer functionality", fmt.Sprintf("Calling API: %v", err))
			return
		}

		_, err := wait.EnableLoadBalancingWaitHandler(ctx, r.client, projectId).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error enabling load balancer functionality", fmt.Sprintf("Waiting for enablement: %v", err))
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create a new load balancer
	createResp, err := r.client.CreateLoadBalancer(ctx, projectId).CreateLoadBalancerPayload(*payload).XRequestID("").Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Calling API: %v", err))
		return
	}

	waitResp, err := wait.CreateLoadBalancerWaitHandler(ctx, r.client, projectId, *createResp.Name).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Load balancer creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Load balancer created")
}

// Read refreshes the Terraform state with the latest data.
func (r *loadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)

	lbResp, err := r.client.GetLoadBalancer(ctx, projectId, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading load balancer", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, lbResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading load balancer", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Load balancer read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *loadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)

	for _, targetPool := range model.TargetPools {
		// Generate API request body from model
		payload, err := toTargetPoolUpdatePayload(ctx, utils.Ptr(targetPool))
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating load balancer", fmt.Sprintf("Creating API payload: %v", err))
			return
		}

		// Update target pool
		_, err = r.client.UpdateTargetPool(ctx, projectId, name, targetPool.Name.ValueString()).UpdateTargetPoolPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating load balancer", fmt.Sprintf("Calling API: %v", err))
			return
		}
	}

	// Get updated load balancer
	getResp, err := r.client.GetLoadBalancer(ctx, projectId, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating load balancer", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, getResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating load balancer", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Load balancer updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *loadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "name", name)

	// Delete load balancer
	_, err := r.client.DeleteLoadBalancer(ctx, projectId, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting load balancer", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.DeleteLoadBalancerWaitHandler(ctx, r.client, projectId, name).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting load balancer", fmt.Sprintf("Load balancer deleting waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Load balancer deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,name
func (r *loadBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing load balancer",
			fmt.Sprintf("Expected import identifier with format: [project_id],[name]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[1])...)
	tflog.Info(ctx, "Load balancer state imported")
}

func toCreatePayload(ctx context.Context, model *Model) (*loadbalancer.CreateLoadBalancerPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	listeners := toListenersPayload(model)
	networks := toNetworksPayload(model)
	options, err := toOptionsPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting options: %w", err)
	}
	targetPools, err := toTargetPoolsPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting target pools: %w", err)
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
			return nil, fmt.Errorf("%w", core.DiagsToError(diags))
		}
	}

	accessControl := &loadbalancer.LoadbalancerOptionAccessControl{}
	if !(optionsModel.ACL.IsNull() || optionsModel.ACL.IsUnknown()) {
		var acl []string
		diags := optionsModel.ACL.ElementsAs(ctx, &acl, false)
		if diags.HasError() {
			return nil, fmt.Errorf("converting acl: %w", core.DiagsToError(diags))
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
		activeHealthCheck, err := toActiveHealthCheckPayload(ctx, utils.Ptr(targetPool))
		if err != nil {
			return nil, fmt.Errorf("converting target pool: %w", err)
		}

		targets := toTargetsPayload(utils.Ptr(targetPool))
		if err != nil {
			return nil, fmt.Errorf("converting target pool: %w", err)
		}

		targetPools = append(targetPools, loadbalancer.TargetPool{
			ActiveHealthCheck: activeHealthCheck,
			Name:              targetPool.Name.ValueStringPointer(),
			TargetPort:        targetPool.TargetPort.ValueInt64Pointer(),
			Targets:           targets,
		})
	}

	return &targetPools, nil
}

func toTargetPoolUpdatePayload(ctx context.Context, targetPool *TargetPool) (*loadbalancer.UpdateTargetPoolPayload, error) {
	if targetPool == nil {
		return nil, fmt.Errorf("nil target pool")
	}

	activeHealthCheck, err := toActiveHealthCheckPayload(ctx, targetPool)
	if err != nil {
		return nil, fmt.Errorf("converting target pool: %w", err)
	}

	targets := toTargetsPayload(targetPool)

	return &loadbalancer.UpdateTargetPoolPayload{
		ActiveHealthCheck: activeHealthCheck,
		Name:              targetPool.Name.ValueStringPointer(),
		TargetPort:        targetPool.TargetPort.ValueInt64Pointer(),
		Targets:           targets,
	}, nil
}

func toActiveHealthCheckPayload(ctx context.Context, targetPool *TargetPool) (*loadbalancer.ActiveHealthCheck, error) {
	if targetPool.ActiveHealthCheck.IsNull() || targetPool.ActiveHealthCheck.IsUnknown() {
		return nil, nil
	}

	var activeHealthCheckModel ActiveHealthCheck
	diags := targetPool.ActiveHealthCheck.As(ctx, &activeHealthCheckModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting active health check: %w", core.DiagsToError(diags))
	}

	return &loadbalancer.ActiveHealthCheck{
		HealthyThreshold:   activeHealthCheckModel.HealthyThreshold.ValueInt64Pointer(),
		Interval:           activeHealthCheckModel.Interval.ValueStringPointer(),
		IntervalJitter:     activeHealthCheckModel.IntervalJitter.ValueStringPointer(),
		Timeout:            activeHealthCheckModel.Timeout.ValueStringPointer(),
		UnhealthyThreshold: activeHealthCheckModel.UnhealthyThreshold.ValueInt64Pointer(),
	}, nil
}

func toTargetsPayload(targetPool *TargetPool) *[]loadbalancer.Target {
	if targetPool.Targets == nil {
		return nil
	}

	var targets []loadbalancer.Target
	for _, target := range targetPool.Targets {
		targets = append(targets, loadbalancer.Target{
			DisplayName: target.DisplayName.ValueStringPointer(),
			Ip:          target.Ip.ValueStringPointer(),
		})
	}

	return &targets
}

func mapFields(ctx context.Context, lb *loadbalancer.LoadBalancer, m *Model) error {
	if lb == nil {
		return fmt.Errorf("response input is nil")
	}
	if m == nil {
		return fmt.Errorf("model input is nil")
	}

	var name string
	if m.Name.ValueString() != "" {
		name = m.Name.ValueString()
	} else if lb.Name != nil {
		name = *lb.Name
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

	m.ExternalAddress = types.StringPointerValue(lb.ExternalAddress)
	m.PrivateAddress = types.StringPointerValue(lb.PrivateAddress)

	mapListeners(lb, m)
	mapNetworks(lb, m)
	err := mapOptions(ctx, lb, m)
	if err != nil {
		return fmt.Errorf("mapping options: %w", err)
	}
	err = mapTargetPools(lb, m)
	if err != nil {
		return fmt.Errorf("mapping target pools: %w", err)
	}

	return nil
}

func mapListeners(lb *loadbalancer.LoadBalancer, m *Model) {
	if lb.Listeners == nil {
		return
	}

	var listeners []Listener
	for _, listener := range *lb.Listeners {
		listeners = append(listeners, Listener{
			DisplayName: types.StringPointerValue(listener.DisplayName),
			Port:        types.Int64PointerValue(listener.Port),
			Protocol:    types.StringPointerValue(listener.Protocol),
			TargetPool:  types.StringPointerValue(listener.TargetPool),
		})
	}
	m.Listeners = listeners
}

func mapNetworks(lb *loadbalancer.LoadBalancer, m *Model) {
	if lb.Networks == nil {
		return
	}

	var networks []Network
	for _, network := range *lb.Networks {
		networks = append(networks, Network{
			NetworkId: types.StringPointerValue(network.NetworkId),
			Role:      types.StringPointerValue(network.Role),
		})
	}
	m.Networks = networks
}

func mapOptions(ctx context.Context, lb *loadbalancer.LoadBalancer, m *Model) error {
	if lb.Options == nil {
		return nil
	}

	var diags diag.Diagnostics
	acl := types.SetNull(types.StringType)
	if lb.Options.AccessControl != nil && lb.Options.AccessControl.AllowedSourceRanges != nil {
		acl, diags = types.SetValueFrom(ctx, types.StringType, *lb.Options.AccessControl.AllowedSourceRanges)
		if diags != nil {
			return fmt.Errorf("converting acl: %w", core.DiagsToError(diags))
		}
	}
	privateNetworkOnly := types.BoolNull()
	if lb.Options.PrivateNetworkOnly != nil {
		privateNetworkOnly = types.BoolValue(*lb.Options.PrivateNetworkOnly)
	}
	if acl.IsNull() && privateNetworkOnly.IsNull() {
		return nil
	}

	optionsValues := map[string]attr.Value{
		"acl":                  acl,
		"private_network_only": privateNetworkOnly,
	}
	options, diags := types.ObjectValue(optionsTypes, optionsValues)
	if diags != nil {
		return fmt.Errorf("converting options: %w", core.DiagsToError(diags))
	}
	m.Options = options

	return nil
}

func mapTargetPools(lb *loadbalancer.LoadBalancer, m *Model) error {
	if lb.TargetPools == nil {
		return nil
	}

	var diags diag.Diagnostics
	var targetPools []TargetPool
	for _, targetPool := range *lb.TargetPools {
		var activeHealthCheck basetypes.ObjectValue
		if targetPool.ActiveHealthCheck != nil {
			activeHealthCheckValues := map[string]attr.Value{
				"healthy_threshold":   types.Int64Value(*targetPool.ActiveHealthCheck.HealthyThreshold),
				"interval":            types.StringValue(*targetPool.ActiveHealthCheck.Interval),
				"interval_jitter":     types.StringValue(*targetPool.ActiveHealthCheck.IntervalJitter),
				"timeout":             types.StringValue(*targetPool.ActiveHealthCheck.Timeout),
				"unhealthy_threshold": types.Int64Value(*targetPool.ActiveHealthCheck.UnhealthyThreshold),
			}
			activeHealthCheck, diags = types.ObjectValue(activeHealthCheckTypes, activeHealthCheckValues)
			if diags != nil {
				return fmt.Errorf("converting active health check: %w", core.DiagsToError(diags))
			}
		}

		var targets []Target
		if targetPool.Targets != nil {
			for _, target := range *targetPool.Targets {
				targets = append(targets, Target{
					DisplayName: types.StringPointerValue(target.DisplayName),
					Ip:          types.StringPointerValue(target.Ip),
				})
			}
		}

		targetPools = append(targetPools, TargetPool{
			ActiveHealthCheck: activeHealthCheck,
			Name:              types.StringPointerValue(targetPool.Name),
			TargetPort:        types.Int64Value(*targetPool.TargetPort),
			Targets:           targets,
		})
	}
	m.TargetPools = targetPools

	return nil
}
