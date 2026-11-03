package members

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
	_ datasource.DataSource = &memberDataSource{}
)

type MemberItem struct {
	Id   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

type Model struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	GroupId        types.String `tfsdk:"group_id"`
	Members        []MemberItem `tfsdk:"members"`
}

func NewGroupMemberDataSource() datasource.DataSource {
	return &memberDataSource{}
}

type memberDataSource struct {
	client identity.DefaultAPI
}

func (r *memberDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	_, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = clients.IdentityV1Client
	tflog.Info(ctx, "Identity Group Member data source client configured")
}

func (r *memberDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_group_members"
}

func (r *memberDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets the list of members for a STACKIT Identity group.",
		Description:         "Gets the list of members for a STACKIT Identity group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID, structured as \"`organization_id`,`group_id`\".",
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
			"group_id": schema.StringAttribute{
				Description: "The internal UUID of the group.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"members": schema.ListNestedAttribute{
				Description: "List of members associated with the group.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the member (user, service account, or group).",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the member (e.g., user, service_account, group).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *memberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	orgId := model.OrganizationId.ValueString()
	groupId := model.GroupId.ValueString()

	membersResp, err := r.client.ListGroupMembers(ctx, orgId, groupId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading identity group members",
			fmt.Sprintf("Failed to read members for group %q in organization %q.", groupId, orgId),
			map[int]string{404: "Group not found or forbidden access."},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	model.Id = types.StringValue(fmt.Sprintf("%s,%s", orgId, groupId))

	var members []MemberItem
	for _, m := range membersResp.Members {
		id := ""
		if m.Id != nil {
			id = *m.Id
		}
		memType := ""
		if m.Type != nil {
			memType = string(*m.Type)
		}
		members = append(members, MemberItem{
			Id:   types.StringValue(id),
			Type: types.StringValue(memType),
		})
	}
	model.Members = members

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
