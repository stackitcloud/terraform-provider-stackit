package securitygrouprule

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// securityGroupRuleDataSourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var securityGroupRuleDataSourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &securityGroupRuleDataSource{}
)

// NewSecurityGroupRuleDataSource is a helper function to simplify the provider implementation.
func NewSecurityGroupRuleDataSource() datasource.DataSource {
	return &securityGroupRuleDataSource{}
}

// securityGroupRuleDataSource is the data source implementation.
type securityGroupRuleDataSource struct {
	client *iaas.APIClient
}

// Metadata returns the data source type name.
func (d *securityGroupRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group_rule"
}

func (d *securityGroupRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var apiClient *iaas.APIClient
	var err error

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !securityGroupRuleDataSourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_security_group_rule", "data source")
		if resp.Diagnostics.HasError() {
			return
		}
		securityGroupRuleDataSourceBetaCheckDone = true
	}

	if providerData.IaaSCustomEndpoint != "" {
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.IaaSCustomEndpoint),
		)
	} else {
		apiClient, err = iaas.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.GetRegion()),
		)
	}
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the data source configuration", err))
		return
	}

	d.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (r *securityGroupRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	directionOptions := []string{"ingress", "egress"}

	resp.Schema = schema.Schema{
		MarkdownDescription: features.AddBetaDescription("Security group datasource schema. Must have a `region` specified in the provider configuration."),
		Description:         "Security group datasource schema. Must have a `region` specified in the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal datasource ID. It is structured as \"`project_id`,`security_group_id`,`security_group_rule_id`\".",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "STACKIT project ID to which the security group rule is associated.",
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
			"security_group_rule_id": schema.StringAttribute{
				Description: "The security group rule ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"direction": schema.StringAttribute{
				Description: "The direction of the traffic which the rule should match. Some of the possible values are: " + utils.SupportedValuesDocumentation(directionOptions),
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the security group rule.",
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
					"number": schema.Int64Attribute{
						Description: "The protocol number which the rule should match.",
						Computed:    true,
					},
				},
			},
			"remote_security_group_id": schema.StringAttribute{
				Description: "The remote security group which the rule should match.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *securityGroupRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	securityGroupId := model.SecurityGroupId.ValueString()
	securityGroupRuleId := model.SecurityGroupRuleId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "security_group_id", securityGroupId)
	ctx = tflog.SetField(ctx, "security_group_rule_id", securityGroupRuleId)

	securityGroupRuleResp, err := d.client.GetSecurityGroupRule(ctx, projectId, securityGroupId, securityGroupRuleId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading security group rule", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapFields(securityGroupRuleResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading security group rule", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "security group rule read")
}
