package postgresql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresql"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresql/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &credentialsResource{}
	_ resource.ResourceWithConfigure   = &credentialsResource{}
	_ resource.ResourceWithImportState = &credentialsResource{}
)

type Model struct {
	Id            types.String `tfsdk:"id"` // needed by TF
	CredentialsId types.String `tfsdk:"credentials_id"`
	InstanceId    types.String `tfsdk:"instance_id"`
	ProjectId     types.String `tfsdk:"project_id"`
	Host          types.String `tfsdk:"host"`
	Hosts         types.List   `tfsdk:"hosts"`
	HttpAPIURI    types.String `tfsdk:"http_api_uri"`
	Name          types.String `tfsdk:"name"`
	Password      types.String `tfsdk:"password"`
	Port          types.Int64  `tfsdk:"port"`
	Uri           types.String `tfsdk:"uri"`
	Username      types.String `tfsdk:"username"`
}

// NewCredentialsResource is a helper function to simplify the provider implementation.
func NewCredentialsResource() resource.Resource {
	return &credentialsResource{}
}

// credentialsResource is the resource implementation.
type credentialsResource struct {
	client *postgresql.APIClient
}

// Metadata returns the resource type name.
func (r *credentialsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_credentials"
}

// Configure adds the provider configured client to the resource.
func (r *credentialsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "PostgreSQL credentials client configured")
}

// Schema defines the schema for the resource.
func (r *credentialsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":           "PostgreSQL credentials resource schema.",
		"id":             "Terraform's internal resource identifier. It is structured as \"`project_id`,`instance_id`,`credentials_id`\".",
		"credentials_id": "The credentials ID.",
		"instance_id":    "ID of the PostgreSQL instance.",
		"project_id":     "STACKIT Project ID to which the instance is associated.",
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
			"credentials_id": schema.StringAttribute{
				Description: descriptions["credentials_id"],
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
				Computed: true,
			},
			"username": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *credentialsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentials", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if credentialsResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentials", "Got empty credentials id")
		return
	}
	credentialsId := *credentialsResp.Id
	ctx = tflog.SetField(ctx, "credentials_id", credentialsId)

	wr, err := wait.CreateCredentialsWaitHandler(ctx, r.client, projectId, instanceId, credentialsId).SetTimeout(1 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentials", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}
	got, ok := wr.(*postgresql.CredentialsResponse)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentials", fmt.Sprintf("Wait result conversion, got %+v", wr))
		return
	}

	// Map response body to schema
	err = mapFields(got, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentials", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "PostgreSQL credentials created")
}

// Read refreshes the Terraform state with the latest data.
func (r *credentialsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	credentialsId := model.CredentialsId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "credentials_id", credentialsId)

	recordSetResp, err := r.client.GetCredentials(ctx, projectId, instanceId, credentialsId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credentials", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(recordSetResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credentials", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "PostgreSQL credentials read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *credentialsResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating credentials", "Credentials can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *credentialsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	credentialsId := model.CredentialsId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	ctx = tflog.SetField(ctx, "credentials_id", credentialsId)

	// Delete existing record set
	err := r.client.DeleteCredentials(ctx, projectId, instanceId, credentialsId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credentials", fmt.Sprintf("Calling API: %v", err))
	}
	_, err = wait.DeleteCredentialsWaitHandler(ctx, r.client, projectId, instanceId, credentialsId).SetTimeout(1 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credentials", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "PostgreSQL credentials deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,credentials_id
func (r *credentialsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing credentials",
			fmt.Sprintf("Expected import identifier with format [project_id],[instance_id],[credentials_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credentials_id"), idParts[2])...)
	tflog.Info(ctx, "PostgreSQL credentials state imported")
}

func mapFields(credentialsResp *postgresql.CredentialsResponse, model *Model) error {
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

	var credentialsId string
	if model.CredentialsId.ValueString() != "" {
		credentialsId = model.CredentialsId.ValueString()
	} else if credentialsResp.Id != nil {
		credentialsId = *credentialsResp.Id
	} else {
		return fmt.Errorf("credentials id not present")
	}

	idParts := []string{
		model.ProjectId.ValueString(),
		model.InstanceId.ValueString(),
		credentialsId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.CredentialsId = types.StringValue(credentialsId)
	model.Hosts = types.ListNull(types.StringType)
	if credentials != nil {
		if credentials.Hosts != nil {
			var hosts []attr.Value
			for _, host := range *credentials.Hosts {
				hosts = append(hosts, types.StringValue(host))
			}
			hostsList, diags := types.ListValue(types.StringType, hosts)
			if diags.HasError() {
				return fmt.Errorf("failed to map hosts: %w", core.DiagsToError(diags))
			}
			model.Hosts = hostsList
		}
		model.Host = types.StringPointerValue(credentials.Host)
		model.HttpAPIURI = types.StringPointerValue(credentials.HttpApiUri)
		model.Name = types.StringPointerValue(credentials.Name)
		model.Password = types.StringPointerValue(credentials.Password)
		model.Port = conversion.ToTypeInt64(credentials.Port)
		model.Uri = types.StringPointerValue(credentials.Uri)
		model.Username = types.StringPointerValue(credentials.Username)
	}
	return nil
}
