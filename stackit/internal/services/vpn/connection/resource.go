package connection

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"

	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/vpn/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &vpnConnectionResource{}
	_ resource.ResourceWithConfigure   = &vpnConnectionResource{}
	_ resource.ResourceWithImportState = &vpnConnectionResource{}
	_ resource.ResourceWithModifyPlan  = &vpnConnectionResource{}
)

type BasePhaseModel struct {
	DhGroups             types.List  `tfsdk:"dh_groups"`
	EncryptionAlgorithms types.List  `tfsdk:"encryption_algorithms"`
	IntegrityAlgorithms  types.List  `tfsdk:"integrity_algorithms"`
	RekeyTime            types.Int32 `tfsdk:"rekey_time"`
}

type Phase1Model struct {
	BasePhaseModel
}

type Phase2Model struct {
	StartAction types.String `tfsdk:"start_action"`
	DpdAction   types.String `tfsdk:"dpd_action"`
	BasePhaseModel
}

type PeeringConfigModel struct {
	LocalAddress  types.String `tfsdk:"local_address"`
	RemoteAddress types.String `tfsdk:"remote_address"`
}

type BGPTunnelConfigModel struct {
	RemoteAsn types.Int64 `tfsdk:"remote_asn"`
}

type TunnelModel struct {
	PreSharedKey          types.String          `tfsdk:"pre_shared_key"`
	PreSharedKeyWo        types.String          `tfsdk:"pre_shared_key_wo"`
	PreSharedKeyWoVersion types.Int64           `tfsdk:"pre_shared_key_wo_version"`
	RemoteAddress         types.String          `tfsdk:"remote_address"`
	Phase1                *Phase1Model          `tfsdk:"phase1"`
	Phase2                *Phase2Model          `tfsdk:"phase2"`
	Peering               *PeeringConfigModel   `tfsdk:"peering"`
	Bgp                   *BGPTunnelConfigModel `tfsdk:"bgp"`
}

