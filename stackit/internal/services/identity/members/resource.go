package members

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	identity "github.com/stackitcloud/stackit-sdk-go/services/identity/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ resource.Resource                = &memberResource{}
	_ resource.ResourceWithConfigure   = &memberResource{}
	_ resource.ResourceWithImportState = &memberResource{}
)

func NewGroupMemberResource() resource.Resource {
	return &memberResource{}
}

type memberResource struct {
	client identity.DefaultAPI
}

func (r *memberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	_, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = clients.IdentityV1Client
	tflog.Info(ctx, "Identity Group Member resource client configured")
}

func (r *memberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_group_members"
}

func (r *memberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the list of members for a STACKIT Identity Group.",
		Description:         "Manages the list of members for a STACKIT Identity Group. This resource will overwrite any existing members on the group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Terraform's internal resource ID, structured as \"`organization_id`,`group_id`\".",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT Organization ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "The internal UUID of the group.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"members": schema.ListNestedAttribute{
				Description: "List of members associated with the group.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the member (user, service account, or group).",
							Required:    true,
							Validators: []validator.String{
								validate.UUID(),
							},
						},
						"type": schema.StringAttribute{
							Description: "The type of the member. Valid values are `user`, `service_account`, `group`.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("user", "service_account", "group"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *memberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	orgId := model.OrganizationId.ValueString()
	groupId := model.GroupId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", orgId)
	ctx = tflog.SetField(ctx, "group_id", groupId)

	payload := toUpdatePayload(model.Members)
	_, err := r.client.UpdateGroupMembers(ctx, orgId, groupId).UpdateGroupMembersPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating identity group members", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	model.Id = types.StringValue(fmt.Sprintf("%s,%s", orgId, groupId))
	time.Sleep(5 * time.Second)

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Identity group members created")
}

func (r *memberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	orgId := model.OrganizationId.ValueString()
	groupId := model.GroupId.ValueString()

	membersResp, err := r.client.ListGroupMembers(ctx, orgId, groupId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading identity group members", fmt.Sprintf("Error calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	mapMembersResponse(membersResp, &model)

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

func (r *memberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	orgId := model.OrganizationId.ValueString()
	groupId := model.GroupId.ValueString()

	payload := toUpdatePayload(model.Members)
	_, err := r.client.UpdateGroupMembers(ctx, orgId, groupId).UpdateGroupMembersPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating identity group members", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

func (r *memberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	orgId := model.OrganizationId.ValueString()
	groupId := model.GroupId.ValueString()

	// Deletion clears all members using an empty slice
	payload := identity.UpdateGroupMembersPayload{
		Members: []identity.UpdateMemberRequest{},
	}

	_, err := r.client.UpdateGroupMembers(ctx, orgId, groupId).UpdateGroupMembersPayload(payload).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting identity group members", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "Identity group members deleted (cleared)")
}

func (r *memberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing identity group members",
			fmt.Sprintf("Expected import identifier with format: [organization_id],[group_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), idParts[1])...)
	tflog.Info(ctx, "Identity group members state imported")
}

func toUpdatePayload(members []MemberItem) identity.UpdateGroupMembersPayload {
	reqs := make([]identity.UpdateMemberRequest, 0, len(members))
	for _, m := range members {
		reqs = append(reqs, identity.UpdateMemberRequest{
			Id:   m.Id.ValueString(),
			Type: identity.UpdateMemberRequestType(m.Type.ValueString()),
		})
	}
	return identity.UpdateGroupMembersPayload{
		Members: reqs,
	}
}

func mapMembersResponse(resp *identity.MembersResponse, model *Model) {
	var members []MemberItem
	for _, m := range resp.Members {
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
}
