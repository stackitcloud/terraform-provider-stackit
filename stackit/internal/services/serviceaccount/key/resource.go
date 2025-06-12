package key

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	serviceaccountUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &serviceAccountKeyResource{}
	_ resource.ResourceWithConfigure = &serviceAccountKeyResource{}
)

// Model represents the schema for the service account key resource in Terraform.
type Model struct {
	Id                  types.String `tfsdk:"id"`
	KeyId               types.String `tfsdk:"key_id"`
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
	ProjectId           types.String `tfsdk:"project_id"`
	RotateWhenChanged   types.Map    `tfsdk:"rotate_when_changed"`
	TtlDays             types.Int64  `tfsdk:"ttl_days"`
	PublicKey           types.String `tfsdk:"public_key"`
	Json                types.String `tfsdk:"json"`
}

// NewServiceAccountKeyResource is a helper function to create a new service account key resource instance.
func NewServiceAccountKeyResource() resource.Resource {
	return &serviceAccountKeyResource{}
}

// serviceAccountKeyResource implements the resource interface for service account key.
type serviceAccountKeyResource struct {
	client *serviceaccount.APIClient
}

// Configure sets up the API client for the service account resource.
func (r *serviceAccountKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := serviceaccountUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

// Metadata sets the resource type name for the service account key resource.
func (r *serviceAccountKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_key"
}

// Schema defines the resource schema for the service account access key.
func (r *serviceAccountKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"id":                    "Terraform's internal resource identifier. It is structured as \"`project_id`,`service_account_email`,`key_id`\".",
		"main":                  "Service account key schema.",
		"project_id":            "The STACKIT project ID associated with the service account key.",
		"key_id":                "The unique identifier for the key associated with the service account.",
		"service_account_email": "The email address associated with the service account, used for account identification and communication.",
		"ttl_days":              "Specifies the key's validity duration in days. If left unspecified, the key is considered valid until it is deleted",
		"rotate_when_changed":   "A map of arbitrary key/value pairs designed to force key recreation when they change, facilitating key rotation based on external factors such as a changing timestamp. Modifying this map triggers the creation of a new resource.",
		"public_key":            "Specifies the public_key (RSA2048 key-pair). If not provided, a certificate from STACKIT will be used to generate a private_key.",
		"json":                  "The raw JSON representation of the service account key json, available for direct use.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("%s%s", descriptions["main"], markdownDescription),
		Description:         descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_account_email": schema.StringAttribute{
				Description: descriptions["service_account_email"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key": schema.StringAttribute{
				Description: descriptions["public_key"],
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl_days": schema.Int64Attribute{
				Description: descriptions["ttl_days"],
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"rotate_when_changed": schema.MapAttribute{
				Description: descriptions["rotate_when_changed"],
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"key_id": schema.StringAttribute{
				Description: descriptions["key_id"],
				Computed:    true,
			},
			"json": schema.StringAttribute{
				Description: descriptions["json"],
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state for service accounts.
func (r *serviceAccountKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the planned values for the resource.
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set logging context with the project ID and service account email.
	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)

	if utils.IsUndefined(model.TtlDays) {
		model.TtlDays = types.Int64Null()
	}

	// Generate the API request payload.
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account key", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Initialize the API request with the required parameters.
	saAccountKeyResp, err := r.client.CreateServiceAccountKey(ctx, projectId, serviceAccountEmail).CreateServiceAccountKeyPayload(*payload).Execute()

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Failed to create service account key", fmt.Sprintf("API call error: %v", err))
		return
	}

	// Map the response to the resource schema.
	err = mapCreateResponse(saAccountKeyResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Service account key created")
}

// Read refreshes the Terraform state with the latest service account data.
func (r *serviceAccountKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	keyId := model.KeyId.ValueString()

	_, err := r.client.GetServiceAccountKey(ctx, projectId, serviceAccountEmail, keyId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		// due to security purposes, attempting to get access key for a non-existent Service Account will return 403.
		if ok && oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusForbidden || oapiErr.StatusCode == http.StatusBadRequest {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// No mapping needed for read response, as private_key is excluded and ID remains unchanged.
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "key read")
}

// Update attempts to update the resource. In this case, service account key cannot be updated.
// Note: This method is intentionally left without update logic because changes
// to 'project_id', 'service_account_email', 'ttl_days' or 'public_key' require the resource to be entirely replaced.
// As a result, the Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (r *serviceAccountKeyResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Service accounts cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating service account key", "Service account key can't be updated")
}

// Delete deletes the service account key and removes it from the Terraform state on success.
func (r *serviceAccountKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	keyId := model.KeyId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)
	ctx = tflog.SetField(ctx, "key_id", keyId)

	// Call API to delete the existing service account key.
	err := r.client.DeleteServiceAccountKey(ctx, projectId, serviceAccountEmail, keyId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting service account key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Service account key deleted")
}

func toCreatePayload(model *Model) (*serviceaccount.CreateServiceAccountKeyPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	// Prepare the payload
	payload := &serviceaccount.CreateServiceAccountKeyPayload{}

	// Set ValidUntil based on TtlDays if specified
	if !utils.IsUndefined(model.TtlDays) {
		validUntil, err := computeValidUntil(model.TtlDays.ValueInt64Pointer())
		if err != nil {
			return nil, err
		}
		payload.ValidUntil = &validUntil
	}

	// Set PublicKey if specified
	if !utils.IsUndefined(model.PublicKey) && model.PublicKey.ValueString() != "" {
		payload.PublicKey = conversion.StringValueToPointer(model.PublicKey)
	}

	return payload, nil
}

// computeValidUntil calculates the timestamp for when the item will no longer be valid.
func computeValidUntil(ttlDays *int64) (time.Time, error) {
	if ttlDays == nil {
		return time.Time{}, fmt.Errorf("ttlDays is nil")
	}
	return time.Now().UTC().Add(time.Duration(*ttlDays) * 24 * time.Hour), nil
}

// mapCreateResponse maps response data from a create operation to the model.
func mapCreateResponse(resp *serviceaccount.CreateServiceAccountKeyResponse, model *Model) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if resp == nil {
		return fmt.Errorf("service account key response is nil")
	}

	if resp.Id == nil {
		return fmt.Errorf("service account key id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.ServiceAccountEmail.ValueString(), *resp.Id)
	model.KeyId = types.StringPointerValue(resp.Id)

	jsonData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("JSON encoding error: %w", err)
	}

	if jsonData != nil {
		model.Json = types.StringValue(string(jsonData))
	}

	return nil
}
