package observability

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	argusCredentialResource "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/argus/credential"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &credentialResource{}
	_ resource.ResourceWithConfigure = &credentialResource{}
	_ resource.ResourceWithMoveState = &credentialResource{}
)

type Model struct {
	Id         types.String `tfsdk:"id"`
	ProjectId  types.String `tfsdk:"project_id"`
	InstanceId types.String `tfsdk:"instance_id"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
}

// NewCredentialResource is a helper function to simplify the provider implementation.
func NewCredentialResource() resource.Resource {
	return &credentialResource{}
}

// credentialResource is the resource implementation.
type credentialResource struct {
	client *observability.APIClient
}

// Metadata returns the resource type name.
func (r *credentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_credential"
}

// Configure adds the provider configured client to the resource.
func (r *credentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *observability.APIClient
	var err error
	if providerData.ObservabilityCustomEndpoint != "" {
		apiClient, err = observability.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ObservabilityCustomEndpoint),
		)
	} else {
		apiClient, err = observability.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.GetRegion()),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Observability credential client configured")
}

func (r *credentialResource) MoveState(_ context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			SourceSchema: &argusCredentialResource.Schema,
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != "stackit_argus_credential" {
					return
				}

				// Checks source provider
				if !strings.HasSuffix(req.SourceProviderAddress, "stackitcloud/stackit") {
					return
				}

				var sourceStateData argusCredentialResource.Model
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				targetStateData := Model{
					Id:         sourceStateData.Id,
					ProjectId:  sourceStateData.ProjectId,
					InstanceId: sourceStateData.InstanceId,
					Username:   sourceStateData.Username,
					Password:   sourceStateData.Password,
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}

func (r *credentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Observability credential resource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`,`username`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the credential is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: "The Observability Instance ID the credential belongs to.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "Credential username",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"password": schema.StringAttribute{
				Description: "Credential password",
				Computed:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
	instanceId := model.InstanceId.ValueString()

	got, err := r.client.CreateCredentials(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Calling API: %v", err))
		return
	}
	err = mapFields(got.Credentials, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability credential created")
}

func mapFields(r *observability.Credentials, model *Model) error {
	if r == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	var userName string
	if model.Username.ValueString() != "" {
		userName = model.Username.ValueString()
	} else if r.Username != nil {
		userName = *r.Username
	} else {
		return fmt.Errorf("username id not present")
	}
	idParts := []string{
		model.ProjectId.ValueString(),
		model.InstanceId.ValueString(),
		userName,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.Username = types.StringPointerValue(r.Username)
	model.Password = types.StringPointerValue(r.Password)
	return nil
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
	instanceId := model.InstanceId.ValueString()
	userName := model.Username.ValueString()
	_, err := r.client.GetCredentials(ctx, instanceId, projectId, userName).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Calling API: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability credential read")
}

func (r *credentialResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating credential", "Credential can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *credentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	userName := model.Username.ValueString()
	_, err := r.client.DeleteCredentials(ctx, instanceId, projectId, userName).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", fmt.Sprintf("Calling API: %v", err))
		return
	}
	tflog.Info(ctx, "Observability credential deleted")
}
