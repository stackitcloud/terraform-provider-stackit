package certcheck

import (
	"context"
	"errors"
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
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	observabilitySdk "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &certCheckResource{}
	_ resource.ResourceWithConfigure   = &certCheckResource{}
	_ resource.ResourceWithImportState = &certCheckResource{}
)

type Model struct {
	Id          types.String `tfsdk:"id"` // needed by TF
	ProjectId   types.String `tfsdk:"project_id"`
	InstanceId  types.String `tfsdk:"instance_id"`
	CertCheckId types.String `tfsdk:"cert_check_id"`
	Source      types.String `tfsdk:"source"`
}

// NewCertCheckResource is a helper function to simplify the provider implementation.
func NewCertCheckResource() resource.Resource {
	return &certCheckResource{}
}

// certCheckResource is the resource implementation.
type certCheckResource struct {
	client observabilitySdk.DefaultAPI
}

// Metadata returns the resource type name.
func (r *certCheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_cert_check"
}

// Configure adds the provider configured client to the resource.
func (r *certCheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !core.ValidateServiceRegion(ctx, &providerData, "Observability", &resp.Diagnostics) {
		return
	}

	r.client = clients.ObservabilityV1Client

	tflog.Info(ctx, "Observability certificate check client configured")
}

// Schema defines the schema for the resource.
func (r *certCheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Observability certificate check resource schema. Certificate checks are unique per source within an instance: creating a check for a source that already exists returns the existing check. " +
			"Two resources with the same `source` therefore manage the same check, and destroying one of them deletes it. " +
			"Not every Observability plan allows certificate checks (e.g. `Observability-Monitoring-Starter-EU01` does not). " +
			"Must have a `region` specified in the provider configuration. This service is only available when the provider is configured with `default_region = \"eu01\"`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`,`cert_check_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the certificate check is associated.",
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
				Description: "The Observability instance ID to which the certificate check is associated.",
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
				Description: "The certificate check ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source": schema.StringAttribute{
				Description: "The endpoint whose certificate is checked, including the port, e.g. `tcp://www.example.com:443` or `https://www.example.com:443`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 270),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *certCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	createResp, err := r.client.CreateCertCheck(ctx, instanceId, projectId).CreateCertCheckPayload(toCreatePayload(&model)).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating certificate check", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	certCheck, err := createdCertCheck(createResp, model.Source.ValueString())
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating certificate check", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":    projectId,
		"instance_id":   instanceId,
		"cert_check_id": certCheck.Id,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	err = mapFields(certCheck, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating certificate check", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability certificate check created")
}

// Read refreshes the Terraform state with the latest data.
func (r *certCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	certCheckId := model.CertCheckId.ValueString()
	if certCheckId == "" {
		// Resource not yet created; ID is unknown.
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "cert_check_id", certCheckId)

	// The API has no endpoint to get a single certificate check
	listResp, err := r.client.ListCertChecks(ctx, instanceId, projectId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading certificate check", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if listResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading certificate check", "Empty response")
		return
	}
	certCheck := findCertCheck(listResp.CertChecks, certCheckId)
	if certCheck == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapFields(certCheck, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading certificate check", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Observability certificate check read")
}

// Update is not supported, all attributes require a replacement.
func (r *certCheckResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating certificate check", "Certificate check can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *certCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	certCheckId := model.CertCheckId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "cert_check_id", certCheckId)

	_, err := r.client.DeleteCertCheck(ctx, instanceId, projectId, certCheckId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting certificate check", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Observability certificate check deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,cert_check_id
func (r *certCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing certificate check",
			fmt.Sprintf("Expected import identifier with format [project_id],[instance_id],[cert_check_id], got %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":    idParts[0],
		"instance_id":   idParts[1],
		"cert_check_id": idParts[2],
	})
	tflog.Info(ctx, "Observability certificate check state imported")
}

func toCreatePayload(model *Model) observabilitySdk.CreateCertCheckPayload {
	return observabilitySdk.CreateCertCheckPayload{
		Source: model.Source.ValueString(),
	}
}

// createdCertCheck returns the certificate check created by a CreateCertCheck call.
// The response contains the created check, and the list of all certificate checks of the instance as a fallback.
// The API does not create duplicates for the same source, so the fallback is expected to find at most one match.
func createdCertCheck(createResp *observabilitySdk.CertCheckResponse, source string) (*observabilitySdk.CertCheckChildResponse, error) {
	if createResp == nil {
		return nil, fmt.Errorf("response is nil")
	}
	if createResp.CertCheck != nil && createResp.CertCheck.Id != "" {
		return createResp.CertCheck, nil
	}

	var match *observabilitySdk.CertCheckChildResponse
	for i := range createResp.CertChecks {
		if createResp.CertChecks[i].Source != source {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("created certificate check not in response and more than one certificate check with source %q found", source)
		}
		match = &createResp.CertChecks[i]
	}
	if match == nil {
		return nil, fmt.Errorf("created certificate check with source %q not found in response", source)
	}
	return match, nil
}

func findCertCheck(certChecks []observabilitySdk.CertCheckChildResponse, id string) *observabilitySdk.CertCheckChildResponse {
	for i := range certChecks {
		if certChecks[i].Id == id {
			return &certChecks[i]
		}
	}
	return nil
}

func mapFields(certCheck *observabilitySdk.CertCheckChildResponse, model *Model) error {
	if certCheck == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	if certCheck.Id == "" {
		return fmt.Errorf("certificate check id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.InstanceId.ValueString(), certCheck.Id)
	model.CertCheckId = types.StringValue(certCheck.Id)
	model.Source = types.StringValue(certCheck.Source)
	return nil
}
