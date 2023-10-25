package loadbalancer

import (
	"context"

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
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
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
	Project         types.String `tfsdk:"project_id"`
	ExternalAddress types.String `tfsdk:"external_address"`
	Listeners       []Listener   `tfsdk:"listeneres"`
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

// Types corresponding to Listener
var listenerTypes = map[string]attr.Type{
	"display_name": basetypes.StringType{},
	"name":         basetypes.StringType{},
	"port":         basetypes.Int64Type{},
	"protocol":     basetypes.StringType{},
	"target_pool":  basetypes.StringType{},
}

// Struct corresponding to each Model.Network
type Network struct {
	NetworkId types.String `tfsdk:"network_id"`
	Role      types.String `tfsdk:"role"`
}

// Types corresponding to Network
var networkTypes = map[string]attr.Type{
	"network_id": basetypes.StringType{},
	"role":       basetypes.StringType{},
}

// Struct corresponding to Model.Options
type Options struct {
	Acls types.Set    `tfsdk:"acls"`
	Role types.String `tfsdk:"role"`
}

// Types corresponding to Options
var optionsTypes = map[string]attr.Type{
	"acls": basetypes.SetType{},
	"role": basetypes.StringType{},
}

// Struct corresponding to each Model.TargetPool
type TargetPool struct {
	ActiveHealthCheck types.Object `tfsdk:"active_health_check"`
	Name              types.String `tfsdk:"name"`
	TargetPort        types.Int64  `tfsdk:"target_port"`
	Targets           types.Object `tfsdk:"protocol"`
}

// Types corresponding to Listener
var targetPoolTypes = map[string]attr.Type{
	"display_name": basetypes.StringType{},
	"name":         basetypes.StringType{},
	"port":         basetypes.Int64Type{},
	"protocol":     basetypes.StringType{},
	"target_pool":  basetypes.StringType{},
}

// Struct corresponding to each Model.TargetPool.ActiveHealthCheck
type ActiveHealthCheck struct {
	HealthyThreshold   types.String `tfsdk:"healthy_threshold"`
	Interval           types.String `tfsdk:"interval"`
	IntervalJitter     types.String `tfsdk:"interval_jitter"`
	Timeout            types.String `tfsdk:"timeout"`
	UnhealthyThreshold types.String `tfsdk:"unhealthy_threshold"`
}

// Types corresponding to ActiveHealthCheck
var activeHealthCheckTypes = map[string]attr.Type{
	"healthy_threshold":   basetypes.StringType{},
	"interval":            basetypes.StringType{},
	"interval_jitter":     basetypes.StringType{},
	"timeout":             basetypes.StringType{},
	"unhealthy_threshold": basetypes.StringType{},
}

// Struct corresponding to each Model.TargetPool.Targets
type Target struct {
	DisplayName types.String `tfsdk:"display_name"`
	Ip          types.String `tfsdk:"ip"`
}

// Types corresponding to Target
var targetTypes = map[string]attr.Type{
	"display_name": basetypes.StringType{},
	"ip":           basetypes.StringType{},
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
	resp.TypeName = req.ProviderTypeName + "_loadbalancer_project"
}

// Configure adds the provider configured client to the resource.
func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":       "Resource Manager project resource schema.",
		"id":         "Terraform's internal resource ID. It is structured as \"`container_id`\".",
		"project_id": "STACKIT project ID to which the Load Balancer is associated.",
		// TODO: Add descriptions according to API docs
		"external_address":       "",
		"listeners":              "",
		"listeners.display_name": "",
		"listeners.name":         "",
		"port":                   "",
		"protocol":               "",
		"target_pool":            "",
		"name":                   "",
		"networks":               "",
		"network_id":             "",
		"role":                   "",
		"options":                "",
		"acls":                   "",
		"private_network_only":   "",
		"private_address":        "",
		"target_pools":           "",
		"active_health_check":    "",
		"healthy_threshold":      "",
		"interval":               "",
		"interval_jitter":        "",
		"timeout":                "",
		"unhealthy_threshold":    "",
		"target_pools.name":      "",
		"target_port":            "",
		"targets":                "",
		"targets.display_name":   "",
		"ip":                     "",
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
							Description: descriptions["listeners.name"],
							Optional:    true,
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
					"acls": schema.SetAttribute{
						Description: descriptions["acls"],
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
								"healthy_threshold": schema.StringAttribute{
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
								"unhealthy_threshold": schema.StringAttribute{
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

func mapFields(ctx context.Context, lbResp *loadbalancer.LoadBalancer, model *Model) (err error) {
	return nil
}

func toCreatePayload(model *Model, serviceAccountEmail string) (*loadbalancer.CreateLoadBalancerPayload, error) {
	return nil, nil
}

func toUpdatePayload(model *Model) (*loadbalancer.UpdateTargetPoolPayload, error) {
	return nil, nil
}
