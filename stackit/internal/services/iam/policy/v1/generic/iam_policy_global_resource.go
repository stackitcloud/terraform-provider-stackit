package generic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	resourcemanagerV1Beta "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &GlobalIAMPolicyResource[resourcemanagerV1Beta.APIClient]{}
	_ resource.ResourceWithConfigure   = &GlobalIAMPolicyResource[resourcemanagerV1Beta.APIClient]{}
	_ resource.ResourceWithImportState = &GlobalIAMPolicyResource[resourcemanagerV1Beta.APIClient]{}
)

type GlobalModel struct {
	Id           types.String       `tfsdk:"id"` // needed by TF
	ResourceId   types.String       `tfsdk:"resource_id"`
	Etag         types.String       `tfsdk:"etag"`
	RoleBindings []roleBindingModel `tfsdk:"role_bindings"`
}

type roleBindingModel struct {
	Role    types.String `tfsdk:"role"`
	Subject types.String `tfsdk:"subject"`
}

func (m *roleBindingModel) GetRole() string {
	return m.Role.ValueString()
}

func (m *roleBindingModel) GetSubject() string {
	return m.Subject.ValueString()
}

func (m *roleBindingModel) SetSubject(v string) {
	m.Subject = types.StringValue(v)
}

func (m *roleBindingModel) SetRole(v string) {
	m.Role = types.StringValue(v)
}

type RoleBinding interface {
	GetRole() string
	SetRole(string)
	GetSubject() string
	SetSubject(string)
}

type IAMPolicy struct {
	RoleBindings []RoleBinding
	Etag         *string
}

// GlobalIAMPolicyResource is the resource implementation.
type GlobalIAMPolicyResource[C any] struct {
	apiClient C

	// Whether the resource should be marked as experimental. Should be the case for all IAM policy resources as of now.
	Experimental bool

	ApiName      string // e.g. "iaas", "secretsmanager", ...
	ResourceType string // e.g. "instance", ...

	// callbacks for lifecyle handling
	ApiClientFactory  func(context.Context, *core.ProviderData, *diag.Diagnostics) C
	ExecReadRequest   func(ctx context.Context, client C, resourceId string) (*IAMPolicy, error)
	ExecUpdateRequest func(ctx context.Context, client C, resourceId string, etag *string, roleBindings []RoleBinding) (*IAMPolicy, error)
}

func (r *GlobalIAMPolicyResource[C]) GetResourceName() string {
	return fmt.Sprintf("stackit_%s_%s_iam_policy_v1", r.ApiName, r.ResourceType)
}

// Metadata returns the resource type name.
func (r *GlobalIAMPolicyResource[C]) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.GetResourceName()
}

