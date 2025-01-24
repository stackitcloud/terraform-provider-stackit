package affinitygroup

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
	"net/http"
	"regexp"
	"strings"
)

// affinityGroupResourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var affinityGroupResourceBetaCheckDone bool

var (
	_ resource.Resource                = &affinityGroupResource{}
	_ resource.ResourceWithConfigure   = &affinityGroupResource{}
	_ resource.ResourceWithImportState = &affinityGroupResource{}
)

// Model is the provider's internal model
type Model struct {
	Id              types.String `tfsdk:"id"`
	ProjectId       types.String `tfsdk:"project_id"`
	AffinityGroupId types.String `tfsdk:"affinity_group_id"`
	Name            types.String `tfsdk:"name"`
	Policy          types.String `tfsdk:"policy"`
	Members         types.List   `tfsdk:"members"`
}

func NewAffinityGroupResource() resource.Resource {
	return &affinityGroupResource{}
}

// affinityGroupResource is the resource implementation.
type affinityGroupResource struct {
	client *iaas.APIClient
}

// Metadata returns the resource type name.
func (r *affinityGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_affinity_group"
}

// Configure adds the provider configured client to the resource.
func (r *affinityGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderDate, got %T", req.ProviderData))
		return
	}

	if !affinityGroupResourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_affinity_group", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		affinityGroupResourceBetaCheckDone = true
	}

	var apiClient *iaas.APIClient
	var err error
	if providerData.IaaSCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "iaas_custom_endpoint", providerData.IaaSCustomEndpoint)
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuratio", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

func (r *affinityGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Affinity Group schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddBetaDescription(description + "\n\n" + exampleUsageWithServer + policies),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. It is structured as \"`project_id`,`bar_id`\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT Project ID to which the affinity group is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"affinity_group_id": schema.StringAttribute{
				Description: "The affinity group ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the affinity group.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+((-|_|\s|\.)[A-Za-z0-9]+)*$`),
						"must match expression"),
				},
			},
			"policy": schema.StringAttribute{
				Description: "The policy of the affinity group.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{},
			},
			"members": schema.ListAttribute{
				Description: "The servers that are part of the affinity group.",
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						validate.UUID(),
					),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *affinityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Create new affinityGroup
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating affinity group", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	affinityGroupResp, err := r.client.CreateAffinityGroup(ctx, projectId).CreateAffinityGroupPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating affinity group", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "affinity_group_id", affinityGroupResp.Id)

	// Map response body to schema
	err = mapFields(ctx, affinityGroupResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating affinity group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Affinity group created")
}

// Read refreshes the Terraform state with the latest data.
func (r *affinityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	affinityGroupId := model.AffinityGroupId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "affinity_group_id", affinityGroupId)

	affinityGroupResp, err := r.client.GetAffinityGroupExecute(ctx, projectId, affinityGroupId)
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading affinity group", fmt.Sprintf("Call API: %v", err))
		return
	}

	err = mapFields(ctx, affinityGroupResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading affinity group", fmt.Sprintf("Processing API payload: %v", err))
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Affinity group read")
}

func (r *affinityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	//TODO implement me
	panic("implement me")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *affinityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	affinityGroupId := model.AffinityGroupId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "affinity_group_id", affinityGroupId)

	// Delete existing affinity group
	err := r.client.DeleteAffinityGroupExecute(ctx, projectId, affinityGroupId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting affinity group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Affinity group deleted")
}

func (r *affinityGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing affinity group",
			fmt.Sprintf("Expected import indentifier with format: [project_id],[affinity_group_id], got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	affinityGroupId := idParts[1]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "affinity_group_id", affinityGroupId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("affinity_group_id"), affinityGroupId)...)
	tflog.Info(ctx, "affinity group state imported")
}

func toCreatePayload(model *Model) (*iaas.CreateAffinityGroupPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	nameValue := conversion.StringValueToPointer(model.Name)
	policyValue := conversion.StringValueToPointer(model.Policy)

	return &iaas.CreateAffinityGroupPayload{
		Name:   nameValue,
		Policy: policyValue,
	}, nil
}

func mapFields(ctx context.Context, affinityGroupResp *iaas.AffinityGroup, model *Model) error {
	if affinityGroupResp == nil {
		return fmt.Errorf("response input is nil")
	}

	var affinityGroupId string
	if model.AffinityGroupId.ValueString() != "" {
		affinityGroupId = model.AffinityGroupId.ValueString()
	} else if affinityGroupResp.Id != nil {
		affinityGroupId = *affinityGroupResp.Id
	} else {
		return fmt.Errorf("affinity group id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		affinityGroupId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	members, diags := types.ListValueFrom(ctx, types.StringType, []string{})
	if diags.HasError() {
		return fmt.Errorf("convert members to StringValue list: %w", core.DiagsToError(diags))
	}
	if affinityGroupResp.Members != nil && len(*affinityGroupResp.Members) > 0 {
		members, diags = types.ListValueFrom(ctx, types.StringType, *affinityGroupResp.Members)
		if diags.HasError() {
			return fmt.Errorf("convert members to StringValue list: %w", core.DiagsToError(diags))
		}
	} else if model.Members.IsNull() {
		members = types.ListNull(types.StringType)
	}

	model.AffinityGroupId = types.StringValue(affinityGroupId)

	model.Name = types.StringPointerValue(affinityGroupResp.Name)
	model.Policy = types.StringPointerValue(affinityGroupResp.Policy)
	model.Members = members

	return nil
}
