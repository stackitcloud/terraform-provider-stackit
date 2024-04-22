package postgresflex

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &instanceDataSource{}
)

// NewInstanceDataSource is a helper function to simplify the provider implementation.
func NewInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

// instanceDataSource is the data source implementation.
type instanceDataSource struct {
	client *postgresflex.APIClient
}

// Metadata returns the data source type name.
func (r *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresflex_instance"
}

// Configure adds the provider configured client to the data source.
func (r *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *postgresflex.APIClient
	var err error
	if providerData.PostgresFlexCustomEndpoint != "" {
		apiClient, err = postgresflex.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.PostgresFlexCustomEndpoint),
		)
	} else {
		apiClient, err = postgresflex.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "PostgresFlex instance client configured")
}

// Schema defines the schema for the data source.
func (r *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "PostgresFlex instance data source schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal data source. ID. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the PostgresFlex instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"name":        "Instance name.",
		"acl":         "The Access Control List (ACL) for the PostgresFlex instance.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
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
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"acl": schema.ListAttribute{
				Description: descriptions["acl"],
				ElementType: types.StringType,
				Computed:    true,
			},
			"backup_schedule": schema.StringAttribute{
				Computed: true,
			},
			"flavor": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"description": schema.StringAttribute{
						Computed: true,
					},
					"cpu": schema.Int64Attribute{
						Computed: true,
					},
					"ram": schema.Int64Attribute{
						Computed: true,
					},
				},
			},
			"replicas": schema.Int64Attribute{
				Computed: true,
			},
			"storage": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"class": schema.StringAttribute{
						Computed: true,
					},
					"size": schema.Int64Attribute{
						Computed: true,
					},
				},
			},
			"version": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	instanceResp, err := r.client.GetInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	if instanceResp != nil && instanceResp.Item != nil && instanceResp.Item.Status != nil && *instanceResp.Item.Status == wait.InstanceStateDeleted {
		resp.State.RemoveResource(ctx)
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", "Instance was deleted successfully")
		return
	}

	var flavor = &flavorModel{}
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		diags = model.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var storage = &storageModel{}
	if !(model.Storage.IsNull() || model.Storage.IsUnknown()) {
		diags = model.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	err = mapFields(ctx, instanceResp, &model, flavor, storage)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "PostgresFlex instance read")
}
