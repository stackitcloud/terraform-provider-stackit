package vpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	iaasAlphaUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &vpcDatasource{}
	_ datasource.DataSourceWithConfigure = &vpcDatasource{}
)

type DatasourceModel struct {
	Model
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// NewVPCDatasource is a helper function to simplify the provider implementation.
func NewVPCDatasource() datasource.DataSource {
	return &vpcDatasource{}
}

// vpcDatasource is the datasource implementation.
type vpcDatasource struct {
	client *iaas.APIClient
}

// Metadata returns the datasource type name.
func (r *vpcDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

// Configure adds the provider configured client to the datasource.
func (r *vpcDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckExperimentEnabled(ctx, &providerData, features.VpcExperiment, "stackit_vpc", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.client = iaasAlphaUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "IaaS v2alpha client configured")
}

// Schema defines the schema for the datasource.
func (r *vpcDatasource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "VPC datasource schema."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.VpcExperiment, core.Datasource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource identifier. It is structured as \"`project_id`,`vpc_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the VPC is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the VPC.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the VPC.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(127),
				},
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *vpcDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, core.DefaultOperationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	projectId := model.ProjectId.ValueString()
	vpcId := model.VpcId.ValueString()

	ctx = core.InitProviderContext(ctx)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "vpc_id", vpcId)

	vpcResp, err := r.client.DefaultAPI.GetVPC(ctx, projectId, vpcId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading vpc",
			fmt.Sprintf("vpc with ID %q does not exist in project %q.", vpcId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, vpcResp, &model.Model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading vpc", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "VPC read")
}
