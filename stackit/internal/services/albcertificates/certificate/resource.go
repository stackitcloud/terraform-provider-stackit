package certificate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
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
	certSdk "github.com/stackitcloud/stackit-sdk-go/services/certificates/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	certUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/albcertificates/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &certificatesResource{}
	_ resource.ResourceWithConfigure   = &certificatesResource{}
	_ resource.ResourceWithImportState = &certificatesResource{}
	_ resource.ResourceWithModifyPlan  = &certificatesResource{}
)

// DataSourceModel - Base fields shared by both Resource and Data Source
type DataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	CertID    types.String `tfsdk:"cert_id"`
	Name      types.String `tfsdk:"name"`
	PublicKey types.String `tfsdk:"public_key"`
}

// Model - For Resource includes the PrivateKey
type Model struct {
	DataSourceModel
	PrivateKey types.String `tfsdk:"private_key"`
}

// NewCertificatesResource is a helper function to simplify the provider implementation.
func NewCertificatesResource() resource.Resource {
	return &certificatesResource{}
}

// certificatesResource is the resource implementation.
type certificatesResource struct {
	client       *certSdk.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *certificatesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alb_certificate"
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *certificatesResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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
func (r *certificatesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := certUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Certificate client configured")
}

// Schema defines the schema for the resource.
func (r *certificatesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "Certificates resource schema.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`cert_id`\".",
		"project_id":  "STACKIT project ID to which the certificate is associated.",
		"region":      "The resource region (e.g. eu01). If not defined, the provider region is used.",
		"cert_id":     "The ID of the certificate.",
		"name":        "Certificate name.",
		"private_key": "The PEM encoded private key part",
		"public_key":  "The PEM encoded public key part",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		MarkdownDescription: `
## Setting up supporting infrastructure` + "\n" + `

The example below creates the supporting infrastructure using the STACKIT Terraform provider, including the automatic creation of a TLS certificate resource.
`,
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
				Validators: []validator.String{
					validate.UUID(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?$`),
						"1-63 characters [0-9] & [a-z] also [-] but not at the beginning or end",
					),
				},
			},
			"cert_id": schema.StringAttribute{
				Description: descriptions["cert_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_key": schema.StringAttribute{
				Description: descriptions["private_key"],
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(8192),
				},
			},
			"public_key": schema.StringAttribute{
				Description: descriptions["public_key"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(8192),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *certificatesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Certificate", fmt.Sprintf("Payload for create: %v", err))
		return
	}

	// Create a new Certificate
	createResp, err := r.client.DefaultAPI.CreateCertificate(ctx, projectId, region).CreateCertificatePayload(*payload).Execute()
	if err != nil {
		errStr := utils.PrettyApiErr(ctx, &resp.Diagnostics, err)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Certificate", fmt.Sprintf("Calling API for create: %v", errStr))
		return
	}
	ctx = core.LogResponse(ctx)

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id": projectId,
		"cert_id":    *createResp.Id,
		"region":     region,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	// Map response body to schema
	err = mapFields(createResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Certificate", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Certificate created")
}

// Read refreshes the Terraform state with the latest data.
func (r *certificatesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	certId := model.CertID.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "cert_id", certId)

	readResp, err := r.client.DefaultAPI.GetCertificate(ctx, projectId, region, certId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		errStr := utils.PrettyApiErr(ctx, &resp.Diagnostics, err)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Certificate", fmt.Sprintf("Calling API: %v", errStr))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(readResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Certificate", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Certificate read")
}

func (r *certificatesResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating certificate", "Certificates can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *certificatesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	certId := model.CertID.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "cert_id", certId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete Certificate
	_, err := r.client.DefaultAPI.DeleteCertificate(ctx, projectId, region, certId).Execute()
	if err != nil {
		errStr := utils.PrettyApiErr(ctx, &resp.Diagnostics, err)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Certificate", fmt.Sprintf("Calling API for delete: %v", errStr))
		return
	}

	ctx = core.LogResponse(ctx)

	tflog.Info(ctx, "Certificate deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id, region, cert_id
func (r *certificatesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing Certificate",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[cert_id]  Got: %q", req.ID),
		)
		return
	}

	ctx = utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]interface{}{
		"project_id": idParts[0],
		"region":     idParts[1],
		"cert_id":    idParts[2],
	})
	tflog.Info(ctx, "Certificate state imported")
}

// toCreatePayload and all other toX functions in this file turn a Terraform Certificate model into a createCertificate to be used with the Certificate API.
func toCreatePayload(model *Model) (*certSdk.CreateCertificatePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	return &certSdk.CreateCertificatePayload{
		Name:       conversion.StringValueToPointer(model.Name),
		PrivateKey: conversion.StringValueToPointer(model.PrivateKey),
		PublicKey:  conversion.StringValueToPointer(model.PublicKey),
	}, nil
}

func mapFields(cert *certSdk.GetCertificateResponse, m *Model, region string) error {
	if m == nil {
		return fmt.Errorf("model input is nil")
	}
	return mapDataFields(cert, &m.DataSourceModel, region)
}

// mapFields and all other map functions in this file translate an API resource into a Terraform model.
func mapDataFields(cert *certSdk.GetCertificateResponse, m *DataSourceModel, region string) error {
	if cert == nil {
		return fmt.Errorf("response input is nil")
	}
	if m == nil {
		return fmt.Errorf("model input is nil")
	}

	var certID string
	if m.CertID.ValueString() != "" {
		certID = m.CertID.ValueString()
	} else if cert.Id != nil {
		certID = *cert.Id
	} else {
		return fmt.Errorf("cert ID not present")
	}
	m.Region = types.StringValue(region)
	m.CertID = types.StringValue(certID)
	m.Id = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), m.Region.ValueString(), certID)
	m.Name = types.StringPointerValue(cert.Name)
	m.PublicKey = types.StringPointerValue(cert.PublicKey)

	return nil
}
