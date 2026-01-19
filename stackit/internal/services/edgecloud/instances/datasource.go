package instances

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/edge"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	edgeutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/edgecloud/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &instancesDataSource{}
)

// DataSourceModel maps the data source schema data.
type DataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Region    types.String `tfsdk:"region"`
	Instances types.List   `tfsdk:"instances"` // Changed from Map to List
}

// instanceTypes defines the attribute types for a single instance object.
var instanceTypes = map[string]attr.Type{
	"instance_id":  types.StringType,
	"display_name": types.StringType,
	"created":      types.StringType,
	"frontend_url": types.StringType,
	"region":       types.StringType,
	"plan_id":      types.StringType,
	"description":  types.StringType,
	"status":       types.StringType,
}

// NewInstancesDataSource creates a new instance of the instancesDataSource.
func NewInstancesDataSource() datasource.DataSource {
	return &instancesDataSource{}
}

// instancesDataSource is the data source implementation.
type instancesDataSource struct {
	client       *edge.APIClient
	providerData core.ProviderData
}

// Configure sets up the API client for the Edge Cloud instance data source.
func (d *instancesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	features.CheckBetaResourcesEnabled(ctx, &d.providerData, &resp.Diagnostics, "stackit_edgecloud_instances", "datasource")
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := edgeutils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "edge cloud client configured")
}

// Metadata provides metadata for the edge datasource.
func (d *instancesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edgecloud_instances"
}

// Schema defines the schema for the Edge Cloud instances data source.
func (d *instancesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Edge Cloud is in private Beta and not generally available.\n You can contact support if you are interested in trying it out.", core.Datasource),
		Description:         "edge cloud instances datasource schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal data source ID, structured as `project_id`,`region`.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the Edge Cloud instances are associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The resource region. If not defined, the provider region is used.",
				Optional:    true,
			},
			"instances": schema.ListNestedAttribute{
				Description: "A list of Edge Cloud instances.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"instance_id": schema.StringAttribute{
							Description: "The ID of the instance.",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "The display name of the instance.",
							Computed:    true,
						},
						"created": schema.StringAttribute{
							Description: "The date and time the instance was created.",
							Computed:    true,
						},
						"frontend_url": schema.StringAttribute{
							Description: "Frontend URL for the Edge Cloud instance.",
							Computed:    true,
						},
						"region": schema.StringAttribute{
							Description: "The region where the instance is located.",
							Computed:    true,
						},
						"plan_id": schema.StringAttribute{
							Description: "The plan ID for the instance.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the instance.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The status of the instance.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read fetches the list of Edge Cloud instances and populates the data source.
func (d *instancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var state DataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := state.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(state.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// Fetch all instances for the project and region
	instancesResp, err := d.client.ListInstances(ctx, projectId, region).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Error reading instances:",
			fmt.Sprintf("Calling API: %v", err),
			map[int]string{
				http.StatusNotFound: fmt.Sprintf("Project %q or region %q not found", projectId, region),
			},
		)
		return
	}

	ctx = core.LogResponse(ctx)

	if instancesResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instances", "API response is nil")
		return
	}
	if instancesResp.Instances == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instances", "instance field in the API response is nil")
		return
	}
	instancesList := buildInstancesList(ctx, instancesResp.Instances, region, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create ListValue
	instancesListValue, diags := types.ListValue(types.ObjectType{AttrTypes: instanceTypes}, instancesList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the Terraform state
	state.Id = types.StringValue(fmt.Sprintf("%s,%s", projectId, region))
	state.Instances = instancesListValue

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read all edgecloud instances")
}

// buildInstancesList constructs a list of instance attributes
func buildInstancesList(ctx context.Context, instances edge.InstanceListGetInstancesAttributeType, region string, diags *diag.Diagnostics) []attr.Value {
	var instancesList []attr.Value

	for _, instance := range *instances {
		instanceAttrs, err := mapInstanceToAttrs(instance, region)
		if err != nil {
			// Keep going in case there are more errors
			instanceId := "without id"
			if instance.Id != nil {
				instanceId = *instance.Id
			}
			core.LogAndAddError(ctx, diags, "Error reading instances", fmt.Sprintf("Could not process instance %q: %v", instanceId, err))
			continue
		}

		instanceObjectValue, objDiags := types.ObjectValue(instanceTypes, instanceAttrs)
		diags.Append(objDiags...)

		if objDiags.HasError() {
			continue
		}
		instancesList = append(instancesList, instanceObjectValue)
	}
	return instancesList
}

func mapInstanceToAttrs(instance edge.Instance, region string) (map[string]attr.Value, error) {
	if instance.Id == nil {
		return nil, fmt.Errorf("instance is missing an 'id'")
	}
	if instance.DisplayName == nil || *instance.DisplayName == "" {
		return nil, fmt.Errorf("instance %q is missing a 'displayName'", *instance.Id)
	}
	if instance.PlanId == nil {
		return nil, fmt.Errorf("instance %q is missing a 'planId'", *instance.Id)
	}
	if instance.FrontendUrl == nil {
		return nil, fmt.Errorf("instance %q is missing a 'frontendUrl'", *instance.Id)
	}
	if instance.Status == nil {
		return nil, fmt.Errorf("instance %q is missing a 'status'", *instance.Id)
	}
	if instance.Created == nil {
		return nil, fmt.Errorf("instance %q is missing a 'created' timestamp", *instance.Id)
	}
	if instance.Description == nil {
		return nil, fmt.Errorf("instance %q is missing a 'description'", *instance.Id)
	}

	attrs := map[string]attr.Value{
		"instance_id":  types.StringValue(*instance.Id),
		"display_name": types.StringValue(*instance.DisplayName),
		"region":       types.StringValue(region),
		"plan_id":      types.StringValue(*instance.PlanId),
		"frontend_url": types.StringValue(*instance.FrontendUrl),
		"status":       types.StringValue(string(instance.GetStatus())),
		"created":      types.StringValue(instance.Created.String()),
		"description":  types.StringValue(*instance.Description),
	}
	return attrs, nil
}
