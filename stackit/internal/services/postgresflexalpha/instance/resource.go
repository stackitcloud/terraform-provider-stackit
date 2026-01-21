package postgresflexalpha

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	postgresflexalpha "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/instance/resources_gen"
	postgresflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/utils"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
	wait "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/wait/postgresflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &instanceResource{}
	_ resource.ResourceWithConfigure      = &instanceResource{}
	_ resource.ResourceWithImportState    = &instanceResource{}
	_ resource.ResourceWithModifyPlan     = &instanceResource{}
	_ resource.ResourceWithValidateConfig = &instanceResource{}
	// _ resource.ResourceWithIdentity       = &instanceResource{}
)

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client       *postgresflex.APIClient
	providerData core.ProviderData
}

func (r *instanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data postgresflexalpha.InstanceModel
	// var data Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Replicas.IsNull() || data.Replicas.IsUnknown() {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("replicas"),
			"Missing Attribute Configuration",
			"Expected replicas to be configured. "+
				"The resource may return unexpected results.",
		)
	}
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel postgresflexalpha.InstanceModel
	// var configModel Model
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel postgresflexalpha.InstanceModel
	// var planModel Model
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

	apiClient := postgresflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Postgres Flex instance client configured")
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = postgresflexalpha.InstanceResourceSchema(ctx)
	resp.Schema = addPlanModifiers(resp.Schema)
}

func addPlanModifiers(s schema.Schema) schema.Schema {
	attr := s.Attributes["backup_schedule"].(schema.StringAttribute)
	attr.PlanModifiers = []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),
	}
	s.Attributes["backup_schedule"] = attr
	return s
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) { // nolint:gocritic // function signature required by Terraform
	var model postgresflexalpha.InstanceModel
	//var model Model
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

	var netAcl []string
	diag := model.Network.Acl.ElementsAs(ctx, &netAcl, false)
	resp.Diagnostics.Append(diags...)
	if diag.HasError() {
		return
	}

	if model.Replicas.ValueInt64() > math.MaxInt32 {
		resp.Diagnostics.AddError("invalid int32 value", "provided int64 value does not fit into int32")
		return
	}
	replVal := int32(model.Replicas.ValueInt64()) // nolint:gosec // check is performed above
	payload := modelToCreateInstancePayload(netAcl, model, replVal)

	// Create new instance
	createResp, err := r.client.CreateInstanceRequest(ctx, projectId, region).CreateInstanceRequestPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	instanceId := *createResp.Id

	model.InstanceId = types.StringValue(instanceId)
	model.Id = utils.BuildInternalTerraformId(projectId, region, instanceId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Wait handler error: %v", err))
		return
	}

	err = mapGetInstanceResponseToModel(ctx, &model, waitResp)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Error creating model: %v", err))
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

func modelToCreateInstancePayload(netAcl []string, model postgresflexalpha.InstanceModel, replVal int32) postgresflex.CreateInstanceRequestPayload {
	payload := postgresflex.CreateInstanceRequestPayload{
		// Acl:            &netAcl,
		BackupSchedule: model.BackupSchedule.ValueStringPointer(),
		Encryption: &postgresflex.InstanceEncryption{
			KekKeyId:       model.Encryption.KekKeyId.ValueStringPointer(),
			KekKeyRingId:   model.Encryption.KekKeyRingId.ValueStringPointer(),
			KekKeyVersion:  model.Encryption.KekKeyVersion.ValueStringPointer(),
			ServiceAccount: model.Encryption.ServiceAccount.ValueStringPointer(),
		},
		FlavorId: model.FlavorId.ValueStringPointer(),
		Name:     model.Name.ValueStringPointer(),
		Network: &postgresflex.InstanceNetwork{
			AccessScope: postgresflex.InstanceNetworkGetAccessScopeAttributeType(
				model.Network.AccessScope.ValueStringPointer(),
			),
			Acl:             &netAcl,
			InstanceAddress: model.Network.InstanceAddress.ValueStringPointer(),
			RouterAddress:   model.Network.RouterAddress.ValueStringPointer(),
		},
		Replicas:      postgresflex.CreateInstanceRequestPayloadGetReplicasAttributeType(&replVal),
		RetentionDays: model.RetentionDays.ValueInt64Pointer(),
		Storage: &postgresflex.StorageCreate{
			PerformanceClass: model.Storage.PerformanceClass.ValueStringPointer(),
			Size:             model.Storage.Size.ValueInt64Pointer(),
		},
		Version: model.Version.ValueStringPointer(),
	}
	return payload
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model postgresflexalpha.InstanceModel
	//var model Model
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

	err = mapGetInstanceResponseToModel(ctx, &model, instanceResp)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Postgres Flex instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model postgresflexalpha.InstanceModel
	//var model Model
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

	var netAcl []string
	diag := model.Network.Acl.ElementsAs(ctx, &netAcl, false)
	resp.Diagnostics.Append(diags...)
	if diag.HasError() {
		return
	}
	replInt32 := int32(model.Replicas.ValueInt64())
	payload := postgresflex.UpdateInstancePartiallyRequestPayload{
		BackupSchedule: model.BackupSchedule.ValueStringPointer(),
		FlavorId:       model.FlavorId.ValueStringPointer(),
		Name:           model.Name.ValueStringPointer(),
		Network: &postgresflex.InstanceNetwork{
			AccessScope: postgresflex.InstanceNetworkGetAccessScopeAttributeType(
				model.Network.AccessScope.ValueStringPointer(),
			),
			Acl: &netAcl,
		},
		Replicas:      postgresflex.UpdateInstancePartiallyRequestPayloadGetReplicasAttributeType(&replInt32),
		RetentionDays: model.RetentionDays.ValueInt64Pointer(),
		Storage: &postgresflex.StorageUpdate{
			Size: model.Storage.Size.ValueInt64Pointer(),
		},
		Version: model.Version.ValueStringPointer(),
	}

	// Update existing instance
	err := r.client.UpdateInstancePartiallyRequest(
		ctx,
		projectId,
		region,
		instanceId,
	).UpdateInstancePartiallyRequestPayload(payload).Execute()
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

	err = mapGetInstanceResponseToModel(ctx, &model, waitResp)
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
	var model postgresflexalpha.InstanceModel
	//var model Model
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

	_, err = r.client.GetInstanceRequest(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode != http.StatusNotFound {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", err.Error())
			return
		}
	}

	resp.State.RemoveResource(ctx)
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
