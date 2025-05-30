package foo

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	fooUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/foo/utils"

	"github.com/stackitcloud/stackit-sdk-go/services/foo" // Import service "foo" from the STACKIT SDK for Go
	// (...)
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &barResource{}
	_ resource.ResourceWithConfigure   = &barResource{}
	_ resource.ResourceWithImportState = &barResource{}
)

// Provider's internal model
type Model struct {
	Id              types.String `tfsdk:"id"` // needed by TF
	ProjectId       types.String `tfsdk:"project_id"`
	BarId           types.String `tfsdk:"bar_id"`
	MyRequiredField types.String `tfsdk:"my_required_field"`
	MyOptionalField types.String `tfsdk:"my_optional_field"`
	MyReadOnlyField types.String `tfsdk:"my_read_only_field"`
}

// NewBarResource is a helper function to simplify the provider implementation.
func NewBarResource() resource.Resource {
	return &barResource{}
}

// barResource is the resource implementation.
type barResource struct {
	client *foo.APIClient
}

// Metadata returns the resource type name.
func (r *barResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_foo_bar"
}

// Configure adds the provider configured client to the resource.
func (r *barResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := fooUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Foo bar client configured")
}

// Schema defines the schema for the resource.
func (r *barResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":               "Foo bar resource schema.",
		"id":                 "Terraform's internal resource identifier. It is structured as \"`project_id`,`bar_id`\".",
		"project_id":         "STACKIT Project ID to which the bar is associated.",
		"bar_id":             "The bar ID.",
		"my_required_field":  "My required field description.",
		"my_optional_field":  "My optional field description.",
		"my_read_only_field": "My read-only field description.",
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
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					// "RequiresReplace" makes the provider recreate the resource when the field is changed in the configuration
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					// Validators can be used to validate the values set to a field
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"bar_id": schema.StringAttribute{
				Description: descriptions["bar_id"],
				Computed:    true,
			},
			"my_required_field": schema.StringAttribute{
				Description: descriptions["my_required_field"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					// "RequiresReplace" makes the provider recreate the resource when the field is changed in the configuration
					stringplanmodifier.RequiresReplace(),
				},
			},
			"my_optional_field": schema.StringAttribute{
				Description: descriptions["my_optional_field"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// "UseStateForUnknown" can be used to copy a prior state value into the planned value. It should be used when it is known that an unconfigured value will remain the same after a resource update
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"my_read_only_field": schema.StringAttribute{
				Description: descriptions["my_read_only_field"],
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *barResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	barId := model.BarId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Create new bar
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	resp, err := r.client.CreateBar(ctx, projectId, barId).CreateBarPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating bar", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "bar_id", resp.BarId)

	// Map response body to schema
	err = mapFields(resp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating bar", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Foo bar created")
}

// Read refreshes the Terraform state with the latest data.
func (r *barResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	barId := model.BarId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bar_id", barId)

	barResp, err := r.client.GetBar(ctx, projectId, barId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading bar", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(barResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading bar", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Foo bar read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *barResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Similar to Create method, calls r.client.UpdateBar instead
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *barResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	barId := model.BarId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bar_id", barId)

	// Delete existing bar
	_, err := r.client.DeleteBar(ctx, projectId, barId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting bar", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "Foo bar deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the bar resource import identifier is: project_id,bar_id
func (r *barResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing bar",
			fmt.Sprintf("Expected import identifier with format [project_id],[bar_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bar_id"), idParts[1])...)
	tflog.Info(ctx, "Foo bar state imported")
}

// Maps bar fields to the provider's internal model
func mapFields(barResp *foo.GetBarResponse, model *Model) error {
	if barResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if barResp.Bar == nil {
		return fmt.Errorf("response bar is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	bar := barResp.Bar
	model.BarId = types.StringPointerValue(bar.BarId)

	idParts := []string{
		model.ProjectId.ValueString(),
		model.BarId.ValueString(),
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	model.MyRequiredField = types.StringPointerValue(bar.MyRequiredField)
	model.MyOptionalField = types.StringPointerValue(bar.MyOptionalField)
	model.MyReadOnlyField = types.StringPointerValue(bar.MyOtherField)
	return nil
}

// Build CreateBarPayload from provider's model
func toCreatePayload(model *Model) (*foo.CreateBarPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	myRequiredFieldValue := conversion.StringValueToPointer(model.MyRequiredField)
	myOptionalFieldValue := conversion.StringValueToPointer(model.MyOptionalField)
	return &foo.CreateBarPayload{
		MyRequiredField: myRequiredFieldValue,
		MyOptionalField: myOptionalFieldValue,
	}, nil
}
