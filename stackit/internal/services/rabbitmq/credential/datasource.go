package rabbitmq

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	rabbitmqUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/rabbitmq/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	rabbitmq "github.com/stackitcloud/stackit-sdk-go/services/rabbitmq/v2api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &credentialDataSource{}
)

type DataSourceModel struct {
	Id           types.String `tfsdk:"id"` // needed by TF
	CredentialId types.String `tfsdk:"credential_id"`
	InstanceId   types.String `tfsdk:"instance_id"`
	ProjectId    types.String `tfsdk:"project_id"`
	Region       types.String `tfsdk:"region"`
	Host         types.String `tfsdk:"host"`
	Hosts        types.List   `tfsdk:"hosts"`
	HttpAPIURI   types.String `tfsdk:"http_api_uri"`
	HttpAPIURIs  types.List   `tfsdk:"http_api_uris"`
	Management   types.String `tfsdk:"management"`
	Password     types.String `tfsdk:"password"`
	Port         types.Int32  `tfsdk:"port"`
	Uri          types.String `tfsdk:"uri"`
	Uris         types.List   `tfsdk:"uris"`
	Username     types.String `tfsdk:"username"`
}

// NewCredentialDataSource is a helper function to simplify the provider implementation.
func NewCredentialDataSource() datasource.DataSource {
	return &credentialDataSource{}
}

// credentialDataSource is the data source implementation.
type credentialDataSource struct {
	client       *rabbitmq.APIClient
	providerData core.ProviderData
}

// Metadata returns the data source type name.
func (r *credentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rabbitmq_credential"
}

// Configure adds the provider configured client to the data source.
func (r *credentialDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := rabbitmqUtils.ConfigureClient(ctx, &r.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "RabbitMQ credential client configured")
}

// Schema defines the schema for the data source.
func (r *credentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{ //nolint:gosec // description for credential id
		"main":          "RabbitMQ credential data source schema. Must have a `region` specified in the provider configuration.",
		"id":            "Terraform's internal data source. identifier. It is structured as \"`project_id`,`region`,`instance_id`,`credential_id`\".",
		"credential_id": "The credential's ID.",
		"instance_id":   "ID of the RabbitMQ instance.",
		"project_id":    "STACKIT project ID to which the instance is associated.",
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
			"http_api_uri": schema.StringAttribute{
				Computed: true,
			},
			"http_api_uris": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"management": schema.StringAttribute{
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
				Computed:  true,
				Sensitive: true,
			},
			"uris": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
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
	tflog.Info(ctx, "RabbitMQ credential read")
}

func mapDataSourceFields(ctx context.Context, credentialsResp *rabbitmq.CredentialsResponse, model *DataSourceModel, region string) error {
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
	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), model.Region.ValueString(), model.InstanceId.ValueString(), credentialId,
	)
	model.CredentialId = types.StringValue(credentialId)

	modelHosts, err := utils.ListValueToStringSlice(model.Hosts)
	if err != nil {
		return err
	}
	modelHttpApiUris, err := utils.ListValueToStringSlice(model.HttpAPIURIs)
	if err != nil {
		return err
	}
	modelUris, err := utils.ListValueToStringSlice(model.Uris)
	if err != nil {
		return err
	}

	model.Hosts = types.ListNull(types.StringType)
	model.Uris = types.ListNull(types.StringType)
	model.HttpAPIURIs = types.ListNull(types.StringType)
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
	if credentials.HttpApiUris != nil {
		respHttpApiUris := credentials.HttpApiUris

		reconciledHttpApiUris := utils.ReconcileStringSlices(modelHttpApiUris, respHttpApiUris)

		httpApiUrisTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledHttpApiUris)
		if diags.HasError() {
			return fmt.Errorf("failed to map httpApiUris: %w", core.DiagsToError(diags))
		}

		model.HttpAPIURIs = httpApiUrisTF
	}

	if credentials.Uris != nil {
		respUris := credentials.Uris

		reconciledUris := utils.ReconcileStringSlices(modelUris, respUris)

		urisTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledUris)
		if diags.HasError() {
			return fmt.Errorf("failed to map uris: %w", core.DiagsToError(diags))
		}

		model.Uris = urisTF
	}

	model.HttpAPIURI = types.StringPointerValue(credentials.HttpApiUri)
	model.Management = types.StringPointerValue(credentials.Management)
	model.Password = types.StringValue(credentials.Password)
	model.Port = types.Int32PointerValue(credentials.Port)
	model.Uri = types.StringPointerValue(credentials.Uri)
	model.Username = types.StringValue(credentials.Username)

	return nil
}
