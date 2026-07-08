package redis

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	redisUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/redis/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	redis "github.com/stackitcloud/stackit-sdk-go/services/redis/v2api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &credentialDataSource{}
)

type DataSourceModel struct {
	Id               types.String `tfsdk:"id"` // needed by TF
	CredentialId     types.String `tfsdk:"credential_id"`
	InstanceId       types.String `tfsdk:"instance_id"`
	ProjectId        types.String `tfsdk:"project_id"`
	Region           types.String `tfsdk:"region"`
	Host             types.String `tfsdk:"host"`
	Hosts            types.List   `tfsdk:"hosts"`
	LoadBalancedHost types.String `tfsdk:"load_balanced_host"`
	Password         types.String `tfsdk:"password"`
	Port             types.Int32  `tfsdk:"port"`
	Uri              types.String `tfsdk:"uri"`
	Username         types.String `tfsdk:"username"`
}

// NewCredentialDataSource is a helper function to simplify the provider implementation.
func NewCredentialDataSource() datasource.DataSource {
	return &credentialDataSource{}
}

// credentialDataSource is the data source implementation.
type credentialDataSource struct {
	client       *redis.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *credentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_credential"
}

// Configure adds the provider configured client to the data source.
func (r *credentialDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := redisUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "Redis credential client configured")
}

// Schema defines the schema for the data source.
func (r *credentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{ //nolint:gosec // description for credential id
		"main":          "Redis credential data source schema. Must have a `region` specified in the provider configuration.",
		"id":            "Terraform's internal data source. identifier. It is structured as \"`project_id`,`region`,`instance_id`,`credential_id`\".",
		"credential_id": "The credential's ID.",
		"instance_id":   "ID of the Redis instance.",
		"project_id":    "STACKIT project ID to which the instance is associated.",
		"uri":           "Connection URI.",
		"region":        "The resource region. If not defined, the provider region is used.",
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
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
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
			"load_balanced_host": schema.StringAttribute{
				Computed: true,
			},
			"password": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"port": schema.Int32Attribute{
				Computed: true,
			},
			"uri": schema.StringAttribute{
				Description: descriptions["uri"],
				Computed:    true,
				Sensitive:   true,
			},
			"username": schema.StringAttribute{
				Computed: true,
			},
			"region": schema.StringAttribute{
				Optional: true,
				// must be computed to allow for storing the override value from the provider
				Computed:    true,
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

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceId := model.InstanceId.ValueString()
	credentialId := model.CredentialId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)

	recordSetResp, err := r.client.DefaultAPI.GetCredentials(ctx, projectId, region, instanceId, credentialId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading credential",
			fmt.Sprintf("Credential with ID %q or instance with ID %q does not exist in project %q.", credentialId, instanceId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapDataSourceFields(ctx, recordSetResp, &model, region)
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
	tflog.Info(ctx, "Redis credential read")
}

func mapDataSourceFields(ctx context.Context, credentialsResp *redis.CredentialsResponse, model *DataSourceModel, region string) error {
	if credentialsResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if credentialsResp.Raw == nil {
		return fmt.Errorf("response credentials raw is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	credentials := credentialsResp.Raw.Credentials

	var credentialId string
	if model.CredentialId.ValueString() != "" {
		credentialId = model.CredentialId.ValueString()
	} else if credentialsResp.Id != "" {
		credentialId = credentialsResp.Id
	} else {
		return fmt.Errorf("credentials id not present")
	}

	model.Region = types.StringValue(region)
	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), model.Region.ValueString(), model.InstanceId.ValueString(), credentialId)

	modelHosts, err := utils.ListValueToStringSlice(model.Hosts)
	if err != nil {
		return err
	}

	model.Hosts = types.ListNull(types.StringType)
	model.CredentialId = types.StringValue(credentialId)

	if credentials.Hosts != nil {
		respHosts := credentials.Hosts

		reconciledHosts := utils.ReconcileStringSlices(modelHosts, respHosts)

		hostsTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledHosts)
		if diags.HasError() {
			return fmt.Errorf("failed to map hosts: %w", core.DiagsToError(diags))
		}

		model.Hosts = hostsTF
	}
	model.Host = types.StringValue(credentials.Host)
	model.LoadBalancedHost = types.StringPointerValue(credentials.LoadBalancedHost)
	model.Password = types.StringValue(credentials.Password)
	model.Port = types.Int32PointerValue(credentials.Port)
	model.Uri = types.StringPointerValue(credentials.Uri)
	model.Username = types.StringValue(credentials.Username)

	return nil
}
