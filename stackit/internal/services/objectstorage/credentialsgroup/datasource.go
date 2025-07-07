package objectstorage

import (
	"context"
	"fmt"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	objectstorageUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &credentialsGroupDataSource{}
)

// NewCredentialsGroupDataSource is a helper function to simplify the provider implementation.
func NewCredentialsGroupDataSource() datasource.DataSource {
	return &credentialsGroupDataSource{}
}

// credentialsGroupDataSource is the data source implementation.
type credentialsGroupDataSource struct {
	client       *objectstorage.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *credentialsGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_credentials_group"
}

// Configure adds the provider configured client to the data source.
func (r *credentialsGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := objectstorageUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "ObjectStorage credentials group client configured")
}

// Schema defines the schema for the data source.
func (r *credentialsGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                 "ObjectStorage credentials group data source schema. Must have a `region` specified in the provider configuration.",
		"id":                   "Terraform's internal data source identifier. It is structured as \"`project_id`,`region`,`credentials_group_id`\".",
		"credentials_group_id": "The credentials group ID.",
		"name":                 "The credentials group's display name.",
		"project_id":           "Object Storage Project ID to which the credentials group is associated.",
		"urn":                  "Credentials group uniform resource name (URN)",
		"region":               "The resource region. If not defined, the provider region is used.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"credentials_group_id": schema.StringAttribute{
				Description: descriptions["credentials_group_id"],
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"urn": schema.StringAttribute{
				Computed:    true,
				Description: descriptions["urn"],
			},
			"region": schema.StringAttribute{
				// the region cannot be found automatically, so it has to be passed
				Optional:    true,
				Description: descriptions["region"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialsGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_group_id", credentialsGroupId)
	ctx = tflog.SetField(ctx, "region", region)

	found, err := readCredentialsGroups(ctx, &model, region, r.client)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credentials group", fmt.Sprintf("getting credential group from list of credentials groups: %v", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credentials group", fmt.Sprintf("Credentials group with ID %q does not exists in project %q", credentialsGroupId, projectId))
		return
	}

	// update the region attribute manually, as it is not contained in the
	// server response
	model.Region = types.StringValue(region)

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage credentials group read")
}
