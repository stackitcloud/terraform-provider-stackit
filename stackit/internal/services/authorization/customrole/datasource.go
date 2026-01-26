package customrole

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	authorizationUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/authorization/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &customRoleDataSource{}
)

// NewAuthorizationDataSource creates a new customrole of the authorizationDataSource.
func NewCustomRoleDataSource() datasource.DataSource {
	return &customRoleDataSource{}
}

// NewProjectRoleAssignmentDataSources is a helper function generate custom role
// data sources for all possible resource types.
func NewCustomRoleDataSources() []func() datasource.DataSource {
	resources := make([]func() datasource.DataSource, 0)
	for _, v := range resourceTypes {
		resources = append(resources, func() datasource.DataSource {
			return &customRoleDataSource{
				resourceType: v,
			}
		})
	}

	return resources
}

// customRoleDataSource is the datasource implementation.
type customRoleDataSource struct {
	resourceType string
	client       *authorization.APIClient
}

// Configure sets up the API client for the authorization customrole resource.
func (d *customRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := authorizationUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	d.client = apiClient

	tflog.Info(ctx, "authorization client configured")
}

// Metadata provides metadata for the custom role datasource.
func (d *customRoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_authorization_%s_custom_role", req.ProviderTypeName, d.resourceType)
}

// Schema defines the schema for the custom role data source.
func (d *customRoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"role_id": schema.StringAttribute{
				Description: descriptions["role_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: descriptions["resource_id"],
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
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Computed:    true,
			},
			"permissions": schema.ListAttribute{
				ElementType: types.StringType,
				Description: descriptions["permissions"],
				Computed:    true,
			},
		},
	}
}

func (d *customRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := model.ResourceId.ValueString()
	roleId := model.RoleId.ValueString()

	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "resource_id", resourceId)
	ctx = tflog.SetField(ctx, "role_id", roleId)

	roleResp, err := d.client.GetRole(ctx, d.resourceType, resourceId, roleId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError

		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading custom role", fmt.Sprintf("Calling API: %v", err))

		return
	}

	ctx = core.LogResponse(ctx)

	if err = mapGetCustomRoleResponse(ctx, roleResp, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading custom role", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	// Set the updated state.
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, fmt.Sprintf("read custom role %s", roleId))
}
