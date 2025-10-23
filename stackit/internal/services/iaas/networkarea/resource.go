package networkarea

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	resourcemanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

const (
	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	defaultValueDefaultPrefixLength = 25

	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	defaultValueMinPrefixLength = 24

	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	defaultValueMaxPrefixLength = 29

	// Deprecated: Will be removed in May 2026.
	deprecationWarningSummary = "Migration to new `stackit_network_area_region` resource needed"
	// Deprecated: Will be removed in May 2026.
	deprecationWarningDetails = "You're using deprecated features of the `stackit_network_area` resource. These will be removed in May 2026. Migrate to the new `stackit_network_area_region` resource instead."
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &networkAreaResource{}
	_ resource.ResourceWithConfigure      = &networkAreaResource{}
	_ resource.ResourceWithImportState    = &networkAreaResource{}
	_ resource.ResourceWithValidateConfig = &networkAreaResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	OrganizationId types.String `tfsdk:"organization_id"`
	NetworkAreaId  types.String `tfsdk:"network_area_id"`
	Name           types.String `tfsdk:"name"`
	ProjectCount   types.Int64  `tfsdk:"project_count"`
	Labels         types.Map    `tfsdk:"labels"`

	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	DefaultNameservers types.List `tfsdk:"default_nameservers"`
	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	MaxPrefixLength types.Int64 `tfsdk:"max_prefix_length"`
	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	NetworkRanges types.List `tfsdk:"network_ranges"`
	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	TransferNetwork types.String `tfsdk:"transfer_network"`
	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	DefaultPrefixLength types.Int64 `tfsdk:"default_prefix_length"`
	// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	MinPrefixLength types.Int64 `tfsdk:"min_prefix_length"`
}

// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider. LegacyMode checks if any of the deprecated fields are set which now relate to the network area region API resource.
func (model *Model) LegacyMode() bool {
	return !model.NetworkRanges.IsNull() || model.NetworkRanges.IsUnknown() || !model.TransferNetwork.IsNull() || model.TransferNetwork.IsUnknown() || !model.DefaultNameservers.IsNull() || model.DefaultNameservers.IsUnknown() || model.DefaultPrefixLength != types.Int64Value(int64(defaultValueDefaultPrefixLength)) || model.MinPrefixLength != types.Int64Value(int64(defaultValueMinPrefixLength)) || model.MaxPrefixLength != types.Int64Value(int64(defaultValueMaxPrefixLength))
}

// Struct corresponding to Model.NetworkRanges[i]
type networkRange struct {
	Prefix         types.String `tfsdk:"prefix"`
	NetworkRangeId types.String `tfsdk:"network_range_id"`
}

// Types corresponding to networkRanges
var networkRangeTypes = map[string]attr.Type{
	"prefix":           types.StringType,
	"network_range_id": types.StringType,
}

// NewNetworkAreaResource is a helper function to simplify the provider implementation.
func NewNetworkAreaResource() resource.Resource {
	return &networkAreaResource{}
}

// networkResource is the resource implementation.
type networkAreaResource struct {
	client                *iaas.APIClient
	resourceManagerClient *resourcemanager.APIClient
}

// Metadata returns the resource type name.
func (r *networkAreaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_area"
}

// Configure adds the provider configured client to the resource.
func (r *networkAreaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	resourceManagerClient := resourcemanagerUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.resourceManagerClient = resourceManagerClient
	tflog.Info(ctx, "IaaS client configured")
}

// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
func (r *networkAreaResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var resourceModel Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &resourceModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if resourceModel.NetworkRanges.IsNull() != resourceModel.TransferNetwork.IsNull() {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring network network area", "You have to either provide both the `network_ranges` and `transfer_network` fields simultaneously or none of them.")
	}

	if (resourceModel.NetworkRanges.IsNull() || resourceModel.TransferNetwork.IsNull()) && (!resourceModel.DefaultNameservers.IsNull() || !resourceModel.DefaultPrefixLength.IsNull() || !resourceModel.MinPrefixLength.IsNull() || !resourceModel.MaxPrefixLength.IsNull()) {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring network network area", "You have to provide both the `network_ranges` and `transfer_network` fields when providing one of these fields: `default_nameservers`, `default_prefix_length`, `max_prefix_length`, `min_prefix_length`")
	}
}

