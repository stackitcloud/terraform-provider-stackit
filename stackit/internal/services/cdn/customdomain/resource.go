package cdn

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	cdnUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/cdn/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	"main":        "The TLS certificate for the custom domain. If omitted, a managed certificate will be used.",
	"type":        "The type of certificate. Can be `managed` or `custom`.",
	"certificate": "The PEM-encoded TLS certificate. Required if `type` is `custom`.",
	"private_key": "The PEM-encoded private key for the certificate. Required if `type` is `custom`.",
	"version":     "A version identifier for the certificate. The certificate will be updated if this field is changed.",
}

var certificateTypes = map[string]attr.Type{
	"type":        types.StringType,
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
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: certificateSchemaDescriptions["type"],
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("managed", "custom"),
						},
					},
					"certificate": schema.StringAttribute{
						Description: certificateSchemaDescriptions["certificate"],
						Optional:    true,
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
	certificate, diags := buildCertificatePayload(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := cdn.PutCustomDomainPayload{
		IntentId:    cdn.PtrString(uuid.NewString()),
		Certificate: certificate,
	}
	respPutCustomDomain, err := r.client.PutCustomDomain(ctx, projectId, distributionId, name).PutCustomDomainPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.CreateCDNCustomDomainWaitHandler(ctx, r.client, projectId, distributionId, name).SetTimeout(5 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating CDN custom domain", fmt.Sprintf("Waiting for create: %v", err))
		return
	}

	err = mapCustomDomainFields(waitResp, &model, respPutCustomDomain.Certificate)
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
	err = mapCustomDomainFields(customDomainResp.CustomDomain, &model, customDomainResp.Certificate)
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

func normalizeCertificate(certInput interface{}) (Certificate, error) {
	var customCert *cdn.GetCustomDomainCustomCertificate
	var managedCert *cdn.GetCustomDomainManagedCertificate

	// We use a type switch to safely extract the inner certificates
	switch c := certInput.(type) {
	case cdn.GetCustomDomainResponseGetCertificateAttributeType:
		if c == nil {
			return Certificate{}, errors.New("input of type GetCustomDomainResponseCertificate is nil")
		}
		customCert = c.GetCustomDomainCustomCertificate
		managedCert = c.GetCustomDomainManagedCertificate

	case cdn.PutCustomDomainResponseGetCertificateAttributeType:
		if c == nil {
			return Certificate{}, errors.New("input of type PutCustomDomainResponseCertificate is nil")
		}
		customCert = c.GetCustomDomainCustomCertificate
		managedCert = c.GetCustomDomainManagedCertificate

	default:
		return Certificate{}, fmt.Errorf("unsupported input type: %T", certInput)
	}

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

// pemToBase64 decodes a PEM-formatted string, extracts the raw data,
// and re-encodes it into a standard single-line Base64 string.
func pemToBase64(pemString string) (string, error) {
	// pem.Decode will find the first PEM formatted block (e.g. -----BEGIN...-----)
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block from the provided string")
	}
	// The block.Bytes field contains the raw data without headers or footers.
	// We re-encode this raw data to standard Base64.
	return base64.StdEncoding.EncodeToString(block.Bytes), nil
}

// buildCertificatePayload constructs the certificate part of the payload for the API request.
// It handles both "managed" and "custom" certificate types based on the Terraform model.
func buildCertificatePayload(ctx context.Context, model *CustomDomainModel) (*cdn.PutCustomDomainPayloadCertificate, diag.Diagnostics) {
	var diags diag.Diagnostics

	// If the certificate block is not specified, default to a managed certificate.
	if model.Certificate.IsNull() {
		managedCert := cdn.NewPutCustomDomainManagedCertificate("managed")
		certPayload := cdn.PutCustomDomainManagedCertificateAsPutCustomDomainPayloadCertificate(managedCert)
		return &certPayload, diags
	}

	// Define a temporary struct to map the Terraform object attributes.
	var certModel struct {
		Type        types.String `tfsdk:"type"`
		Certificate types.String `tfsdk:"certificate"`
		PrivateKey  types.String `tfsdk:"private_key"`
		Version     types.Int64  `tfsdk:"version"`
	}

	// Unpack the Terraform object into the temporary struct.
	respDiags := model.Certificate.As(ctx, &certModel, basetypes.ObjectAsOptions{})
	diags.Append(respDiags...)
	if diags.HasError() {
		return nil, diags
	}

	var certPayload cdn.PutCustomDomainPayloadCertificate
	certType := certModel.Type.ValueString()

	switch certType {
	case "managed":
		// Create a payload for a managed certificate.
		managedCert := cdn.NewPutCustomDomainManagedCertificate("managed")
		certPayload = cdn.PutCustomDomainManagedCertificateAsPutCustomDomainPayloadCertificate(managedCert)

	case "custom":
		// For a custom certificate, validate that the certificate and private key are provided.
		if certModel.Certificate.IsNull() || certModel.Certificate.ValueString() == "" {
			diags.AddAttributeError(
				path.Root("certificate").AtName("certificate"),
				"Missing Certificate",
				"The 'certificate' attribute is required when the certificate type is 'custom'.",
			)
		}
		if certModel.PrivateKey.IsNull() || certModel.PrivateKey.ValueString() == "" {
			diags.AddAttributeError(
				path.Root("certificate").AtName("private_key"),
				"Missing Private Key",
				"The 'private_key' attribute is required when the certificate type is 'custom'.",
			)
		}
		if diags.HasError() {
			return nil, diags
		}

		// Process the PEM strings to get raw Base64 strings for the SDK.
		certBase64, err := pemToBase64(certModel.Certificate.ValueString())
		if err != nil {
			diags.AddAttributeError(
				path.Root("certificate").AtName("certificate"),
				"Invalid Certificate Format",
				"The provided certificate is not a valid PEM-encoded string.",
			)
		}

		keyBase64, err := pemToBase64(certModel.PrivateKey.ValueString())
		if err != nil {
			diags.AddAttributeError(
				path.Root("certificate").AtName("private_key"),
				"Invalid Private Key Format",
				"The provided private key is not a valid PEM-encoded string.",
			)
		}

		if diags.HasError() {
			return nil, diags
		}

		// Create a payload for a custom certificate using the processed Base64 strings.
		customCert := cdn.NewPutCustomDomainCustomCertificate(
			certBase64,
			keyBase64,
			"custom",
		)
		certPayload = cdn.PutCustomDomainCustomCertificateAsPutCustomDomainPayloadCertificate(customCert)

	default:
		// This case should ideally be caught by the schema validator, but it's good practice to handle it.
		diags.AddAttributeError(
			path.Root("certificate").AtName("type"),
			"Invalid Certificate Type",
			fmt.Sprintf("Expected 'managed' or 'custom', but got: %s", certType),
		)
		return nil, diags
	}

	return &certPayload, diags
}

func mapCustomDomainFields(customDomain *cdn.CustomDomain, model *CustomDomainModel, certificateInput interface{}) error {
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
	certificate, err := normalizeCertificate(certificateInput)
	if err != nil {
		return fmt.Errorf("Certificate error in nomalizer: %s", err)
	}
	var diags diag.Diagnostics
	certAttributes := map[string]attr.Value{
		"type":        types.StringValue(certificate.Type),
		"version":     types.Int64Null(),
		"certificate": types.StringNull(),
		"private_key": types.StringNull(),
	}
	if certificate.Version != nil {
		certAttributes["version"] = types.Int64Value(*certificate.Version)
	}
	certificateObj, conversionDiags := types.ObjectValue(certificateTypes, certAttributes)
	diags.Append(conversionDiags...)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	model.Certificate = certificateObj
	model.ID = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.DistributionId.ValueString(), *customDomain.Name)
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
