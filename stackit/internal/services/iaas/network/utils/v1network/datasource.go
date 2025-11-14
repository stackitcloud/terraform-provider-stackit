package v1network

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	networkModel "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/model"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func DatasourceRead(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse, client *iaas.APIClient) { // nolint:gocritic // function signature required by Terraform
	var model networkModel.DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = core.InitProviderContext(ctx)
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	networkResp, err := client.GetNetwork(ctx, projectId, networkId).Execute()
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
	ctx = core.LogResponse(ctx)

	err = mapDataSourceFields(ctx, networkResp, &model)
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

func mapDataSourceFields(ctx context.Context, networkResp *iaas.Network, model *networkModel.DataSourceModel) error {
	if networkResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var networkId string
	if model.NetworkId.ValueString() != "" {
		networkId = model.NetworkId.ValueString()
	} else if networkResp.NetworkId != nil {
		networkId = *networkResp.NetworkId
	} else {
		return fmt.Errorf("network id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), networkId)

	labels, err := iaasUtils.MapLabels(ctx, networkResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	// IPv4

	if networkResp.Nameservers == nil {
		model.Nameservers = types.ListNull(types.StringType)
		model.IPv4Nameservers = types.ListNull(types.StringType)
	} else {
		respNameservers := *networkResp.Nameservers
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

	if networkResp.Prefixes == nil {
		model.Prefixes = types.ListNull(types.StringType)
		model.IPv4Prefixes = types.ListNull(types.StringType)
	} else {
		respPrefixes := *networkResp.Prefixes
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

	model.IPv4Gateway = types.StringNull()
	if networkResp.Gateway != nil {
		model.IPv4Gateway = types.StringPointerValue(networkResp.GetGateway())
	}

	// IPv6

	if networkResp.NameserversV6 == nil {
		model.IPv6Nameservers = types.ListNull(types.StringType)
	} else {
		respIPv6Nameservers := *networkResp.NameserversV6
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

	if networkResp.PrefixesV6 == nil {
		model.IPv6Prefixes = types.ListNull(types.StringType)
	} else {
		respPrefixesV6 := *networkResp.PrefixesV6
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

	model.IPv6Gateway = types.StringNull()
	if networkResp.Gatewayv6 != nil {
		model.IPv6Gateway = types.StringPointerValue(networkResp.GetGatewayv6())
	}

	model.NetworkId = types.StringValue(networkId)
	model.Name = types.StringPointerValue(networkResp.Name)
	model.PublicIP = types.StringPointerValue(networkResp.PublicIp)
	model.Labels = labels
	model.Routed = types.BoolPointerValue(networkResp.Routed)
	model.RoutingTableID = types.StringNull()
	model.Region = types.StringNull()

	return nil
}
