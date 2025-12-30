package postgresflexalpha

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha/wait"
	postgresflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &instanceResource{}
	_ resource.ResourceWithConfigure      = &instanceResource{}
	_ resource.ResourceWithImportState    = &instanceResource{}
	_ resource.ResourceWithModifyPlan     = &instanceResource{}
	_ resource.ResourceWithValidateConfig = &instanceResource{}
	_ resource.ResourceWithIdentity       = &instanceResource{}
)

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client       *postgresflex.APIClient
	providerData core.ProviderData
}

func (r *instanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Replicas.IsNull() || data.Replicas.IsUnknown() {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("replicas"),
			"Missing Attribute Configuration",
			"Expected replicas to be configured. "+
				"The resource may return unexpected results.",
		)
	}
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { // nolint:gocritic // function signature required by Terraform
	var configModel Model
	// skip initial empty configuration to avoid follow-up errors
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

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresflexalpha_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := postgresflexUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Postgres Flex instance client configured")
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":               "Postgres Flex instance resource schema. Must have a `region` specified in the provider configuration.",
		"id":                 "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`instance_id`\".",
		"instance_id":        "ID of the PostgresFlex instance.",
		"project_id":         "STACKIT project ID to which the instance is associated.",
		"name":               "Instance name.",
		"backup_schedule":    "The schedule for on what time and how often the database backup will be created. The schedule is written as a cron schedule.",
		"retention_days":     "The days of the retention period.",
		"flavor":             "The block that defines the flavor data.",
		"flavor_id":          "The ID of the flavor.",
		"flavor_description": "The flavor detailed flavor name.",
		"flavor_cpu":         "The CPU count of the flavor.",
		"flavor_ram":         "The RAM count of the flavor.",
		"flavor_node_type":   "The node type of the flavor. (Single or Replicas)",
		"replicas":           "The number of replicas.",
		"storage":            "The block of the storage configuration.",
		"storage_class":      "The storage class used.",
		"storage_size":       "The disk size of the storage.",
		"region":             "The resource region. If not defined, the provider region is used.",
		"version":            "The database version used.",
		"encryption":         "The encryption block.",
		"keyring_id":         "KeyRing ID of the encryption key.",
		"key_id":             "Key ID of the encryption key.",
		"key_version":        "Key version of the encryption key.",
		"service_account":    "The service account ID of the service account.",
		"network":            "The network block configuration.",
		"access_scope":       "The access scope. (Either SNA or PUBLIC)",
		"acl":                "The Access Control List (ACL) for the PostgresFlex instance.",
		"instance_address":   "The returned instance address.",
		"router_address":     "The returned router address.",
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
			"backup_schedule": schema.StringAttribute{
				Required: true,
			},
			"retention_days": schema.Int64Attribute{
				Description: descriptions["retention_days"],
				Required:    true,
			},
			"flavor": schema.SingleNestedAttribute{
				Required:    true,
				Description: descriptions["flavor"],
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: descriptions["flavor_id"],
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							UseStateForUnknownIfFlavorUnchanged(req),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"description": schema.StringAttribute{
						Computed:    true,
						Description: descriptions["flavor_description"],
						PlanModifiers: []planmodifier.String{
							UseStateForUnknownIfFlavorUnchanged(req),
						},
					},
					"cpu": schema.Int64Attribute{
						Description: descriptions["flavor_cpu"],
						Required:    true,
					},
					"ram": schema.Int64Attribute{
						Description: descriptions["flavor_ram"],
						Required:    true,
					},
					"node_type": schema.StringAttribute{
						Description: descriptions["flavor_node_type"],
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							// TODO @mhenselin anschauen
							UseStateForUnknownIfFlavorUnchanged(req),
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"replicas": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"storage": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"class": schema.StringAttribute{
						Required:    true,
						Description: descriptions["storage_class"],
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"size": schema.Int64Attribute{
						Description: descriptions["storage_size"],
						Required:    true,
						// PlanModifiers: []planmodifier.Int64{
						// TODO - req replace if new size smaller than state size
						// int64planmodifier.RequiresReplaceIf(),
						// },
					},
				},
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Required:    true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
				Description: descriptions["region"],
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"encryption": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"key_id": schema.StringAttribute{
						Description: descriptions["key_id"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
					"key_version": schema.StringAttribute{
						Description: descriptions["key_version"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
					"keyring_id": schema.StringAttribute{
						Description: descriptions["keyring_id"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
					"service_account": schema.StringAttribute{
						Description: descriptions["service_account"],
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
						},
					},
				},
				Description: descriptions["encryption"],
				//Validators:          nil,
				PlanModifiers: []planmodifier.Object{},
			},
			"network": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"access_scope": schema.StringAttribute{
						Default: stringdefault.StaticString(
							"PUBLIC",
						),
						Description: descriptions["access_scope"],
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
						Validators: []validator.String{
							validate.NoSeparator(),
							stringvalidator.OneOf("SNA", "PUBLIC"),
						},
					},
					"acl": schema.ListAttribute{
						Description: descriptions["acl"],
						ElementType: types.StringType,
						Required:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"instance_address": schema.StringAttribute{
						Description: descriptions["instance_address"],
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"router_address": schema.StringAttribute{
						Description: descriptions["router_address"],
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
				Description: descriptions["network"],
				//MarkdownDescription: "",
				//Validators:          nil,
				PlanModifiers: []planmodifier.Object{},
			},
		},
	}
}

func (r *instanceResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true, // must be set during import by the practitioner
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	var storage = &storageModel{}
	if !model.Storage.IsNull() && !model.Storage.IsUnknown() {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var flavor = &flavorModel{}
	if !model.Flavor.IsNull() && !model.Flavor.IsUnknown() {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client, &model, flavor, storage)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}

	if flavor.Id.IsNull() || flavor.Id.IsUnknown() {
		err := loadFlavorId(ctx, r.client, &model, flavor, storage)
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), err.Error())
			return
		}
		flavorValues := map[string]attr.Value{
			"id":          flavor.Id,
			"description": flavor.Description,
			"cpu":         flavor.CPU,
			"ram":         flavor.RAM,
			"node_type":   flavor.NodeType,
		}
		var flavorObject basetypes.ObjectValue
		flavorObject, diags = types.ObjectValue(flavorTypes, flavorValues)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
		model.Flavor = flavorObject
	}

	var encryption = &encryptionModel{}
	if !model.Encryption.IsNull() && !model.Encryption.IsUnknown() {
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var network = &networkModel{}
	if !model.Network.IsNull() && !model.Network.IsUnknown() {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var acl []string
	if !network.ACL.IsNull() && !network.ACL.IsUnknown() {
		diags = network.ACL.ElementsAs(ctx, &acl, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, flavor, storage, encryption, network)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new instance
	createResp, err := r.client.CreateInstanceRequest(ctx, projectId, region).CreateInstanceRequestPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)
	instanceId := *createResp.Id

	model.InstanceId = types.StringValue(instanceId)
	model.Id = utils.BuildInternalTerraformId(projectId, region, instanceId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set data returned by API in identity
	identity := IdentityModel{
		ID: utils.BuildInternalTerraformId(projectId, region, instanceId),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)

	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Wait handler error: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, storage, encryption, network, region)
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
	tflog.Info(ctx, "Postgres Flex instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read identity data
	var identityData IdentityModel
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	var flavor = &flavorModel{}
	if !model.Flavor.IsNull() && !model.Flavor.IsUnknown() {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var storage = &storageModel{}
	if !model.Storage.IsNull() && !model.Storage.IsUnknown() {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var network = &networkModel{}
	if !model.Network.IsNull() && !model.Network.IsUnknown() {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var encryption = &encryptionModel{}
	if !model.Encryption.IsNull() && !model.Encryption.IsUnknown() {
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	instanceResp, err := r.client.GetInstanceRequest(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, instanceResp, &model, flavor, storage, encryption, network, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identityData.ID = model.Id
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identityData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Postgres Flex instance read")
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

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	// nolint:gocritic // need that code later
	// var acl []string
	// if !(model.ACL.IsNull() || model.ACL.IsUnknown()) {
	//	diags = model.ACL.ElementsAs(ctx, &acl, false)
	//	resp.Diagnostics.Append(diags...)
	//	if resp.Diagnostics.HasError() {
	//		return
	//	}
	// }

	var storage = &storageModel{}
	if !model.Storage.IsNull() && !model.Storage.IsUnknown() {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var flavor = &flavorModel{}
	if !model.Flavor.IsNull() && !model.Flavor.IsUnknown() {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := loadFlavorId(ctx, r.client, &model, flavor, storage)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Loading flavor ID: %v", err))
			return
		}
	}

	var network = &networkModel{}
	if !model.Network.IsNull() && !model.Network.IsUnknown() {
		diags = model.Network.As(ctx, network, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var encryption = &encryptionModel{}
	if !model.Encryption.IsNull() && !model.Encryption.IsUnknown() {
		diags = model.Encryption.As(ctx, encryption, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model, flavor, storage, network)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing instance
	err = r.client.UpdateInstancePartiallyRequest(ctx, projectId, region, instanceId).UpdateInstancePartiallyRequestPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", err.Error())
		return
	}

	ctx = core.LogResponse(ctx)

	waitResp, err := wait.PartialUpdateInstanceWaitHandler(ctx, r.client, projectId, region, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model, flavor, storage, encryption, network, region)
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

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	region := model.Region.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing instance
	err := r.client.DeleteInstanceRequest(ctx, projectId, region, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client, projectId, region, instanceId).SetTimeout(45 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "Postgres Flex instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[2])...)
	tflog.Info(ctx, "Postgres Flex instance state imported")
}