// Configure adds the provider configured client to the resource.
func (r *GlobalIAMPolicyResource[C]) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if r.Experimental {
		features.CheckExperimentEnabled(ctx, &providerData, features.IamExperiment, fmt.Sprintf("stackit_%s_%s_iam_policy_v1", r.ApiName, r.ResourceType), core.Resource, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	r.apiClient = r.ApiClientFactory(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s %s client configured", r.ApiName, r.ResourceType))
}

// Schema defines the schema for the resource.
func (r *GlobalIAMPolicyResource[C]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	const warning = "~> Destruction of the IAM policy resource will only remove it from the state and won't modify the IAM policy on API side. Previous listed subjects will keep their role bindings."

	description := "IAM policy resource schema."
	if r.Experimental {
		description = features.AddExperimentDescription(description, features.IamExperiment, core.Resource)
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("%s\n\n%s", description, warning),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. It is structured as \"`resource_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: "The identifier of the resource to apply this IAM policy to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_bindings": schema.ListNestedAttribute{
				Description: "The role bindings of the policy.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							Description: "A valid role defined for the resource.",
							Required:    true,
						},
						"subject": schema.StringAttribute{
							Description: "Identifier of user, service account or client. Usually email address or name in case of clients.",
							Required:    true,
						},
					},
				},
			},
			"etag": schema.StringAttribute{
				Description: "Internal etag used for API concurrency control.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *GlobalIAMPolicyResource[C]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model GlobalModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	resourceId := model.ResourceId.ValueString()
	etag := model.Etag.ValueStringPointer()

	ctx = tflog.SetField(ctx, "resource_id", resourceId)
	ctx = tflog.SetField(ctx, "etag", etag)

	roleBindings := utils.Map(model.RoleBindings, func(t roleBindingModel) RoleBinding {
		return &t
	})

	roleBindingResp, err := r.ExecUpdateRequest(ctx, r.apiClient, resourceId, etag, roleBindings)
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusConflict {
			core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Etag failure during creation of %s %s IAM policy", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
			return
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error creating %s %s IAM policy", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx) //nolint:tflogresponse // false positive - SDK should actually be called in the callback implementations above

	err = mapFieldsGlobal(roleBindingResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error creating %s %s IAM policy", r.ApiName, r.ResourceType), fmt.Sprintf("Processing API payload: %v", err))
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

	tflog.Info(ctx, fmt.Sprintf("%s %s IAM policy updated", r.ApiName, r.ResourceType))
}

// Read refreshes the Terraform state with the latest data.
func (r *GlobalIAMPolicyResource[C]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model GlobalModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	resourceId := model.ResourceId.ValueString()

	ctx = tflog.SetField(ctx, "resource_id", resourceId)

	iamPolicy, err := r.ExecReadRequest(ctx, r.apiClient, resourceId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error reading %s %s IAM policy", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx) //nolint:tflogresponse // false positive - SDK should actually be called in the callback implementations above

	// Map response body to schema
	err = mapFieldsGlobal(iamPolicy, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error reading %s %s IAM policy", r.ApiName, r.ResourceType), fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	model.Etag = types.StringPointerValue(iamPolicy.Etag)

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s %s IAM policy read", r.ApiName, r.ResourceType))
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *GlobalIAMPolicyResource[C]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model GlobalModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	resourceId := model.ResourceId.ValueString()
	etag := model.Etag.ValueStringPointer()

	ctx = tflog.SetField(ctx, "resource_id", resourceId)
	ctx = tflog.SetField(ctx, "etag", etag)

	roleBindings := utils.Map(model.RoleBindings, func(t roleBindingModel) RoleBinding {
		return &t
	})

	roleBindingResp, err := r.ExecUpdateRequest(ctx, r.apiClient, resourceId, etag, roleBindings)
	if err != nil {
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok && oapiErr.StatusCode == http.StatusConflict {
			core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Etag failure during update of %s %s IAM policy", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
			return
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error updating %s %s IAM policy", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx) //nolint:tflogresponse // false positive - SDK should actually be called in the callback implementations above

	err = mapFieldsGlobal(roleBindingResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error updating %s %s IAM policy", r.ApiName, r.ResourceType), fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	model.Etag = types.StringPointerValue(roleBindingResp.Etag)

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

	tflog.Info(ctx, fmt.Sprintf("%s %s IAM policy updated", r.ApiName, r.ResourceType))
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *GlobalIAMPolicyResource[C]) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddWarning(ctx, &resp.Diagnostics, "IAM Policy left unchanged", "Destruction of the IAM policy resource only removed it from the state and didn't modify the IAM policy on API side. Previous listed subjects will keep their role bindings.")
}

// ImportState imports a resource into the Terraform state on success.
func (r *GlobalIAMPolicyResource[C]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 1 || idParts[0] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			fmt.Sprintf("Error importing %s %s IAM policy", r.ApiName, r.ResourceType),
			fmt.Sprintf("Expected import identifier with format [resource_id] got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"resource_id": idParts[0],
	})

	tflog.Info(ctx, fmt.Sprintf("%s %s IAM policy state imported", r.ApiName, r.ResourceType))
}

func mapFieldsGlobal(resp *IAMPolicy, model *GlobalModel) error {
	if resp == nil {
		return fmt.Errorf("nil response")
	} else if model == nil {
		return fmt.Errorf("nil model")
	}

	model.Id = utils.BuildInternalTerraformId(model.ResourceId.ValueString())
	model.Etag = types.StringPointerValue(resp.Etag)

	model.RoleBindings = make([]roleBindingModel, len(resp.RoleBindings))
	for i, roleBinding := range resp.RoleBindings {
		model.RoleBindings[i] = roleBindingModel{
			Role:    types.StringValue(roleBinding.GetRole()),
			Subject: types.StringValue(roleBinding.GetSubject()),
		}
	}

	return nil
}
