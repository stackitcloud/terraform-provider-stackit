package mariadb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	mariadbUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/mariadb/utils"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	mariadb "github.com/stackitcloud/stackit-sdk-go/services/mariadb/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/mariadb/v2api/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &credentialResource{}
	_ resource.ResourceWithConfigure   = &credentialResource{}
	_ resource.ResourceWithImportState = &credentialResource{}
	_ resource.ResourceWithModifyPlan  = &credentialResource{}
)

type Model struct {
	Id           types.String `tfsdk:"id"` // needed by TF
	CredentialId types.String `tfsdk:"credential_id"`
	InstanceId   types.String `tfsdk:"instance_id"`
	ProjectId    types.String `tfsdk:"project_id"`
	Host         types.String `tfsdk:"host"`
	Hosts        types.List   `tfsdk:"hosts"`
	Name         types.String `tfsdk:"name"`
	Password     types.String `tfsdk:"password"`
	Port         types.Int32  `tfsdk:"port"`
	Uri          types.String `tfsdk:"uri"`
	Username     types.String `tfsdk:"username"`
	// RotateWhenChanged is a map of arbitrary key/value pairs that will force
	// recreation of the resource when they change, enabling resource rotation based on
	// external conditions such as a rotating timestamp. Changing this forces a new
	// resource to be created.
	RotateWhenChanged types.Map    `tfsdk:"rotate_when_changed"`
	Region            types.String `tfsdk:"region"`
}

// NewCredentialResource is a helper function to simplify the provider implementation.
func NewCredentialResource() resource.Resource {
	return &credentialResource{}
}

// credentialResource is the resource implementation.
type credentialResource struct {
	client       *mariadb.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *credentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mariadb_credential"
}

// Configure adds the provider configured client to the resource.
func (r *credentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := mariadbUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "MariaDB credential client configured")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *credentialResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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
func (r *credentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{ //nolint:gosec // description for credential id
		"main":          "MariaDB credential resource schema. Must have a `region` specified in the provider configuration.",
		"id":            "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`,`credential_id`\".",
		"credential_id": "The credential's ID.",
		"instance_id":   "ID of the MariaDB instance.",
		"project_id":    "STACKIT Project ID to which the instance is associated.",
		"region":        "The resource region. If not defined, the provider region is used.",
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
			"credential_id": schema.StringAttribute{
				Description: descriptions["credential_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
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
			"host": schema.StringAttribute{
				Computed: true,
			},
			"hosts": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"password": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"port": schema.Int32Attribute{
				Computed: true,
			},
			"uri": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"username": schema.StringAttribute{
				Computed: true,
			},
			"rotate_when_changed": schema.MapAttribute{
				Description: "A map of arbitrary key/value pairs that will force " +
					"recreation of the resource when they change, enabling resource rotation " +
					"based on external conditions such as a rotating timestamp. Changing " +
					"this forces a new resource to be created.",
				Optional:    true,
				Required:    false,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
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
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *credentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// Create new recordset
	credentialsResp, err := r.client.DefaultAPI.CreateCredentials(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	credentialId := credentialsResp.Id
	// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":    projectId,
		"region":        region,
		"instance_id":   instanceId,
		"credential_id": credentialId,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	waitResp, err := wait.CreateCredentialsWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId, credentialId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "MariaDB credential created")
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	credentialId := model.CredentialId.ValueString()
	if credentialId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)

	recordSetResp, err := r.client.DefaultAPI.GetCredentials(ctx, projectId, region, instanceId, credentialId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, recordSetResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "MariaDB credential read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *credentialResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating credential", "Credential can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *credentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	credentialId := model.CredentialId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)

	// Delete existing record set
	err := r.client.DefaultAPI.DeleteCredentials(ctx, projectId, region, instanceId, credentialId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteCredentialsWaitHandler(ctx, r.client.DefaultAPI, projectId, region, instanceId, credentialId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "MariaDB credential deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,region,instance_id,credential_id
func (r *credentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing credential",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[instance_id],[credential_id], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":    idParts[0],
		"region":        idParts[1],
		"instance_id":   idParts[2],
		"credential_id": idParts[3],
	})
	tflog.Info(ctx, "MariaDB credential state imported")
}

func mapFields(ctx context.Context, credentialsResp *mariadb.CredentialsResponse, model *Model, region string) error {
	if credentialsResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if credentialsResp.Raw == nil {
		return fmt.Errorf("response credentials raw is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	credentials := credentialsResp.Raw.Credentials

	var credentialId string
	if model.CredentialId.ValueString() != "" {
		credentialId = model.CredentialId.ValueString()
	} else if credentialsResp.Id != "" {
		credentialId = credentialsResp.Id
	} else {
		return fmt.Errorf("credentials id not present")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		model.InstanceId.ValueString(),
		credentialId,
	)
	model.Region = types.StringValue(region)

	modelHosts, err := utils.ListValueToStringSlice(model.Hosts)
	if err != nil {
		return err
	}

	model.Hosts = types.ListNull(types.StringType)
	model.CredentialId = types.StringValue(credentialId)

	if credentials.Hosts != nil {
		respHosts := credentials.Hosts

		reconciledHosts := utils.ReconcileStringSlices(modelHosts, respHosts)

		hostsTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledHosts)
		if diags.HasError() {
			return fmt.Errorf("failed to map hosts: %w", core.DiagsToError(diags))
		}

		model.Hosts = hostsTF
	}
	model.Host = types.StringValue(credentials.Host)
	model.Name = types.StringPointerValue(credentials.Name)
	model.Password = types.StringValue(credentials.Password)
	model.Port = types.Int32PointerValue(credentials.Port)
	model.Uri = types.StringPointerValue(credentials.Uri)
	model.Username = types.StringValue(credentials.Username)

	return nil
}
