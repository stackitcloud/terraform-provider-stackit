package rabbitmq

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	rabbitmqUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/rabbitmq/utils"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq/wait"
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
	HttpAPIURIs  types.List   `tfsdk:"http_api_uris"`
	Management   types.String `tfsdk:"management"`
	Password     types.String `tfsdk:"password"`
	Port         types.Int64  `tfsdk:"port"`
	Uri          types.String `tfsdk:"uri"`
	Uris         types.List   `tfsdk:"uris"`
	Username     types.String `tfsdk:"username"`
}

// NewCredentialResource is a helper function to simplify the provider implementation.
func NewCredentialResource() resource.Resource {
	return &credentialResource{}
}

// credentialResource is the resource implementation.
type credentialResource struct {
	client *rabbitmq.APIClient
}

// Metadata returns the resource type name.
func (r *credentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rabbitmq_credential"
}

// Configure adds the provider configured client to the resource.
func (r *credentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := rabbitmqUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "RabbitMQ credential client configured")
}

// Schema defines the schema for the resource.
func (r *credentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":          "RabbitMQ credential resource schema. Must have a `region` specified in the provider configuration.",
		"id":            "Terraform's internal resource identifier. It is structured as \"`project_id`,`instance_id`,`credential_id`\".",
		"credential_id": "The credential's ID.",
		"instance_id":   "ID of the RabbitMQ instance.",
		"project_id":    "STACKIT Project ID to which the instance is associated.",
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
			"port": schema.Int64Attribute{
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
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *credentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// Create new recordset
	credentialsResp, err := r.client.CreateCredentials(ctx, projectId, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if credentialsResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", "Got empty credential id")
		return
	}
	credentialId := *credentialsResp.Id
	ctx = tflog.SetField(ctx, "credential_id", credentialId)

	waitResp, err := wait.CreateCredentialsWaitHandler(ctx, r.client, projectId, instanceId, credentialId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "RabbitMQ credential created")
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	credentialId := model.CredentialId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)

	recordSetResp, err := r.client.GetCredentials(ctx, projectId, instanceId, credentialId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, recordSetResp, &model)
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

// Update updates the resource and sets the updated Terraform state on success.
func (r *credentialResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating credential", "Credential can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *credentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	credentialId := model.CredentialId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "credential_id", credentialId)

	// Delete existing record set
	err := r.client.DeleteCredentials(ctx, projectId, instanceId, credentialId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", fmt.Sprintf("Calling API: %v", err))
	}
	_, err = wait.DeleteCredentialsWaitHandler(ctx, r.client, projectId, instanceId, credentialId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "RabbitMQ credential deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,credential_id
func (r *credentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing credential",
			fmt.Sprintf("Expected import identifier with format [project_id],[instance_id],[credential_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credential_id"), idParts[2])...)
	tflog.Info(ctx, "RabbitMQ credential state imported")
}

func mapFields(ctx context.Context, credentialsResp *rabbitmq.CredentialsResponse, model *Model) error {
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
	} else if credentialsResp.Id != nil {
		credentialId = *credentialsResp.Id
	} else {
		return fmt.Errorf("credentials id not present")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(), model.InstanceId.ValueString(), credentialId,
	)
	model.CredentialId = types.StringValue(credentialId)

	modelHosts, err := utils.ListValuetoStringSlice(model.Hosts)
	if err != nil {
		return err
	}
	modelHttpApiUris, err := utils.ListValuetoStringSlice(model.HttpAPIURIs)
	if err != nil {
		return err
	}
	modelUris, err := utils.ListValuetoStringSlice(model.Uris)
	if err != nil {
		return err
	}

	model.Hosts = types.ListNull(types.StringType)
	model.Uris = types.ListNull(types.StringType)
	model.HttpAPIURIs = types.ListNull(types.StringType)
	if credentials != nil {
		if credentials.Hosts != nil {
			respHosts := *credentials.Hosts
			reconciledHosts := utils.ReconcileStringSlices(modelHosts, respHosts)

			hostsTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledHosts)
			if diags.HasError() {
				return fmt.Errorf("failed to map hosts: %w", core.DiagsToError(diags))
			}

			model.Hosts = hostsTF
		}
		model.Host = types.StringPointerValue(credentials.Host)
		if credentials.HttpApiUris != nil {
			respHttpApiUris := *credentials.HttpApiUris

			reconciledHttpApiUris := utils.ReconcileStringSlices(modelHttpApiUris, respHttpApiUris)

			httpApiUrisTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledHttpApiUris)
			if diags.HasError() {
				return fmt.Errorf("failed to map httpApiUris: %w", core.DiagsToError(diags))
			}

			model.HttpAPIURIs = httpApiUrisTF
		}

		if credentials.Uris != nil {
			respUris := *credentials.Uris

			reconciledUris := utils.ReconcileStringSlices(modelUris, respUris)

			urisTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledUris)
			if diags.HasError() {
				return fmt.Errorf("failed to map uris: %w", core.DiagsToError(diags))
			}

			model.Uris = urisTF
		}

		model.HttpAPIURI = types.StringPointerValue(credentials.HttpApiUri)
		model.Management = types.StringPointerValue(credentials.Management)
		model.Password = types.StringPointerValue(credentials.Password)
		model.Port = types.Int64PointerValue(credentials.Port)
		model.Uri = types.StringPointerValue(credentials.Uri)
		model.Username = types.StringPointerValue(credentials.Username)
	}
	return nil
}
