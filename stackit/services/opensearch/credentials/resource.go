package opensearch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/validate"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &openSearchCredentialsResource{}
	_ resource.ResourceWithConfigure   = &openSearchCredentialsResource{}
	_ resource.ResourceWithImportState = &openSearchCredentialsResource{}
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
	return &openSearchCredentialsResource{}
}

// credentialsResource is the resource implementation.
type openSearchCredentialsResource struct {
	client *opensearch.APIClient
}

// Metadata returns the resource type name.
func (r *openSearchCredentialsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_opensearch_credentials"
}

// Configure adds the provider configured client to the resource.
func (r *openSearchCredentialsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected stackit.ProviderData, got %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}

	var apiClient *opensearch.APIClient
	var err error
	if providerData.OpenSearchCustomEndpoint != "" {
		apiClient, err = opensearch.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.OpenSearchCustomEndpoint),
		)
	} else {
		apiClient, err = opensearch.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		resp.Diagnostics.AddError("Could not Configure API Client", err.Error())
		return
	}

	tflog.Info(ctx, "OpenSearch zone client configured")
	r.client = apiClient
}

// Schema defines the schema for the resource.
func (r *openSearchCredentialsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":           "OpenSearch credentials resource schema.",
		"id":             "Terraform's internal resource identifier. It is structured as \"`project_id`,`instance_id`,`credentials_id`\".",
		"credentials_id": "The credentials ID.",
		"instance_id":    "ID of the OpenSearch instance.",
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
func (r *openSearchCredentialsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
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

	wr, err := opensearch.CreateCredentialsWaitHandler(ctx, r.client, projectId, instanceId, credentialsId).SetTimeout(1 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentials", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}
	got, ok := wr.(*opensearch.CredentialsResponse)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credentials", fmt.Sprintf("Wait result conversion, got %+v", got))
		return
	}

	// Map response body to schema and populate Computed attribute values
	err = mapFields(got, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error mapping fields", err.Error())
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "OpenSearch credentials created")
}

// Read refreshes the Terraform state with the latest data.
func (r *openSearchCredentialsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credentials", err.Error())
		return
	}

	// Map response body to schema and populate Computed attribute values
	err = mapFields(recordSetResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error mapping fields", err.Error())
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "OpenSearch credentials read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *openSearchCredentialsResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	resp.Diagnostics.AddError("Error updating credentials", "credentials can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *openSearchCredentialsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
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
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credentials", err.Error())
	}
	_, err = opensearch.DeleteCredentialsWaitHandler(ctx, r.client, projectId, instanceId, credentialsId).SetTimeout(1 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credentials", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "OpenSearch credentials deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id,credentials_id
func (r *openSearchCredentialsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format [project_id],[instance_id],[credentials_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credentials_id"), idParts[2])...)
	tflog.Info(ctx, "OpenSearch credentials state imported")
}

func mapFields(credentialsResp *opensearch.CredentialsResponse, model *Model) error {
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
