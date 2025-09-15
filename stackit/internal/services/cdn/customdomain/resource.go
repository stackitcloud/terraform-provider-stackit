package cdn

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
var certificateSchemaDescriptions = map[string]string{
	"main":        "The TLS certificate for the custom domain. If omitted, a managed certificate will be used. If the block is specified, a custom certificate is used.",
	"certificate": "The PEM-encoded TLS certificate. Required for custom certificates.",
	"private_key": "The PEM-encoded private key for the certificate. Required for custom certificates. The certificate will be updated if this field is changed.",
	"version":     "A version identifier for the certificate. Required for custom certificates. The certificate will be updated if this field is changed.",
}

var certificateTypes = map[string]attr.Type{
	"version":     types.Int64Type,
	"certificate": types.StringType,
	"private_key": types.StringType,
}

var customDomainSchemaDescriptions = map[string]string{
	"id":              "Terraform's internal resource identifier. It is structured as \"`project_id`,`distribution_id`\".",
	"distribution_id": "CDN distribution ID",
	"project_id":      "STACKIT project ID associated with the distribution",
	"status":          "Status of the distribution",
	"errors":          "List of distribution errors",
}

type CertificateModel struct {
	Certificate types.String `tfsdk:"certificate"`
	PrivateKey  types.String `tfsdk:"private_key"`
	Version     types.Int64  `tfsdk:"version"`
}

type CustomDomainModel struct {
	ID             types.String `tfsdk:"id"`              // Required by Terraform
	DistributionId types.String `tfsdk:"distribution_id"` // DistributionID associated with the cdn distribution
	ProjectId      types.String `tfsdk:"project_id"`      // ProjectId associated with the cdn distribution
	Name           types.String `tfsdk:"name"`            // The custom domain
	Status         types.String `tfsdk:"status"`          // The status of the cdn distribution
	Errors         types.List   `tfsdk:"errors"`          // Any errors that the distribution has
	Certificate    types.Object `tfsdk:"certificate"`     // the certificate of the custom domain
}

type customDomainResource struct {
	client *cdn.APIClient
}

func NewCustomDomainResource() resource.Resource {
	return &customDomainResource{}
}

type Certificate struct {
	Type    string
	Version *int64
}

