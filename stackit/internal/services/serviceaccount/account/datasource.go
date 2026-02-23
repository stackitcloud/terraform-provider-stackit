package account

import (
	"context"
	"fmt"
	"regexp"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	serviceaccountUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &serviceAccountDataSource{}
)

// DatasourceModel represents the schema for the service account data source.
type DatasourceModel struct {
	Model
	EmailRegex types.String `tfsdk:"email_regex"`
}

// NewServiceAccountDataSource creates a new instance of the serviceAccountDataSource.
func NewServiceAccountDataSource() datasource.DataSource {
	return &serviceAccountDataSource{}
}

// serviceAccountDataSource is the datasource implementation for service accounts.
type serviceAccountDataSource struct {
	client *serviceaccount.APIClient
}

// Configure initializes the serviceAccountDataSource with the provided provider data.
func (r *serviceAccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := serviceaccountUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

// Metadata provides metadata for the service account datasource.
func (r *serviceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

// Schema defines the schema for the service account data source.
func (r *serviceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"id":          "Terraform's internal resource ID, structured as \"`project_id`,`email`\".",
		"project_id":  "STACKIT project ID to which the service account is associated.",
		"name":        "Name of the service account.",
		"email":       "Email of the service account. Either email or email_regex must be provided.",
		"email_regex": "Regular expression to match the email of the service account. The first service account matching this regex will be used. Either email or email_regex must be provided.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Service account data source schema.",
		Description:         "Service account data source schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"email": schema.StringAttribute{
				Description: descriptions["email"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("email_regex")),
				},
			},
			"email_regex": schema.StringAttribute{
				Description: descriptions["email_regex"],
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("email")),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
		},
	}
}

// Read reads all service accounts from the API and updates the state with the latest information.
func (r *serviceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DatasourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	// Extract the project ID from the model configuration
	projectId := model.ProjectId.ValueString()

	// Compile the regex if provided
	var compiledRegex *regexp.Regexp
	var err error
	if !model.EmailRegex.IsNull() && model.EmailRegex.ValueString() != "" {
		compiledRegex, err = regexp.Compile(model.EmailRegex.ValueString())
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Invalid email_regex", err.Error())
			return
		}
	}

	// Call the API to list service accounts in the specified project
	listSaResp, err := r.client.ListServiceAccounts(ctx, projectId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading service account",
			fmt.Sprintf("Forbidden access for service account in project %q.", projectId),
			map[int]string{},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Iterate over the service accounts returned by the API to find the one matching the email or regex
	serviceAccounts := *listSaResp.Items
	for i := range serviceAccounts {
		saEmail := *serviceAccounts[i].Email
		match := false

		// Determine if the current service account matches the criteria
		if !model.Email.IsNull() && model.Email.ValueString() != "" {
			match = saEmail == model.Email.ValueString()
		} else if compiledRegex != nil {
			match = compiledRegex.MatchString(saEmail)
		}

		// Skip if it doesn't match
		if !match {
			continue
		}

		// Map the API response to the model, updating its fields with the service account data
		err = mapFields(&serviceAccounts[i], &model.Model)
		if err != nil {
			core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account", fmt.Sprintf("Error processing API response: %v", err))
			return
		}

		// If matched by regex, ensure the exact email is saved back to the state for downstream references
		model.Email = types.StringValue(saEmail)

		// Try to parse the name from the provided email address
		name, err := parseNameFromEmail(saEmail)
		if name != "" && err == nil {
			model.Name = types.StringValue(name)
		}

		// Update the state with the service account model
		diags = resp.State.Set(ctx, &model)
		resp.Diagnostics.Append(diags...)
		return
	}

	// If no matching service account is found, data sources must return an error
	core.LogAndAddError(
		ctx,
		&resp.Diagnostics,
		"Service Account not found",
		"No service account matching the provided email or email_regex was found in the project.",
	)
	resp.State.RemoveResource(ctx)
}
