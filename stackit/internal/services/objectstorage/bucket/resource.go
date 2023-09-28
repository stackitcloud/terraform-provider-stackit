package objectstorage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &bucketResource{}
	_ resource.ResourceWithConfigure   = &bucketResource{}
	_ resource.ResourceWithImportState = &bucketResource{}
)

type Model struct {
	Id                    types.String `tfsdk:"id"` // needed by TF
	BucketName            types.String `tfsdk:"bucket_name"`
	ProjectId             types.String `tfsdk:"project_id"`
	URLPathStyle          types.String `tfsdk:"url_path_style"`
	URLVirtualHostedStyle types.String `tfsdk:"url_virtual_hosted_style"`
}

// NewBucketResource is a helper function to simplify the provider implementation.
func NewBucketResource() resource.Resource {
	return &bucketResource{}
}

// bucketResource is the resource implementation.
type bucketResource struct {
	client *objectstorage.APIClient
}

// Metadata returns the resource type name.
func (r *bucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objectstorage_bucket"
}

// Configure adds the provider configured client to the resource.
func (r *bucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *objectstorage.APIClient
	var err error
	if providerData.ObjectStorageCustomEndpoint != "" {
		apiClient, err = objectstorage.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ObjectStorageCustomEndpoint),
		)
	} else {
		apiClient, err = objectstorage.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "ObjectStorage bucket client configured")
}

// Schema defines the schema for the resource.
func (r *bucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":                     "ObjectStorage bucket resource schema.",
		"id":                       "Terraform's internal resource identifier. It is structured as \"`project_id`,`bucket_name`\".",
		"bucket_name":              "The bucket name. Has to be DNS conform.",
		"project_id":               "STACKIT Project ID to which the bucket is associated.",
		"url_path_style":           "URL in path style.",
		"url_virtual_hosted_style": "URL in virtual hosted style.",
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
			"bucket_name": schema.StringAttribute{
				Description: descriptions["bucket_name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
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
			"url_path_style": schema.StringAttribute{
				Computed: true,
			},
			"url_virtual_hosted_style": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	bucketName := model.BucketName.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bucket_name", bucketName)

	// Create new recordset
	_, err := r.client.CreateBucket(ctx, projectId, bucketName).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating bucket", fmt.Sprintf("Calling API: %v", err))
		return
	}

	wr, err := objectstorage.CreateBucketWaitHandler(ctx, r.client, projectId, bucketName).SetTimeout(1 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating bucket", fmt.Sprintf("Bucket creation waiting: %v", err))
		return
	}
	got, ok := wr.(*objectstorage.GetBucketResponse)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating bucket", fmt.Sprintf("Wait result conversion, got %+v", wr))
		return
	}

	// Map response body to schema
	err = mapFields(got, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating bucket", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage bucket created")
}

// Read refreshes the Terraform state with the latest data.
func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	bucketName := model.BucketName.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bucket_name", bucketName)

	bucketResp, err := r.client.GetBucket(ctx, projectId, bucketName).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading bucket", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(bucketResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading bucket", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "ObjectStorage bucket read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *bucketResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Update shouldn't be called
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating bucket", "Bucket can't be updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	bucketName := model.BucketName.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bucket_name", bucketName)

	// Delete existing bucket
	_, err := r.client.DeleteBucket(ctx, projectId, bucketName).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting bucket", fmt.Sprintf("Calling API: %v", err))
	}
	_, err = objectstorage.DeleteBucketWaitHandler(ctx, r.client, projectId, bucketName).SetTimeout(1 * time.Minute).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting bucket", fmt.Sprintf("Bucket deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "ObjectStorage bucket deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,bucket_name
func (r *bucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing bucket",
			fmt.Sprintf("Expected import identifier with format [project_id],[bucket_name], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket_name"), idParts[1])...)
	tflog.Info(ctx, "ObjectStorage bucket state imported")
}

func mapFields(bucketResp *objectstorage.GetBucketResponse, model *Model) error {
	if bucketResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if bucketResp.Bucket == nil {
		return fmt.Errorf("response bucket is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	bucket := bucketResp.Bucket

	idParts := []string{
		model.ProjectId.ValueString(),
		model.BucketName.ValueString(),
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)
	model.URLPathStyle = types.StringPointerValue(bucket.UrlPathStyle)
	model.URLVirtualHostedStyle = types.StringPointerValue(bucket.UrlVirtualHostedStyle)
	return nil
}
