package v2network

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	networkModel "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/model"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func DatasourceRead(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse, client *iaasalpha.APIClient, providerData core.ProviderData) { // nolint:gocritic // function signature required by Terraform
	var model networkModel.DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	region := providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	networkResp, err := client.GetNetwork(ctx, projectId, region, networkId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading network",
			fmt.Sprintf("Network with ID %q does not exist in project %q.", networkId, projectId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Project with ID %q not found or forbidden access", projectId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapDataSourceFields(ctx, networkResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network read")
}

func mapDataSourceFields(ctx context.Context, networkResp *iaasalpha.Network, model *networkModel.DataSourceModel, region string) error {
	if networkResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkId string
	if model.NetworkId.ValueString() != "" {
		networkId = model.NetworkId.ValueString()
	} else if networkResp.Id != nil {
		networkId = *networkResp.Id
	} else {
		return fmt.Errorf("network id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, networkId)

	labels, err := iaasUtils.MapLabels(ctx, networkResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	// IPv4

	model.Nameservers = types.ListNull(types.StringType)
	model.IPv4Nameservers = types.ListNull(types.StringType)
	model.Prefixes = types.ListNull(types.StringType)
	model.IPv4Prefixes = types.ListNull(types.StringType)
	model.IPv4Gateway = types.StringNull()
	model.PublicIP = types.StringNull()
	if networkResp.Ipv4 != nil {
		if networkResp.Ipv4.Nameservers != nil {
			respNameservers := *networkResp.Ipv4.Nameservers
			modelNameservers, err := utils.ListValuetoStringSlice(model.Nameservers)
			modelIPv4Nameservers, errIpv4 := utils.ListValuetoStringSlice(model.IPv4Nameservers)
			if err != nil {
				return fmt.Errorf("get current network nameservers from model: %w", err)
			}
			if errIpv4 != nil {
				return fmt.Errorf("get current IPv4 network nameservers from model: %w", errIpv4)
			}

			reconciledNameservers := utils.ReconcileStringSlices(modelNameservers, respNameservers)
			reconciledIPv4Nameservers := utils.ReconcileStringSlices(modelIPv4Nameservers, respNameservers)

			nameserversTF, diags := types.ListValueFrom(ctx, types.StringType, reconciledNameservers)
			ipv4NameserversTF, ipv4Diags := types.ListValueFrom(ctx, types.StringType, reconciledIPv4Nameservers)
			if diags.HasError() {
				return fmt.Errorf("map network nameservers: %w", core.DiagsToError(diags))
			}
			if ipv4Diags.HasError() {
				return fmt.Errorf("map IPv4 network nameservers: %w", core.DiagsToError(ipv4Diags))
			}

			model.Nameservers = nameserversTF
			model.IPv4Nameservers = ipv4NameserversTF
		}

		if networkResp.Ipv4.Prefixes != nil {
			respPrefixes := *networkResp.Ipv4.Prefixes
			prefixesTF, diags := types.ListValueFrom(ctx, types.StringType, respPrefixes)
			if diags.HasError() {
				return fmt.Errorf("map network prefixes: %w", core.DiagsToError(diags))
			}
			if len(respPrefixes) > 0 {
				model.IPv4Prefix = types.StringValue(respPrefixes[0])
				_, netmask, err := net.ParseCIDR(respPrefixes[0])
				if err != nil {
					// silently ignore parsing error for the netmask
					model.IPv4PrefixLength = types.Int64Null()
				} else {
					ones, _ := netmask.Mask.Size()
					model.IPv4PrefixLength = types.Int64Value(int64(ones))
				}
			}

			model.Prefixes = prefixesTF
			model.IPv4Prefixes = prefixesTF
		}

		if networkResp.Ipv4.Gateway != nil {
			model.IPv4Gateway = types.StringPointerValue(networkResp.Ipv4.GetGateway())
		}

		if networkResp.Ipv4.PublicIp != nil {
			model.PublicIP = types.StringPointerValue(networkResp.Ipv4.PublicIp)
		}
	}

	// IPv6

	model.IPv6Nameservers = types.ListNull(types.StringType)
	model.IPv6Prefixes = types.ListNull(types.StringType)
	model.IPv6Gateway = types.StringNull()
	if networkResp.Ipv6 != nil {
		if networkResp.Ipv6.Nameservers != nil {
			respIPv6Nameservers := *networkResp.Ipv6.Nameservers
			modelIPv6Nameservers, errIpv6 := utils.ListValuetoStringSlice(model.IPv6Nameservers)
			if errIpv6 != nil {
				return fmt.Errorf("get current IPv6 network nameservers from model: %w", errIpv6)
			}

			reconciledIPv6Nameservers := utils.ReconcileStringSlices(modelIPv6Nameservers, respIPv6Nameservers)

			ipv6NameserversTF, ipv6Diags := types.ListValueFrom(ctx, types.StringType, reconciledIPv6Nameservers)
			if ipv6Diags.HasError() {
				return fmt.Errorf("map IPv6 network nameservers: %w", core.DiagsToError(ipv6Diags))
			}

			model.IPv6Nameservers = ipv6NameserversTF
		}

		if networkResp.Ipv6.Prefixes != nil {
			respPrefixesV6 := *networkResp.Ipv6.Prefixes
			prefixesV6TF, diags := types.ListValueFrom(ctx, types.StringType, respPrefixesV6)
			if diags.HasError() {
				return fmt.Errorf("map network IPv6 prefixes: %w", core.DiagsToError(diags))
			}
			if len(respPrefixesV6) > 0 {
				model.IPv6Prefix = types.StringValue(respPrefixesV6[0])
				_, netmask, err := net.ParseCIDR(respPrefixesV6[0])
				if err != nil {
					// silently ignore parsing error for the netmask
					model.IPv6PrefixLength = types.Int64Null()
				} else {
					ones, _ := netmask.Mask.Size()
					model.IPv6PrefixLength = types.Int64Value(int64(ones))
				}
			}
			model.IPv6Prefixes = prefixesV6TF
		}

		if networkResp.Ipv6.Gateway != nil {
			model.IPv6Gateway = types.StringPointerValue(networkResp.Ipv6.GetGateway())
		}
	}

	model.RoutingTableID = types.StringNull()
	if networkResp.RoutingTableId != nil {
		model.RoutingTableID = types.StringValue(*networkResp.RoutingTableId)
	}

	model.NetworkId = types.StringValue(networkId)
	model.Name = types.StringPointerValue(networkResp.Name)
	model.Labels = labels
	model.Routed = types.BoolPointerValue(networkResp.Routed)
	model.Region = types.StringValue(region)

	return nil
}
