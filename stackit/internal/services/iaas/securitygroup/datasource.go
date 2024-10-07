package securitygroup

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// securityGroupDataSourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var securityGroupDataSourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &securityGroupDataSource{}
)

// NewSecurityGroupDataSource is a helper function to simplify the provider implementation.
func NewSecurityGroupDataSource() datasource.DataSource {
	return &securityGroupDataSource{}
}

// securityGroupDataSource is the data source implementation.
type securityGroupDataSource struct {
	client *iaasalpha.APIClient
}

// Metadata returns the data source type name.
func (d *securityGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

func (d *securityGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var apiClient *iaasalpha.APIClient
	var err error

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !securityGroupDataSourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_security_group", "data source")
		if resp.Diagnostics.HasError() {
			return
		}
		securityGroupDataSourceBetaCheckDone = true
	}

	if providerData.IaaSCustomEndpoint != "" {
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaasalpha.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.Region),
		)
	}
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	d.client = apiClient
	tflog.Info(ctx, "iaasalpha client configured")
}

// Schema defines the schema for the resource.
func (r *securityGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Security group datasource schema. Must have a `region` specified in the provider configuration."),
		Description:         "Security group datasource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`project_id`,`security_group_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the security group is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"security_group_id": schema.StringAttribute{
				Description: "The security group ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the security group.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the security group.",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
			"stateful": schema.BoolAttribute{
				Description: "Shows if a security group is stateful or stateless. There can only be one security groups per network interface/server.",
				Computed:    true,
			},
			"rules": schema.ListNestedAttribute{
				Description: "The rules of the security group.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Description: "The rule description.",
							Computed:    true,
						},
						"direction": schema.StringAttribute{
							Description: "The direction of the traffic which the rule should match.",
							Computed:    true,
						},
						"ether_type": schema.StringAttribute{
							Description: "The ethertype which the rule should match.",
							Computed:    true,
						},
						"icmp_parameters": schema.SingleNestedAttribute{
							Description: "ICMP Parameters.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"code": schema.Int64Attribute{
									Description: "ICMP code. Can be set if the protocol is ICMP.",
									Computed:    true,
								},
								"type": schema.Int64Attribute{
									Description: "ICMP type. Can be set if the protocol is ICMP.",
									Computed:    true,
								},
							},
						},
						"id": schema.StringAttribute{
							Description: "UUID",
							Computed:    true,
						},
						"ip_range": schema.StringAttribute{
							Description: "The remote IP range which the rule should match.",
							Computed:    true,
						},
						"port_range": schema.SingleNestedAttribute{
							Description: "The range of ports.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"max": schema.Int64Attribute{
									Description: "The maximum port number. Should be greater or equal to the minimum.",
									Computed:    true,
								},
								"min": schema.Int64Attribute{
									Description: "The minimum port number. Should be less or equal to the minimum.",
									Computed:    true,
								},
							},
						},
						"protocol": schema.SingleNestedAttribute{
							Description: "The internet protocol which the rule should match.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Description: "The protocol name which the rule should match.",
									Computed:    true,
								},
								"protocol": schema.Int64Attribute{
									Description: "The protocol number which the rule should match.",
									Computed:    true,
								},
							},
						},
						"remote_security_group_id": schema.StringAttribute{
							Description: "The remote security group which the rule should match.",
							Computed:    true,
						},
						"security_group_id": schema.StringAttribute{
							Description: "UUID",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *securityGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	securityGroupId := model.SecurityGroupId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)

	securityGroupResp, err := d.client.GetSecurityGroup(ctx, projectId, securityGroupId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading security group", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(ctx, securityGroupResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading security group", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "security group read")
}
