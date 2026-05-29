package link

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	telemetrylink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1betaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1betaapi/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/telemetrylink/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

const (
	resourceTypeOrganization = "organization"
	resourceTypeFolder       = "folder"
	resourceTypeProject      = "project"
)

var (
	_ resource.Resource                = &telemetryLinkResource{}
	_ resource.ResourceWithConfigure   = &telemetryLinkResource{}
	_ resource.ResourceWithImportState = &telemetryLinkResource{}
	_ resource.ResourceWithModifyPlan  = &telemetryLinkResource{}

	resourceTypes = []string{resourceTypeOrganization, resourceTypeFolder, resourceTypeProject}
)

var schemaDescriptions = map[string]string{
	"id":      "Terraform's internal resource identifier. It is structured as \"`resource_type`, `resource_id`,`region`\".",
	"link_id": "The TelemetryLink ID",
	"region":  "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"resource_type": fmt.Sprintf(
		"The resource type of the TelemetryLink resource, possible values: %s",
		tfutils.FormatPossibleValues(resourceTypes...),
	),
	"resource_id":         "STACKIT project ID, folder ID, or organization ID associated with the Telemetry Link resource.",
	"display_name":        "The displayed name of the Telemetry Link resource.",
	"description":         "The description of the Telemetry Link resource.",
	"telemetry_router_id": "The Telemetry Router ID.",
	"access_token":        "The access token of the Telemetry Router instance.",
	"create_time":         "The time the Telemetry Link was created.",
	"status": fmt.Sprintf(
		"The status of the TelemetryLink, possible values: %s",
		tfutils.FormatPossibleValues("active", "inactive", "failed"),
	),
}

