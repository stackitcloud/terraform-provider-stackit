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
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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

type Phase1Model struct {
	DhGroups             types.List  `tfsdk:"dh_groups"`
	EncryptionAlgorithms types.List  `tfsdk:"encryption_algorithms"`
	IntegrityAlgorithms  types.List  `tfsdk:"integrity_algorithms"`
	RekeyTime            types.Int32 `tfsdk:"rekey_time"`
}

type Phase2Model struct {
	DhGroups             types.List   `tfsdk:"dh_groups"`
	EncryptionAlgorithms types.List   `tfsdk:"encryption_algorithms"`
	IntegrityAlgorithms  types.List   `tfsdk:"integrity_algorithms"`
	RekeyTime            types.Int32  `tfsdk:"rekey_time"`
	StartAction          types.String `tfsdk:"start_action"`
	DpdAction            types.String `tfsdk:"dpd_action"`
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

var schemaDescriptions = map[string]string{
	"id":            "Terraform's internal resource identifier. Structured as \"`project_id`,`region`,`gateway_id`,`connection_id`\".",
	"connection_id": "The server-generated UUID of the VPN connection.",
	"project_id":    "STACKIT project ID.",
	"region":        "STACKIT region.",
	"gateway_id":    "The UUID of the parent VPN gateway.",
	"display_name":  "A user-friendly name for the connection. Must start and end with an alphanumeric character, may contain hyphens, and be 1-63 characters long.",
	"enabled":       "Whether this connection is enabled. Defaults to true.",
	"remote_subnet": "List of remote IPv4 CIDRs accessible via this connection. Optional for route-based and BGP configurations (defaults to 0.0.0.0/0). Mandatory for policy-based.",
	"local_subnet":  "List of local IPv4 CIDRs to route through this connection. Optional for route-based and BGP configurations (defaults to 0.0.0.0/0). Mandatory for policy-based.",
	"static_routes": "List of static routes (IPv4 CIDRs) for route-based VPN. Mandatory for ROUTE_BASED gateways.",
	"tunnel1":       "Configuration for the first IPsec tunnel.",
	"tunnel2":       "Configuration for the second IPsec tunnel.",
	"labels":        "Map of custom labels.",
}

var tunnelSchemaDescriptions = map[string]string{
	"tunnel":                       "Configuration for the IPsec tunnel.",
	"pre_shared_key":               "Pre-shared key for the IPsec tunnel. Minimum 20 characters. Write-only argument `pre_shared_key_wo` should be preferred.",
	"pre_shared_key_wo":            "Pre-shared key for the IPsec tunnel. Minimum 20 characters. Write-only - never stored in state and never returned by the API. To rotate the key, update this value AND increment pre_shared_key_wo_version. Changing this field alone will NOT trigger an update.",
	"pre_shared_key_wo_version":    "User-managed rotation counter for the pre-shared key. Must be incremented every time pre_shared_key_wo is changed. Terraform diffs this field to detect key rotations - changing pre_shared_key_wo alone will NOT trigger an update because it is write-only and never stored in state.",
	"remote_address":               "Remote IPv4 address for the tunnel endpoint.",
	"phase1_dh_groups":             fmt.Sprintf("Diffie-Hellman groups for key exchange. %s", tfutils.FormatPossibleValues(dhGroupValues...)),
	"phase1_encryption_algorithms": fmt.Sprintf("Encryption algorithms for Phase 1. %s", tfutils.FormatPossibleValues(encryptionAlgorithmValues...)),
	"phase1_integrity_algorithms":  fmt.Sprintf("Integrity algorithms for Phase 1. %s", tfutils.FormatPossibleValues(integrityAlgorithmValues...)),
	"phase1_rekey_time":            "Time to schedule an IKE re-keying in seconds. Range: 900-28800. Default: 14400.",
	"phase2_dh_groups":             fmt.Sprintf("Diffie-Hellman groups for Phase 2. %s", tfutils.FormatPossibleValues(dhGroupValues...)),
	"phase2_encryption_algorithms": fmt.Sprintf("Encryption algorithms for Phase 2. %s", tfutils.FormatPossibleValues(encryptionAlgorithmValues...)),
	"phase2_integrity_algorithms":  fmt.Sprintf("Integrity algorithms for Phase 2. %s", tfutils.FormatPossibleValues(integrityAlgorithmValues...)),
	"phase2_rekey_time":            "Time to schedule a Child SA re-keying in seconds. Range: 900-3600. Default: 3600.",
	"phase2_start_action":          fmt.Sprintf("Action to perform after loading the connection configuration. Default: 'start'. %s", tfutils.FormatPossibleValues(startActionValues...)),
	"phase2_dpd_action":            fmt.Sprintf("Action to perform on DPD timeout. Default: 'restart'. %s", tfutils.FormatPossibleValues(dpdActionValues...)),
	"peering_local_address":        "Local tunnel interface IPv4 address.",
	"peering_remote_address":       "Remote tunnel interface IPv4 address.",
	"bgp_remote_asn":               "Remote ASN for BGP peering (private ASN range, 64512-4294967294).",
}

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

func (r *vpnConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	tunnelSchema := schema.SingleNestedAttribute{
		Description:         tunnelSchemaDescriptions["tunnel"],
		MarkdownDescription: fmt.Sprintf("%s \n\n-> **Note:** Write-Only argument `pre_shared_key_wo` is available to use in place of `pre_shared_key`. Write-Only arguments are supported in HashiCorp Terraform 1.11.0 and later. [Learn more](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments).", tunnelSchemaDescriptions["tunnel"]),
		Required:            true,
		Validators: []validator.Object{
			objectvalidator.ExactlyOneOf(
				path.MatchRelative().AtName("pre_shared_key"),
				path.MatchRelative().AtName("pre_shared_key_wo"),
			),
		},
		Attributes: map[string]schema.Attribute{
			"pre_shared_key": schema.StringAttribute{
				Description: tunnelSchemaDescriptions["pre_shared_key"],
				Required:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(20),
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("pre_shared_key_wo"),
						path.MatchRelative().AtParent().AtName("pre_shared_key_wo_version"),
					),
					stringvalidator.PreferWriteOnlyAttribute(path.MatchRelative().AtParent().AtName("key_payload_base64_wo")),
				},
			},
			"pre_shared_key_wo": schema.StringAttribute{
				Description: tunnelSchemaDescriptions["pre_shared_key_wo"],
				Required:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(20),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("pre_shared_key")),
				},
			},
			"pre_shared_key_wo_version": schema.Int64Attribute{
				Description: tunnelSchemaDescriptions["pre_shared_key_wo_version"],
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRelative().AtParent().AtName("pre_shared_key_wo")),
					int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("pre_shared_key")),
				},
			},
			"remote_address": schema.StringAttribute{
				Description: tunnelSchemaDescriptions["remote_address"],
				Required:    true,
				Validators: []validator.String{
					validate.IP(true),
				},
			},
			"phase1": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"dh_groups": schema.ListAttribute{
						Description: tunnelSchemaDescriptions["phase1_dh_groups"],
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(dhGroupValues...),
							),
						},
					},
					"encryption_algorithms": schema.ListAttribute{
						Description: tunnelSchemaDescriptions["phase1_encryption_algorithms"],
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(encryptionAlgorithmValues...),
							),
						},
					},
					"integrity_algorithms": schema.ListAttribute{
						Description: tunnelSchemaDescriptions["phase1_integrity_algorithms"],
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(integrityAlgorithmValues...),
							),
						},
					},
					"rekey_time": schema.Int32Attribute{
						Description: tunnelSchemaDescriptions["phase1_rekey_time"],
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
						Description: tunnelSchemaDescriptions["phase2_dh_groups"],
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(dhGroupValues...),
							),
						},
					},
					"encryption_algorithms": schema.ListAttribute{
						Description: tunnelSchemaDescriptions["phase2_encryption_algorithms"],
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(encryptionAlgorithmValues...),
							),
						},
					},
					"integrity_algorithms": schema.ListAttribute{
						Description: tunnelSchemaDescriptions["phase2_integrity_algorithms"],
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(
								stringvalidator.OneOf(integrityAlgorithmValues...),
							),
						},
					},
					"rekey_time": schema.Int32Attribute{
						Description: tunnelSchemaDescriptions["phase2_rekey_time"],
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int32{
							int32validator.Between(900, 3600),
						},
					},
					"start_action": schema.StringAttribute{
						Description: tunnelSchemaDescriptions["phase2_start_action"],
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(startActionValues...),
						},
					},
					"dpd_action": schema.StringAttribute{
						Description: tunnelSchemaDescriptions["phase2_dpd_action"],
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
						Description: tunnelSchemaDescriptions["peering_local_address"],
						Required:    true,
						Validators: []validator.String{
							validate.IP(true),
						},
					},
					"remote_address": schema.StringAttribute{
						Description: tunnelSchemaDescriptions["peering_remote_address"],
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
						Description: tunnelSchemaDescriptions["bgp_remote_asn"],
						Required:    true,
						Validators: []validator.Int64{
							int64validator.Between(64512, 4294967294),
						},
					},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("VPN Connection resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_id": schema.StringAttribute{
				Description: schemaDescriptions["connection_id"],
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
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gateway_id": schema.StringAttribute{
				Description: schemaDescriptions["gateway_id"],
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
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`),
						"must start and end with an alphanumeric character, may contain hyphens, and be 1-63 characters long",
					),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: schemaDescriptions["enabled"],
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"remote_subnet": schema.ListAttribute{
				Description: schemaDescriptions["remote_subnet"],
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 100),
					listvalidator.ValueStringsAre(validate.CIDR()),
				},
			},
			"local_subnet": schema.ListAttribute{
				Description: schemaDescriptions["local_subnet"],
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 100),
					listvalidator.ValueStringsAre(validate.CIDR()),
				},
			},
			"static_routes": schema.ListAttribute{
				Description: schemaDescriptions["static_routes"],
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(validate.CIDR()),
				},
			},
			"tunnel1": tunnelSchema,
			"tunnel2": tunnelSchema,
			"labels": schema.MapAttribute{
				Description: schemaDescriptions["labels"],
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
	model.Tunnel1.PreSharedKey = configModel.Tunnel1.PreSharedKey
	model.Tunnel1.PreSharedKeyWo = configModel.Tunnel1.PreSharedKeyWo
	model.Tunnel2.PreSharedKey = configModel.Tunnel2.PreSharedKey
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

	if createResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN connection", "Got empty connection id")
		return
	}
	connectionId := *createResp.Id

	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":    projectId,
		"region":        region,
		"gateway_id":    gatewayId,
		"connection_id": connectionId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	connResp, err := r.client.DefaultAPI.GetGatewayConnection(ctx, projectId, region, gatewayId, connectionId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating VPN connection", fmt.Sprintf("Reading created connection: %v", err))
		return
	}

	err = mapFields(ctx, connResp, &model, region)
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
		pv := model.Tunnel1.PreSharedKeyWoVersion.ValueInt64()
		sv := stateModel.Tunnel1.PreSharedKeyWoVersion.ValueInt64()
		if pv < sv {
			resp.Diagnostics.AddAttributeError(
				path.Root("tunnel1").AtName("pre_shared_key_wo_version"),
				"Version must not decrease",
				fmt.Sprintf("`pre_shared_key_wo_version` must be incremented to rotate the pre-shared key, got %d (current: %d).", pv, sv),
			)
			return
		}
		if pv > sv {
			// Secret must be read from Config, not Plan — write-only values are always null in plan.
			model.Tunnel1.PreSharedKeyWo = configModel.Tunnel1.PreSharedKeyWo
		}
	}

	// tunnel2 PSK rotation
	if !tfutils.IsUndefined(model.Tunnel2.PreSharedKeyWoVersion) {
		pv := model.Tunnel2.PreSharedKeyWoVersion.ValueInt64()
		sv := stateModel.Tunnel2.PreSharedKeyWoVersion.ValueInt64()
		if pv < sv {
			resp.Diagnostics.AddAttributeError(
				path.Root("tunnel2").AtName("pre_shared_key_wo_version"),
				"Version must not decrease",
				fmt.Sprintf("`pre_shared_key_wo_version` must be incremented to rotate the pre-shared key, got %d (current: %d).", pv, sv),
			)
			return
		}
		if pv > sv {
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

	_, err = r.client.DefaultAPI.UpdateGatewayConnection(ctx, projectId, region, gatewayId, connectionId).UpdateGatewayConnectionPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN connection", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	connResp, err := r.client.DefaultAPI.GetGatewayConnection(ctx, projectId, region, gatewayId, connectionId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating VPN connection", fmt.Sprintf("Reading updated connection: %v", err))
		return
	}

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

	fields, err := toConnectionFields(ctx, model)
	if err != nil {
		return nil, err
	}

	return &vpn.CreateGatewayConnectionPayload{
		DisplayName:   fields.displayName,
		Tunnel1:       fields.tunnel1,
		Tunnel2:       fields.tunnel2,
		Enabled:       fields.enabled,
		RemoteSubnets: fields.remoteSubnets,
		LocalSubnets:  fields.localSubnets,
		StaticRoutes:  fields.staticRoutes,
		Labels:        &fields.labels,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model) (*vpn.UpdateGatewayConnectionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	fields, err := toConnectionFields(ctx, model)
	if err != nil {
		return nil, err
	}

	return &vpn.UpdateGatewayConnectionPayload{
		DisplayName:   fields.displayName,
		Tunnel1:       fields.tunnel1,
		Tunnel2:       fields.tunnel2,
		Enabled:       fields.enabled,
		RemoteSubnets: fields.remoteSubnets,
		LocalSubnets:  fields.localSubnets,
		StaticRoutes:  fields.staticRoutes,
		Labels:        &fields.labels,
	}, nil
}

type connectionFields struct {
	displayName   string
	tunnel1       vpn.TunnelConfiguration
	tunnel2       vpn.TunnelConfiguration
	enabled       *bool
	remoteSubnets []string
	localSubnets  []string
	staticRoutes  []string
	labels        map[string]string
}

func toConnectionFields(ctx context.Context, model *Model) (*connectionFields, error) {
	tunnel1, err := toTunnelConfiguration(model.Tunnel1)
	if err != nil {
		return nil, fmt.Errorf("converting tunnel1: %w", err)
	}

	tunnel2, err := toTunnelConfiguration(model.Tunnel2)
	if err != nil {
		return nil, fmt.Errorf("converting tunnel2: %w", err)
	}

	fields := &connectionFields{
		displayName: model.DisplayName.ValueString(),
		tunnel1:     *tunnel1,
		tunnel2:     *tunnel2,
	}

	if !tfutils.IsUndefined(model.Enabled) {
		enabled := model.Enabled.ValueBool()
		fields.enabled = &enabled
	}

	if !tfutils.IsUndefined(model.RemoteSubnet) {
		remoteSubnets, err := tfutils.ListValueToStringSlice(model.RemoteSubnet)
		if err != nil {
			return nil, fmt.Errorf("converting remote_subnet: %w", err)
		}
		fields.remoteSubnets = remoteSubnets
	}

	if !tfutils.IsUndefined(model.LocalSubnet) {
		localSubnets, err := tfutils.ListValueToStringSlice(model.LocalSubnet)
		if err != nil {
			return nil, fmt.Errorf("converting local_subnet: %w", err)
		}
		fields.localSubnets = localSubnets
	}

	if !tfutils.IsUndefined(model.StaticRoutes) {
		staticRoutes, err := tfutils.ListValueToStringSlice(model.StaticRoutes)
		if err != nil {
			return nil, fmt.Errorf("converting static_routes: %w", err)
		}
		fields.staticRoutes = staticRoutes
	}

	fields.labels, err = tfutils.LabelsToPayload(ctx, model.Labels)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

func toTunnelConfiguration(tunnel *TunnelModel) (*vpn.TunnelConfiguration, error) {
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

func mapFields(ctx context.Context, conn *vpn.ConnectionResponse, model *Model, region string) error {
	if conn == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var connectionId string
	if conn.Id != nil {
		connectionId = *conn.Id
	} else if model.ConnectionID.ValueString() != "" {
		connectionId = model.ConnectionID.ValueString()
	} else {
		return fmt.Errorf("connection id not present")
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), region, model.GatewayID.ValueString(), connectionId)
	model.ConnectionID = types.StringValue(connectionId)
	model.DisplayName = types.StringValue(conn.DisplayName)
	model.Region = types.StringValue(region)

	if conn.Enabled != nil {
		model.Enabled = types.BoolValue(*conn.Enabled)
	} else {
		model.Enabled = types.BoolValue(true)
	}

	if conn.RemoteSubnets != nil {
		list, diags := types.ListValueFrom(ctx, types.StringType, conn.RemoteSubnets)
		if diags.HasError() {
			return fmt.Errorf("mapping remote_subnet: %w", core.DiagsToError(diags))
		}
		model.RemoteSubnet = list
	} else {
		model.RemoteSubnet = types.ListNull(types.StringType)
	}

	if conn.LocalSubnets != nil {
		list, diags := types.ListValueFrom(ctx, types.StringType, conn.LocalSubnets)
		if diags.HasError() {
			return fmt.Errorf("mapping local_subnet: %w", core.DiagsToError(diags))
		}
		model.LocalSubnet = list
	} else {
		model.LocalSubnet = types.ListNull(types.StringType)
	}

	if conn.StaticRoutes != nil {
		list, diags := types.ListValueFrom(ctx, types.StringType, conn.StaticRoutes)
		if diags.HasError() {
			return fmt.Errorf("mapping static_routes: %w", core.DiagsToError(diags))
		}
		model.StaticRoutes = list
	} else {
		model.StaticRoutes = types.ListNull(types.StringType)
	}

	tunnel1, err := mapTunnel(ctx, &conn.Tunnel1, model.Tunnel1)
	if err != nil {
		return fmt.Errorf("mapping tunnel1: %w", err)
	}
	model.Tunnel1 = tunnel1

	tunnel2, err := mapTunnel(ctx, &conn.Tunnel2, model.Tunnel2)
	if err != nil {
		return fmt.Errorf("mapping tunnel2: %w", err)
	}
	model.Tunnel2 = tunnel2

	labels, err := tfutils.MapLabels(ctx, conn.Labels, model.Labels)
	if err != nil {
		return fmt.Errorf("mapping labels: %w", err)
	}
	model.Labels = labels

	return nil
}

func mapTunnel(ctx context.Context, apiTunnel *vpn.TunnelConfiguration, cuurrentTunnel *TunnelModel) (*TunnelModel, error) {
	tunnel := &TunnelModel{
		RemoteAddress: types.StringValue(string(apiTunnel.RemoteAddress)),
	}
	phase1 := &Phase1Model{}
	if len(apiTunnel.Phase1.DhGroups) > 0 {
		list, diags := types.ListValueFrom(ctx, types.StringType, apiTunnel.Phase1.DhGroups)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping phase1 dh_groups: %w", core.DiagsToError(diags))
		}
		phase1.DhGroups = list
	} else {
		phase1.DhGroups = types.ListNull(types.StringType)
	}
	if len(apiTunnel.Phase1.EncryptionAlgorithms) > 0 {
		list, diags := types.ListValueFrom(ctx, types.StringType, apiTunnel.Phase1.EncryptionAlgorithms)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping phase1 encryption_algorithms: %w", core.DiagsToError(diags))
		}
		phase1.EncryptionAlgorithms = list
	} else {
		phase1.EncryptionAlgorithms = types.ListNull(types.StringType)
	}
	if len(apiTunnel.Phase1.IntegrityAlgorithms) > 0 {
		list, diags := types.ListValueFrom(ctx, types.StringType, apiTunnel.Phase1.IntegrityAlgorithms)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping phase1 integrity_algorithms: %w", core.DiagsToError(diags))
		}
		phase1.IntegrityAlgorithms = list
	} else {
		phase1.IntegrityAlgorithms = types.ListNull(types.StringType)
	}
	if apiTunnel.Phase1.RekeyTime != nil {
		phase1.RekeyTime = types.Int32Value(*apiTunnel.Phase1.RekeyTime)
	} else {
		phase1.RekeyTime = types.Int32Null()
	}
	tunnel.Phase1 = phase1

	phase2 := &Phase2Model{}
	if len(apiTunnel.Phase2.DhGroups) > 0 {
		list, diags := types.ListValueFrom(ctx, types.StringType, apiTunnel.Phase2.DhGroups)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping phase2 dh_groups: %w", core.DiagsToError(diags))
		}
		phase2.DhGroups = list
	} else {
		phase2.DhGroups = types.ListNull(types.StringType)
	}
	if len(apiTunnel.Phase2.EncryptionAlgorithms) > 0 {
		list, diags := types.ListValueFrom(ctx, types.StringType, apiTunnel.Phase2.EncryptionAlgorithms)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping phase2 encryption_algorithms: %w", core.DiagsToError(diags))
		}
		phase2.EncryptionAlgorithms = list
	} else {
		phase2.EncryptionAlgorithms = types.ListNull(types.StringType)
	}
	if len(apiTunnel.Phase2.IntegrityAlgorithms) > 0 {
		list, diags := types.ListValueFrom(ctx, types.StringType, apiTunnel.Phase2.IntegrityAlgorithms)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping phase2 integrity_algorithms: %w", core.DiagsToError(diags))
		}
		phase2.IntegrityAlgorithms = list
	} else {
		phase2.IntegrityAlgorithms = types.ListNull(types.StringType)
	}
	if apiTunnel.Phase2.RekeyTime != nil {
		phase2.RekeyTime = types.Int32Value(*apiTunnel.Phase2.RekeyTime)
	} else {
		phase2.RekeyTime = types.Int32Null()
	}
	if apiTunnel.Phase2.StartAction != nil {
		phase2.StartAction = types.StringValue(string(*apiTunnel.Phase2.StartAction))
	} else {
		phase2.StartAction = types.StringNull()
	}
	if apiTunnel.Phase2.DpdAction != nil {
		phase2.DpdAction = types.StringValue(string(*apiTunnel.Phase2.DpdAction))
	} else {
		phase2.DpdAction = types.StringNull()
	}
	tunnel.Phase2 = phase2

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
		tunnel.Peering = peering
	}

	if apiTunnel.Bgp != nil {
		tunnel.Bgp = &BGPTunnelConfigModel{
			RemoteAsn: types.Int64Value(int64(apiTunnel.Bgp.RemoteAsn)),
		}
	}

	// could be nil for Read after a terraform import
	if cuurrentTunnel != nil {
		tunnel.PreSharedKeyWoVersion = cuurrentTunnel.PreSharedKeyWoVersion
	} else {
		tunnel.PreSharedKeyWoVersion = types.Int64Null()
	}

	return tunnel, nil
}