// Schema defines the schema for the resource.
func (r *networkAreaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	deprecationMsg := "Deprecated because of the IaaS API v1 -> v2 migration. Will be removed in May 2026. Use the new `stackit_network_area_region` resource instead."
	description := "Network area resource schema."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`organization_id`,`network_area_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT organization ID to which the network area is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_area_id": schema.StringAttribute{
				Description: "The network area ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the network area.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"project_count": schema.Int64Attribute{
				Description: "The amount of projects currently referencing this area.",
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
			"default_nameservers": schema.ListAttribute{
				Description:        "List of DNS Servers/Nameservers for configuration of network area for region `eu01`.",
				DeprecationMessage: deprecationMsg,
				Optional:           true,
				ElementType:        types.StringType,
			},
			// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
			"network_ranges": schema.ListNestedAttribute{
				Description:        "List of Network ranges for configuration of network area for region `eu01`.",
				DeprecationMessage: deprecationMsg,
				Optional:           true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(64),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_range_id": schema.StringAttribute{
							DeprecationMessage: deprecationMsg,
							Computed:           true,
							Validators: []validator.String{
								validate.UUID(),
								validate.NoSeparator(),
							},
						},
						"prefix": schema.StringAttribute{
							DeprecationMessage: deprecationMsg,
							Description:        "Classless Inter-Domain Routing (CIDR).",
							Required:           true,
						},
					},
				},
			},
			// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
			"transfer_network": schema.StringAttribute{
				DeprecationMessage: deprecationMsg,
				Description:        "Classless Inter-Domain Routing (CIDR) for configuration of network area for region `eu01`.",
				Optional:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
			"default_prefix_length": schema.Int64Attribute{
				DeprecationMessage: deprecationMsg,
				Description:        "The default prefix length for networks in the network area for region `eu01`.",
				Optional:           true,
				Computed:           true,
				Validators: []validator.Int64{
					int64validator.AtLeast(24),
					int64validator.AtMost(29),
				},
				Default: int64default.StaticInt64(defaultValueDefaultPrefixLength),
			},
			// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
			"max_prefix_length": schema.Int64Attribute{
				DeprecationMessage: deprecationMsg,
				Description:        "The maximal prefix length for networks in the network area for region `eu01`.",
				Optional:           true,
				Computed:           true,
				Validators: []validator.Int64{
					int64validator.AtLeast(24),
					int64validator.AtMost(29),
				},
				Default: int64default.StaticInt64(defaultValueMaxPrefixLength),
			},
			// Deprecated: Will be removed in May 2026. Only kept to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
			"min_prefix_length": schema.Int64Attribute{
				DeprecationMessage: deprecationMsg,
				Description:        "The minimal prefix length for networks in the network area for region `eu01`.",
				Optional:           true,
				Computed:           true,
				Validators: []validator.Int64{
					int64validator.AtLeast(8),
					int64validator.AtMost(29),
				},
				Default: int64default.StaticInt64(defaultValueMinPrefixLength),
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *networkAreaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network area
	networkArea, err := r.client.CreateNetworkArea(ctx, organizationId).CreateNetworkAreaPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	networkAreaId := *networkArea.Id
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	// Map response body to schema
	err = mapFields(ctx, networkArea, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	if model.LegacyMode() {
		core.LogAndAddWarning(ctx, &resp.Diagnostics, deprecationWarningSummary, deprecationWarningDetails)

		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		regionCreatePayload, err := toRegionCreatePayload(ctx, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area region", fmt.Sprintf("Creating API payload: %v", err))
			return
		}

		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		_, err = r.client.CreateNetworkAreaRegion(ctx, organizationId, networkAreaId, "eu01").CreateNetworkAreaRegionPayload(*regionCreatePayload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area region", fmt.Sprintf("Calling API: %v", err))
			return
		}

		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		networkAreaRegionResp, err := wait.CreateNetworkAreaRegionWaitHandler(ctx, r.client, organizationId, networkAreaId, "eu01").WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error waiting for network area region creation", fmt.Sprintf("Calling API: %v", err))
			return
		}

		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		err = mapNetworkAreaRegionFields(ctx, networkAreaRegionResp, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network area region", fmt.Sprintf("Processing API payload: %v", err))
			return
		}
	} else {
		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		model.NetworkRanges = types.ListNull(types.ObjectType{AttrTypes: networkRangeTypes})
		model.DefaultNameservers = types.ListNull(types.StringType)
		model.TransferNetwork = types.StringNull()
		model.DefaultPrefixLength = types.Int64Value(defaultValueDefaultPrefixLength)
		model.MinPrefixLength = types.Int64Value(defaultValueMinPrefixLength)
		model.MaxPrefixLength = types.Int64Value(defaultValueMaxPrefixLength)
	}

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area created")
}

// Read refreshes the Terraform state with the latest data.
func (r *networkAreaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	networkAreaResp, err := r.client.GetNetworkArea(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, networkAreaResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	if model.LegacyMode() {
		core.LogAndAddWarning(ctx, &resp.Diagnostics, deprecationWarningSummary, deprecationWarningDetails)

		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		networkAreaRegionResp, err := r.client.GetNetworkAreaRegion(ctx, organizationId, networkAreaId, "eu01").Execute()
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			ok := errors.As(err, &oapiErr)
			if !(ok && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusBadRequest)) { // TODO: iaas api returns http 400 in case network area region is not found
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area region", fmt.Sprintf("Calling API: %v", err))
				return
			}

			model.NetworkRanges = types.ListNull(types.ObjectType{AttrTypes: networkRangeTypes})
			model.DefaultNameservers = types.ListNull(types.StringType)
			model.TransferNetwork = types.StringNull()
			model.DefaultPrefixLength = types.Int64Value(defaultValueDefaultPrefixLength)
			model.MinPrefixLength = types.Int64Value(defaultValueMinPrefixLength)
			model.MaxPrefixLength = types.Int64Value(defaultValueMaxPrefixLength)
		} else {
			// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
			err = mapNetworkAreaRegionFields(ctx, networkAreaRegionResp, &model)
			if err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area region", fmt.Sprintf("Processing API payload: %v", err))
				return
			}
		}
	} else {
		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		model.NetworkRanges = types.ListNull(types.ObjectType{AttrTypes: networkRangeTypes})
		model.DefaultNameservers = types.ListNull(types.StringType)
		model.TransferNetwork = types.StringNull()
		model.DefaultPrefixLength = types.Int64Value(defaultValueDefaultPrefixLength)
		model.MinPrefixLength = types.Int64Value(defaultValueMinPrefixLength)
		model.MaxPrefixLength = types.Int64Value(defaultValueMaxPrefixLength)
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *networkAreaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	ranges := []networkRange{}
	if !(model.NetworkRanges.IsNull() || model.NetworkRanges.IsUnknown()) {
		resp.Diagnostics.Append(model.NetworkRanges.ElementsAs(ctx, &ranges, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Retrieve values from state
	var stateModel Model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing network
	networkAreaUpdateResp, err := r.client.PartialUpdateNetworkArea(ctx, organizationId, networkAreaId).PartialUpdateNetworkAreaPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, networkAreaUpdateResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
	if model.LegacyMode() {
		core.LogAndAddWarning(ctx, &resp.Diagnostics, deprecationWarningSummary, deprecationWarningDetails)

		// Deprecated: Update network area region payload creation. Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		regionUpdatePayload, err := toRegionUpdatePayload(ctx, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area region", fmt.Sprintf("Creating API payload: %v", err))
			return
		}

		// Deprecated: Update network area region. Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		networkAreaRegionUpdateResp, err := r.client.UpdateNetworkAreaRegion(ctx, organizationId, networkAreaId, "eu01").UpdateNetworkAreaRegionPayload(*regionUpdatePayload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area region", fmt.Sprintf("Calling API: %v", err))
			return
		}

		// Deprecated: Update network area region. Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		err = mapNetworkAreaRegionFields(ctx, networkAreaRegionUpdateResp, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area region", fmt.Sprintf("Processing API payload: %v", err))
			return
		}

		// Deprecated: Update network ranges. Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		err = updateNetworkRanges(ctx, organizationId, networkAreaId, ranges, r.client)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network area region", fmt.Sprintf("Updating Network ranges: %v", err))
			return
		}

		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		networkAreaRegionResp, err := r.client.GetNetworkAreaRegion(ctx, organizationId, networkAreaId, "eu01").Execute()
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			ok := errors.As(err, &oapiErr)
			if ok && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusBadRequest) { // TODO: iaas api returns http 400 in case network area region is not found
				return
			}
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area region", fmt.Sprintf("Calling API: %v", err))
			return
		}

		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		err = mapNetworkAreaRegionFields(ctx, networkAreaRegionResp, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area region", fmt.Sprintf("Processing API payload: %v", err))
			return
		}
	} else {
		// Deprecated: Will be removed in May 2026. Only introduced to make the IaaS v1 -> v2 API migration non-breaking in the Terraform provider.
		model.NetworkRanges = types.ListNull(types.ObjectType{AttrTypes: networkRangeTypes})
		model.DefaultNameservers = types.ListNull(types.StringType)
		model.TransferNetwork = types.StringNull()
		model.DefaultPrefixLength = types.Int64Value(defaultValueDefaultPrefixLength)
		model.MinPrefixLength = types.Int64Value(defaultValueMinPrefixLength)
		model.MaxPrefixLength = types.Int64Value(defaultValueMaxPrefixLength)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *networkAreaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	_, err := wait.ReadyForNetworkAreaDeletionWaitHandler(ctx, r.client, r.resourceManagerClient, organizationId, networkAreaId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area", fmt.Sprintf("Network area ready for deletion waiting: %v", err))
		return
	}

	// Get all configured regions so we can delete them one by one before deleting the network area
	regionsListResp, err := r.client.ListNetworkAreaRegions(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area region", fmt.Sprintf("Calling API to list configured regions: %v", err))
		return
	}

	// Delete network region configurations
	for region := range *regionsListResp.Regions {
		err = r.client.DeleteNetworkAreaRegion(ctx, organizationId, networkAreaId, region).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area region", fmt.Sprintf("Calling API: %v", err))
			return
		}

		_, err = wait.DeleteNetworkAreaRegionWaitHandler(ctx, r.client, organizationId, networkAreaId, region).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area region", fmt.Sprintf("Waiting for networea deletion: %v", err))
			return
		}
	}

	// Delete existing network area
	err = r.client.DeleteNetworkArea(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network area", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Network area deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,network_id
func (r *networkAreaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network area",
			fmt.Sprintf("Expected import identifier with format: [organization_id],[network_area_id]  Got: %q", req.ID),
		)
		return
	}

	organizationId := idParts[0]
	networkAreaId := idParts[1]
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_area_id"), networkAreaId)...)
	tflog.Info(ctx, "Network state imported")
}

func mapFields(ctx context.Context, networkAreaResp *iaas.NetworkArea, model *Model) error {
	if networkAreaResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkAreaId string
	if model.NetworkAreaId.ValueString() != "" {
		networkAreaId = model.NetworkAreaId.ValueString()
	} else if networkAreaResp.Id != nil {
		networkAreaId = *networkAreaResp.Id
	} else {
		return fmt.Errorf("network area id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.OrganizationId.ValueString(), networkAreaId)

	labels, err := iaasUtils.MapLabels(ctx, networkAreaResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.NetworkAreaId = types.StringValue(networkAreaId)
	model.Name = types.StringPointerValue(networkAreaResp.Name)
	model.ProjectCount = types.Int64PointerValue(networkAreaResp.ProjectCount)
	model.Labels = labels

	return nil
}

// Deprecated: mapRegionFields maps the region configuration for eu01 to avoid a breaking change in the Terraform provider during the IaaS v1 -> v2 API migration. Will be removed in May 2026.
func mapNetworkAreaRegionFields(ctx context.Context, networkAreaRegionResp *iaas.RegionalArea, model *Model) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	if networkAreaRegionResp == nil {
		return fmt.Errorf("response input is nil")
	}

	// map default nameservers
	if networkAreaRegionResp.Ipv4 == nil || networkAreaRegionResp.Ipv4.DefaultNameservers == nil {
		model.DefaultNameservers = types.ListNull(types.StringType)
	} else {
		respDefaultNameservers := *networkAreaRegionResp.Ipv4.DefaultNameservers
		modelDefaultNameservers, err := utils.ListValuetoStringSlice(model.DefaultNameservers)
		if err != nil {
			return fmt.Errorf("get current network area default nameservers from model: %w", err)
		}

		reconciledDefaultNameservers := utils.ReconcileStringSlices(modelDefaultNameservers, respDefaultNameservers)

		defaultNameserversTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledDefaultNameservers)
		if diags.HasError() {
			return fmt.Errorf("map network area default nameservers: %w", core.DiagsToError(diags))
		}

		model.DefaultNameservers = defaultNameserversTF
	}

	// map network ranges
	if networkAreaRegionResp.Ipv4 == nil || networkAreaRegionResp.Ipv4.NetworkRanges == nil {
		model.NetworkRanges = types.ListNull(types.ObjectType{AttrTypes: networkRangeTypes})
	} else {
		err := mapNetworkRanges(ctx, networkAreaRegionResp.Ipv4.NetworkRanges, model)
		if err != nil {
			return fmt.Errorf("mapping network ranges: %w", err)
		}
	}

	// map remaining fields
	if networkAreaRegionResp.Ipv4 != nil {
		model.TransferNetwork = types.StringPointerValue(networkAreaRegionResp.Ipv4.TransferNetwork)
		model.DefaultPrefixLength = types.Int64PointerValue(networkAreaRegionResp.Ipv4.DefaultPrefixLen)
		model.MaxPrefixLength = types.Int64PointerValue(networkAreaRegionResp.Ipv4.MaxPrefixLen)
		model.MinPrefixLength = types.Int64PointerValue(networkAreaRegionResp.Ipv4.MinPrefixLen)
	}

	return nil
}

// Deprecated: mapNetworkRanges will be removed in May 2026. Implementation won't be needed anymore because of the IaaS API v1 -> v2 migration. Func was only kept to circumvent breaking changes.
func mapNetworkRanges(ctx context.Context, networkAreaRangesList *[]iaas.NetworkRange, model *Model) error {
	var diags diag.Diagnostics

	if networkAreaRangesList == nil {
		return fmt.Errorf("nil network area ranges list")
	}
	if len(*networkAreaRangesList) == 0 {
		model.NetworkRanges = types.ListNull(types.ObjectType{AttrTypes: networkRangeTypes})
		return nil
	}

	ranges := []networkRange{}
	if !(model.NetworkRanges.IsNull() || model.NetworkRanges.IsUnknown()) {
		diags = model.NetworkRanges.ElementsAs(ctx, &ranges, false)
		if diags.HasError() {
			return fmt.Errorf("map network ranges: %w", core.DiagsToError(diags))
		}
	}

	modelNetworkRangePrefixes := []string{}
	for _, m := range ranges {
		modelNetworkRangePrefixes = append(modelNetworkRangePrefixes, m.Prefix.ValueString())
	}

	apiNetworkRangePrefixes := []string{}
	for _, n := range *networkAreaRangesList {
		apiNetworkRangePrefixes = append(apiNetworkRangePrefixes, *n.Prefix)
	}

	reconciledRangePrefixes := utils.ReconcileStringSlices(modelNetworkRangePrefixes, apiNetworkRangePrefixes)

	networkRangesList := []attr.Value{}
	for i, prefix := range reconciledRangePrefixes {
		var networkRangeId string
		for _, networkRangeElement := range *networkAreaRangesList {
			if *networkRangeElement.Prefix == prefix {
				networkRangeId = *networkRangeElement.Id
				break
			}
		}
		networkRangeMap := map[string]attr.Value{
			"prefix":           types.StringValue(prefix),
			"network_range_id": types.StringValue(networkRangeId),
		}

		networkRangeTF, diags := types.ObjectValue(networkRangeTypes, networkRangeMap)
		if diags.HasError() {
			return fmt.Errorf("mapping index %d: %w", i, core.DiagsToError(diags))
		}

		networkRangesList = append(networkRangesList, networkRangeTF)
	}

	networkRangesTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: networkRangeTypes},
		networkRangesList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	model.NetworkRanges = networkRangesTF
	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateNetworkAreaPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.CreateNetworkAreaPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: &labels,
	}, nil
}

// Deprecated: toRegionCreatePayload will be removed in May 2026. Implementation won't be needed anymore because of the IaaS API v1 -> v2 migration. Func was only introduced to circumvent breaking changes.
func toRegionCreatePayload(ctx context.Context, model *Model) (*iaas.CreateNetworkAreaRegionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelDefaultNameservers, err := toDefaultNameserversPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting default nameservers: %w", err)
	}

	networkRangesPayload, err := toNetworkRangesPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting network ranges: %w", err)
	}

	return &iaas.CreateNetworkAreaRegionPayload{
		Ipv4: &iaas.RegionalAreaIPv4{
			DefaultNameservers: &modelDefaultNameservers,
			DefaultPrefixLen:   conversion.Int64ValueToPointer(model.DefaultPrefixLength),
			MaxPrefixLen:       conversion.Int64ValueToPointer(model.MaxPrefixLength),
			MinPrefixLen:       conversion.Int64ValueToPointer(model.MinPrefixLength),
			TransferNetwork:    conversion.StringValueToPointer(model.TransferNetwork),
			NetworkRanges:      networkRangesPayload,
		},
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.PartialUpdateNetworkAreaPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.PartialUpdateNetworkAreaPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: &labels,
	}, nil
}

// Deprecated: toRegionUpdatePayload will be removed in May 2026. Implementation won't be needed anymore because of the IaaS API v1 -> v2 migration. Func was only introduced to circumvent breaking changes.
func toRegionUpdatePayload(ctx context.Context, model *Model) (*iaas.UpdateNetworkAreaRegionPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	modelDefaultNameservers, err := toDefaultNameserversPayload(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("converting default nameservers: %w", err)
	}

	return &iaas.UpdateNetworkAreaRegionPayload{
		Ipv4: &iaas.UpdateRegionalAreaIPv4{
			DefaultNameservers: &modelDefaultNameservers,
			DefaultPrefixLen:   conversion.Int64ValueToPointer(model.DefaultPrefixLength),
			MaxPrefixLen:       conversion.Int64ValueToPointer(model.MaxPrefixLength),
			MinPrefixLen:       conversion.Int64ValueToPointer(model.MinPrefixLength),
		},
	}, nil
}

// Deprecated: toDefaultNameserversPayload will be removed in May 2026. Implementation won't be needed anymore because of the IaaS API v1 -> v2 migration. Func was only introduced to circumvent breaking changes.
func toDefaultNameserversPayload(_ context.Context, model *Model) ([]string, error) {
	modelDefaultNameservers := []string{}
	for _, ns := range model.DefaultNameservers.Elements() {
		nameserverString, ok := ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelDefaultNameservers = append(modelDefaultNameservers, nameserverString.ValueString())
	}

	return modelDefaultNameservers, nil
}

// Deprecated: toNetworkRangesPayload will be removed in May 2026. Implementation won't be needed anymore because of the IaaS API v1 -> v2 migration. Func was only introduced to circumvent breaking changes.
func toNetworkRangesPayload(ctx context.Context, model *Model) (*[]iaas.NetworkRange, error) {
	if model.NetworkRanges.IsNull() || model.NetworkRanges.IsUnknown() {
		return nil, nil
	}

	networkRangesModel := []networkRange{}
	diags := model.NetworkRanges.ElementsAs(ctx, &networkRangesModel, false)
	if diags.HasError() {
		return nil, core.DiagsToError(diags)
	}

	if len(networkRangesModel) == 0 {
		return nil, nil
	}

	payload := []iaas.NetworkRange{}
	for i := range networkRangesModel {
		networkRangeModel := networkRangesModel[i]
		payload = append(payload, iaas.NetworkRange{
			Prefix: conversion.StringValueToPointer(networkRangeModel.Prefix),
		})
	}

	return &payload, nil
}

// Deprecated: updateNetworkRanges creates and deletes network ranges so that network area ranges are the ones in the model. This was only kept to make the v1 -> v2 IaaS API migration non-breaking in the Terraform provider.
func updateNetworkRanges(ctx context.Context, organizationId, networkAreaId string, ranges []networkRange, client *iaas.APIClient) error {
	// Get network ranges current state
	currentNetworkRangesResp, err := client.ListNetworkAreaRanges(ctx, organizationId, networkAreaId, "eu01").Execute()
	if err != nil {
		return fmt.Errorf("error reading network area ranges: %w", err)
	}

	type networkRangeState struct {
		isInModel bool
		isCreated bool
		id        string
	}

	networkRangesState := make(map[string]*networkRangeState)
	for _, nwRange := range ranges {
		networkRangesState[nwRange.Prefix.ValueString()] = &networkRangeState{
			isInModel: true,
		}
	}

	for _, networkRange := range *currentNetworkRangesResp.Items {
		prefix := *networkRange.Prefix
		if _, ok := networkRangesState[prefix]; !ok {
			networkRangesState[prefix] = &networkRangeState{}
		}
		networkRangesState[prefix].isCreated = true
		networkRangesState[prefix].id = *networkRange.Id
	}

	// Delete network ranges
	for prefix, state := range networkRangesState {
		if !state.isInModel && state.isCreated {
			err := client.DeleteNetworkAreaRange(ctx, organizationId, networkAreaId, "eu01", state.id).Execute()
			if err != nil {
				return fmt.Errorf("deleting network area range '%v': %w", prefix, err)
			}
		}
	}

	// Create network ranges
	for prefix, state := range networkRangesState {
		if state.isInModel && !state.isCreated {
			payload := iaas.CreateNetworkAreaRangePayload{
				Ipv4: &[]iaas.NetworkRange{
					{
						Prefix: sdkUtils.Ptr(prefix),
					},
				},
			}

			_, err := client.CreateNetworkAreaRange(ctx, organizationId, networkAreaId, "eu01").CreateNetworkAreaRangePayload(payload).Execute()
			if err != nil {
				return fmt.Errorf("creating network range '%v': %w", prefix, err)
			}
		}
	}

	return nil
}
