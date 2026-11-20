package groups

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	identity "github.com/stackitcloud/stackit-sdk-go/services/identity/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &groupsDataSource{}
)

// GroupItem represents a single identity group inside the list.
type GroupItem struct {
	GroupId     types.String `tfsdk:"group_id"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
}

// Model represents the Model for the plural data source.
type Model struct {
	Id               types.String `tfsdk:"id"`
	OrganizationId   types.String `tfsdk:"organization_id"`
	DisplayNameRegex types.String `tfsdk:"display_name_regex"`
	SortAscending    types.Bool   `tfsdk:"sort_ascending"`
	Items            []GroupItem  `tfsdk:"items"`
}

func NewGroupsDataSource() datasource.DataSource {
	return &groupsDataSource{}
}

type groupsDataSource struct {
	client identity.DefaultAPI
}

func (r *groupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	_, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = clients.IdentityV1Client
	tflog.Info(ctx, "Identity Groups (plural) client configured")
}

func (r *groupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_groups"
}

func (r *groupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists STACKIT Identity groups in an organization, optionally filtered by display name.",
		Description:         "Lists STACKIT Identity groups in an organization, optionally filtered by display name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID, structured as \"`organization_id`\".",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT Organization ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"display_name_regex": schema.StringAttribute{
				Description: "Optional regular expression to filter groups by display name.",
				Optional:    true,
			},
			"sort_ascending": schema.BoolAttribute{
				Description: "If set to `true`, groups are sorted in ascending lexicographical order by display name. Defaults to `false` (descending).",
				Optional:    true,
			},
			"items": schema.ListNestedAttribute{
				Description: "The list of identity groups matching the provided filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.StringAttribute{
							Description: "The internal UUID of the group.",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "Display name of the group.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the group.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *groupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	orgId := model.OrganizationId.ValueString()

	var compiledRegex *regexp.Regexp
	var err error
	if !model.DisplayNameRegex.IsNull() && model.DisplayNameRegex.ValueString() != "" {
		compiledRegex, err = regexp.Compile(model.DisplayNameRegex.ValueString())
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Invalid display_name_regex", err.Error())
			return
		}
	}

	listGroupsResp, err := r.client.ListGroups(ctx, orgId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading identity groups",
			fmt.Sprintf("Failed to read groups in organization %q.", orgId),
			map[int]string{},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapGroupsDataSourceFields(listGroupsResp.Groups, &model, compiledRegex)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading identity groups", fmt.Sprintf("Error processing API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

func mapGroupsDataSourceFields(apiItems []identity.GetGroupResponse, model *Model, compiledRegex *regexp.Regexp) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var matchedItems []GroupItem

	for _, group := range apiItems {
		name := ""
		if group.DisplayName != nil {
			name = *group.DisplayName
		}

		if compiledRegex != nil && !compiledRegex.MatchString(name) {
			continue
		}

		item := GroupItem{
			GroupId:     types.StringValue(*group.Id),
			DisplayName: types.StringValue(name),
		}

		if group.Description != nil {
			item.Description = types.StringValue(*group.Description)
		} else {
			item.Description = types.StringNull()
		}

		matchedItems = append(matchedItems, item)
	}

	sortAsc := false
	if !model.SortAscending.IsNull() && !model.SortAscending.IsUnknown() {
		sortAsc = model.SortAscending.ValueBool()
	}

	sort.SliceStable(matchedItems, func(i, j int) bool {
		nameA := matchedItems[i].DisplayName.ValueString()
		nameB := matchedItems[j].DisplayName.ValueString()
		if sortAsc {
			return nameA < nameB
		}
		return nameA > nameB
	})

	model.Id = model.OrganizationId
	model.Items = matchedItems

	return nil
}
