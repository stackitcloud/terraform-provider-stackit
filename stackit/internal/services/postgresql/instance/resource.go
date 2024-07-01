package postgresql

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresql"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	InstanceId         types.String `tfsdk:"instance_id"`
	ProjectId          types.String `tfsdk:"project_id"`
	CfGuid             types.String `tfsdk:"cf_guid"`
	CfSpaceGuid        types.String `tfsdk:"cf_space_guid"`
	DashboardUrl       types.String `tfsdk:"dashboard_url"`
	ImageUrl           types.String `tfsdk:"image_url"`
	Name               types.String `tfsdk:"name"`
	CfOrganizationGuid types.String `tfsdk:"cf_organization_guid"`
	Parameters         types.Object `tfsdk:"parameters"`
	Version            types.String `tfsdk:"version"`
	PlanName           types.String `tfsdk:"plan_name"`
	PlanId             types.String `tfsdk:"plan_id"`
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client *postgresql.APIClient
}

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_instance"
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

	var apiClient *postgresql.APIClient
	var err error
	if providerData.PostgreSQLCustomEndpoint != "" {
		apiClient, err = postgresql.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.PostgreSQLCustomEndpoint),
		)
	} else {
		apiClient, err = postgresql.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "PostgreSQL instance client configured")
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main": "PostgreSQL instance resource schema. Must have a `region` specified in the provider configuration.",
		"deprecation_message": strings.Join(
			[]string{
				"The STACKIT PostgreSQL service has reached its end of support on June 30th 2024.",
				"Resources of this type have stopped working since then.",
				"Use stackit_postgresflex_instance instead.",
				"Check https://docs.stackit.cloud/stackit/en/bring-your-data-to-stackit-postgresql-flex-138347648.html on how to backup and restore an instance from PostgreSQL to PostgreSQL Flex, then import the resource to Terraform using an \"import\" block (https://developer.hashicorp.com/terraform/language/import)",
			},
			" ",
		),
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the PostgreSQL instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"name":        "Instance name.",
		"version":     "The service version.",
		"plan_name":   "The selected plan name.",
		"plan_id":     "The selected plan ID.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		// Callout block: https://developer.hashicorp.com/terraform/registry/providers/docs#callouts
		MarkdownDescription: fmt.Sprintf("%s\n\n!> %s", descriptions["main"], descriptions["deprecation_message"]),
		DeprecationMessage:  descriptions["deprecation_message"],
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Required:    true,
			},
			"plan_name": schema.StringAttribute{
				Description: descriptions["plan_name"],
				Required:    true,
			},
			"plan_id": schema.StringAttribute{
				Description: descriptions["plan_id"],
				Computed:    true,
			},
			"parameters": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"enable_monitoring": schema.BoolAttribute{
						Optional: true,
					},
					"metrics_frequency": schema.Int64Attribute{
						Optional: true,
					},
					"metrics_prefix": schema.StringAttribute{
						Optional: true,
					},
					"monitoring_instance_id": schema.StringAttribute{
						Optional: true,
					},
					"plugins": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"sgw_acl": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
				},
				Optional: true,
				Computed: true,
			},
			"cf_guid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cf_space_guid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"image_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cf_organization_guid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(ctx context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead.")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead.")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *instanceResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead.")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing instance", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead.")
}
