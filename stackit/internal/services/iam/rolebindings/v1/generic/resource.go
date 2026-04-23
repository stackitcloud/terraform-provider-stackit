package generic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	secretsmanagerV1Alpha "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &RoleBindingResource[secretsmanagerV1Alpha.APIClient]{}
	_ resource.ResourceWithConfigure   = &RoleBindingResource[secretsmanagerV1Alpha.APIClient]{}
	_ resource.ResourceWithImportState = &RoleBindingResource[secretsmanagerV1Alpha.APIClient]{}
)

type Model struct {
	Id         types.String `tfsdk:"id"` // needed by TF
	Region     types.String `tfsdk:"region"`
	ResourceId types.String `tfsdk:"resource_id"`
	Role       types.String `tfsdk:"role"`
	Subject    types.String `tfsdk:"subject"`
}

type GenericRoleBindingResponse interface {
	GetRole() string
	GetSubject() string
}

// RoleBindingResource is the resource implementation.
type RoleBindingResource[C any] struct {
	providerData core.ProviderData
	apiClient    *C

	ApiName      string // e.g. "iaas", "secretsmanager", ...
	ResourceType string // e.g. "instance", ...

	// callbacks for lifecyle handling
	ApiClientFactory  func(context.Context, *core.ProviderData, *diag.Diagnostics) *C
	ExecReadRequest   func(ctx context.Context, client *C, region, resourceId, role, subject string) (GenericRoleBindingResponse, error)
	ExecCreateRequest func(ctx context.Context, client *C, region, resourceId, role, subject string) (GenericRoleBindingResponse, error)
	ExecUpdateRequest func(ctx context.Context, client *C, region, resourceId, role, subject string) (GenericRoleBindingResponse, error)
	ExecDeleteRequest func(ctx context.Context, client *C, region, resourceId, role, subject string) error
}

// Metadata returns the resource type name.
func (r *RoleBindingResource[C]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_%s_role_binding_v1", req.ProviderTypeName, r.ApiName, r.ResourceType)
}

// Configure adds the provider configured client to the resource.
func (r *RoleBindingResource[C]) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &providerData, features.IamExperiment, fmt.Sprintf("stackit_%s_%s_role_binding_v1", r.ApiName, r.ResourceType), core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apiClient = r.ApiClientFactory(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s %s client configured", r.ApiName, r.ResourceType))
}

// Schema defines the schema for the resource.
func (r *RoleBindingResource[C]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: features.AddExperimentDescription("IAM role binding resource schema.", features.IamExperiment, core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. It is structured as \"`region`,`resource_id`,`role`,`subject`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: "The identifier of the resource to apply this role binding to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "A valid role defined for the resource.",
				Required:    true,
			},
			"subject": schema.StringAttribute{
				Description: "Identifier of user, service account or client. Usually email address or name in case of clients.",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: "The resource region. If not defined, the provider region is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *RoleBindingResource[C]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	region := r.providerData.GetRegionWithOverride(model.Region)
	resourceId := model.ResourceId.ValueString()

	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "resource_id", resourceId)

	roleBindingResp, err := r.ExecCreateRequest(ctx, r.apiClient, region, resourceId, model.Role.ValueString(), model.Subject.ValueString())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error creating %s %s role binding", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(roleBindingResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error creating %s %s role binding", r.ApiName, r.ResourceType), fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(10 * time.Second): // safety sleep due to api cache
		// continue
	}

	tflog.Info(ctx, fmt.Sprintf("%s %s role binding created", r.ApiName, r.ResourceType))
}

// Read refreshes the Terraform state with the latest data.
func (r *RoleBindingResource[C]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	region := r.providerData.GetRegionWithOverride(model.Region)
	resourceId := model.ResourceId.ValueString()

	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "resource_id", resourceId)

	// Note: The API won't return HTTP 404 errors here, at least there are no HTTP 404 errors documented for the distributed role binding API
	roleBindingResp, err := r.ExecReadRequest(ctx, r.apiClient, region, resourceId, model.Role.ValueString(), model.Subject.ValueString())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error reading %s %s role binding", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(roleBindingResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error reading %s %s role binding", r.ApiName, r.ResourceType), fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s %s role binding read", r.ApiName, r.ResourceType))
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *RoleBindingResource[C]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	region := r.providerData.GetRegionWithOverride(model.Region)
	resourceId := model.ResourceId.ValueString()

	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "resource_id", resourceId)

	roleBindingResp, err := r.ExecUpdateRequest(ctx, r.apiClient, region, resourceId, model.Role.ValueString(), model.Subject.ValueString())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error updating %s %s role binding", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(roleBindingResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error updating %s %s role binding", r.ApiName, r.ResourceType), fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(10 * time.Second): // safety sleep due to api cache
		// continue
	}

	tflog.Info(ctx, fmt.Sprintf("%s %s role binding updated", r.ApiName, r.ResourceType))
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *RoleBindingResource[C]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	region := r.providerData.GetRegionWithOverride(model.Region)
	resourceId := model.ResourceId.ValueString()

	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "resource_id", resourceId)

	// Note: The API won't return HTTP 404 errors here, at least there are no HTTP 404 errors documented for the distributed role binding API
	err := r.ExecDeleteRequest(ctx, r.apiClient, region, resourceId, model.Role.ValueString(), model.Subject.ValueString())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error deleting %s %s role binding", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(10 * time.Second): // safety sleep due to api cache
		// continue
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, fmt.Sprintf("%s %s role binding deleted", r.ApiName, r.ResourceType))
}

// ImportState imports a resource into the Terraform state on success.
func (r *RoleBindingResource[C]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			fmt.Sprintf("Error importing %s %s role binding", r.ApiName, r.ResourceType),
			fmt.Sprintf("Expected import identifier with format [region],[resource_id],[role],[subject], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"region":      idParts[0],
		"resource_id": idParts[1],
		"role":        idParts[2],
		"subject":     idParts[3],
	})

	tflog.Info(ctx, fmt.Sprintf("%s %s role binding state imported", r.ApiName, r.ResourceType))
}

func mapFields(resp GenericRoleBindingResponse, model *Model, region string) error {
	if resp == nil {
		return fmt.Errorf("nil response")
	} else if model == nil {
		return fmt.Errorf("nil model")
	}

	role := resp.GetRole()
	subject := resp.GetSubject()

	model.Id = utils.BuildInternalTerraformId(region, model.ResourceId.ValueString(), role, subject)
	model.Region = types.StringValue(region)
	model.Role = types.StringValue(role)
	model.Subject = types.StringValue(subject)

	return nil
}