type Model struct {
	ID           types.String `tfsdk:"id"`
	ConnectionID types.String `tfsdk:"connection_id"`
	ProjectID    types.String `tfsdk:"project_id"`
	Region       types.String `tfsdk:"region"`
	GatewayID    types.String `tfsdk:"gateway_id"`
	DisplayName  types.String `tfsdk:"display_name"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	RemoteSubnet types.List   `tfsdk:"remote_subnet"`
	LocalSubnet  types.List   `tfsdk:"local_subnet"`
	StaticRoutes types.List   `tfsdk:"static_routes"`
	Tunnel1      *TunnelModel `tfsdk:"tunnel1"`
	Tunnel2      *TunnelModel `tfsdk:"tunnel2"`
	Labels       types.Map    `tfsdk:"labels"`
}

var (
	dhGroupValues             = sdkUtils.EnumSliceToStringSlice(vpn.AllowedPhaseDhGroupsInnerEnumValues)
	encryptionAlgorithmValues = sdkUtils.EnumSliceToStringSlice(vpn.AllowedPhaseEncryptionAlgorithmsInnerEnumValues)
	integrityAlgorithmValues  = sdkUtils.EnumSliceToStringSlice(vpn.AllowedPhaseIntegrityAlgorithmsInnerEnumValues)
	startActionValues         = sdkUtils.EnumSliceToStringSlice(vpn.AllowedTunnelConfigurationPhase2AllOfStartActionEnumValues)
	dpdActionValues           = sdkUtils.EnumSliceToStringSlice(vpn.AllowedTunnelConfigurationPhase2AllOfDpdActionEnumValues)
)

type vpnConnectionResource struct {
	client       *vpn.APIClient
	providerData core.ProviderData
}

func NewVpnConnectionResource() resource.Resource {
	return &vpnConnectionResource{}
}

func (r *vpnConnectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	tflog.Info(ctx, "VPN client configured")
}

func (r *vpnConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_connection"
}

func tunnelSchema(rootAttribute string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description:         fmt.Sprintf("Configuration for the IPsec %s.", rootAttribute),
		MarkdownDescription: fmt.Sprintf("Configuration for the IPsec %s \n\n~> Write-Only argument `pre_shared_key_wo` is available to use in place of `pre_shared_key`. Write-Only arguments are supported in HashiCorp Terraform 1.11.0 and later. [Learn more](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments).", rootAttribute),
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"pre_shared_key": schema.StringAttribute{
				Description: "Pre-shared key for the IPsec tunnel. Minimum 20 characters. Write-only argument `pre_shared_key_wo` should be preferred.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(20),
					stringvalidator.PreferWriteOnlyAttribute(path.MatchRoot(rootAttribute).AtName("pre_shared_key_wo")),
				},
			},
			"pre_shared_key_wo": schema.StringAttribute{
				Description: "Pre-shared key for the IPsec tunnel. Minimum 20 characters. Write-only - never stored in state and never returned by the API. To rotate the key, update this value AND increment pre_shared_key_wo_version. Changing this field alone will NOT trigger an update.",
				Optional:    true,
				WriteOnly:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(20),
					stringvalidator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("pre_shared_key"),
						path.MatchRelative().AtParent().AtName("pre_shared_key_wo"),
					),
				},
			},
			"pre_shared_key_wo_version": schema.Int64Attribute{
				Description: "User-managed rotation counter for the pre-shared key. Must be incremented every time pre_shared_key_wo is changed. Terraform diffs this field to detect key rotations - changing pre_shared_key_wo alone will NOT trigger an update because it is write-only and never stored in state.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRelative().AtParent().AtName("pre_shared_key_wo")),
				},
			},
			"remote_address": schema.StringAttribute{
				Description: "Remote IPv4 address for the tunnel endpoint.",
				Required:    true,
				Validators: []validator.String{
					validate.IP(true),
				},
			},
			"phase1": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"dh_groups": schema.ListAttribute{
						Description: fmt.Sprintf("Diffie-Hellman groups for key exchange. %s", tfutils.FormatPossibleValues(dhGroupValues...)),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(dhGroupValues...),
							),
						},
					},
					"encryption_algorithms": schema.ListAttribute{
						Description: fmt.Sprintf("Encryption algorithms for Phase 1. %s", tfutils.FormatPossibleValues(encryptionAlgorithmValues...)),
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(encryptionAlgorithmValues...),
							),
						},
					},
					"integrity_algorithms": schema.ListAttribute{
						Description: fmt.Sprintf("Integrity algorithms for Phase 1. %s", tfutils.FormatPossibleValues(integrityAlgorithmValues...)),
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(integrityAlgorithmValues...),
							),
						},
					},
					"rekey_time": schema.Int32Attribute{
						Description: "Time to schedule an IKE re-keying in seconds. Range: 900-28800. Default: 14400.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int32{
							int32validator.Between(900, 28800),
						},
					},
				},
			},
			"phase2": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"dh_groups": schema.ListAttribute{
						Description: fmt.Sprintf("Diffie-Hellman groups for Phase 2. %s", tfutils.FormatPossibleValues(dhGroupValues...)),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(dhGroupValues...),
							),
						},
					},
					"encryption_algorithms": schema.ListAttribute{
						Description: fmt.Sprintf("Encryption algorithms for Phase 2. %s", tfutils.FormatPossibleValues(encryptionAlgorithmValues...)),
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(encryptionAlgorithmValues...),
							),
						},
					},
					"integrity_algorithms": schema.ListAttribute{
						Description: fmt.Sprintf("Integrity algorithms for Phase 2. %s", tfutils.FormatPossibleValues(integrityAlgorithmValues...)),
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(integrityAlgorithmValues...),
							),
						},
					},
					"rekey_time": schema.Int32Attribute{
						Description: "Time to schedule a Child SA re-keying in seconds. Range: 900-3600. Default: 3600.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int32{
							int32validator.Between(900, 3600),
						},
					},
					"start_action": schema.StringAttribute{
						Description: fmt.Sprintf("Action to perform after loading the connection configuration. Default: 'start'. %s", tfutils.FormatPossibleValues(startActionValues...)),
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(startActionValues...),
						},
					},
					"dpd_action": schema.StringAttribute{
						Description: fmt.Sprintf("Action to perform on DPD timeout. Default: 'restart'. %s", tfutils.FormatPossibleValues(dpdActionValues...)),
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(dpdActionValues...),
						},
					},
				},
			},
			"peering": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"local_address": schema.StringAttribute{
						Description: "Local tunnel interface IPv4 address.",
						Required:    true,
						Validators: []validator.String{
							validate.IP(true),
						},
					},
					"remote_address": schema.StringAttribute{
						Description: "Remote tunnel interface IPv4 address.",
						Required:    true,
						Validators: []validator.String{
							validate.IP(true),
						},
					},
				},
			},
			"bgp": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"remote_asn": schema.Int64Attribute{
						Description: "Remote ASN for BGP peering (private ASN range, 64512-4294967294).",
						Required:    true,
						Validators: []validator.Int64{
							int64validator.Between(64512, 4294967294),
						},
					},
				},
			},
		},
	}
}

func (r *vpnConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("VPN Connection resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`gateway_id`,`connection_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_id": schema.StringAttribute{
				Description: "The server-generated UUID of the VPN connection.",
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "STACKIT region.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gateway_id": schema.StringAttribute{
				Description: "The UUID of the parent VPN gateway.",
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
				Description: "A user-friendly name for the connection. Must start and end with an alphanumeric character, may contain hyphens, and be 1-63 characters long.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`),
						"must start and end with an alphanumeric character, may contain hyphens, and be 1-63 characters long",
					),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether this connection is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"remote_subnet": schema.ListAttribute{
				Description: "List of remote IPv4 CIDRs accessible via this connection. Optional for route-based and BGP configurations (defaults to 0.0.0.0/0). Mandatory for policy-based.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 100),
					listvalidator.ValueStringsAre(validate.CIDR()),
				},
			},
			"local_subnet": schema.ListAttribute{
				Description: "List of local IPv4 CIDRs to route through this connection. Optional for route-based and BGP configurations (defaults to 0.0.0.0/0). Mandatory for policy-based.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 100),
					listvalidator.ValueStringsAre(validate.CIDR()),
				},
			},
			"static_routes": schema.ListAttribute{
				Description: "List of static routes (IPv4 CIDRs) for route-based VPN. Mandatory for ROUTE_BASED gateways.",
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(validate.CIDR()),
				},
			},
			"tunnel1": tunnelSchema("tunnel1"),
			"tunnel2": tunnelSchema("tunnel2"),
			"labels": schema.MapAttribute{
				Description: "Map of custom labels.",
				Optional:    true,
				ElementType: types.StringType,
				Validators:  validate.LabelValidators(),
			},
		},
	}
}

