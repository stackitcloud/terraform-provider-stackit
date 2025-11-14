package keypair

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &keyPairResource{}
	_ resource.ResourceWithConfigure   = &keyPairResource{}
	_ resource.ResourceWithImportState = &keyPairResource{}
)

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	Labels      types.Map    `tfsdk:"labels"`
}

// NewKeyPairResource is a helper function to simplify the provider implementation.
func NewKeyPairResource() resource.Resource {
	return &keyPairResource{}
}

// keyPairResource is the resource implementation.
type keyPairResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *keyPairResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key_pair"
}

// Configure adds the provider configured client to the resource.
func (r *keyPairResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *keyPairResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Key pair resource schema. Must have a `region` specified in the provider configuration. Allows uploading an SSH public key to be used for server authentication."

	resp.Schema = schema.Schema{
		MarkdownDescription: description + "\n\n" + exampleUsageWithServer,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It takes the value of the key pair \"`name`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the SSH key pair.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_key": schema.StringAttribute{
				Description: "A string representation of the public SSH key. E.g., `ssh-rsa <key_data>` or `ssh-ed25519 <key-data>`.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"fingerprint": schema.StringAttribute{
				Description: "The fingerprint of the public SSH key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container.",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

// ModifyPlan will be called in the Plan phase.
// It will check if the plan contains a change that requires replacement. If yes, it will show a warning to the user.
func (r *keyPairResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	// If the state is empty we are creating a new resource
	// If the plan is empty we are deleting the resource
	// In both cases we don't need to check for replacement
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var planModel Model
	diags := req.Plan.Get(ctx, &planModel)
	resp.Diagnostics.Append(diags...)

	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)

	if planModel.PublicKey.ValueString() != stateModel.PublicKey.ValueString() {
		core.LogAndAddWarning(ctx, &resp.Diagnostics, "Key pair public key change", "Changing the public key will trigger a replacement of the key pair resource. The new key pair will not be valid to access servers on which the old key was used, as the key is only registered during server creation.")
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *keyPairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := model.Name.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "name", name)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating key pair", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new key pair

	keyPair, err := r.client.CreateKeyPair(ctx).CreateKeyPairPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating key pair", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, keyPair, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating key pair", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key pair created")
}

// Read refreshes the Terraform state with the latest data.
func (r *keyPairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := model.Name.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "name", name)

	keyPairResp, err := r.client.GetKeyPair(ctx, name).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading key pair", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, keyPairResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading key pair", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Key pair read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *keyPairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := model.Name.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "name", name)

	// Retrieve values from state
	var stateModel Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, stateModel.Labels)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating key pair", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing key pair
	updatedKeyPair, err := r.client.UpdateKeyPair(ctx, name).UpdateKeyPairPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating key pair", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, updatedKeyPair, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating key pair", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "key pair updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *keyPairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := model.Name.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "name", name)

	// Delete existing key pair
	err := r.client.DeleteKeyPair(ctx, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting key pair", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Key pair deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,key_pair_id
func (r *keyPairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 1 || idParts[0] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing key pair",
			fmt.Sprintf("Expected import identifier with format: [name]  Got: %q", req.ID),
		)
		return
	}

	name := idParts[0]
	ctx = tflog.SetField(ctx, "name", name)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	tflog.Info(ctx, "Key pair state imported")
}

func mapFields(ctx context.Context, keyPairResp *iaas.Keypair, model *Model) error {
	if keyPairResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var name string
	if model.Name.ValueString() != "" {
		name = model.Name.ValueString()
	} else if keyPairResp.Name != nil {
		name = *keyPairResp.Name
	} else {
		return fmt.Errorf("key pair name not present")
	}

	model.Id = types.StringValue(name)
	model.PublicKey = types.StringPointerValue(keyPairResp.PublicKey)
	model.Fingerprint = types.StringPointerValue(keyPairResp.Fingerprint)

	var err error
	model.Labels, err = iaasUtils.MapLabels(ctx, keyPairResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	return nil
}

func toCreatePayload(ctx context.Context, model *Model) (*iaas.CreateKeyPairPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.CreateKeyPairPayload{
		Name:      conversion.StringValueToPointer(model.Name),
		PublicKey: conversion.StringValueToPointer(model.PublicKey),
		Labels:    &labels,
	}, nil
}

func toUpdatePayload(ctx context.Context, model *Model, currentLabels types.Map) (*iaas.UpdateKeyPairPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	return &iaas.UpdateKeyPairPayload{
		Labels: &labels,
	}, nil
}
