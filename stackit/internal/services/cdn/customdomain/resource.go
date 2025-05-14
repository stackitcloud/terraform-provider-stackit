package cdn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	cdnUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &customDomainResource{}
	_ resource.ResourceWithConfigure   = &customDomainResource{}
	_ resource.ResourceWithImportState = &customDomainResource{}
)

var customDomainSchemaDescriptions = map[string]string{
	"id":              "Terraform's internal resource identifier. It is structured as \"`project_id`,`distribution_id`\".",
	"distribution_id": "CDN distribution ID",
	"project_id":      "STACKIT project ID associated with the distribution",
	"status":          "Status of the distribution",
	"errors":          "List of distribution errors",
}

type CustomDomainModel struct {
	ID             types.String `tfsdk:"id"`              // Required by Terraform
	DistributionId types.String `tfsdk:"distribution_id"` // DistributionID associated with the cdn distribution
	ProjectId      types.String `tfsdk:"project_id"`      // ProjectId associated with the cdn distribution
	Name           types.String `tfsdk:"name"`            // The custom domain
	Status         types.String `tfsdk:"status"`          // The status of the cdn distribution
	Errors         types.List   `tfsdk:"errors"`          // Any errors that the distribution has
}

type customDomainResource struct {
	client *cdn.APIClient
}

func NewCustomDomainResource() resource.Resource {
	return &customDomainResource{}
}

func (r *customDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_cdn_custom_domain", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := cdnUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "CDN client configured")
}

func (r *customDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_custom_domain"
}

func (r *customDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("CDN distribution data source schema."),
		Description:         "CDN distribution data source schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: customDomainSchemaDescriptions["id"],
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: customDomainSchemaDescriptions["name"],
				Required:    true,
				Optional:    false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"distribution_id": schema.StringAttribute{
				Description: customDomainSchemaDescriptions["distribution_id"],
				Required:    true,
				Optional:    false,
				Validators:  []validator.String{validate.UUID()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: customDomainSchemaDescriptions["project_id"],
				Required:    true,
				Optional:    false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: customDomainSchemaDescriptions["status"],
			},
			"errors": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: customDomainSchemaDescriptions["errors"],
			},
		},
	}
}

func (r *customDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model CustomDomainModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	distributionId := model.DistributionId.ValueString()
	ctx = tflog.SetField(ctx, "distribution_id", distributionId)
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "name", name)

	payload := cdn.PutCustomDomainPayload{IntentId: cdn.PtrString(uuid.NewString())}

	_, err := r.client.PutCustomDomain(ctx, projectId, distributionId, name).PutCustomDomainPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.CreateCDNCustomDomainWaitHandler(ctx, r.client, projectId, distributionId, name).SetTimeout(5 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Waiting for create: %v", err))
		return
	}

	err = mapCustomDomainFields(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "CDN custom domain created")
}

func (r *customDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model CustomDomainModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	distributionId := model.DistributionId.ValueString()
	ctx = tflog.SetField(ctx, "distribution_id", distributionId)
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "name", name)

	customDomainResp, err := r.client.GetCustomDomain(ctx, projectId, distributionId, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		// n.b. err is caught here if of type *oapierror.GenericOpenAPIError, which the stackit SDK client returns
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN custom domain", fmt.Sprintf("Calling API: %v", err))
		return
	}
	err = mapCustomDomainFields(customDomainResp.CustomDomain, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading CDN custom domain", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "CDN custom domain read")
}

func (r *customDomainResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called; custom domains have only computed fields and fields that require replacement when changed
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating CDN custom domain", "Custom domain cannot be updated")
}

func (r *customDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model CustomDomainModel
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	distributionId := model.DistributionId.ValueString()
	ctx = tflog.SetField(ctx, "distribution_id", distributionId)
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "name", name)

	_, err := r.client.DeleteCustomDomain(ctx, projectId, distributionId, name).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Delete CDN custom domain", fmt.Sprintf("Delete custom domain: %v", err))
	}
	_, err = wait.DeleteCDNCustomDomainWaitHandler(ctx, r.client, projectId, distributionId, name).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Delete CDN custom domain", fmt.Sprintf("Waiting for deletion: %v", err))
		return
	}
	tflog.Info(ctx, "CDN custom domain deleted")
}

func (r *customDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing CDN custom domain", fmt.Sprintf("Expected import identifier on the format: [project_id]%q[distribution_id]%q[custom_domain_name], got %q", core.Separator, core.Separator, req.ID))
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("distribution_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[2])...)
	tflog.Info(ctx, "CDN custom domain state imported")
}

func mapCustomDomainFields(customDomain *cdn.CustomDomain, model *CustomDomainModel) error {
	if customDomain == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if customDomain.Name == nil {
		return fmt.Errorf("Name is missing in response")
	}

	if customDomain.Status == nil {
		return fmt.Errorf("Status missing in response")
	}

	id := model.ProjectId.ValueString() + core.Separator + model.DistributionId.ValueString() + core.Separator + *customDomain.Name

	model.ID = types.StringValue(id)
	model.Status = types.StringValue(string(*customDomain.Status))

	customDomainErrors := []attr.Value{}
	if customDomain.Errors != nil {
		for _, e := range *customDomain.Errors {
			if e.En == nil {
				return fmt.Errorf("Error description missing")
			}
			customDomainErrors = append(customDomainErrors, types.StringValue(*e.En))
		}
	}
	modelErrors, diags := types.ListValue(types.StringType, customDomainErrors)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.Errors = modelErrors

	return nil
}
