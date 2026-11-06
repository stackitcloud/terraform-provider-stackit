package group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	identity "github.com/stackitcloud/stackit-sdk-go/services/identity/v1api"
)

var (
	_ datasource.DataSource = &groupDataSource{}
)

// NewGroupDataSource creates a new instance of the groupDataSource.
func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

// groupDataSource is the datasource implementation for identity groups.
type groupDataSource struct {
	client identity.DefaultAPI
}

// Configure initializes the groupDataSource with the provided provider data.
func (r *groupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	_, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.client = clients.IdentityV1Client

	tflog.Info(ctx, "Identity Group client configured")
}

// Metadata provides metadata for the group datasource.
func (r *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_group"
}

// Schema defines the schema for the group data source.
func (r *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"id":              "Terraform's internal resource ID, structured as \"`organization_id`,`group_id`\".",
		"organization_id": "STACKIT Organization ID to which the group is associated.",
		"group_id":        "The internal UUID of the group to look up.",
		"display_name":    "Display name of the group.",
		"description":     "Description of the group's purpose.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets details about a STACKIT Identity group in an organization.",
		Description:         "Gets details about a STACKIT Identity group in an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: descriptions["organization_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: descriptions["group_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Computed:    true,
			},
		},
	}
}

// Read reads the group from the API and updates the state.
func (r *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	orgId := model.OrganizationId.ValueString()
	groupId := model.GroupId.ValueString()

	// Call the Identity API to get the specific group
	groupResp, err := r.client.GetGroup(ctx, orgId, groupId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading identity group",
			fmt.Sprintf("Failed to read group %q in organization %q.", groupId, orgId),
			map[int]string{
				404: "Group not found or forbidden access.",
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	model.Id = types.StringValue(fmt.Sprintf("%s,%s", orgId, groupId))

	if groupResp.DisplayName != nil {
		model.DisplayName = types.StringValue(*groupResp.DisplayName)
	} else {
		model.DisplayName = types.StringNull()
	}

	if groupResp.Description != nil {
		model.Description = types.StringValue(*groupResp.Description)
	} else {
		model.Description = types.StringNull()
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
