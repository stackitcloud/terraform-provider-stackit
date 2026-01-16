// Copyright (c) STACKIT

package sqlserverflex

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	sqlserverflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/sqlserverflexalpha/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sqlserverflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha/wait"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
	_ resource.ResourceWithModifyPlan  = &instanceResource{}
)

//nolint:unused // TODO: remove if not needed later
var validNodeTypes []string = []string{
	"Single",
	"Replica",
}

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	InstanceId     types.String `tfsdk:"instance_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	BackupSchedule types.String `tfsdk:"backup_schedule"`
	FlavorId       types.String `tfsdk:"flavor_id"`
	Encryption     types.Object `tfsdk:"encryption"`
	IsDeletable    types.Bool   `tfsdk:"is_deletable"`
	Storage        types.Object `tfsdk:"storage"`
	Status         types.String `tfsdk:"status"`
	Version        types.String `tfsdk:"version"`
	Replicas       types.Int64  `tfsdk:"replicas"`
	Region         types.String `tfsdk:"region"`
	Network        types.Object `tfsdk:"network"`
	Edition        types.String `tfsdk:"edition"`
	RetentionDays  types.Int64  `tfsdk:"retention_days"`
}

type encryptionModel struct {
	KeyRingId      types.String `tfsdk:"keyring_id"`
	KeyId          types.String `tfsdk:"key_id"`
	KeyVersion     types.String `tfsdk:"key_version"`
	ServiceAccount types.String `tfsdk:"service_account"`
}

var encryptionTypes = map[string]attr.Type{
	"keyring_id":      basetypes.StringType{},
	"key_id":          basetypes.StringType{},
	"key_version":     basetypes.StringType{},
	"service_account": basetypes.StringType{},
}

type networkModel struct {
	ACL             types.List   `tfsdk:"acl"`
	AccessScope     types.String `tfsdk:"access_scope"`
	InstanceAddress types.String `tfsdk:"instance_address"`
	RouterAddress   types.String `tfsdk:"router_address"`
}

var networkTypes = map[string]attr.Type{
	"acl":              basetypes.ListType{ElemType: types.StringType},
	"access_scope":     basetypes.StringType{},
	"instance_address": basetypes.StringType{},
	"router_address":   basetypes.StringType{},
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
	resp.TypeName = req.ProviderTypeName + "_sqlserverflexalpha_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
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
func (r *instanceResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) { // nolint:gocritic // function signature required by Terraform
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

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":             "SQLServer Flex ALPHA instance resource schema. Must have a `region` specified in the provider configuration.",
		"id":               "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
		"instance_id":      "ID of the SQLServer Flex instance.",
		"project_id":       "STACKIT project ID to which the instance is associated.",
		"name":             "Instance name.",
		"access_scope":     "The access scope of the instance. (SNA | PUBLIC)",
		"flavor_id":        "The flavor ID of the instance.",
		"acl":              "The Access Control List (ACL) for the SQLServer Flex instance.",
		"backup_schedule":  `The backup schedule. Should follow the cron scheduling system format (e.g. "0 0 * * *")`,
		"region":           "The resource region. If not defined, the provider region is used.",
		"encryption":       "The encryption block.",
		"replicas":         "The number of replicas of the SQLServer Flex instance.",
		"network":          "The network block.",
		"keyring_id":       "STACKIT KMS - KeyRing ID of the encryption key to use.",
		"key_id":           "STACKIT KMS - Key ID of the encryption key to use.",
		"key_version":      "STACKIT KMS - Key version to use in the encryption key.",
		"service:account":  "STACKIT KMS - service account to use in the encryption key.",
		"instance_address": "The returned instance IP address of the SQLServer Flex instance.",
		"router_address":   "The returned router IP address of the SQLServer Flex instance.",
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
			"backup_schedule": schema.StringAttribute{
				Description: descriptions["backup_schedule"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_deletable": schema.BoolAttribute{
				Description: descriptions["is_deletable"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"flavor_id": schema.StringAttribute{
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Required: true,
			},
			"replicas": schema.Int64Attribute{
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"storage": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"class": schema.StringAttribute{
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"size": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"version": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"edition": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"retention_days": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
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
			"status": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["status"],
			},
			"encryption": schema.SingleNestedAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"key_id": schema.StringAttribute{
						Description: descriptions["key_id"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
					"key_version": schema.StringAttribute{
						Description: descriptions["key_version"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
					"keyring_id": schema.StringAttribute{
						Description: descriptions["keyring_id"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
					"service_account": schema.StringAttribute{
						Description: descriptions["service_account"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
				},
				Description: descriptions["encryption"],
			},
			"network": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"access_scope": schema.StringAttribute{
						Description: descriptions["access_scope"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
					"acl": schema.ListAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
						Required:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"instance_address": schema.StringAttribute{
						Description: descriptions["instance_address"],
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"router_address": schema.StringAttribute{
						Description: descriptions["router_address"],
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
				Description: descriptions["network"],
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) { // nolint:gocritic // function signature required by Terraform
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

	var storage = &storageModel{}
	if !model.Storage.IsNull() && !model.Storage.IsUnknown() {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var encryption = &encryptionModel{}
	if !model.Encryption.IsNull() && !model.Encryption.IsUnknown() {
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var network = &networkModel{}
	if !model.Network.IsNull() && !model.Network.IsUnknown() {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, storage, encryption, network)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating instance",
			fmt.Sprintf("Creating API payload: %v", err),
		)
		return
	}
	// Create new instance
	createResp, err := r.client.CreateInstanceRequest(
		ctx,
		projectId,
		region,
	).CreateInstanceRequestPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	instanceId := *createResp.Id
	utils.SetAndLogStateFields(
		ctx, &resp.Diagnostics, &resp.State, map[string]any{
			"id":          utils.BuildInternalTerraformId(projectId, region, instanceId),
			"instance_id": instanceId,
		},
	)
	if resp.Diagnostics.HasError() {
		return
	}

	// The creation waiter sometimes returns an error from the API: "instance with id xxx has unexpected status Failure"
	// which can be avoided by sleeping before wait
	waitResp, err := wait.CreateInstanceWaitHandler(
		ctx,
		r.client,
		projectId,
		instanceId,
		region,
	).SetSleepBeforeWait(30 * time.Second).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating instance",
			fmt.Sprintf("Instance creation waiting: %v", err),
		)
		return
	}

	if waitResp.FlavorId == nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating instance",
			"Instance creation waiting: returned flavor id is nil",
		)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, storage, encryption, network, region)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error creating instance",
			fmt.Sprintf("Processing API payload: %v", err),
		)
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// After the instance creation, database might not be ready to accept connections immediately.
	// That is why we add a sleep
	// TODO - can get removed?
	time.Sleep(120 * time.Second)

	tflog.Info(ctx, "SQLServer Flex instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	var storage = &storageModel{}
	if !model.Storage.IsNull() && !model.Storage.IsUnknown() {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var encryption = &encryptionModel{}
	if !model.Encryption.IsNull() && !model.Encryption.IsUnknown() {
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var network = &networkModel{}
	if !model.Network.IsNull() && !model.Network.IsUnknown() {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	instanceResp, err := r.client.GetInstanceRequest(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, instanceResp, &model, storage, encryption, network, region)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error reading instance",
			fmt.Sprintf("Processing API payload: %v", err),
		)
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
func (r *instanceResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) { // nolint:gocritic // function signature required by Terraform
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

	var storage = &storageModel{}
	if !model.Storage.IsNull() && !model.Storage.IsUnknown() {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var encryption = &encryptionModel{}
	if !model.Encryption.IsNull() && !model.Encryption.IsUnknown() {
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var network = &networkModel{}
	if !model.Network.IsNull() && !model.Network.IsUnknown() {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model, storage, network)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error updating instance",
			fmt.Sprintf("Creating API payload: %v", err),
		)
		return
	}
	// Update existing instance
	err = r.client.UpdateInstanceRequest(
		ctx,
		projectId,
		region,
		instanceId,
	).UpdateInstanceRequestPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.UpdateInstanceWaitHandler(ctx, r.client, projectId, instanceId, region).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error updating instance",
			fmt.Sprintf("Instance update waiting: %v", err),
		)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, storage, encryption, network, region)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error updating instance",
			fmt.Sprintf("Processing API payload: %v", err),
		)
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
func (r *instanceResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) { // nolint:gocritic // function signature required by Terraform
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
	err := r.client.DeleteInstanceRequest(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client, projectId, instanceId, region).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(
			ctx,
			&resp.Diagnostics,
			"Error deleting instance",
			fmt.Sprintf("Instance deletion waiting: %v", err),
		)
		return
	}
	tflog.Info(ctx, "SQLServer Flex instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(
			ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	tflog.Info(ctx, "SQLServer Flex instance state imported")
}
