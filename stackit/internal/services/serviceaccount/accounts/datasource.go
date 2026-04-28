package accounts

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	serviceaccountUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/serviceaccount/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &serviceAccountsDataSource{}
)

// ServiceAccountItem represents a single service account inside the list.
type ServiceAccountItem struct {
	ServiceAccountId types.String `tfsdk:"service_account_id"`
	Email            types.String `tfsdk:"email"`
	Name             types.String `tfsdk:"name"`
}

// ServiceAccountsModel represents the Model for the plural data source.
type ServiceAccountsModel struct {
	Id            types.String         `tfsdk:"id"`
	ProjectId     types.String         `tfsdk:"project_id"`
	EmailRegex    types.String         `tfsdk:"email_regex"`
	EmailSuffix   types.String         `tfsdk:"email_suffix"`
	SortAscending types.Bool           `tfsdk:"sort_ascending"`
	Items         []ServiceAccountItem `tfsdk:"items"`
}

// NewServiceAccountsDataSource creates a new instance of the plural data source.
func NewServiceAccountsDataSource() datasource.DataSource {
	return &serviceAccountsDataSource{}
}

// serviceAccountsDataSource is the datasource implementation for querying multiple service accounts.
type serviceAccountsDataSource struct {
	client *serviceaccount.APIClient
}

func (r *serviceAccountsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := serviceaccountUtils.ConfigureV2Client(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Service Accounts (plural) client configured")
}

func (r *serviceAccountsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_accounts"
}

func (r *serviceAccountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Service accounts plural data source schema. Returns a list of all service accounts in a project, optionally filtered.",
		Description:         "Service accounts plural data source schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID, structured as \"`project_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"email_regex": schema.StringAttribute{
				Description: "Optional regular expression to filter service accounts by email.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("email_suffix")),
				},
			},
			"email_suffix": schema.StringAttribute{
				Description: "Optional suffix to filter service accounts by email (e.g.,`@sa.stackit.cloud`, `@ske.sa.stackit.cloud`).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("email_regex")),
				},
			},
			"sort_ascending": schema.BoolAttribute{
				Description: "If set to `true`, service accounts are sorted in ascending lexicographical order by email. Defaults to `false` (descending).",
				Optional:    true,
			},
			"items": schema.ListNestedAttribute{
				Description: "The list of service accounts matching the provided filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"service_account_id": schema.StringAttribute{
							Description: "The internal UUID of the service account.",
							Computed:    true,
							Validators: []validator.String{
								validate.UUID(),
							},
						},
						"email": schema.StringAttribute{
							Description: "Email of the service account.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the service account.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *serviceAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic
	var model ServiceAccountsModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)
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

	// Fetch all service accounts
	listSaResp, err := r.client.DefaultAPI.ListServiceAccounts(ctx, projectId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading service accounts",
			fmt.Sprintf("Forbidden access for service accounts in project %q.", projectId),
			map[int]string{},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map the response data (filter, sort, and assign) to the model.
	err = mapDataSourceFields(listSaResp.Items, &model, compiledRegex)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service accounts", fmt.Sprintf("Error processing API response: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

// mapDataSourceFields filters, sorts, and maps a list of ServiceAccount API responses to the plural model.
func mapDataSourceFields(apiItems []serviceaccount.ServiceAccount, model *ServiceAccountsModel, compiledRegex *regexp.Regexp) error {
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var matchedItems []ServiceAccountItem
	emailSuffix := model.EmailSuffix.ValueString()

	for _, sa := range apiItems {
		email := sa.Email

		// Apply Filters (If neither is set, these checks simply pass)
		if compiledRegex != nil && !compiledRegex.MatchString(email) {
			continue
		}
		if emailSuffix != "" && !strings.HasSuffix(email, emailSuffix) {
			continue
		}

		// Parse name, ignore errors if the format is non-standard, just leave name empty
		nameStr, _ := serviceaccountUtils.ParseNameFromEmail(email)

		matchedItems = append(matchedItems, ServiceAccountItem{
			ServiceAccountId: types.StringValue(sa.Id),
			Email:            types.StringValue(email),
			Name:             types.StringValue(nameStr),
		})
	}

	// Sorting logic
	sortAsc := false
	if !model.SortAscending.IsNull() && !model.SortAscending.IsUnknown() {
		sortAsc = model.SortAscending.ValueBool()
	}

	sort.SliceStable(matchedItems, func(i, j int) bool {
		emailA := matchedItems[i].Email.ValueString()
		emailB := matchedItems[j].Email.ValueString()
		if sortAsc {
			return emailA < emailB
		}
		return emailA > emailB
	})

	// Assign values to the model
	model.Id = model.ProjectId // Use the project ID directly from the model as the data source ID
	model.Items = matchedItems

	return nil
}
