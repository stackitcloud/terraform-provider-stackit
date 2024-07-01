package postgresql

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                = &credentialResource{}
	_ resource.ResourceWithConfigure   = &credentialResource{}
	_ resource.ResourceWithImportState = &credentialResource{}
)

type Model struct {
	Id           types.String `tfsdk:"id"` // needed by TF
	CredentialId types.String `tfsdk:"credential_id"`
	InstanceId   types.String `tfsdk:"instance_id"`
	ProjectId    types.String `tfsdk:"project_id"`
	Host         types.String `tfsdk:"host"`
	Hosts        types.List   `tfsdk:"hosts"`
	HttpAPIURI   types.String `tfsdk:"http_api_uri"`
	Name         types.String `tfsdk:"name"`
	Password     types.String `tfsdk:"password"`
	Port         types.Int64  `tfsdk:"port"`
	Uri          types.String `tfsdk:"uri"`
	Username     types.String `tfsdk:"username"`
}

// NewCredentialResource is a helper function to simplify the provider implementation.
func NewCredentialResource() resource.Resource {
	return &credentialResource{}
}

// credentialResource is the resource implementation.
type credentialResource struct {
	client *postgresql.APIClient
}

// Metadata returns the resource type name.
func (r *credentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_credential"
}

// Configure adds the provider configured client to the resource.
func (r *credentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	tflog.Info(ctx, "PostgreSQL credential client configured")
}

// Schema defines the schema for the resource.
func (r *credentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main": "PostgreSQL credential resource schema. Must have a `region` specified in the provider configuration.",
		"deprecation_message": strings.Join(
			[]string{
				"The STACKIT PostgreSQL service has reached its end of support on June 30th 2024.",
				"Resources of this type have stopped working since then.",
				"Use stackit_postgresflex_user instead.",
				"For more details, check https://docs.stackit.cloud/stackit/en/bring-your-data-to-stackit-postgresql-flex-138347648.html",
			},
			" ",
		),
		"id":            "Terraform's internal resource identifier. It is structured as \"`project_id`,`instance_id`,`credential_id`\".",
		"credential_id": "The credential's ID.",
		"instance_id":   "ID of the PostgreSQL instance.",
		"project_id":    "STACKIT Project ID to which the instance is associated.",
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
			"credential_id": schema.StringAttribute{
				Description: descriptions["credential_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
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
			"host": schema.StringAttribute{
				Computed: true,
			},
			"hosts": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"http_api_uri": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"password": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"port": schema.Int64Attribute{
				Computed: true,
			},
			"uri": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"username": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *credentialResource) Create(ctx context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead. Check https://docs.stackit.cloud/stackit/en/bring-your-data-to-stackit-postgresql-flex-138347648.html on how to backup and restore an instance from PostgreSQL to PostgreSQL Flex, then import the resource to Terraform using an \"import\" block (https://developer.hashicorp.com/terraform/language/import)")
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead. Check https://docs.stackit.cloud/stackit/en/bring-your-data-to-stackit-postgresql-flex-138347648.html on how to backup and restore an instance from PostgreSQL to PostgreSQL Flex, then import the resource to Terraform using an \"import\" block (https://developer.hashicorp.com/terraform/language/import)")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *credentialResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating credential", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead. Check https://docs.stackit.cloud/stackit/en/bring-your-data-to-stackit-postgresql-flex-138347648.html on how to backup and restore an instance from PostgreSQL to PostgreSQL Flex, then import the resource to Terraform using an \"import\" block (https://developer.hashicorp.com/terraform/language/import)")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *credentialResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead. Check https://docs.stackit.cloud/stackit/en/bring-your-data-to-stackit-postgresql-flex-138347648.html on how to backup and restore an instance from PostgreSQL to PostgreSQL Flex, then import the resource to Terraform using an \"import\" block (https://developer.hashicorp.com/terraform/language/import)")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *credentialResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing credential", "The STACKIT PostgreSQL service has reached its end of support on June 30th 2024. Resources of this type have stopped working since then. Use stackit_postgresflex_instance instead. Check https://docs.stackit.cloud/stackit/en/bring-your-data-to-stackit-postgresql-flex-138347648.html on how to backup and restore an instance from PostgreSQL to PostgreSQL Flex, then import the resource to Terraform using an \"import\" block (https://developer.hashicorp.com/terraform/language/import)")
}
