package postgresflex

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	InstanceId     types.String `tfsdk:"instance_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	ACL            types.List   `tfsdk:"acl"`
	BackupSchedule types.String `tfsdk:"backup_schedule"`
	Flavor         types.Object `tfsdk:"flavor"`
	Replicas       types.Int64  `tfsdk:"replicas"`
	Storage        types.Object `tfsdk:"storage"`
	Version        types.String `tfsdk:"version"`
}

// Struct corresponding to Model.Flavor
type flavorModel struct {
	Id          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	CPU         types.Int64  `tfsdk:"cpu"`
	RAM         types.Int64  `tfsdk:"ram"`
}

// Types corresponding to flavorModel
var flavorTypes = map[string]attr.Type{
	"id":          basetypes.StringType{},
	"description": basetypes.StringType{},
	"cpu":         basetypes.Int64Type{},
	"ram":         basetypes.Int64Type{},
}

// Struct corresponding to Model.Storage
type storageModel struct {
	Class types.String `tfsdk:"class"`
	Size  types.Int64  `tfsdk:"size"`
}

// Types corresponding to storageModel
var storageTypes = map[string]attr.Type{
	"class": basetypes.StringType{},
	"size":  basetypes.Int64Type{},
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client *postgresflex.APIClient
}

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresflex_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *postgresflex.APIClient
	var err error
	if providerData.PostgresFlexCustomEndpoint != "" {
		apiClient, err = postgresflex.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.PostgresFlexCustomEndpoint),
		)
	} else {
		apiClient, err = postgresflex.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "PostgresFlex instance client configured")
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "PostgresFlex instance resource schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the PostgresFlex instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"name":        "Instance name.",
		"acl":         "The Access Control List (ACL) for the PostgresFlex instance.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$"),
						"must start with a letter, must have lower case letters, numbers or hyphens, and no hyphen at the end",
					),
				},
			},
			"acl": schema.ListAttribute{
				Description: descriptions["acl"],
				ElementType: types.StringType,
				Required:    true,
			},
			"backup_schedule": schema.StringAttribute{
				Required: true,
			},
			"flavor": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"description": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"cpu": schema.Int64Attribute{
						Required: true,
					},
					"ram": schema.Int64Attribute{
						Required: true,
					},
				},
			},
			"replicas": schema.Int64Attribute{
				Required: true,
			},
			"storage": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"class": schema.StringAttribute{
						Required: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"size": schema.Int64Attribute{
						Required: true,
					},
				},
			},
			"version": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	var acl []string
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client, &model, flavor)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, acl, flavor, storage)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new instance
	createResp, err := r.client.CreateInstance(ctx, projectId).CreateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	instanceId := *createResp.Id
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, storage)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "PostgresFlex instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	instanceResp, err := r.client.GetInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", err.Error())
		return
	}
	if instanceResp != nil && instanceResp.Item != nil && instanceResp.Item.Status != nil && *instanceResp.Item.Status == wait.InstanceStateDeleted {
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, instanceResp, &model, flavor, storage)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "PostgresFlex instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	var acl []string
	if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
		diags = model.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client, &model, flavor)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model, acl, flavor, storage)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing instance
	_, err = r.client.PartialUpdateInstance(ctx, projectId, instanceId).PartialUpdateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", err.Error())
		return
	}
	waitResp, err := wait.PartialUpdateInstanceWaitHandler(ctx, r.client, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, storage)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Postgresflex instance updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// Delete existing instance
	err := r.client.DeleteInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client, projectId, instanceId).SetTimeout(45 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "PostgresFlex instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	tflog.Info(ctx, "Postgresql instance state imported")
}

func mapFields(ctx context.Context, resp *postgresflex.InstanceResponse, model *Model, flavor *flavorModel, storage *storageModel) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if resp.Item == nil {
		return fmt.Errorf("no instance provided")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	instance := resp.Item

	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if instance.Id != nil {
		instanceId = *instance.Id
	} else {
		return fmt.Errorf("instance id not present")
	}

	var aclList basetypes.ListValue
	var diags diag.Diagnostics
	if instance.Acl == nil || instance.Acl.Items == nil {
		aclList = types.ListNull(types.StringType)
	} else {
		respACL := *instance.Acl.Items
		modelACL, err := utils.ListValuetoStringSlice(model.ACL)
		if err != nil {
			return err
		}

		reconciledACL := utils.ReconcileStringSlices(modelACL, respACL)

		aclList, diags = types.ListValueFrom(ctx, types.StringType, reconciledACL)
		if diags.HasError() {
			return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
		}
	}

	var flavorValues map[string]attr.Value
	if instance.Flavor == nil {
		flavorValues = map[string]attr.Value{
			"id":          flavor.Id,
			"description": flavor.Description,
			"cpu":         flavor.CPU,
			"ram":         flavor.RAM,
		}
	} else {
		flavorValues = map[string]attr.Value{
			"id":          types.StringValue(*instance.Flavor.Id),
			"description": types.StringValue(*instance.Flavor.Description),
			"cpu":         types.Int64PointerValue(instance.Flavor.Cpu),
			"ram":         types.Int64PointerValue(instance.Flavor.Memory),
		}
	}
	flavorObject, diags := types.ObjectValue(flavorTypes, flavorValues)
	if diags.HasError() {
		return fmt.Errorf("creating flavor: %w", core.DiagsToError(diags))
	}

	var storageValues map[string]attr.Value
	if instance.Storage == nil {
		storageValues = map[string]attr.Value{
			"class": storage.Class,
			"size":  storage.Size,
		}
	} else {
		storageValues = map[string]attr.Value{
			"class": types.StringValue(*instance.Storage.Class),
			"size":  types.Int64PointerValue(instance.Storage.Size),
		}
	}
	storageObject, diags := types.ObjectValue(storageTypes, storageValues)
	if diags.HasError() {
		return fmt.Errorf("creating storage: %w", core.DiagsToError(diags))
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		instanceId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.InstanceId = types.StringValue(instanceId)
	model.Name = types.StringPointerValue(instance.Name)
	model.ACL = aclList
	model.BackupSchedule = types.StringPointerValue(instance.BackupSchedule)
	model.Flavor = flavorObject
	model.Replicas = types.Int64PointerValue(instance.Replicas)
	model.Storage = storageObject
	model.Version = types.StringPointerValue(instance.Version)
	return nil
}

func toCreatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel) (*postgresflex.CreateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if acl == nil {
		return nil, fmt.Errorf("nil acl")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	return &postgresflex.CreateInstancePayload{
		Acl: &postgresflex.ACL{
			Items: &acl,
		},
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		Replicas:       conversion.Int64ValueToPointer(model.Replicas),
		Storage: &postgresflex.Storage{
			Class: conversion.StringValueToPointer(storage.Class),
			Size:  conversion.Int64ValueToPointer(storage.Size),
		},
		Version: conversion.StringValueToPointer(model.Version),
	}, nil
}

func toUpdatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel) (*postgresflex.PartialUpdateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if acl == nil {
		return nil, fmt.Errorf("nil acl")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	return &postgresflex.PartialUpdateInstancePayload{
		Acl: &postgresflex.ACL{
			Items: &acl,
		},
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		Replicas:       conversion.Int64ValueToPointer(model.Replicas),
		Storage: &postgresflex.Storage{
			Class: conversion.StringValueToPointer(storage.Class),
			Size:  conversion.Int64ValueToPointer(storage.Size),
		},
		Version: conversion.StringValueToPointer(model.Version),
	}, nil
}

type postgresFlexClient interface {
	ListFlavorsExecute(ctx context.Context, projectId string) (*postgresflex.ListFlavorsResponse, error)
}

func loadFlavorId(ctx context.Context, client postgresFlexClient, model *Model, flavor *flavorModel) error {
	if model == nil {
		return fmt.Errorf("nil model")
	}
	if flavor == nil {
		return fmt.Errorf("nil flavor")
	}
	cpu := conversion.Int64ValueToPointer(flavor.CPU)
	if cpu == nil {
		return fmt.Errorf("nil CPU")
	}
	ram := conversion.Int64ValueToPointer(flavor.RAM)
	if ram == nil {
		return fmt.Errorf("nil RAM")
	}

	projectId := model.ProjectId.ValueString()
	res, err := client.ListFlavorsExecute(ctx, projectId)
	if err != nil {
		return fmt.Errorf("listing postgresflex flavors: %w", err)
	}

	avl := ""
	if res.Flavors == nil {
		return fmt.Errorf("finding flavors for project %s", projectId)
	}
	for _, f := range *res.Flavors {
		if f.Id == nil || f.Cpu == nil || f.Memory == nil {
			continue
		}
		if *f.Cpu == *cpu && *f.Memory == *ram {
			flavor.Id = types.StringValue(*f.Id)
			flavor.Description = types.StringValue(*f.Description)
			break
		}
		avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM", avl, *f.Cpu, *f.Cpu)
	}
	if flavor.Id.ValueString() == "" {
		return fmt.Errorf("couldn't find flavor, available specs are:%s", avl)
	}

	return nil
}
