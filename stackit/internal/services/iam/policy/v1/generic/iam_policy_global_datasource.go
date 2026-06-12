package generic

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	resourcemanagerV1Beta "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &globalIAMPolicyDatasource[resourcemanagerV1Beta.APIClient]{}
	_ datasource.DataSourceWithConfigure = &globalIAMPolicyDatasource[resourcemanagerV1Beta.APIClient]{}
)

// NewDatasource allows creating a IAM policy datasource implementation from a IAM policy resource definition
func NewDatasource[C any](r GlobalIAMPolicyResource[C]) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &globalIAMPolicyDatasource[C]{
			datasourceName: r.GetResourceName(),
			experimental:   r.Experimental,

			apiName:      r.ApiName,
			resourceType: r.ResourceType,

			apiClientFactory: r.ApiClientFactory,
			execReadRequest:  r.ExecReadRequest,
		}
	}
}

// globalIAMPolicyDatasource is the datasource implementation.
type globalIAMPolicyDatasource[C any] struct {
	apiClient C

	// datasourceName is the name of the datasource, e.g. "stackit_resourcemanager_folder_iam_policy_v1"
	datasourceName string
	// Defines whether the datasource should be marked as experimental. Should be the case for all IAM policy resources as of now.
	experimental bool

	apiName      string // e.g. "iaas", "secretsmanager", ...
	resourceType string // e.g. "instance", ...

	// callbacks for lifecyle handling
	apiClientFactory func(context.Context, *core.ProviderData, *diag.Diagnostics) C
	execReadRequest  func(ctx context.Context, client C, resourceId string) (*IAMPolicy, error)
}

// Metadata returns the datasource type name.
func (d *globalIAMPolicyDatasource[C]) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.datasourceName
}

// Configure adds the provider configured client to the datasource.
func (d *globalIAMPolicyDatasource[C]) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if d.experimental {
		features.CheckExperimentEnabled(ctx, &providerData, features.IamExperiment, d.datasourceName, core.Datasource, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	d.apiClient = d.apiClientFactory(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s %s client configured", d.apiName, d.resourceType))
}

// Schema defines the schema for the datasource.
func (d *globalIAMPolicyDatasource[C]) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "IAM policy datasource schema."

	if d.experimental {
		description = features.AddExperimentDescription(description, features.IamExperiment, core.Resource)
	}

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal datasource identifier. It is structured as \"`resource_id`\".",
				Computed:    true,
			},
			"resource_id": schema.StringAttribute{
				Description: "The identifier of the resource to read the IAM policy from.",
				Required:    true,
			},
			"role_bindings": schema.ListNestedAttribute{
				Description: "The role bindings of the policy.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							Description: "A valid role defined for the resource.",
							Computed:    true,
						},
						"subject": schema.StringAttribute{
							Description: "Identifier of user, service account or client. Usually email address or name in case of clients.",
							Computed:    true,
						},
					},
				},
			},
			"etag": schema.StringAttribute{
				Description: "Internal etag used for API concurrency control.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *globalIAMPolicyDatasource[C]) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model GlobalModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	resourceId := model.ResourceId.ValueString()

	ctx = tflog.SetField(ctx, "resource_id", resourceId)

	iamPolicy, err := d.execReadRequest(ctx, d.apiClient, resourceId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error reading %s %s IAM policy", d.apiName, d.resourceType), fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFieldsGlobal(iamPolicy, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, fmt.Sprintf("Error reading %s %s IAM policy", d.apiName, d.resourceType), fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	model.Etag = types.StringPointerValue(iamPolicy.Etag)

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("%s %s IAM policy read", d.apiName, d.resourceType))
}
