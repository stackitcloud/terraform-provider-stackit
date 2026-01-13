package certcheck

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	observabilityUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/observability/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &observabilityCertCheckResource{}
	_ resource.ResourceWithConfigure   = &observabilityCertCheckResource{}
	_ resource.ResourceWithImportState = &observabilityCertCheckResource{}
)

type Model struct {
	Id          types.String `tfsdk:"id"`
	ProjectId   types.String `tfsdk:"project_id"`
	InstanceId  types.String `tfsdk:"instance_id"`
	CertCheckId types.String `tfsdk:"cert_check_id"`
	Source      types.String `tfsdk:"source"`
}

var descriptions = map[string]string{
	"id":            "Terraform resource ID in format `project_id,instance_id,cert_check_id`.",
	"project_id":    "STACKIT project ID.",
	"instance_id":   "STACKIT Observability instance ID.",
	"cert_check_id": "Unique ID of the cert-check.",
	"source":        "The cert source to check, e.g. tcp://stackit.de:443 Must start with `tcp://`.",
}

// NewCertCheckResource is a helper function to simplify the provider implementation.
func NewCertCheckResource() resource.Resource {
	return &observabilityCertCheckResource{}
}

// observabilityCertCheckResource is the resource implementation.
type observabilityCertCheckResource struct {
	client *observability.APIClient
}

// Metadata returns the resource type name.
func (r *observabilityCertCheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_cert_check"
}

// Configure adds the provider configured client to the resource.
func (r *observabilityCertCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_observability_cert_check", "resource")
	if resp.Diagnostics.HasError() {
		return
	}

	r.client = observabilityUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability client configured")
}

// Schema defines the schema for the resource.
func (r *observabilityCertCheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource for managing Cert checks in STACKIT Observability. It ships Telegraf X509-Cert metrics to the observability instance, as documented here: `https://docs.influxdata.com/telegraf/v1/input-plugins/x509_cert/`",
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
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cert_check_id": schema.StringAttribute{
				Description: descriptions["cert_check_id"],
				Computed:    true,
			},
			"source": schema.StringAttribute{
				Description: descriptions["source"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(5),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^tcp://`),
						"The source must start with tcp://.",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *observabilityCertCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	certCheckSource := model.Source.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "cert_check_source", certCheckSource)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	createCertCheck, err := r.client.CreateCertCheck(
		ctx,
		instanceId,
		projectId,
	).CreateCertCheckPayload(observability.CreateCertCheckPayload{Source: &certCheckSource}).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating cert-check", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Log success message from API
	if createCertCheck.Message != nil {
		tflog.Info(ctx, fmt.Sprintf("Create cert-check response message: %s", *createCertCheck.Message))
	}

	if createCertCheck.CertChecks == nil || len(*createCertCheck.CertChecks) == 0 {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating cert-check", "Response is empty")
		return
	}

	for _, certCheck := range *createCertCheck.CertChecks {
		if certCheck.Source != nil && *certCheck.Source == certCheckSource {
			if err := mapFields(ctx, &certCheck, &model); err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating cert-check", "Unable to map cert-check model")
				return
			}
			break
		}
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "cert-check created")
}

// Read refreshes the Terraform state with the latest data.
func (r *observabilityCertCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	certCheckSource := model.Source.ValueString()
	certCheckId := model.CertCheckId.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "cert_check_source", certCheckSource)
	ctx = tflog.SetField(ctx, "cert_check_id", certCheckId)

	listCertCheck, err := r.client.ListCertChecks(ctx, instanceId, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing cert-checks", fmt.Sprintf("List API payload: %v", err))
		return
	}

	if listCertCheck.CertChecks == nil || len(*listCertCheck.CertChecks) == 0 {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error listing cert-checks", "Response is empty")
		return
	}

	for _, certCheck := range *listCertCheck.CertChecks {
		// we also check if cert-ids are matching to support import functionality
		if (certCheck.Source != nil && *certCheck.Source == certCheckSource) || (certCheck.Id != nil && *certCheck.Id == certCheckId) {
			if err := mapFields(ctx, &certCheck, &model); err != nil {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading cert-check", "Unable to map cert-check model")
				return
			}
			break
		}
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "cert-check read")
}

// Update attempts to update the resource. In this case, cert-checks cannot be updated.
// The Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (r *observabilityCertCheckResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating cert-check", "Observability cert-checks can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *observabilityCertCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	certCheckId := model.CertCheckId.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "cert_check_id", certCheckId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	_, err := r.client.DeleteCertCheck(ctx, instanceId, projectId, certCheckId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting cert-check", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	tflog.Info(ctx, "cert check deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,cert_check_id
func (r *observabilityCertCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing cert-check",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id],[cert_check_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cert_check_id"), idParts[2])...)
	tflog.Info(ctx, "Observability cert-check state imported")
}

// mapFields maps certCheck response to the model.
func mapFields(_ context.Context, certCheck *observability.CertCheckChildResponse, model *Model) error {
	if certCheck == nil {
		return fmt.Errorf("cert-check input is nil")
	}

	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if certCheck.Id == nil {
		return fmt.Errorf("cert-check id is nil")
	}

	if certCheck.Source == nil {
		return fmt.Errorf("cert-check source is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.InstanceId.ValueString(), *certCheck.Id)
	model.CertCheckId = types.StringValue(*certCheck.Id)
	model.Source = types.StringValue(*certCheck.Source)

	return nil
}
