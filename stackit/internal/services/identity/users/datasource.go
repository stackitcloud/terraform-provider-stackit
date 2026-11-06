package users

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
	_ datasource.DataSource = &usersDataSource{}
)

type Item struct {
	UserId      types.String `tfsdk:"user_id"`
	Email       types.String `tfsdk:"email"`
	DisplayName types.String `tfsdk:"display_name"`
}

type Model struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	EmailRegex     types.String `tfsdk:"email_regex"`
	SortAscending  types.Bool   `tfsdk:"sort_ascending"`
	Items          []Item       `tfsdk:"items"`
}

func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

type usersDataSource struct {
	client identity.DefaultAPI
}

func (r *usersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	_, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = clients.IdentityV1Client
	tflog.Info(ctx, "Identity Users (plural) client configured")
}

func (r *usersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_users"
}

func (r *usersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists STACKIT Identity users in an organization, optionally filtered by email.",
		Description:         "Lists STACKIT Identity users in an organization, optionally filtered by email.",
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
			"email_regex": schema.StringAttribute{
				Description: "Optional regular expression to filter users by email.",
				Optional:    true,
			},
			"sort_ascending": schema.BoolAttribute{
				Description: "If set to `true`, users are sorted in ascending lexicographical order by email. Defaults to `false` (descending).",
				Optional:    true,
			},
			"items": schema.ListNestedAttribute{
				Description: "The list of identity users matching the provided filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id": schema.StringAttribute{
							Description: "The internal UUID of the user.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "Email of the user.",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "Display name of the user.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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
	if !model.EmailRegex.IsNull() && model.EmailRegex.ValueString() != "" {
		compiledRegex, err = regexp.Compile(model.EmailRegex.ValueString())
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Invalid email_regex", err.Error())
			return
		}
	}

	listUsersResp, err := r.client.ListUsers(ctx, orgId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading identity users",
			fmt.Sprintf("Failed to read users in organization %q.", orgId),
			map[int]string{},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapUsersDataSourceFields(listUsersResp.Users, &model, compiledRegex)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading identity users", fmt.Sprintf("Error processing API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

func mapUsersDataSourceFields(apiItems []identity.GetUserResponse, model *Model, compiledRegex *regexp.Regexp) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var matchedItems []Item

	for _, u := range apiItems {
		email := ""
		if u.Email != nil {
			email = *u.Email
		}

		if compiledRegex != nil && !compiledRegex.MatchString(email) {
			continue
		}

		item := Item{
			UserId: types.StringValue(*u.Id),
			Email:  types.StringValue(email),
		}

		if u.DisplayName != nil {
			item.DisplayName = types.StringValue(*u.DisplayName)
		} else {
			item.DisplayName = types.StringNull()
		}

		matchedItems = append(matchedItems, item)
	}

	sortAsc := false
	if !model.SortAscending.IsNull() && !model.SortAscending.IsUnknown() {
		sortAsc = model.SortAscending.ValueBool()
	}

	sort.SliceStable(matchedItems, func(i, j int) bool {
		emailA := matchedItems[i].Email.ValueString()
		emailB := matchedItems[j].Email.ValueString()
		if sortAsc {
			return emailA < emailB
		}
		return emailA > emailB
	})

	model.Id = model.OrganizationId
	model.Items = matchedItems

	return nil
}
