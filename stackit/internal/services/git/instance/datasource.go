package instance

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	gitUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/git/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/git"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &gitDataSource{}
)

// NewGitDataSource creates a new instance of the gitDataSource.
func NewGitDataSource() datasource.DataSource {
	return &gitDataSource{}
}

// gitDataSource is the datasource implementation.
type gitDataSource struct {
	client *git.APIClient
}

// Configure sets up the API client for the git instance resource.
func (g *gitDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_git", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := gitUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	g.client = apiClient
	tflog.Info(ctx, "git client configured")
}

// Metadata provides metadata for the git datasource.
func (g *gitDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_git"
}

// Schema defines the schema for the git data source.
func (g *gitDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Git Instance datasource schema."),
		Description:         "Git Instance datasource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"acl": schema.ListAttribute{
				Description: descriptions["acl"],
				Computed:    true,
				ElementType: types.StringType,
			},
			"consumed_disk": schema.StringAttribute{
				Description: descriptions["consumed_disk"],
				Computed:    true,
			},
			"consumed_object_storage": schema.StringAttribute{
				Description: descriptions["consumed_object_storage"],
				Computed:    true,
			},
			"created": schema.StringAttribute{
				Description: descriptions["created"],
				Computed:    true,
			},
			"flavor": schema.StringAttribute{
				Description: descriptions["flavor"],
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: descriptions["url"],
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Computed:    true,
			},
		},
	}
}

func (g *gitDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract the project ID and instance id of the model
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()

	// Read the current git instance via id
	gitInstanceResp, err := g.client.GetInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading git instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, gitInstanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading git instance", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, fmt.Sprintf("read git instance %s", instanceId))
}
