package group

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
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
)

// Model maps the Terraform state to the resource schema.
type Model struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	GroupId        types.String `tfsdk:"group_id"`
	DisplayName    types.String `tfsdk:"display_name"`
	Description    types.String `tfsdk:"description"`
}

func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResource struct {
	client identity.DefaultAPI
}

func (r *groupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	_, clients, ok := core.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.client = clients.IdentityV1Client

	tflog.Info(ctx, "Identity Group client configured")
}

func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_group"
}

func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"id":              "Terraform's internal resource ID, structured as \"`organization_id`,`group_id`\".",
		"organization_id": "STACKIT Organization ID to which the group is associated.",
		"group_id":        "The internal UUID of the group.",
		"display_name":    "Display name of the group.",
		"description":     "Description of the group.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a STACKIT Identity Group in an organization.",
		Description:         "Manages a STACKIT Identity Group in an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   descriptions["id"],
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				Description: descriptions["organization_id"],
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
				Description:   descriptions["group_id"],
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Optional:    true,
			},
		},
	}
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	orgId := model.OrganizationId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", orgId)

	payload := identity.CreateGroupPayload{
		DisplayName: model.DisplayName.ValueString(),
	}
	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		desc := model.Description.ValueString()
		payload.Description = &desc
	}

	groupResp, err := r.client.CreateGroup(ctx, orgId).CreateGroupPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating identity group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	model.Id = types.StringValue(fmt.Sprintf("%s,%s", orgId, *groupResp.Id))
	model.GroupId = types.StringValue(*groupResp.Id)
	if groupResp.DisplayName != nil {
		model.DisplayName = types.StringValue(*groupResp.DisplayName)
	}
	if groupResp.Description != nil {
		model.Description = types.StringValue(*groupResp.Description)
	}

	time.Sleep(5 * time.Second)

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Identity group created")
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	orgId := model.OrganizationId.ValueString()
	groupId := model.GroupId.ValueString()

	groupResp, err := r.client.GetGroup(ctx, orgId, groupId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading identity group", fmt.Sprintf("Error calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if groupResp.DisplayName != nil {
		model.DisplayName = types.StringValue(*groupResp.DisplayName)
	}
	if groupResp.Description != nil {
		model.Description = types.StringValue(*groupResp.Description)
	} else {
		model.Description = types.StringNull()
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	orgId := model.OrganizationId.ValueString()
	groupId := model.GroupId.ValueString()

	payload := identity.UpdateGroupPayload{
		DisplayName: model.DisplayName.ValueString(),
		Description: model.Description.ValueString(),
	}

	updateResp, err := r.client.UpdateGroup(ctx, orgId, groupId).UpdateGroupPayload(payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating identity group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	if updateResp.DisplayName != nil {
		model.DisplayName = types.StringValue(*updateResp.DisplayName)
	}
	if updateResp.Description != nil {
		model.Description = types.StringValue(*updateResp.Description)
	} else {
		model.Description = types.StringNull()
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
	orgId := model.OrganizationId.ValueString()
	groupId := model.GroupId.ValueString()

	err := r.client.DeleteGroup(ctx, orgId, groupId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting identity group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "Identity group deleted")
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing identity group",
			fmt.Sprintf("Expected import identifier with format: [organization_id],[group_id]  Got: %q", req.ID),
		)
		return
	}

	orgId := idParts[0]
	groupId := idParts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), orgId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupId)...)
	tflog.Info(ctx, "Identity group state imported")
}

func toCreatePayload(model *Model) (*identity.CreateGroupPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payload := &identity.CreateGroupPayload{
		DisplayName: model.DisplayName.ValueString(),
	}

	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		desc := model.Description.ValueString()
		payload.Description = &desc
	}

	return payload, nil
}

func mapFields(resp *identity.GetGroupResponse, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	orgId := model.OrganizationId.ValueString()

	if resp.Id != nil {
		model.Id = types.StringValue(fmt.Sprintf("%s,%s", orgId, *resp.Id))
		model.GroupId = types.StringValue(*resp.Id)
	}

	if resp.DisplayName != nil {
		model.DisplayName = types.StringValue(*resp.DisplayName)
	}

	if resp.Description != nil {
		model.Description = types.StringValue(*resp.Description)
	} else {
		model.Description = types.StringNull()
	}

	return nil
}
