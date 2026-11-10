package user

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
	_ datasource.DataSource = &userDataSource{}
)

// Model maps the Terraform state to the datasource schema.
type Model struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	UserId         types.String `tfsdk:"user_id"`
	Email          types.String `tfsdk:"email"`
	DisplayName    types.String `tfsdk:"display_name"`
}

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

type userDataSource struct {
	client identity.DefaultAPI
}

func (r *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	_, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = clients.IdentityV1Client
	tflog.Info(ctx, "Identity User client configured")
}

func (r *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_user"
}

func (r *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets details about a STACKIT Identity user in an organization.",
		Description:         "Gets details about a STACKIT Identity user in an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID, structured as \"`organization_id`,`user_id`\".",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT Organization ID to which the user belongs.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "The internal UUID of the user to look up.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"email": schema.StringAttribute{
				Description: "Email address of the user.",
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "Display name of the user.",
				Computed:    true,
			},
		},
	}
}

func (r *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	orgId := model.OrganizationId.ValueString()
	userId := model.UserId.ValueString()

	userResp, err := r.client.GetUser(ctx, orgId, userId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading identity user",
			fmt.Sprintf("Failed to read user %q in organization %q.", userId, orgId),
			map[int]string{
				404: "User not found or forbidden access.",
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	model.Id = types.StringValue(fmt.Sprintf("%s,%s", orgId, userId))

	if userResp.Email != nil {
		model.Email = types.StringValue(*userResp.Email)
	} else {
		model.Email = types.StringNull()
	}

	if userResp.DisplayName != nil {
		model.DisplayName = types.StringValue(*userResp.DisplayName)
	} else {
		model.DisplayName = types.StringNull()
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