func (r *vpnConnectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

func (r *vpnConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing VPN connection",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[gateway_id],[connection_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":    idParts[0],
		"region":        idParts[1],
		"gateway_id":    idParts[2],
		"connection_id": idParts[3],
	})
	tflog.Info(ctx, "VPN connection state imported")
}

func (r *vpnConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configModel Model
	diags = req.Config.Get(ctx, &configModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Tunnel1.PreSharedKeyWo = configModel.Tunnel1.PreSharedKeyWo
	model.Tunnel2.PreSharedKeyWo = configModel.Tunnel2.PreSharedKeyWo

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	gatewayId := model.GatewayID.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN connection", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	createResp, err := r.client.DefaultAPI.CreateGatewayConnection(ctx, projectId, region, gatewayId).CreateGatewayConnectionPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN connection", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, createResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN connection", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN connection created")
}

func (r *vpnConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	gatewayId := model.GatewayID.ValueString()
	connectionId := model.ConnectionID.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "connection_id", connectionId)

	connResp, err := r.client.DefaultAPI.GetGatewayConnection(ctx, projectId, region, gatewayId, connectionId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN connection", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, connResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading VPN connection", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN connection read")
}

func (r *vpnConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configModel Model
	diags = req.Config.Get(ctx, &configModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// tunnel1 PSK rotation
	if !tfutils.IsUndefined(model.Tunnel1.PreSharedKeyWoVersion) {
		planVersion := model.Tunnel1.PreSharedKeyWoVersion.ValueInt64()
		stateVersion := stateModel.Tunnel1.PreSharedKeyWoVersion.ValueInt64()
		if planVersion < stateVersion {
			resp.Diagnostics.AddAttributeError(
				path.Root("tunnel1").AtName("pre_shared_key_wo_version"),
				"Version must not decrease",
				fmt.Sprintf("`pre_shared_key_wo_version` must be incremented to rotate the pre-shared key, got %d (current: %d).", planVersion, stateVersion),
			)
			return
		}
		if planVersion > stateVersion {
			// Secret must be read from Config, not Plan — write-only values are always null in plan.
			model.Tunnel1.PreSharedKeyWo = configModel.Tunnel1.PreSharedKeyWo
		}
	}

	// tunnel2 PSK rotation
	if !tfutils.IsUndefined(model.Tunnel2.PreSharedKeyWoVersion) {
		planVersion := model.Tunnel2.PreSharedKeyWoVersion.ValueInt64()
		stateVersion := stateModel.Tunnel2.PreSharedKeyWoVersion.ValueInt64()
		if planVersion < stateVersion {
			resp.Diagnostics.AddAttributeError(
				path.Root("tunnel2").AtName("pre_shared_key_wo_version"),
				"Version must not decrease",
				fmt.Sprintf("`pre_shared_key_wo_version` must be incremented to rotate the pre-shared key, got %d (current: %d).", planVersion, stateVersion),
			)
			return
		}
		if planVersion > stateVersion {
			// Secret must be read from Config, not Plan — write-only values are always null in plan.
			model.Tunnel2.PreSharedKeyWo = configModel.Tunnel2.PreSharedKeyWo
		}
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	gatewayId := model.GatewayID.ValueString()
	connectionId := model.ConnectionID.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "connection_id", connectionId)

	payload, err := toUpdatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN connection", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	connResp, err := r.client.DefaultAPI.UpdateGatewayConnection(ctx, projectId, region, gatewayId, connectionId).UpdateGatewayConnectionPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN connection", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, connResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN connection", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPN connection updated")
}

