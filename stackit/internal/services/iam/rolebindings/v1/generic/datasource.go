package generic

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	secretsmanagerV1Alpha "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &RoleBindingDatasource[secretsmanagerV1Alpha.APIClient]{}
)

type DatasourceModel struct {
	Id           types.String        `tfsdk:"id"` // needed by TF
	Region       types.String        `tfsdk:"region"`
	ResourceId   types.String        `tfsdk:"resource_id"`
	RoleBindings []nestedRoleBinding `tfsdk:"role_bindings"`
}

type nestedRoleBinding struct {
	Role    types.String `tfsdk:"role"`
	Subject types.String `tfsdk:"subject"`
}

// RoleBindingDatasource is the resource implementation.
type RoleBindingDatasource[C any] struct {
	providerData core.ProviderData
	apiClient    *C

	ApiName      string // e.g. "iaas", "secretsmanager", ...
	ResourceType string // e.g. "instance", ...

	// callbacks for lifecyle handling
	ApiClientFactory func(context.Context, *core.ProviderData, *diag.Diagnostics) *C
	ExecReadRequest  func(ctx context.Context, client *C, region, resourceId string) ([]GenericRoleBindingResponse, error)
}

// Metadata returns the resource type name.
func (r *RoleBindingDatasource[C]) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_%s_role_bindings_v1", req.ProviderTypeName, r.ApiName, r.ResourceType)
}

// Configure adds the provider configured client to the resource.
func (r *RoleBindingDatasource[C]) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &providerData, features.IamExperiment, fmt.Sprintf("stackit_%s_%s_role_binding", r.ApiName, r.ResourceType), core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apiClient = r.ApiClientFactory(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s %s client configured", r.ApiName, r.ResourceType))
}

// Schema defines the schema for the resource.
func (r *RoleBindingDatasource[C]) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: features.AddExperimentDescription("IAM role binding datasource schema.", features.IamExperiment, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. It is structured as \"`region`,`resource_id`\".",
				Computed:    true,
			},
			"resource_id": schema.StringAttribute{
				Description: "The identifier of the resource to get the role bindings for.",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// the region cannot be found automatically, so it has to be passed
				Description: "The resource region. If not defined, the provider region is used.",
			},
			"role_bindings": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of role bindings.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							Computed:    true,
							Description: "A valid role defined for the resource.",
						},
						"subject": schema.StringAttribute{
							Computed:    true,
							Description: "Identifier of user, service account or client. Usually email address or name in case of clients.",
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *RoleBindingDatasource[C]) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DatasourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	region := r.providerData.GetRegionWithOverride(model.Region)
	resourceId := model.ResourceId.ValueString()

	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "resource_id", resourceId)

	roleBindingResp, err := r.ExecReadRequest(ctx, r.apiClient, region, resourceId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error reading %s %s role bindings", r.ApiName, r.ResourceType), fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapDatasourceFields(roleBindingResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error reading %s %s role bindings", r.ApiName, r.ResourceType), fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s %s role bindings read", r.ApiName, r.ResourceType))
}

func mapDatasourceFields(resp []GenericRoleBindingResponse, model *DatasourceModel, region string) error {
	if model == nil {
		return fmt.Errorf("nil model")
	}

	model.Id = utils.BuildInternalTerraformId(region, model.ResourceId.ValueString())
	model.Region = types.StringValue(region)

	model.RoleBindings = make([]nestedRoleBinding, len(resp))
	for i, roleBinding := range resp {
		model.RoleBindings[i] = nestedRoleBinding{
			Role:    types.StringValue(roleBinding.GetRole()),
			Subject: types.StringValue(roleBinding.GetSubject()),
		}
	}

	return nil
}
