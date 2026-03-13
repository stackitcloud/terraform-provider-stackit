package image

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	authorizationUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/authorization/utils"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &imageShareResource{}
	_ resource.ResourceWithConfigure   = &imageShareResource{}
	_ resource.ResourceWithImportState = &imageShareResource{}
	_ resource.ResourceWithModifyPlan  = &imageShareResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"`
	ProjectId          types.String `tfsdk:"project_id"`
	Region             types.String `tfsdk:"region"`
	ImageId            types.String `tfsdk:"image_id"`
	ParentOrganization types.Bool   `tfsdk:"parent_organization"`
	Projects           types.Set    `tfsdk:"projects"`
}

func NewImageShareResource() resource.Resource {
	return &imageShareResource{}
}

type imageShareResource struct {
	client       *iaas.APIClient
	providerData core.ProviderData
}

func (r *imageShareResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image_share"
}

func (r *imageShareResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

func (r *imageShareResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
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

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *imageShareResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Image Share resource schema. Manages the sharing settings of a STACKIT Image.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the image belongs.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image_id": schema.StringAttribute{
				Description: "The image ID to share.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"parent_organization": schema.BoolAttribute{
				Description: "If set to true, the image is shared with all projects inside the image owners organization. Mutually exclusive with `projects`.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRoot("projects")),
				},
			},
			"projects": schema.SetAttribute{
				Description: "List of project IDs to share the image with. Mutually exclusive with `parent_organization`.",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Set{
					setvalidator.ConflictsWith(path.MatchRoot("parent_organization")),
					setvalidator.ValueStringsAre(
						validate.UUID(),
						validate.NoSeparator(),
					),
				},
			},
		},
	}
}

func (r *imageShareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	imageId := model.ImageId.ValueString()

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "image_id", imageId)
	ctx = core.InitProviderContext(ctx)

	// Locking needed to prevent race conditions during creation checks
	lockKey := fmt.Sprintf("%s,%s,%s", projectId, imageId, region)
	// move to standard utils
	unlock := authorizationUtils.LockAssignment(lockKey)
	defer unlock()

	// 1. Check if a image share exists
	shareResp, err := r.client.GetImageShare(ctx, projectId, region, imageId).Execute()
	var exists bool
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if !(errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound) {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error checking existing image share", fmt.Sprintf("Calling API: %v", err))
			return
		}
	} else {
		// If endpoint returns 200, we check if it is "active".
		// We only block creation if there is an existing ACTIVE share.
		// If the existing share is "inactive" (false/empty), we allow overwriting it
		// because the user's intent is to manage the share state (even if they set it to false).
		exists = checkImageShareActive(shareResp)
	}

	if exists {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Image Share already exists",
			fmt.Sprintf("A share configuration for image %q already exists. Please import it or delete it first.", imageId))
		return
	}

	// 2.
	payload, err := toSetImageSharePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating image share", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// 3.
	createResp, err := r.client.SetImageShare(ctx, projectId, region, imageId).SetImageSharePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating image share", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// 4.
	err = mapFields(ctx, createResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error processing API response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Sleep to ensure consistency
	time.Sleep(1 * time.Second)
	tflog.Info(ctx, "Image share created")
}

func (r *imageShareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	imageId := model.ImageId.ValueString()

	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "image_id", imageId)

	shareResp, err := r.client.GetImageShare(ctx, projectId, region, imageId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading image share", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	// NOTE: We do NOT remove the resource here if checkImageShareActive(shareResp) is false.
	// If the user configured `parent_organization = false` or `projects = []`, the resource
	// exists in Terraform and should track that "inactive" state.

	err = mapFields(ctx, shareResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error processing API response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Update is not supported because fields are set to RequiresReplace
func (r *imageShareResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
}

func (r *imageShareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	imageId := model.ImageId.ValueString()

	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "image_id", imageId)

	err := r.client.DeleteImageShare(ctx, projectId, region, imageId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting image share", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	tflog.Info(ctx, "Image share deleted")
}

func (r *imageShareResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing image share",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[image_id]  Got: %q", req.ID),
		)
		return
	}

	utils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id": idParts[0],
		"region":     idParts[1],
		"image_id":   idParts[2],
	})
	tflog.Info(ctx, "Image share imported")
}

// mapFields
func mapFields(ctx context.Context, resp *iaas.ImageShare, model *Model, region string) error {
	if resp == nil || model == nil {
		return fmt.Errorf("response or model is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, model.ImageId.ValueString())
	model.Region = types.StringValue(region)

	// 1. Map Parent Organization
	apiParentOrg := false
	if resp.ParentOrganization != nil {
		apiParentOrg = *resp.ParentOrganization
	}

	if apiParentOrg {
		model.ParentOrganization = types.BoolValue(true)
	} else {
		// API is false. Check the Plan/State to decide between False and Null.
		if !model.ParentOrganization.IsNull() {
			model.ParentOrganization = types.BoolValue(false)
		} else {
			model.ParentOrganization = types.BoolNull()
		}
	}

	// 2. Map Projects
	// If Parent Org is active, Projects MUST be null to enforce schema exclusivity
	if apiParentOrg {
		model.Projects = types.SetNull(types.StringType)
	} else {
		// Parent Org is inactive, so check projects
		var apiProjects []string
		if resp.Projects != nil {
			apiProjects = *resp.Projects
		}

		if len(apiProjects) == 0 {
			// API is empty. Check Plan/State to decide between Empty Set and Null.
			if !model.Projects.IsNull() {
				// Create explicit empty set
				emptySet, diags := types.SetValueFrom(ctx, types.StringType, []string{})
				if diags.HasError() {
					return fmt.Errorf("creating empty projects set: %w", core.DiagsToError(diags))
				}
				model.Projects = emptySet
			} else {
				model.Projects = types.SetNull(types.StringType)
			}
		} else {
			// API has data, map it
			projects, diags := types.SetValueFrom(ctx, types.StringType, apiProjects)
			if diags.HasError() {
				return fmt.Errorf("mapping projects: %w", core.DiagsToError(diags))
			}
			model.Projects = projects
		}
	}

	return nil
}

func toSetImageSharePayload(ctx context.Context, model *Model) (*iaas.SetImageSharePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	// Case A: Share with Organization
	// If user explicitly configured ParentOrganization (true OR false)
	if !model.ParentOrganization.IsNull() {
		return &iaas.SetImageSharePayload{
			ParentOrganization: conversion.BoolValueToPointer(model.ParentOrganization),
			Projects:           nil, // Explicitly nil to omit key from JSON
		}, nil
	}

	// Case B: Share with Projects
	// Use empty slice [] instead of nil for empty projects, to send "projects": []
	projects := make([]string, 0)
	if !model.Projects.IsNull() && !model.Projects.IsUnknown() {
		diags := model.Projects.ElementsAs(ctx, &projects, false)
		if diags.HasError() {
			return nil, fmt.Errorf("mapping projects to strings: %w", core.DiagsToError(diags))
		}
	}

	return &iaas.SetImageSharePayload{
		ParentOrganization: nil, // Explicitly nil to omit key from JSON
		Projects:           &projects,
	}, nil
}

func checkImageShareActive(shareResp *iaas.ImageShare) bool {
	if shareResp == nil {
		return false
	}
	if shareResp.ParentOrganization != nil && *shareResp.ParentOrganization {
		return true
	}
	if shareResp.Projects != nil && len(*shareResp.Projects) > 0 {
		return true
	}
	return false
}
