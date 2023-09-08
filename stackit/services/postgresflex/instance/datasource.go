package postgresflex

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/validate"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex"
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

// Metadata returns the resource type name.
func (r *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresflex_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected stackit.ProviderData, got %T", req.ProviderData))
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
		resp.Diagnostics.AddError("Could not Configure API Client", err.Error())
		return
	}

	tflog.Info(ctx, "Postgresflex instance client configured")
	r.client = apiClient
}

// Schema defines the schema for the resource.
func (r *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "PostgresFlex instance data source schema.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
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
	var state Model
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := state.ProjectId.ValueString()
	instanceId := state.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	instanceResp, err := r.client.GetInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Unable to read instance", err.Error())
		return
	}

	var flavor = &flavorModel{}
	if !(state.Flavor.IsNull() || state.Flavor.IsUnknown()) {
		diags = state.Flavor.As(ctx, flavor, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var storage = &storageModel{}
	if !(state.Storage.IsNull() || state.Storage.IsUnknown()) {
		diags = state.Storage.As(ctx, storage, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	err = mapFields(instanceResp, &state, flavor, storage)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Mapping fields", err.Error())
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Postgresql instance read")
}