type Model struct {
	ID                types.String `tfsdk:"id"` // Required by Terraform
	LinkID            types.String `tfsdk:"link_id"`
	Region            types.String `tfsdk:"region"`
	ResourceType      types.String `tfsdk:"resource_type"`
	ResourceID        types.String `tfsdk:"resource_id"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	TelemetryRouterID types.String `tfsdk:"telemetry_router_id"`
	AccessToken       types.String `tfsdk:"access_token"`
	CreateTime        types.String `tfsdk:"create_time"`
	Status            types.String `tfsdk:"status"`
}

type telemetryLinkResource struct {
	client       *telemetrylink.APIClient
	providerData core.ProviderData
}

func NewTelemetryLinkResource() resource.Resource {
	return &telemetryLinkResource{}
}

func (r *telemetryLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := utils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	r.providerData = providerData
	tflog.Info(ctx, "TelemetryLink client configured")
}

func (r *telemetryLinkResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tfutils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *telemetryLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_telemetrylink"
}

func (r *telemetryLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("TelemetryLink instance resource schema. %s", core.ResourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"link_id": schema.StringAttribute{
				Description: schemaDescriptions["link_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type": schema.StringAttribute{
				Description: schemaDescriptions["resource_type"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(resourceTypes...),
					validate.NoSeparator(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: schemaDescriptions["resource_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"telemetry_router_id": schema.StringAttribute{
				Description: schemaDescriptions["telemetry_router_id"],
				Required:    true,
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Optional:    true,
				// must be computed to allow for storing the override value from the provider
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_token": schema.StringAttribute{
				Description: schemaDescriptions["access_token"],
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_time": schema.StringAttribute{
				Description: schemaDescriptions["create_time"],
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
		},
	}
}

func (r *telemetryLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	resourceType := model.ResourceType.ValueString()
	resourceID := model.ResourceID.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "resource_type", resourceType)
	ctx = tflog.SetField(ctx, "resource_id", resourceID)
	ctx = tflog.SetField(ctx, "region", region)

	regionId := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "region", regionId)

	var response *telemetrylink.TelemetryLinkResponse
	switch model.ResourceType.ValueString() {
	case resourceTypeOrganization:
		payload, err := toCreateOrUpdateOrganizationTelemetryLinkPayload(ctx, resp.Diagnostics, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Creating API payload: %v", err))
			return
		}

		createResp, err := r.client.DefaultAPI.CreateOrUpdateOrganizationTelemetryLink(ctx, resourceID, regionId).CreateOrUpdateOrganizationTelemetryLinkPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Calling API: %v", err))
			return
		}

		ctx = core.LogResponse(ctx)

		if createResp == nil || createResp.Id == "" {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", "Create API response: Incomplete response (id missing)")
			return
		}

		// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
		ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
			"resource_type": resourceType,
			"resource_id":   resourceID,
			"region":        region,
		})
		if resp.Diagnostics.HasError() {
			return
		}

		response, err = wait.CreateOrUpdateOrganizationTelemetryLinkWaitHandler(ctx, r.client.DefaultAPI, resourceID, regionId).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Waiting for TelemetryLink to become active: %v", err))
			return
		}

	case resourceTypeFolder:
		payload, err := toCreateOrUpdateFolderTelemetryLinkPayload(ctx, resp.Diagnostics, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Creating API payload: %v", err))
			return
		}

		createResp, err := r.client.DefaultAPI.CreateOrUpdateFolderTelemetryLink(ctx, resourceID, regionId).CreateOrUpdateFolderTelemetryLinkPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Calling API: %v", err))
			return
		}

		ctx = core.LogResponse(ctx)

		if createResp == nil || createResp.Id == "" {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", "Create API response: Incomplete response (id missing)")
			return
		}

		// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
		ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
			"resource_type": resourceType,
			"resource_id":   resourceID,
			"region":        region,
		})
		if resp.Diagnostics.HasError() {
			return
		}

		response, err = wait.CreateOrUpdateFolderTelemetryLinkWaitHandler(ctx, r.client.DefaultAPI, resourceID, regionId).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Waiting for TelemetryLink to become active: %v", err))
			return
		}
	case resourceTypeProject:
		payload, err := toCreateOrUpdateProjectTelemetryLinkPayload(ctx, resp.Diagnostics, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Creating API payload: %v", err))
			return
		}

		createResp, err := r.client.DefaultAPI.CreateOrUpdateProjectTelemetryLink(ctx, resourceID, regionId).CreateOrUpdateProjectTelemetryLinkPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Calling API: %v", err))
			return
		}

		ctx = core.LogResponse(ctx)

		if createResp == nil || createResp.Id == "" {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", "Create API response: Incomplete response (id missing)")
			return
		}

		// Write id attributes to state before polling via the wait handler - just in case anything goes wrong during the wait handler
		ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
			"resource_type": resourceType,
			"resource_id":   resourceID,
			"region":        region,
		})
		if resp.Diagnostics.HasError() {
			return
		}

		response, err = wait.CreateOrUpdateProjectTelemetryLinkWaitHandler(ctx, r.client.DefaultAPI, resourceID, regionId).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Waiting for TelemetryLink to become active: %v", err))
			return
		}
	default:
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Unsupported resource type: %s", model.ResourceType.ValueString()))
		return
	}

	err := mapFields(ctx, response, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating TelemetryLink", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryLink created")
}

func (r *telemetryLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	resourceType := model.ResourceType.ValueString()
	resourceID := model.ResourceID.ValueString()
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "resource_type", resourceType)
	ctx = tflog.SetField(ctx, "resource_id", resourceID)
	ctx = tflog.SetField(ctx, "region", region)

	var err error
	var response *telemetrylink.TelemetryLinkResponse
	switch resourceType {
	case resourceTypeOrganization:
		response, err = r.client.DefaultAPI.GetOrganizationTelemetryLink(ctx, resourceID, region).Execute()
	case resourceTypeFolder:
		response, err = r.client.DefaultAPI.GetFolderTelemetryLink(ctx, resourceID, region).Execute()
	case resourceTypeProject:
		response, err = r.client.DefaultAPI.GetProjectTelemetryLink(ctx, resourceID, region).Execute()
	default:
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryLink", fmt.Sprintf("Unsupported resource type: %s", model.ResourceType.ValueString()))
		return
	}
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryLink", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, response, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryLink", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryLink read", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

func (r *telemetryLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	resourceType := model.ResourceType.ValueString()
	resourceID := model.ResourceID.ValueString()
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "resource_type", resourceType)
	ctx = tflog.SetField(ctx, "resource_id", resourceID)
	ctx = tflog.SetField(ctx, "region", region)

	regionId := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "region", regionId)

	var response *telemetrylink.TelemetryLinkResponse
	switch model.ResourceType.ValueString() {
	case resourceTypeOrganization:
		payload, err := toPartialUpdateOrganizationTelemetryLinkPayload(ctx, resp.Diagnostics, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Creating API payload: %v", err))
			return
		}

		_, err = r.client.DefaultAPI.PartialUpdateOrganizationTelemetryLink(ctx, resourceID, regionId).PartialUpdateOrganizationTelemetryLinkPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Calling API: %v", err))
			return
		}

		ctx = core.LogResponse(ctx)

		response, err = wait.PartialUpdateOrganizationTelemetryLinkWaitHandler(ctx, r.client.DefaultAPI, resourceID, regionId).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Waiting for TelemetryLink to become active: %v", err))
			return
		}
	case resourceTypeFolder:
		payload, err := toPartialUpdateFolderTelemetryLinkPayload(ctx, resp.Diagnostics, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Creating API payload: %v", err))
			return
		}

		_, err = r.client.DefaultAPI.PartialUpdateFolderTelemetryLink(ctx, resourceID, regionId).PartialUpdateFolderTelemetryLinkPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Calling API: %v", err))
			return
		}

		ctx = core.LogResponse(ctx)

		response, err = wait.PartialUpdateFolderTelemetryLinkWaitHandler(ctx, r.client.DefaultAPI, resourceID, regionId).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Waiting for TelemetryLink to become active: %v", err))
			return
		}
	case resourceTypeProject:
		payload, err := toPartialUpdateProjectTelemetryLinkPayload(ctx, resp.Diagnostics, &model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Creating API payload: %v", err))
			return
		}

		_, err = r.client.DefaultAPI.PartialUpdateProjectTelemetryLink(ctx, resourceID, regionId).PartialUpdateProjectTelemetryLinkPayload(*payload).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Calling API: %v", err))
			return
		}

		ctx = core.LogResponse(ctx)

		response, err = wait.PartialUpdateProjectTelemetryLinkWaitHandler(ctx, r.client.DefaultAPI, resourceID, regionId).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Waiting for TelemetryLink to become active: %v", err))
			return
		}
	default:
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Unsupported resource type: %s", model.ResourceType.ValueString()))
		return
	}

	err := mapFields(ctx, response, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating TelemetryLink", fmt.Sprintf("Processing response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryLink updated", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

func (r *telemetryLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	resourceType := model.ResourceType.ValueString()
	resourceID := model.ResourceID.ValueString()
	region := model.Region.ValueString()

	ctx = tflog.SetField(ctx, "resource_type", resourceType)
	ctx = tflog.SetField(ctx, "resource_id", resourceID)
	ctx = tflog.SetField(ctx, "region", region)

	var err error
	switch resourceType {
	case resourceTypeOrganization:
		err = r.client.DefaultAPI.DeleteOrganizationTelemetryLink(ctx, resourceID, region).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryLink", fmt.Sprintf("Calling API: %v", err))
			return
		}

		ctx = core.LogResponse(ctx)

		_, err = wait.DeleteOrganizationTelemetryLinkWaitHandler(ctx, r.client.DefaultAPI, resourceID, region).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryLink", fmt.Sprintf("Waiting for TelemetryLink to become deleted: %v", err))
			return
		}
	case resourceTypeFolder:
		err = r.client.DefaultAPI.DeleteFolderTelemetryLink(ctx, resourceID, region).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryLink", fmt.Sprintf("Calling API: %v", err))
			return
		}

		ctx = core.LogResponse(ctx)

		_, err = wait.DeleteFolderTelemetryLinkWaitHandler(ctx, r.client.DefaultAPI, resourceID, region).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryLink", fmt.Sprintf("Waiting for TelemetryLink to become deleted: %v", err))
			return
		}
	case resourceTypeProject:
		err = r.client.DefaultAPI.DeleteProjectTelemetryLink(ctx, resourceID, region).Execute()
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryLink", fmt.Sprintf("Calling API: %v", err))
			return
		}

		ctx = core.LogResponse(ctx)

		_, err = wait.DeleteProjectTelemetryLinkWaitHandler(ctx, r.client.DefaultAPI, resourceID, region).WaitWithContext(ctx)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryLink", fmt.Sprintf("Waiting for TelemetryLink to become deleted: %v", err))
			return
		}
	default:
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting TelemetryLink", fmt.Sprintf("Unsupported resource type: %s", resourceType))
		return
	}

	tflog.Info(ctx, "TelemetryLink deleted")
}

func (r *telemetryLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing TelemetryLink", fmt.Sprintf("Invalid import ID %q: expected format is `project_id`,`region`,`instance_id`", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_type"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[2])...)
	tflog.Info(ctx, "TelemetryLink state imported")
}

func toPartialUpdateOrganizationTelemetryLinkPayload(_ context.Context, _ diag.Diagnostics, model *Model) (*telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	return &telemetrylink.PartialUpdateOrganizationTelemetryLinkPayload{
		DisplayName:       model.DisplayName.ValueStringPointer(),
		Description:       model.Description.ValueStringPointer(),
		TelemetryRouterId: model.TelemetryRouterID.ValueStringPointer(),
		AccessToken:       model.AccessToken.ValueStringPointer(),
	}, nil
}

func toPartialUpdateFolderTelemetryLinkPayload(_ context.Context, _ diag.Diagnostics, model *Model) (*telemetrylink.PartialUpdateFolderTelemetryLinkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	return &telemetrylink.PartialUpdateFolderTelemetryLinkPayload{
		DisplayName:       model.DisplayName.ValueStringPointer(),
		Description:       model.Description.ValueStringPointer(),
		TelemetryRouterId: model.TelemetryRouterID.ValueStringPointer(),
		AccessToken:       model.AccessToken.ValueStringPointer(),
	}, nil
}

func toPartialUpdateProjectTelemetryLinkPayload(_ context.Context, _ diag.Diagnostics, model *Model) (*telemetrylink.PartialUpdateProjectTelemetryLinkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	return &telemetrylink.PartialUpdateProjectTelemetryLinkPayload{
		DisplayName:       model.DisplayName.ValueStringPointer(),
		Description:       model.Description.ValueStringPointer(),
		TelemetryRouterId: model.TelemetryRouterID.ValueStringPointer(),
		AccessToken:       model.AccessToken.ValueStringPointer(),
	}, nil
}

func toCreateOrUpdateOrganizationTelemetryLinkPayload(_ context.Context, _ diag.Diagnostics, model *Model) (*telemetrylink.CreateOrUpdateOrganizationTelemetryLinkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	return &telemetrylink.CreateOrUpdateOrganizationTelemetryLinkPayload{
		DisplayName:       model.DisplayName.ValueString(),
		Description:       model.Description.ValueStringPointer(),
		TelemetryRouterId: model.TelemetryRouterID.ValueString(),
		AccessToken:       model.AccessToken.ValueString(),
	}, nil
}

func toCreateOrUpdateFolderTelemetryLinkPayload(_ context.Context, _ diag.Diagnostics, model *Model) (*telemetrylink.CreateOrUpdateFolderTelemetryLinkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	return &telemetrylink.CreateOrUpdateFolderTelemetryLinkPayload{
		DisplayName:       model.DisplayName.ValueString(),
		Description:       model.Description.ValueStringPointer(),
		TelemetryRouterId: model.TelemetryRouterID.ValueString(),
		AccessToken:       model.AccessToken.ValueString(),
	}, nil
}

func toCreateOrUpdateProjectTelemetryLinkPayload(_ context.Context, _ diag.Diagnostics, model *Model) (*telemetrylink.CreateOrUpdateProjectTelemetryLinkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("missing model")
	}

	return &telemetrylink.CreateOrUpdateProjectTelemetryLinkPayload{
		DisplayName:       model.DisplayName.ValueString(),
		Description:       model.Description.ValueStringPointer(),
		TelemetryRouterId: model.TelemetryRouterID.ValueString(),
		AccessToken:       model.AccessToken.ValueString(),
	}, nil
}

func mapFields(_ context.Context, link *telemetrylink.TelemetryLinkResponse, model *Model, region string) error {
	if link == nil {
		return fmt.Errorf("link is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	var linkID string
	if model.LinkID.ValueString() != "" {
		linkID = model.LinkID.ValueString()
	} else {
		linkID = link.Id
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ResourceType.ValueString(), model.ResourceID.ValueString(), region)
	model.Region = types.StringValue(region)
	model.LinkID = types.StringValue(linkID)
	model.DisplayName = types.StringValue(link.DisplayName)
	model.Description = types.StringPointerValue(link.Description)
	model.TelemetryRouterID = types.StringValue(link.TelemetryRouterId)
	model.CreateTime = types.StringValue(link.CreateTime.Format(time.RFC3339))
	model.Status = types.StringValue(link.Status)

	return nil
}
