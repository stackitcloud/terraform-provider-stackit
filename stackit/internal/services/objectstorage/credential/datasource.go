package objectstorage

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	objectstorageUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/objectstorage/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &credentialDataSource{}
)

type DataSourceModel struct {
	Id                  types.String `tfsdk:"id"` // needed by TF
	CredentialId        types.String `tfsdk:"credential_id"`
	CredentialsGroupId  types.String `tfsdk:"credentials_group_id"`
	ProjectId           types.String `tfsdk:"project_id"`
	Name                types.String `tfsdk:"name"`
	ExpirationTimestamp types.String `tfsdk:"expiration_timestamp"`
	Region              types.String `tfsdk:"region"`
}

// NewCredentialDataSource is a helper function to simplify the provider implementation.
func NewCredentialDataSource() datasource.DataSource {
	return &credentialDataSource{}
}

// credentialDataSource is the resource implementation.
type credentialDataSource struct {
	client       *objectstorage.APIClient
	providerData core.ProviderData
}

// Metadata returns the resource type name.
func (r *credentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_credential"
}

// Configure adds the provider configured client to the datasource.
func (r *credentialDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := objectstorageUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "ObjectStorage credential client configured")
}

// Schema defines the schema for the datasource.
func (r *credentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                 "ObjectStorage credential data source schema. Must have a `region` specified in the provider configuration.",
		"id":                   "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`credentials_group_id`,`credential_id`\".",
		"credential_id":        "The credential ID.",
		"credentials_group_id": "The credential group ID.",
		"project_id":           "STACKIT Project ID to which the credential group is associated.",
		"region":               "The resource region. If not defined, the provider region is used.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"credential_id": schema.StringAttribute{
				Description: descriptions["credential_id"],
				Required:    true,
			},
			"credentials_group_id": schema.StringAttribute{
				Description: descriptions["credentials_group_id"],
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"expiration_timestamp": schema.StringAttribute{
				Computed: true,
			},
			"region": schema.StringAttribute{
				// the region cannot be found automatically, so it has to be passed
				Optional:    true,
				Description: descriptions["region"],
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	credentialsGroupId := model.CredentialsGroupId.ValueString()
	credentialId := model.CredentialId.ValueString()
	var region string
	if utils.IsUndefined(model.Region) {
		region = r.providerData.GetRegion()
	} else {
		region = model.Region.ValueString()
	}

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "credentials_group_id", credentialsGroupId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)
	ctx = tflog.SetField(ctx, "region", region)

	credentialsGroupResp, err := r.client.ListAccessKeys(ctx, projectId, region).CredentialsGroup(credentialsGroupId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading credential",
			fmt.Sprintf("Credential group with ID %q does not exist in project %q.", credentialsGroupId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	if credentialsGroupResp == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Reading credentials", fmt.Sprintf("Response is nil: %v", err))
		return
	}

	credential := findCredential(*credentialsGroupResp, credentialId)
	if credential == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Reading credential", fmt.Sprintf("Credential with ID %q not found in credentials group %q", credentialId, credentialsGroupId))
		return
	}

	err = mapDataSourceFields(credential, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage credential read")
}

func mapDataSourceFields(credentialResp *objectstorage.AccessKey, model *DataSourceModel, region string) error {
	if credentialResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var credentialId string
	if model.CredentialId.ValueString() != "" {
		credentialId = model.CredentialId.ValueString()
	} else if credentialResp.KeyId != nil {
		credentialId = *credentialResp.KeyId
	} else {
		return fmt.Errorf("credential id not present")
	}

	if credentialResp.Expires == nil {
		model.ExpirationTimestamp = types.StringNull()
	} else {
		// Harmonize the timestamp format
		// Eg. "2027-01-02T03:04:05.000Z" = "2027-01-02T03:04:05Z"
		expirationTimestamp, err := time.Parse(time.RFC3339, *credentialResp.Expires)
		if err != nil {
			return fmt.Errorf("unable to parse payload expiration timestamp '%v': %w", *credentialResp.Expires, err)
		}
		model.ExpirationTimestamp = types.StringValue(expirationTimestamp.Format(time.RFC3339))
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		region,
		model.CredentialsGroupId.ValueString(),
		credentialId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.CredentialId = types.StringValue(credentialId)
	model.Name = types.StringPointerValue(credentialResp.DisplayName)
	model.Region = types.StringValue(region)
	return nil
}

// Returns the access key if found otherwise nil
func findCredential(credentialsGroupResp objectstorage.ListAccessKeysResponse, credentialId string) *objectstorage.AccessKey {
	for _, credential := range *credentialsGroupResp.AccessKeys {
		if credential.KeyId == nil || *credential.KeyId != credentialId {
			continue
		}
		return &credential
	}
	return nil
}
