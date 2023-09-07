package ske

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &projectDataSource{}
)

// NewProjectDataSource is a helper function to simplify the provider implementation.
func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

// projectDataSource is the data source implementation.
type projectDataSource struct {
	client *ske.APIClient
}

// Metadata returns the resource type name.
func (r *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_project"
}

// Configure adds the provider configured client to the resource.
func (r *projectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected stackit.ProviderData, got %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}

	var apiClient *ske.APIClient
	var err error
	if providerData.SKECustomEndpoint != "" {
		apiClient, err = ske.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.SKECustomEndpoint),
		)
	} else {
		apiClient, err = ske.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		resp.Diagnostics.AddError("Could not Configure API Client", err.Error())
		return
	}

	tflog.Info(ctx, "SKE client configured")
	r.client = apiClient
}

// Schema defines the schema for the resource.
func (r *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT Project ID in which the kubernetes project is enabled.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var state Model
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := state.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	_, err := r.client.GetProject(ctx, projectId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Unable to read project", err.Error())
		return
	}
	state.Id = types.StringValue(projectId)
	state.ProjectId = types.StringValue(projectId)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "SKE project read")
}
