package objectstorage

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	objectstorageUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/utils"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &credentialResource{}
	_ resource.ResourceWithConfigure   = &credentialResource{}
	_ resource.ResourceWithImportState = &credentialResource{}
	_ resource.ResourceWithModifyPlan  = &credentialResource{}
)

type Model struct {
	Id                  types.String `tfsdk:"id"` // needed by TF
	CredentialId        types.String `tfsdk:"credential_id"`
	CredentialsGroupId  types.String `tfsdk:"credentials_group_id"`
	ProjectId           types.String `tfsdk:"project_id"`
	Name                types.String `tfsdk:"name"`
	AccessKey           types.String `tfsdk:"access_key"`
	SecretAccessKey     types.String `tfsdk:"secret_access_key"`
	ExpirationTimestamp types.String `tfsdk:"expiration_timestamp"`
	Region              types.String `tfsdk:"region"`
}

// NewCredentialResource is a helper function to simplify the provider implementation.
func NewCredentialResource() resource.Resource {
	return &credentialResource{}
}

// credentialResource is the resource implementation.
type credentialResource struct {
	client       *objectstorage.APIClient
	providerData core.ProviderData
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
func (r *credentialResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	r.modifyPlanRegion(ctx, &req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	r.modifyPlanExpiration(ctx, &req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *credentialResource) modifyPlanRegion(ctx context.Context, req *resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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

// ModifyPlan implements resource.ResourceWithModifyPlan.
func (r *credentialResource) modifyPlanExpiration(ctx context.Context, req *resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	p := path.Root("expiration_timestamp")
	var (
		stateDate time.Time
		planDate  time.Time
	)

	resp.Diagnostics.Append(utils.GetTimeFromStringAttribute(ctx, p, req.State, time.RFC3339, &stateDate)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(utils.GetTimeFromStringAttribute(ctx, p, resp.Plan, time.RFC3339, &planDate)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// replace the planned expiration time with the current state date, iff they represent
	// the same point in time (but perhaps with different textual representation)
	// this will prevent no-op updates
	if stateDate.Equal(planDate) && !stateDate.IsZero() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p, types.StringValue(stateDate.Format(time.RFC3339)))...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

// Metadata returns the resource type name.
func (r *credentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_credential"
}

// Configure adds the provider configured client to the resource.
func (r *credentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := objectstorageUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "ObjectStorage credential client configured")
}

// Schema defines the schema for the resource.
func (r *credentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                 "ObjectStorage credential resource schema. Must have a `region` specified in the provider configuration.",
		"id":                   "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`credentials_group_id`,`credential_id`\".",
		"credential_id":        "The credential ID.",
		"credentials_group_id": "The credential group ID.",
		"project_id":           "STACKIT Project ID to which the credential group is associated.",
		"expiration_timestamp": "Expiration timestamp, in RFC339 format without fractional seconds. Example: \"2025-01-01T00:00:00Z\". If not set, the credential never expires.",
		"region":               "The resource region. If not defined, the provider region is used.",
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
			"credentials_group_id": schema.StringAttribute{
				Description: descriptions["credentials_group_id"],
				Required:    true,
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
				Computed: true,
			},
			"access_key": schema.StringAttribute{
				Computed: true,
			},
			"secret_access_key": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"expiration_timestamp": schema.StringAttribute{
				Description: descriptions["expiration_timestamp"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validate.RFC3339SecondsOnly(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
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
	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_group_id", credentialsGroupId)
	ctx = tflog.SetField(ctx, "region", region)

	// Handle project init
	err := enableProject(ctx, &model, region, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Enabling object storage project before creation: %v", err))
		return
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new credential
	credentialResp, err := r.client.CreateAccessKey(ctx, projectId, region).CredentialsGroup(credentialsGroupId).CreateAccessKeyPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if credentialResp.KeyId == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", "Got empty credential id")
		return
	}
	credentialId := *credentialResp.KeyId
	ctx = tflog.SetField(ctx, "credential_id", credentialId)

	// Map response body to schema
	err = mapFields(credentialResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	if !utils.IsUndefined(model.ExpirationTimestamp) {
		var (
			actualDate time.Time
			planDate   time.Time
		)
		resp.Diagnostics.Append(utils.ToTime(ctx, time.RFC3339, model.ExpirationTimestamp, &actualDate)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(utils.GetTimeFromStringAttribute(ctx, path.Root("expiration_timestamp"), req.Plan, time.RFC3339, &planDate)...)
		if resp.Diagnostics.HasError() {
			return
		}
		// replace the planned expiration date with the original date, iff
		// they represent the same point in time, (perhaps with different textual representations)
		if actualDate.Equal(planDate) {
			model.ExpirationTimestamp = types.StringValue(planDate.Format(time.RFC3339))
		}
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage credential created")
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	credentialId := model.CredentialId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_group_id", credentialsGroupId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)
	ctx = tflog.SetField(ctx, "region", region)

	found, err := readCredentials(ctx, &model, region, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Finding credential: %v", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	var (
		currentApiDate time.Time
		stateDate      time.Time
	)

	if !utils.IsUndefined(model.ExpirationTimestamp) {
		resp.Diagnostics.Append(utils.ToTime(ctx, time.RFC3339, model.ExpirationTimestamp, &currentApiDate)...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(utils.GetTimeFromStringAttribute(ctx, path.Root("expiration_timestamp"), req.State, time.RFC3339, &stateDate)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// replace the resulting expiration date with the original date, iff
		// they represent the same point in time, (perhaps with different textual representations)
		if currentApiDate.Equal(stateDate) {
			model.ExpirationTimestamp = types.StringValue(stateDate.Format(time.RFC3339))
		}
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage credential read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *credentialResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	/*
		While a credential cannot be updated, the Update call must not be prevented with an error:
		When the expiration timestamp has been updated to the same point in time, but e.g. with a different timezone,
		terraform will still trigger an Update due to the computed attributes. These will not change,
		but terraform has no way of knowing this without calling this function. So it is
		still updated as a no-op.
		A possible enhancement would be to emit an error, if it is attempted to change one of the not computed attributes
		and abort with an error in this case.
	*/
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *credentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	credentialId := model.CredentialId.ValueString()
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_group_id", credentialsGroupId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing credential
	_, err := r.client.DeleteAccessKey(ctx, projectId, region, credentialId).CredentialsGroup(credentialsGroupId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "ObjectStorage credential deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,credentials_group_id,credential_id
func (r *credentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing credential",
			fmt.Sprintf("Expected import identifier with format [project_id],[region],[credentials_group_id],[credential_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credentials_group_id"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credential_id"), idParts[3])...)
	tflog.Info(ctx, "ObjectStorage credential state imported")
}

type objectStorageClient interface {
	EnableServiceExecute(ctx context.Context, projectId, region string) (*objectstorage.ProjectStatus, error)
}

// enableProject enables object storage for the specified project. If the project is already enabled, nothing happens
func enableProject(ctx context.Context, model *Model, region string, client objectStorageClient) error {
	projectId := model.ProjectId.ValueString()

	// From the object storage OAS: Creation will also be successful if the project is already enabled, but will not create a duplicate
	_, err := client.EnableServiceExecute(ctx, projectId, region)
	if err != nil {
		return fmt.Errorf("failed to create object storage project: %w", err)
	}
	return nil
}

func toCreatePayload(model *Model) (*objectstorage.CreateAccessKeyPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	if model.ExpirationTimestamp.IsNull() || model.ExpirationTimestamp.IsUnknown() {
		return &objectstorage.CreateAccessKeyPayload{}, nil
	}

	expirationTimestampValue := conversion.StringValueToPointer(model.ExpirationTimestamp)
	if expirationTimestampValue == nil {
		return &objectstorage.CreateAccessKeyPayload{}, nil
	}
	expirationTimestamp, err := time.Parse(time.RFC3339, *expirationTimestampValue)
	if err != nil {
		return nil, fmt.Errorf("unable to parse expiration timestamp '%v': %w", *expirationTimestampValue, err)
	}
	return &objectstorage.CreateAccessKeyPayload{
		Expires: &expirationTimestamp,
	}, nil
}

func mapFields(credentialResp *objectstorage.CreateAccessKeyResponse, model *Model, region string) error {
	if credentialResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var credentialId string
	if model.CredentialId.ValueString() != "" {
		credentialId = model.CredentialId.ValueString()
	} else if credentialResp.KeyId != nil {
		credentialId = *credentialResp.KeyId
	} else {
		return fmt.Errorf("credential id not present")
	}

	if credentialResp.Expires == nil {
		model.ExpirationTimestamp = types.StringNull()
	} else {
		// Harmonize the timestamp format
		// Eg. "2027-01-02T03:04:05.000Z" = "2027-01-02T03:04:05Z"
		expirationTimestamp, err := time.Parse(time.RFC3339, *credentialResp.Expires)
		if err != nil {
			return fmt.Errorf("unable to parse payload expiration timestamp '%v': %w", *credentialResp.Expires, err)
		}
		model.ExpirationTimestamp = types.StringValue(expirationTimestamp.Format(time.RFC3339))
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), region, model.CredentialsGroupId.ValueString(), credentialId,
	)
	model.CredentialId = types.StringValue(credentialId)
	model.Name = types.StringPointerValue(credentialResp.DisplayName)
	model.AccessKey = types.StringPointerValue(credentialResp.AccessKey)
	model.SecretAccessKey = types.StringPointerValue(credentialResp.SecretAccessKey)
	model.Region = types.StringValue(region)
	return nil
}

// readCredentials gets all the existing credentials for the specified credentials group,
// finds the credential that is being read and updates the state.
// Returns True if the credential was found, False otherwise.
func readCredentials(ctx context.Context, model *Model, region string, client *objectstorage.APIClient) (bool, error) {
	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	credentialId := model.CredentialId.ValueString()

	credentialsGroupResp, err := client.ListAccessKeys(ctx, projectId, region).CredentialsGroup(credentialsGroupId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("getting credentials groups: %w", err)
	}
	if credentialsGroupResp == nil {
		return false, fmt.Errorf("getting credentials groups: nil response")
	}

	foundCredential := false
	for _, credential := range *credentialsGroupResp.AccessKeys {
		if credential.KeyId == nil || *credential.KeyId != credentialId {
			continue
		}

		foundCredential = true

		model.Id = utils.BuildInternalTerraformId(projectId, region, credentialsGroupId, credentialId)
		model.Name = types.StringPointerValue(credential.DisplayName)

		if credential.Expires == nil {
			model.ExpirationTimestamp = types.StringNull()
		} else {
			// Harmonize the timestamp format
			// Eg. "2027-01-02T03:04:05.000Z" = "2027-01-02T03:04:05Z"
			expirationTimestamp, err := time.Parse(time.RFC3339, *credential.Expires)
			if err != nil {
				return foundCredential, fmt.Errorf("unable to parse payload expiration timestamp '%v': %w", *credential.Expires, err)
			}
			model.ExpirationTimestamp = types.StringValue(expirationTimestamp.Format(time.RFC3339))
		}
		break
	}
	model.Region = types.StringValue(region)

	return foundCredential, nil
}
