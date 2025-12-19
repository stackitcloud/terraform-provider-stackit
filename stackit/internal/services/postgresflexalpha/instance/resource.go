package postgresflex

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha/wait"
	postgresflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
	_ resource.ResourceWithModifyPlan  = &instanceResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	InstanceId     types.String `tfsdk:"instance_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	ACL            types.List   `tfsdk:"acl"`
	BackupSchedule types.String `tfsdk:"backup_schedule"`
	Flavor         types.Object `tfsdk:"flavor"`
	Replicas       types.Int64  `tfsdk:"replicas"`
	Storage        types.Object `tfsdk:"storage"`
	Version        types.String `tfsdk:"version"`
	Region         types.String `tfsdk:"region"`
	Encryption     types.Object `tfsdk:"encryption"`
	Network        types.Object `tfsdk:"network"`
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
	AccessScope types.String `tfsdk:"access_scope"`
}

var networkTypes = map[string]attr.Type{
	"access_scope": basetypes.StringType{},
}

// Struct corresponding to Model.Flavor
type flavorModel struct {
	Id          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	CPU         types.Int64  `tfsdk:"cpu"`
	RAM         types.Int64  `tfsdk:"ram"`
}

// Types corresponding to flavorModel
var flavorTypes = map[string]attr.Type{
	"id":          basetypes.StringType{},
	"description": basetypes.StringType{},
	"cpu":         basetypes.Int64Type{},
	"ram":         basetypes.Int64Type{},
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
	client       *postgresflexalpha.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresflexalpha_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := postgresflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Postgres Flex instance client configured")
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "Postgres Flex instance resource schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
		"instance_id": "ID of the PostgresFlex instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"name":        "Instance name.",
		"acl":         "The Access Control List (ACL) for the PostgresFlex instance.",
		"region":      "The resource region. If not defined, the provider region is used.",
		"encryption":  "The encryption block.",
		"key_id":      "Key ID of the encryption key.",
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
					stringplanmodifier.UseStateForUnknown(),
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
			"acl": schema.ListAttribute{
				Description: descriptions["acl"],
				ElementType: types.StringType,
				Required:    true,
			},
			"backup_schedule": schema.StringAttribute{
				Required: true,
			},
			"flavor": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							UseStateForUnknownIfFlavorUnchanged(req),
						},
					},
					"description": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							UseStateForUnknownIfFlavorUnchanged(req),
						},
					},
					"cpu": schema.Int64Attribute{
						Required: true,
					},
					"ram": schema.Int64Attribute{
						Required: true,
					},
				},
			},
			"replicas": schema.Int64Attribute{
				Required: true,
			},
			"storage": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"class": schema.StringAttribute{
						Required: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"size": schema.Int64Attribute{
						Required: true,
					},
				},
			},
			"version": schema.StringAttribute{
				Required: true,
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
			"encryption": schema.SingleNestedAttribute{
				Required: true,
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
					"key_ring_id": schema.StringAttribute{
						Description: descriptions["key_ring_id"],
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
				//Blocks:              nil,
				//CustomType:          nil,
				Description: descriptions["encryption"],
				//MarkdownDescription: "",
				//DeprecationMessage:  "",
				//Validators:          nil,
				PlanModifiers: []planmodifier.Object{},
			},
			"network": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"access_scope": schema.StringAttribute{
						Description: descriptions["access_scope"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
				},
				//Blocks:              nil,
				//CustomType:          nil,
				Description: descriptions["network"],
				//MarkdownDescription: "",
				//DeprecationMessage:  "",
				//Validators:          nil,
				PlanModifiers: []planmodifier.Object{},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
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

	var acl []string
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client, &model, flavor)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var encryption = &encryptionModel{}
	if !(model.Encryption.IsNull() || model.Encryption.IsUnknown()) {
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var network = &networkModel{}
	if !(model.Network.IsNull() || model.Network.IsUnknown()) {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, acl, flavor, storage, encryption, network)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new instance
	createResp, err := r.client.CreateInstanceRequest(ctx, projectId, region).CreateInstanceRequestPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	instanceId := *createResp.Id
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, storage, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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

	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
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

	if instanceResp != nil && instanceResp.Status != nil && *instanceResp.Status == wait.InstanceStateDeleted {
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, instanceResp, &model, flavor, storage, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgres Flex instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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

	var acl []string
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client, &model, flavor)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model, acl, flavor, storage)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing instance
	err = r.client.UpdateInstancePartiallyRequest(ctx, projectId, region, instanceId).UpdateInstancePartiallyRequestPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.PartialUpdateInstanceWaitHandler(ctx, r.client, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, storage, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgresflex instance updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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

	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client, projectId, region, instanceId).SetTimeout(45 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "Postgres Flex instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	tflog.Info(ctx, "Postgres Flex instance state imported")
}

func mapFields(ctx context.Context, resp *postgresflexalpha.GetInstanceResponse, model *Model, flavor *flavorModel, storage *storageModel, region string) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	instance := resp

	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if instance.Id != nil {
		instanceId = *instance.Id
	} else {
		return fmt.Errorf("instance id not present")
	}

	var aclList basetypes.ListValue
	var diags diag.Diagnostics
	if instance.Acl == nil {
		aclList = types.ListNull(types.StringType)
	} else {
		respACL := *instance.Acl
		modelACL, err := utils.ListValuetoStringSlice(model.ACL)
		if err != nil {
			return err
		}

		reconciledACL := utils.ReconcileStringSlices(modelACL, respACL)

		aclList, diags = types.ListValueFrom(ctx, types.StringType, reconciledACL)
		if diags.HasError() {
			return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
		}
	}

	var flavorValues map[string]attr.Value
	if instance.FlavorId == nil {
		flavorValues = map[string]attr.Value{
			"id":          flavor.Id,
			"description": flavor.Description,
			"cpu":         flavor.CPU,
			"ram":         flavor.RAM,
		}
	} else {
		// TODO
		//flavorValues = map[string]attr.Value{
		//	"id":          types.StringValue(*instance.FlavorId),
		//	"description": types.StringValue(*instance.FlavorId.Description),
		//	"cpu":         types.Int64PointerValue(instance.FlavorId.Cpu),
		//	"ram":         types.Int64PointerValue(instance.FlavorId.Memory),
		//}
	}
	flavorObject, diags := types.ObjectValue(flavorTypes, flavorValues)
	if diags.HasError() {
		return fmt.Errorf("creating flavor: %w", core.DiagsToError(diags))
	}

	var storageValues map[string]attr.Value
	if instance.Storage == nil {
		storageValues = map[string]attr.Value{
			"class": storage.Class,
			"size":  storage.Size,
		}
	} else {
		storageValues = map[string]attr.Value{
			"class": types.StringValue(*instance.Storage.PerformanceClass),
			"size":  types.Int64PointerValue(instance.Storage.Size),
		}
	}
	storageObject, diags := types.ObjectValue(storageTypes, storageValues)
	if diags.HasError() {
		return fmt.Errorf("creating storage: %w", core.DiagsToError(diags))
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instanceId)
	model.InstanceId = types.StringValue(instanceId)
	model.Name = types.StringPointerValue(instance.Name)
	model.ACL = aclList
	model.BackupSchedule = types.StringPointerValue(instance.BackupSchedule)
	model.Flavor = flavorObject
	// TODO - verify working
	model.Replicas = types.Int64Value(int64(*instance.Replicas))
	model.Storage = storageObject
	model.Version = types.StringPointerValue(instance.Version)
	model.Region = types.StringValue(region)
	//model.Encryption = types.ObjectValue()
	//model.Network = networkModel
	return nil
}

func toCreatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel, enc *encryptionModel, net *networkModel) (*postgresflexalpha.CreateInstanceRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if acl == nil {
		return nil, fmt.Errorf("nil acl")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	replVal := int32(model.Replicas.ValueInt64())
	return &postgresflexalpha.CreateInstanceRequestPayload{
		// TODO - verify working
		Acl: &[]string{
			strings.Join(acl, ","),
		},
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		// TODO - verify working
		Replicas: postgresflexalpha.CreateInstanceRequestPayloadGetReplicasAttributeType(&replVal),
		// TODO - verify working
		Storage: postgresflexalpha.CreateInstanceRequestPayloadGetStorageAttributeType(&postgresflexalpha.Storage{
			PerformanceClass: conversion.StringValueToPointer(storage.Class),
			Size:             conversion.Int64ValueToPointer(storage.Size),
		}),
		Version: conversion.StringValueToPointer(model.Version),
		// TODO - verify working
		Encryption: postgresflexalpha.CreateInstanceRequestPayloadGetEncryptionAttributeType(
			&postgresflexalpha.InstanceEncryption{
				KekKeyId:       conversion.StringValueToPointer(enc.KeyId), // model.Encryption.Attributes(),
				KekKeyRingId:   conversion.StringValueToPointer(enc.KeyRingId),
				KekKeyVersion:  conversion.StringValueToPointer(enc.KeyVersion),
				ServiceAccount: conversion.StringValueToPointer(enc.ServiceAccount),
			},
		),
		Network: &postgresflexalpha.InstanceNetwork{
			AccessScope: postgresflexalpha.InstanceNetworkGetAccessScopeAttributeType(
				conversion.StringValueToPointer(net.AccessScope),
			),
		},
	}, nil
}

func toUpdatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel) (*postgresflexalpha.UpdateInstancePartiallyRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if acl == nil {
		return nil, fmt.Errorf("nil acl")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	return &postgresflexalpha.UpdateInstancePartiallyRequestPayload{
		//Acl: postgresflexalpha.UpdateInstancePartiallyRequestPayloadGetAclAttributeType{
		//	Items: &acl,
		//},
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		//Replicas:       conversion.Int64ValueToPointer(model.Replicas),
		Storage: &postgresflexalpha.StorageUpdate{
			Size: conversion.Int64ValueToPointer(storage.Size),
		},
		Version: conversion.StringValueToPointer(model.Version),
	}, nil
}

type postgresflexClient interface {
	GetFlavorsRequestExecute(ctx context.Context, projectId string, region string) (*postgresflexalpha.GetFlavorsResponse, error)
}

func loadFlavorId(ctx context.Context, client postgresflexClient, model *Model, flavor *flavorModel) error {
	if model == nil {
		return fmt.Errorf("nil model")
	}
	if flavor == nil {
		return fmt.Errorf("nil flavor")
	}
	cpu := conversion.Int64ValueToPointer(flavor.CPU)
	if cpu == nil {
		return fmt.Errorf("nil CPU")
	}
	ram := conversion.Int64ValueToPointer(flavor.RAM)
	if ram == nil {
		return fmt.Errorf("nil RAM")
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	res, err := client.GetFlavorsRequestExecute(ctx, projectId, region)
	if err != nil {
		return fmt.Errorf("listing postgresflex flavors: %w", err)
	}

	avl := ""
	if res.Flavors == nil {
		return fmt.Errorf("finding flavors for project %s", projectId)
	}
	for _, f := range *res.Flavors {
		if f.Id == nil || f.Cpu == nil || f.Memory == nil {
			continue
		}
		if *f.Cpu == *cpu && *f.Memory == *ram {
			flavor.Id = types.StringValue(*f.Id)
			flavor.Description = types.StringValue(*f.Description)
			break
		}
		avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM", avl, *f.Cpu, *f.Memory)
	}
	if flavor.Id.ValueString() == "" {
		return fmt.Errorf("couldn't find flavor, available specs are:%s", avl)
	}

	return nil
}
