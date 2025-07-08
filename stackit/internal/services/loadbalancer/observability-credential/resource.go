package loadbalancer

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	loadbalancerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/loadbalancer/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &observabilityCredentialResource{}
	_ resource.ResourceWithConfigure   = &observabilityCredentialResource{}
	_ resource.ResourceWithImportState = &observabilityCredentialResource{}
	_ resource.ResourceWithModifyPlan  = &observabilityCredentialResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	ProjectId      types.String `tfsdk:"project_id"`
	DisplayName    types.String `tfsdk:"display_name"`
	Username       types.String `tfsdk:"username"`
	Password       types.String `tfsdk:"password"`
	CredentialsRef types.String `tfsdk:"credentials_ref"`
	Region         types.String `tfsdk:"region"`
}

// NewObservabilityCredentialResource is a helper function to simplify the provider implementation.
func NewObservabilityCredentialResource() resource.Resource {
	return &observabilityCredentialResource{}
}

// observabilityCredentialResource is the resource implementation.
type observabilityCredentialResource struct {
	client       *loadbalancer.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *observabilityCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_loadbalancer_observability_credential"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *observabilityCredentialResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

// Configure adds the provider configured client to the resource.
func (r *observabilityCredentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := loadbalancerUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Load Balancer client configured")
}

// Schema defines the schema for the resource.
func (r *observabilityCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":            "Load balancer observability credential resource schema. Must have a `region` specified in the provider configuration. These contain the username and password for the observability service (e.g. Argus) where the load balancer logs/metrics will be pushed into",
		"id":              "Terraform's internal resource ID. It is structured as \"`project_id`\",\"region\",\"`credentials_ref`\".",
		"credentials_ref": "The credentials reference is used by the Load Balancer to define which credentials it will use.",
		"project_id":      "STACKIT project ID to which the load balancer observability credential is associated.",
		"display_name":    "Observability credential name.",
		"username":        "The password for the observability service (e.g. Argus) where the logs/metrics will be pushed into.",
		"password":        "The username for the observability service (e.g. Argus) where the logs/metrics will be pushed into.",
		"region":          "The resource region. If not defined, the provider region is used.",
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
			"credentials_ref": schema.StringAttribute{
				Description: descriptions["credentials_ref"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: descriptions["username"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Description: descriptions["password"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
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
func (r *observabilityCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating observability credential", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new observability credentials
	createResp, err := r.client.CreateCredentials(ctx, projectId, region).CreateCredentialsPayload(*payload).XRequestID(uuid.NewString()).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating observability credential", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "credentials_ref", createResp.Credential.CredentialsRef)

	// Map response body to schema
	err = mapFields(createResp.Credential, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating observability credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Load balancer observability credential created")
}

// Read refreshes the Terraform state with the latest data.
func (r *observabilityCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsRef := model.CredentialsRef.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_ref", credentialsRef)
	ctx = tflog.SetField(ctx, "region", region)

	// Get credentials
	credResp, err := r.client.GetCredentials(ctx, projectId, region, credentialsRef).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading observability credential", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(credResp.Credential, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading observability credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Load balancer observability credential read")
}

func (r *observabilityCredentialResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating observability credential", "Observability credential can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *observabilityCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsRef := model.CredentialsRef.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_ref", credentialsRef)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete credentials
	_, err := r.client.DeleteCredentials(ctx, projectId, region, credentialsRef).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting observability credential", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Load balancer observability credential deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,name
func (r *observabilityCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing observability credential",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[credentials_ref]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credentials_ref"), idParts[2])...)
	tflog.Info(ctx, "Load balancer observability credential state imported")
}

func toCreatePayload(model *Model) (*loadbalancer.CreateCredentialsPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &loadbalancer.CreateCredentialsPayload{
		DisplayName: conversion.StringValueToPointer(model.DisplayName),
		Username:    conversion.StringValueToPointer(model.Username),
		Password:    conversion.StringValueToPointer(model.Password),
	}, nil
}

func mapFields(cred *loadbalancer.CredentialsResponse, m *Model, region string) error {
	if cred == nil {
		return fmt.Errorf("response input is nil")
	}
	if m == nil {
		return fmt.Errorf("model input is nil")
	}

	var credentialsRef string
	if m.CredentialsRef.ValueString() != "" {
		credentialsRef = m.CredentialsRef.ValueString()
	} else if cred.CredentialsRef != nil {
		credentialsRef = *cred.CredentialsRef
	} else {
		return fmt.Errorf("credentials ref not present")
	}
	m.CredentialsRef = types.StringValue(credentialsRef)
	m.DisplayName = types.StringPointerValue(cred.DisplayName)
	var username string
	if m.Username.ValueString() != "" {
		username = m.Username.ValueString()
	} else if cred.Username != nil {
		username = *cred.Username
	} else {
		return fmt.Errorf("username not present")
	}
	m.Username = types.StringValue(username)
	m.Region = types.StringValue(region)
	m.Id = utils.BuildInternalTerraformId(
		m.ProjectId.ValueString(),
		m.Region.ValueString(),
		m.CredentialsRef.ValueString(),
	)

	return nil
}