func (r *customDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_cdn_custom_domain", "resource")
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
		MarkdownDescription: features.AddBetaDescription("CDN distribution data source schema.", core.Resource),
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
			"certificate": schema.SingleNestedAttribute{
				Description: certificateSchemaDescriptions["main"],
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"certificate": schema.StringAttribute{
						Description: certificateSchemaDescriptions["certificate"],
						Optional:    true,
						Sensitive:   true,
					},
					"private_key": schema.StringAttribute{
						Description: certificateSchemaDescriptions["private_key"],
						Optional:    true,
						Sensitive:   true,
					},
					"version": schema.Int64Attribute{
						Description: certificateSchemaDescriptions["version"],
						Computed:    true,
					},
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
	certificate, err := buildCertificatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	payload := cdn.PutCustomDomainPayload{
		IntentId:    cdn.PtrString(uuid.NewString()),
		Certificate: certificate,
	}
	_, err = r.client.PutCustomDomain(ctx, projectId, distributionId, name).PutCustomDomainPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.CreateCDNCustomDomainWaitHandler(ctx, r.client, projectId, distributionId, name).SetTimeout(5 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Waiting for create: %v", err))
		return
	}

	respCustomDomain, err := r.client.GetCustomDomainExecute(ctx, projectId, distributionId, name)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Calling API: %v", err))
		return
	}
	err = mapCustomDomainResourceFields(respCustomDomain, &model)
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
	err = mapCustomDomainResourceFields(customDomainResp, &model)
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

func (r *customDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
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

	certificate, err := buildCertificatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	payload := cdn.PutCustomDomainPayload{
		IntentId:    cdn.PtrString(uuid.NewString()),
		Certificate: certificate,
	}
	_, err = r.client.PutCustomDomain(ctx, projectId, distributionId, name).PutCustomDomainPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating CDN custom domain certificate", fmt.Sprintf("Calling API: %v", err))
		return
	}

	_, err = wait.CreateCDNCustomDomainWaitHandler(ctx, r.client, projectId, distributionId, name).SetTimeout(5 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating CDN custom domain certificate", fmt.Sprintf("Waiting for update: %v", err))
		return
	}

	respCustomDomain, err := r.client.GetCustomDomainExecute(ctx, projectId, distributionId, name)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating CDN custom domain certificate", fmt.Sprintf("Calling API to read final state: %v", err))
		return
	}
	err = mapCustomDomainResourceFields(respCustomDomain, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating CDN custom domain certificate", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "CDN custom domain certificate updated")
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

func normalizeCertificate(certInput cdn.GetCustomDomainResponseGetCertificateAttributeType) (Certificate, error) {
	var customCert *cdn.GetCustomDomainCustomCertificate
	var managedCert *cdn.GetCustomDomainManagedCertificate

	if certInput == nil {
		return Certificate{}, errors.New("input of type GetCustomDomainResponseCertificate is nil")
	}
	customCert = certInput.GetCustomDomainCustomCertificate
	managedCert = certInput.GetCustomDomainManagedCertificate

	// Now we process the extracted certificates
	if customCert != nil && customCert.Type != nil && customCert.Version != nil {
		return Certificate{
			Type:    *customCert.Type,
			Version: customCert.Version, // Converts from *int64 to int
		}, nil
	}

	if managedCert != nil && managedCert.Type != nil {
		// The version will be the zero value for int (0), as requested
		return Certificate{
			Type: *managedCert.Type,
		}, nil
	}

	return Certificate{}, errors.New("certificate structure is empty, neither custom nor managed is set")
}

// buildCertificatePayload constructs the certificate part of the payload for the API request.
// It defaults to a managed certificate if the certificate block is omitted, otherwise it creates a custom certificate.
func buildCertificatePayload(ctx context.Context, model *CustomDomainModel) (*cdn.PutCustomDomainPayloadCertificate, error) {
	// If the certificate block is not specified, default to a managed certificate.
	if model.Certificate.IsNull() {
		managedCert := cdn.NewPutCustomDomainManagedCertificate("managed")
		certPayload := cdn.PutCustomDomainManagedCertificateAsPutCustomDomainPayloadCertificate(managedCert)
		return &certPayload, nil
	}

	var certModel CertificateModel
	// Unpack the Terraform object into the temporary struct.
	respDiags := model.Certificate.As(ctx, &certModel, basetypes.ObjectAsOptions{})
	if respDiags.HasError() {
		return nil, fmt.Errorf("invalid certificate or private key: %w", core.DiagsToError(respDiags))
	}
	certStr := base64.StdEncoding.EncodeToString([]byte(certModel.Certificate.ValueString()))
	keyStr := base64.StdEncoding.EncodeToString([]byte(certModel.PrivateKey.ValueString()))

	if certStr == "" || keyStr == "" {
		return nil, errors.New("invalid certificate or private key. Please check if the string of the public certificate and private key in PEM format")
	}

	customCert := cdn.NewPutCustomDomainCustomCertificate(
		certStr,
		keyStr,
		"custom",
	)
	certPayload := cdn.PutCustomDomainCustomCertificateAsPutCustomDomainPayloadCertificate(customCert)

	return &certPayload, nil
}

func mapCustomDomainResourceFields(customDomainResponse *cdn.GetCustomDomainResponse, model *CustomDomainModel) error {
	if customDomainResponse == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if customDomainResponse.CustomDomain.Name == nil {
		return fmt.Errorf("name is missing in response")
	}

	if customDomainResponse.CustomDomain.Status == nil {
		return fmt.Errorf("status missing in response")
	}
	normalizedCert, err := normalizeCertificate(customDomainResponse.Certificate)
	if err != nil {
		return fmt.Errorf("Certificate error in normalizer: %w", err)
	}

	// If the certificate is managed, the certificate block in the state should be null.
	if normalizedCert.Type == "managed" {
		model.Certificate = types.ObjectNull(certificateTypes)
	} else {
		// If the certificate is custom, we need to preserve the user-configured
		// certificate and private key from the plan/state, and only update the computed version.
		certAttributes := map[string]attr.Value{
			"certificate": types.StringNull(), // Default to null
			"private_key": types.StringNull(), // Default to null
			"version":     types.Int64Null(),
		}

		// Get existing values from the model's certificate object if it exists
		if !model.Certificate.IsNull() {
			existingAttrs := model.Certificate.Attributes()
			if val, ok := existingAttrs["certificate"]; ok {
				certAttributes["certificate"] = val
			}
			if val, ok := existingAttrs["private_key"]; ok {
				certAttributes["private_key"] = val
			}
		}

		// Set the computed version from the API response
		if normalizedCert.Version != nil {
			certAttributes["version"] = types.Int64Value(*normalizedCert.Version)
		}

		certificateObj, diags := types.ObjectValue(certificateTypes, certAttributes)
		if diags.HasError() {
			return fmt.Errorf("failed to map certificate: %w", core.DiagsToError(diags))
		}
		model.Certificate = certificateObj
	}

	model.ID = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.DistributionId.ValueString(), *customDomainResponse.CustomDomain.Name)
	model.Status = types.StringValue(string(*customDomainResponse.CustomDomain.Status))

	customDomainErrors := []attr.Value{}
	if customDomainResponse.CustomDomain.Errors != nil {
		for _, e := range *customDomainResponse.CustomDomain.Errors {
			if e.En == nil {
				return fmt.Errorf("error description missing")
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
