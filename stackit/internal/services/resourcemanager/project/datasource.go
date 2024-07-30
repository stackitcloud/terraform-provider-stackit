package project

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &projectDataSource{}
)

// NewProjectDataSource is a helper function to simplify the provider implementation.
func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

// projectDataSource is the data source implementation.
type projectDataSource struct {
	resourceManagerClient *resourcemanager.APIClient
	membershipClient      *authorization.APIClient
}

// Metadata returns the data source type name.
func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resourcemanager_project"
}

func (d *projectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var rmClient *resourcemanager.APIClient
	var err error
	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if providerData.ResourceManagerCustomEndpoint != "" {
		rmClient, err = resourcemanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithServiceAccountEmail(providerData.ServiceAccountEmail),
			config.WithEndpoint(providerData.ResourceManagerCustomEndpoint),
		)
	} else {
		rmClient, err = resourcemanager.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithServiceAccountEmail(providerData.ServiceAccountEmail),
		)
	}
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	var aClient *authorization.APIClient
	if providerData.AuthorizationCustomEndpoint != "" {
		ctx = tflog.SetField(ctx, "authorization_custom_endpoint", providerData.AuthorizationCustomEndpoint)
		aClient, err = authorization.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.AuthorizationCustomEndpoint),
		)
	} else {
		aClient, err = authorization.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring Membership API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	d.resourceManagerClient = rmClient
	d.membershipClient = aClient
	tflog.Info(ctx, "Resource Manager project client configured")
}

// Schema defines the schema for the data source.
func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                            "Resource Manager project data source schema. To identify the project, you need to provider either project_id or container_id. If you provide both, project_id will be used.",
		"id":                              "Terraform's internal data source. ID. It is structured as \"`container_id`\".",
		"project_id":                      "Project UUID identifier. This is the ID that can be used in most of the other resources to identify the project.",
		"container_id":                    "Project container ID. Globally unique, user-friendly identifier.",
		"parent_container_id":             "Parent resource identifier. Both container ID (user-friendly) and UUID are supported",
		"name":                            "Project name.",
		"labels":                          `Labels are key-value string pairs which can be attached to a resource container. A label key must match the regex [A-ZÄÜÖa-zäüöß0-9_-]{1,64}. A label value must match the regex ^$|[A-ZÄÜÖa-zäüöß0-9_-]{1,64}`,
		"owner_email":                     "Email address of the owner of the project. This value is only considered during creation. Changing it afterwards will have no effect.",
		"owner_email_deprecation_message": "The \"owner_email\" field has been deprecated in favor of the \"members\" field. Please use the \"members\" field to assign the owner role to a user, by setting the \"role\" field to `owner`.",
		"members":                         "The members assigned to the project. At least one subject needs to be a user, and not a client or service account.",
		"members.role":                    fmt.Sprintf("The role of the member in the project. Legacy roles (%s) are not supported.", strings.Join(utils.QuoteValues(utils.LegacyProjectRoles), ", ")),
		"members.subject":                 "Unique identifier of the user, service account or client. This is usually the email address for users or service accounts, and the name in case of clients.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Optional:    true,
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"container_id": schema.StringAttribute{
				Description: descriptions["container_id"],
				Optional:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"parent_container_id": schema.StringAttribute{
				Description: descriptions["parent_container_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				ElementType: types.StringType,
				Computed:    true,
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`[A-ZÄÜÖa-zäüöß0-9_-]{1,64}`),
							"must match expression"),
					),
					mapvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`[A-ZÄÜÖa-zäüöß0-9_-]{1,64}`),
							"must match expression"),
					),
				},
			},
			"owner_email": schema.StringAttribute{
				Description:         descriptions["owner_email"],
				DeprecationMessage:  descriptions["owner_email_deprecation_message"],
				MarkdownDescription: fmt.Sprintf("%s\n\n!> %s", descriptions["owner_email"], descriptions["owner_email_deprecation_message"]),
				Optional:            true,
			},
			"members": schema.ListNestedAttribute{
				Description: descriptions["members"],
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							Description: descriptions["members.role"],
							Computed:    true,
							Validators: []validator.String{
								validate.NonLegacyProjectRole(),
							},
						},
						"subject": schema.StringAttribute{
							Description: descriptions["members.subject"],
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	containerId := model.ContainerId.ValueString()
	ctx = tflog.SetField(ctx, "container_id", containerId)

	if containerId == "" && projectId == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", "Either container_id or project_id must be set")
		return
	}

	// set project identifier. If projectId is provided, it takes precedence over containerId
	var identifier = containerId
	if projectId != "" {
		identifier = projectId
	}

	projectResp, err := d.resourceManagerClient.GetProject(ctx, identifier).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusForbidden {
			resp.State.RemoveResource(ctx)
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapProjectFields(ctx, projectResp, &model, &resp.State)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Processing API response: %v", err))
		return
	}

	membersResp, err := d.membershipClient.ListMembersExecute(ctx, projectResourceType, *projectResp.ProjectId)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Reading members: %v", err))
		return
	}

	err = mapMembersFields(ctx, membersResp.Members, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading project", fmt.Sprintf("Processing API response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Resource Manager project read")
}