func (r *vpnConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	gatewayId := model.GatewayID.ValueString()
	connectionId := model.ConnectionID.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "gateway_id", gatewayId)
	ctx = tflog.SetField(ctx, "connection_id", connectionId)

	err := r.client.DefaultAPI.DeleteGatewayConnection(ctx, projectId, region, gatewayId, connectionId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting VPN connection", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "VPN connection deleted")
}

func toCreatePayload(ctx context.Context, model *Model) (*vpn.CreateGatewayConnectionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payload := &vpn.CreateGatewayConnectionPayload{}
	err := toConnectionPayload(ctx, model, payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func toUpdatePayload(ctx context.Context, model *Model) (*vpn.UpdateGatewayConnectionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payload := &vpn.UpdateGatewayConnectionPayload{}
	err := toConnectionPayload(ctx, model, payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

type connectionFields interface {
	SetDisplayName(string)
	SetTunnel1(vpn.TunnelConfiguration)
	SetTunnel2(vpn.TunnelConfiguration)
	SetEnabled(bool)
	SetRemoteSubnets([]string)
	SetLocalSubnets([]string)
	SetStaticRoutes([]string)
	SetLabels(map[string]string)
}

func toConnectionPayload(ctx context.Context, model *Model, payload connectionFields) error {
	if payload == nil {
		return fmt.Errorf("payload can not be nil")
	}

	tunnel1, err := toTunnelPayload(model.Tunnel1)
	if err != nil && tunnel1 != nil {
		return fmt.Errorf("converting tunnel1: %w", err)
	}
	payload.SetTunnel1(*tunnel1)

	tunnel2, err := toTunnelPayload(model.Tunnel2)
	if err != nil && tunnel2 != nil {
		return fmt.Errorf("converting tunnel2: %w", err)
	}
	payload.SetTunnel2(*tunnel2)

	payload.SetDisplayName(model.DisplayName.ValueString())

	if !tfutils.IsUndefined(model.Enabled) {
		payload.SetEnabled(model.Enabled.ValueBool())
	}

	if !tfutils.IsUndefined(model.RemoteSubnet) {
		remoteSubnets, err := tfutils.ListValueToStringSlice(model.RemoteSubnet)
		if err != nil {
			return fmt.Errorf("converting remote_subnet: %w", err)
		}
		payload.SetRemoteSubnets(remoteSubnets)
	}

	if !tfutils.IsUndefined(model.LocalSubnet) {
		localSubnets, err := tfutils.ListValueToStringSlice(model.LocalSubnet)
		if err != nil {
			return fmt.Errorf("converting local_subnet: %w", err)
		}
		payload.SetLocalSubnets(localSubnets)
	}

	if !tfutils.IsUndefined(model.StaticRoutes) {
		staticRoutes, err := tfutils.ListValueToStringSlice(model.StaticRoutes)
		if err != nil {
			return fmt.Errorf("converting static_routes: %w", err)
		}
		payload.SetStaticRoutes(staticRoutes)
	}

	labels, err := tfutils.LabelsToPayload(ctx, model.Labels)
	if err != nil {
		return err
	}
	payload.SetLabels(labels)

	return nil
}

func toTunnelPayload(tunnel *TunnelModel) (*vpn.TunnelConfiguration, error) {
	if tunnel == nil {
		return nil, fmt.Errorf("nil tunnel model")
	}

	config := &vpn.TunnelConfiguration{
		RemoteAddress: tunnel.RemoteAddress.ValueString(),
	}

	if !tfutils.IsUndefined(tunnel.PreSharedKeyWo) {
		config.PreSharedKey = tunnel.PreSharedKeyWo.ValueStringPointer()
	} else if !tfutils.IsUndefined(tunnel.PreSharedKey) {
		config.PreSharedKey = tunnel.PreSharedKey.ValueStringPointer()
	}

	if tunnel.Phase1 != nil {
		phase1 := vpn.TunnelConfigurationPhase1{}

		if !tfutils.IsUndefined(tunnel.Phase1.DhGroups) {
			dhGroups, err := tfutils.ListValueToStringSlice(tunnel.Phase1.DhGroups)
			if err != nil {
				return nil, fmt.Errorf("converting phase1 dh_groups: %w", err)
			}
			dhGroupsInner := []vpn.PhaseDhGroupsInner{}
			for _, item := range dhGroups {
				dhGroupsInner = append(dhGroupsInner, vpn.PhaseDhGroupsInner(item))
			}
			phase1.DhGroups = dhGroupsInner
		}

		encAlgs, err := tfutils.ListValueToStringSlice(tunnel.Phase1.EncryptionAlgorithms)
		if err != nil {
			return nil, fmt.Errorf("converting phase1 encryption_algorithms: %w", err)
		}
		encAlgsInner := []vpn.PhaseEncryptionAlgorithmsInner{}
		for _, item := range encAlgs {
			encAlgsInner = append(encAlgsInner, vpn.PhaseEncryptionAlgorithmsInner(item))
		}
		phase1.EncryptionAlgorithms = encAlgsInner

		intAlgs, err := tfutils.ListValueToStringSlice(tunnel.Phase1.IntegrityAlgorithms)
		if err != nil {
			return nil, fmt.Errorf("converting phase1 integrity_algorithms: %w", err)
		}
		intAlgsInner := []vpn.PhaseIntegrityAlgorithmsInner{}
		for _, item := range intAlgs {
			intAlgsInner = append(intAlgsInner, vpn.PhaseIntegrityAlgorithmsInner(item))
		}
		phase1.IntegrityAlgorithms = intAlgsInner

		if !tfutils.IsUndefined(tunnel.Phase1.RekeyTime) {
			rekeyTime := tunnel.Phase1.RekeyTime.ValueInt32()
			phase1.RekeyTime = &rekeyTime
		}

		config.Phase1 = phase1
	}

	if tunnel.Phase2 != nil {
		phase2 := vpn.TunnelConfigurationPhase2{}
		if !tfutils.IsUndefined(tunnel.Phase2.DhGroups) {
			dhGroups, err := tfutils.ListValueToStringSlice(tunnel.Phase2.DhGroups)
			if err != nil {
				return nil, fmt.Errorf("converting phase2 dh_groups: %w", err)
			}
			dhGroupsInner := []vpn.PhaseDhGroupsInner{}
			for _, item := range dhGroups {
				dhGroupsInner = append(dhGroupsInner, vpn.PhaseDhGroupsInner(item))
			}
			phase2.DhGroups = dhGroupsInner
		}
		encAlgs, err := tfutils.ListValueToStringSlice(tunnel.Phase2.EncryptionAlgorithms)
		if err != nil {
			return nil, fmt.Errorf("converting phase2 encryption_algorithms: %w", err)
		}
		encAlgsInner := []vpn.PhaseEncryptionAlgorithmsInner{}
		for _, item := range encAlgs {
			encAlgsInner = append(encAlgsInner, vpn.PhaseEncryptionAlgorithmsInner(item))
		}
		phase2.EncryptionAlgorithms = encAlgsInner
		intAlgs, err := tfutils.ListValueToStringSlice(tunnel.Phase2.IntegrityAlgorithms)
		if err != nil {
			return nil, fmt.Errorf("converting phase2 integrity_algorithms: %w", err)
		}
		intAlgsInner := []vpn.PhaseIntegrityAlgorithmsInner{}
		for _, item := range intAlgs {
			intAlgsInner = append(intAlgsInner, vpn.PhaseIntegrityAlgorithmsInner(item))
		}
		phase2.IntegrityAlgorithms = intAlgsInner
		if !tfutils.IsUndefined(tunnel.Phase2.RekeyTime) {
			rekeyTime := tunnel.Phase2.RekeyTime.ValueInt32()
			phase2.RekeyTime = &rekeyTime
		}
		if !tfutils.IsUndefined(tunnel.Phase2.StartAction) {
			startAction := tunnel.Phase2.StartAction.ValueString()
			phase2.StartAction = vpn.TunnelConfigurationPhase2AllOfStartAction(startAction).Ptr()
		}
		if !tfutils.IsUndefined(tunnel.Phase2.DpdAction) {
			dpdAction := tunnel.Phase2.DpdAction.ValueString()
			phase2.DpdAction = vpn.TunnelConfigurationPhase2AllOfDpdAction(dpdAction).Ptr()
		}
		config.Phase2 = phase2
	}

	if tunnel.Peering != nil {
		localAddr := tunnel.Peering.LocalAddress.ValueString()
		remoteAddr := tunnel.Peering.RemoteAddress.ValueString()
		config.Peering = &vpn.PeeringConfig{
			LocalAddress:  &localAddr,
			RemoteAddress: &remoteAddr,
		}
	}

	if tunnel.Bgp != nil {
		remoteAsn := tunnel.Bgp.RemoteAsn.ValueInt64()
		config.Bgp = &vpn.BGPTunnelConfig{
			RemoteAsn: remoteAsn,
		}
	}

	return config, nil
}

type connectionResponse interface {
	GetIdOk() (*string, bool)
	GetDisplayName() string
	GetTunnel1() vpn.TunnelConfiguration
	GetTunnel2() vpn.TunnelConfiguration
	GetEnabledOk() (*bool, bool)
	GetRemoteSubnetsOk() ([]string, bool)
	GetLocalSubnetsOk() ([]string, bool)
	GetStaticRoutesOk() ([]string, bool)
	GetLabelsOk() (*map[string]string, bool)
}

func mapFields(ctx context.Context, conn connectionResponse, model *Model, region string) error {
	if conn == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var connectionId string
	if respConnectionId, _ := conn.GetIdOk(); respConnectionId != nil {
		connectionId = *respConnectionId
	} else if model.ConnectionID.ValueString() != "" {
		connectionId = model.ConnectionID.ValueString()
	} else {
		return fmt.Errorf("connection id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), region, model.GatewayID.ValueString(), connectionId)
	model.ConnectionID = types.StringValue(connectionId)
	model.DisplayName = types.StringValue(conn.GetDisplayName())
	model.Region = types.StringValue(region)

	if enabled, _ := conn.GetEnabledOk(); enabled != nil {
		model.Enabled = types.BoolValue(*enabled)
	} else {
		model.Enabled = types.BoolValue(true)
	}

	if remoteSubnets, _ := conn.GetRemoteSubnetsOk(); remoteSubnets != nil {
		list, diags := types.ListValueFrom(ctx, types.StringType, remoteSubnets)
		if diags.HasError() {
			return fmt.Errorf("mapping remote_subnet: %w", core.DiagsToError(diags))
		}
		model.RemoteSubnet = list
	} else {
		model.RemoteSubnet = types.ListNull(types.StringType)
	}

	if localSubnets, _ := conn.GetLocalSubnetsOk(); localSubnets != nil {
		list, diags := types.ListValueFrom(ctx, types.StringType, localSubnets)
		if diags.HasError() {
			return fmt.Errorf("mapping local_subnet: %w", core.DiagsToError(diags))
		}
		model.LocalSubnet = list
	} else {
		model.LocalSubnet = types.ListNull(types.StringType)
	}

	if staticRoutes, _ := conn.GetStaticRoutesOk(); staticRoutes != nil {
		list, diags := types.ListValueFrom(ctx, types.StringType, staticRoutes)
		if diags.HasError() {
			return fmt.Errorf("mapping static_routes: %w", core.DiagsToError(diags))
		}
		model.StaticRoutes = list
	} else {
		model.StaticRoutes = types.ListNull(types.StringType)
	}

	err := mapTunnel(ctx, conn.GetTunnel1(), model.Tunnel1)
	if err != nil {
		return fmt.Errorf("mapping tunnel1: %w", err)
	}

	err = mapTunnel(ctx, conn.GetTunnel2(), model.Tunnel2)
	if err != nil {
		return fmt.Errorf("mapping tunnel2: %w", err)
	}

	respLabels, _ := conn.GetLabelsOk()
	labels, err := tfutils.MapLabels(ctx, respLabels, model.Labels)
	if err != nil {
		return fmt.Errorf("mapping labels: %w", err)
	}
	model.Labels = labels

	return nil
}

func mapTunnel(ctx context.Context, apiTunnel vpn.TunnelConfiguration, tfTunnel *TunnelModel) error {
	if tfTunnel == nil {
		tfTunnel = &TunnelModel{
			PreSharedKeyWoVersion: types.Int64Null(),
		}
	}

	tfTunnel.RemoteAddress = types.StringValue(string(apiTunnel.RemoteAddress))

	phase1, err := mapPhase1(ctx, &apiTunnel.Phase1)
	if err != nil {
		return err
	}
	tfTunnel.Phase1 = phase1

	phase2, err := mapPhase2(ctx, &apiTunnel.Phase2)
	if err != nil {
		return err
	}
	tfTunnel.Phase2 = phase2

	if apiTunnel.Peering != nil {
		peering := &PeeringConfigModel{}
		if apiTunnel.Peering.LocalAddress != nil {
			peering.LocalAddress = types.StringValue(*apiTunnel.Peering.LocalAddress)
		} else {
			peering.LocalAddress = types.StringNull()
		}
		if apiTunnel.Peering.RemoteAddress != nil {
			peering.RemoteAddress = types.StringValue(*apiTunnel.Peering.RemoteAddress)
		} else {
			peering.RemoteAddress = types.StringNull()
		}
		tfTunnel.Peering = peering
	} else {
		tfTunnel.Peering = nil
	}

	if apiTunnel.Bgp != nil {
		tfTunnel.Bgp = &BGPTunnelConfigModel{
			RemoteAsn: types.Int64Value(int64(apiTunnel.Bgp.RemoteAsn)),
		}
	} else {
		tfTunnel.Bgp = nil
	}

	return nil
}

type BasePhaseFields interface {
	GetDhGroupsOk() ([]vpn.PhaseDhGroupsInner, bool)
	GetEncryptionAlgorithmsOk() ([]vpn.PhaseEncryptionAlgorithmsInner, bool)
	GetIntegrityAlgorithmsOk() ([]vpn.PhaseIntegrityAlgorithmsInner, bool)
	GetRekeyTimeOk() (*int32, bool)
}

func mapBasePhase(ctx context.Context, apiPhase BasePhaseFields) (phase BasePhaseModel, err error) {
	if apiPhase == nil {
		return phase, fmt.Errorf("api phase can not be nil")
	}

	if dhGroups, _ := apiPhase.GetDhGroupsOk(); len(dhGroups) > 0 {
		list, diags := types.ListValueFrom(ctx, types.StringType, dhGroups)
		if diags.HasError() {
			return phase, fmt.Errorf("mapping base phase dh_groups: %w", core.DiagsToError(diags))
		}
		phase.DhGroups = list
	} else {
		phase.DhGroups = types.ListNull(types.StringType)
	}

	if encryptionAlgorithms, _ := apiPhase.GetEncryptionAlgorithmsOk(); len(encryptionAlgorithms) > 0 {
		list, diags := types.ListValueFrom(ctx, types.StringType, encryptionAlgorithms)
		if diags.HasError() {
			return phase, fmt.Errorf("mapping base phase encryption_algorithms: %w", core.DiagsToError(diags))
		}
		phase.EncryptionAlgorithms = list
	} else {
		phase.EncryptionAlgorithms = types.ListNull(types.StringType)
	}

	if integrityAlgorithms, _ := apiPhase.GetIntegrityAlgorithmsOk(); len(integrityAlgorithms) > 0 {
		list, diags := types.ListValueFrom(ctx, types.StringType, integrityAlgorithms)
		if diags.HasError() {
			return phase, fmt.Errorf("mapping base phase integrity_algorithms: %w", core.DiagsToError(diags))
		}
		phase.IntegrityAlgorithms = list
	} else {
		phase.IntegrityAlgorithms = types.ListNull(types.StringType)
	}

	rekeyTime, _ := apiPhase.GetRekeyTimeOk()
	phase.RekeyTime = types.Int32PointerValue(rekeyTime)

	return phase, nil
}

func mapPhase1(ctx context.Context, apiPhase1 *vpn.TunnelConfigurationPhase1) (*Phase1Model, error) {
	basePhase, err := mapBasePhase(ctx, apiPhase1)
	if err != nil {
		return nil, err
	}

	return &Phase1Model{
		BasePhaseModel: basePhase,
	}, nil
}

func mapPhase2(ctx context.Context, apiPhase2 *vpn.TunnelConfigurationPhase2) (*Phase2Model, error) {
	if apiPhase2 == nil {
		return nil, fmt.Errorf("phase can not be nil")
	}

	basePhase, err := mapBasePhase(ctx, apiPhase2)
	if err != nil {
		return nil, err
	}

	phase2 := &Phase2Model{
		BasePhaseModel: basePhase,
	}

	if apiPhase2.StartAction != nil {
		phase2.StartAction = types.StringValue(string(*apiPhase2.StartAction))
	} else {
		phase2.StartAction = types.StringNull()
	}

	if apiPhase2.DpdAction != nil {
		phase2.DpdAction = types.StringValue(string(*apiPhase2.DpdAction))
	} else {
		phase2.DpdAction = types.StringNull()
	}

	return phase2, nil
}
