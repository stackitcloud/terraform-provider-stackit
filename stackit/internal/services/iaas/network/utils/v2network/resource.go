package v2network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	networkModel "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/model"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, client *iaasalpha.APIClient) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model networkModel.Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network

	network, err := client.CreateNetwork(ctx, projectId, region).CreateNetworkPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	networkId := *network.Id
	network, err = wait.CreateNetworkWaitHandler(ctx, client, projectId, region, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Network creation waiting: %v", err))
		return
	}

	ctx = tflog.SetField(ctx, "network_id", networkId)

	// Map response body to schema
	err = mapFields(ctx, network, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network created")
}

func Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse, client *iaasalpha.APIClient, providerData core.ProviderData) { // nolint:gocritic // function signature required by Terraform
	var model networkModel.Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	region := providerData.GetRegionWithOverride(model.Region)
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "region", region)

	networkResp, err := client.GetNetwork(ctx, projectId, region, networkId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	// Map response body to schema
	err = mapFields(ctx, networkResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network read")
}

func Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse, client *iaasalpha.APIClient) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model networkModel.Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	region := model.Region.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "region", region)

	// Retrieve values from state
	var stateModel networkModel.Model
	diags = req.State.Get(ctx, &stateModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(ctx, &model, &stateModel)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing network
	err = client.PartialUpdateNetwork(ctx, projectId, region, networkId).PartialUpdateNetworkPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)
	waitResp, err := wait.UpdateNetworkWaitHandler(ctx, client, projectId, region, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Network update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network updated")
}

func Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse, client *iaasalpha.APIClient) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model networkModel.Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	region := model.Region.ValueString()
	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)
	ctx = tflog.SetField(ctx, "region", region)

	// Delete existing network
	err := client.DeleteNetwork(ctx, projectId, region, networkId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)
	_, err = wait.DeleteNetworkWaitHandler(ctx, client, projectId, region, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network", fmt.Sprintf("Network deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Network deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,region,network_id
func ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network",
			fmt.Sprintf("Expected import identifier with format: [project_id],[region],[network_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	region := idParts[1]
	networkId := idParts[2]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), region)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_id"), networkId)...)
	tflog.Info(ctx, "Network state imported")
}

func mapFields(ctx context.Context, networkResp *iaasalpha.Network, model *networkModel.Model, region string) error {
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

	if networkResp.Ipv4 == nil || networkResp.Ipv4.Nameservers == nil {
		model.Nameservers = types.ListNull(types.StringType)
		model.IPv4Nameservers = types.ListNull(types.StringType)
	} else {
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

	if networkResp.Ipv4 == nil || networkResp.Ipv4.Prefixes == nil {
		model.Prefixes = types.ListNull(types.StringType)
		model.IPv4Prefixes = types.ListNull(types.StringType)
	} else {
		respPrefixes := *networkResp.Ipv4.Prefixes
		prefixesTF, diags := types.ListValueFrom(ctx, types.StringType, respPrefixes)
		if diags.HasError() {
			return fmt.Errorf("map network prefixes: %w", core.DiagsToError(diags))
		}
		if len(respPrefixes) > 0 {
			model.IPv4Prefix = types.StringValue(respPrefixes[0])
			_, netmask, err := net.ParseCIDR(respPrefixes[0])
			if err != nil {
				tflog.Error(ctx, fmt.Sprintf("ipv4_prefix_length: %+v", err))
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

	if networkResp.Ipv4 == nil || networkResp.Ipv4.Gateway == nil {
		model.IPv4Gateway = types.StringNull()
	} else {
		model.IPv4Gateway = types.StringPointerValue(networkResp.Ipv4.GetGateway())
	}

	if networkResp.Ipv4 == nil || networkResp.Ipv4.PublicIp == nil {
		model.PublicIP = types.StringNull()
	} else {
		model.PublicIP = types.StringPointerValue(networkResp.Ipv4.PublicIp)
	}

	// IPv6

	if networkResp.Ipv6 == nil || networkResp.Ipv6.Nameservers == nil {
		model.IPv6Nameservers = types.ListNull(types.StringType)
	} else {
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

	if networkResp.Ipv6 == nil || networkResp.Ipv6.Prefixes == nil {
		model.IPv6Prefixes = types.ListNull(types.StringType)
	} else {
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

	if networkResp.Ipv6 == nil || networkResp.Ipv6.Gateway == nil {
		model.IPv6Gateway = types.StringNull()
	} else {
		model.IPv6Gateway = types.StringPointerValue(networkResp.Ipv6.GetGateway())
	}

	if networkResp.RoutingTableId != nil {
		model.RoutingTableID = types.StringPointerValue(networkResp.RoutingTableId)
	} else {
		model.RoutingTableID = types.StringNull()
	}

	model.NetworkId = types.StringValue(networkId)
	model.Name = types.StringPointerValue(networkResp.Name)
	model.Labels = labels
	model.Routed = types.BoolPointerValue(networkResp.Routed)
	model.Region = types.StringValue(region)

	return nil
}

func toCreatePayload(ctx context.Context, model *networkModel.Model) (*iaasalpha.CreateNetworkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var modelIPv6Nameservers []string
	// Is true when IPv6Nameservers is not null or unset
	if !utils.IsUndefined(model.IPv6Nameservers) {
		// If ipv6Nameservers is empty, modelIPv6Nameservers will be set to an empty slice.
		// empty slice != nil slice. Empty slice will result in an empty list in the payload []. Nil slice will result in a payload without the property set
		modelIPv6Nameservers = []string{}
		for _, ipv6ns := range model.IPv6Nameservers.Elements() {
			ipv6NameserverString, ok := ipv6ns.(types.String)
			if !ok {
				return nil, fmt.Errorf("type assertion failed")
			}
			modelIPv6Nameservers = append(modelIPv6Nameservers, ipv6NameserverString.ValueString())
		}
	}

	var ipv6Body *iaasalpha.CreateNetworkIPv6
	if !utils.IsUndefined(model.IPv6PrefixLength) {
		ipv6Body = &iaasalpha.CreateNetworkIPv6{
			CreateNetworkIPv6WithPrefixLength: &iaasalpha.CreateNetworkIPv6WithPrefixLength{
				PrefixLength: conversion.Int64ValueToPointer(model.IPv6PrefixLength),
			},
		}
		// IPv6 nameservers should only be set, if it contains any value. If the slice is nil, it should NOT be set.
		// Setting it to a nil slice would result in a payload, where nameservers is set to null in the json payload,
		// but it should actually be unset. Setting it to "null" will result in an error, because it's NOT nullable.
		if modelIPv6Nameservers != nil {
			ipv6Body.CreateNetworkIPv6WithPrefixLength.Nameservers = &modelIPv6Nameservers
		}
	} else if !utils.IsUndefined(model.IPv6Prefix) {
		var gateway *iaasalpha.NullableString
		if model.NoIPv6Gateway.ValueBool() {
			gateway = iaasalpha.NewNullableString(nil)
		} else if !(model.IPv6Gateway.IsUnknown() || model.IPv6Gateway.IsNull()) {
			gateway = iaasalpha.NewNullableString(conversion.StringValueToPointer(model.IPv6Gateway))
		}

		ipv6Body = &iaasalpha.CreateNetworkIPv6{
			CreateNetworkIPv6WithPrefix: &iaasalpha.CreateNetworkIPv6WithPrefix{
				Gateway: gateway,
				Prefix:  conversion.StringValueToPointer(model.IPv6Prefix),
			},
		}
		// IPv6 nameservers should only be set, if it contains any value. If the slice is nil, it should NOT be set.
		// Setting it to a nil slice would result in a payload, where nameservers is set to null in the json payload,
		// but it should actually be unset. Setting it to "null" will result in an error, because it's NOT nullable.
		if modelIPv6Nameservers != nil {
			ipv6Body.CreateNetworkIPv6WithPrefix.Nameservers = &modelIPv6Nameservers
		}
	}

	modelIPv4Nameservers := []string{}
	var modelIPv4List []attr.Value

	if !(model.IPv4Nameservers.IsNull() || model.IPv4Nameservers.IsUnknown()) {
		modelIPv4List = model.IPv4Nameservers.Elements()
	} else {
		modelIPv4List = model.Nameservers.Elements()
	}

	for _, ipv4ns := range modelIPv4List {
		ipv4NameserverString, ok := ipv4ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelIPv4Nameservers = append(modelIPv4Nameservers, ipv4NameserverString.ValueString())
	}

	var ipv4Body *iaasalpha.CreateNetworkIPv4
	if !utils.IsUndefined(model.IPv4PrefixLength) {
		ipv4Body = &iaasalpha.CreateNetworkIPv4{
			CreateNetworkIPv4WithPrefixLength: &iaasalpha.CreateNetworkIPv4WithPrefixLength{
				Nameservers:  &modelIPv4Nameservers,
				PrefixLength: conversion.Int64ValueToPointer(model.IPv4PrefixLength),
			},
		}
	} else if !utils.IsUndefined(model.IPv4Prefix) {
		var gateway *iaasalpha.NullableString
		if model.NoIPv4Gateway.ValueBool() {
			gateway = iaasalpha.NewNullableString(nil)
		} else if !(model.IPv4Gateway.IsUnknown() || model.IPv4Gateway.IsNull()) {
			gateway = iaasalpha.NewNullableString(conversion.StringValueToPointer(model.IPv4Gateway))
		}

		ipv4Body = &iaasalpha.CreateNetworkIPv4{
			CreateNetworkIPv4WithPrefix: &iaasalpha.CreateNetworkIPv4WithPrefix{
				Nameservers: &modelIPv4Nameservers,
				Prefix:      conversion.StringValueToPointer(model.IPv4Prefix),
				Gateway:     gateway,
			},
		}
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	payload := iaasalpha.CreateNetworkPayload{
		Name:           conversion.StringValueToPointer(model.Name),
		Labels:         &labels,
		Routed:         conversion.BoolValueToPointer(model.Routed),
		Ipv4:           ipv4Body,
		Ipv6:           ipv6Body,
		RoutingTableId: conversion.StringValueToPointer(model.RoutingTableID),
	}

	return &payload, nil
}

func toUpdatePayload(ctx context.Context, model, stateModel *networkModel.Model) (*iaasalpha.PartialUpdateNetworkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	var modelIPv6Nameservers []string
	// Is true when IPv6Nameservers is not null or unset
	if !utils.IsUndefined(model.IPv6Nameservers) {
		// If ipv6Nameservers is empty, modelIPv6Nameservers will be set to an empty slice.
		// empty slice != nil slice. Empty slice will result in an empty list in the payload []. Nil slice will result in a payload without the property set
		modelIPv6Nameservers = []string{}
		for _, ipv6ns := range model.IPv6Nameservers.Elements() {
			ipv6NameserverString, ok := ipv6ns.(types.String)
			if !ok {
				return nil, fmt.Errorf("type assertion failed")
			}
			modelIPv6Nameservers = append(modelIPv6Nameservers, ipv6NameserverString.ValueString())
		}
	}

	var ipv6Body *iaasalpha.UpdateNetworkIPv6Body
	if modelIPv6Nameservers != nil || !utils.IsUndefined(model.NoIPv6Gateway) || !utils.IsUndefined(model.IPv6Gateway) {
		ipv6Body = &iaasalpha.UpdateNetworkIPv6Body{}
		// IPv6 nameservers should only be set, if it contains any value. If the slice is nil, it should NOT be set.
		// Setting it to a nil slice would result in a payload, where nameservers is set to null in the json payload,
		// but it should actually be unset. Setting it to "null" will result in an error, because it's NOT nullable.
		if modelIPv6Nameservers != nil {
			ipv6Body.Nameservers = &modelIPv6Nameservers
		}

		if model.NoIPv6Gateway.ValueBool() {
			ipv6Body.Gateway = iaasalpha.NewNullableString(nil)
		} else if !(model.IPv6Gateway.IsUnknown() || model.IPv6Gateway.IsNull()) {
			ipv6Body.Gateway = iaasalpha.NewNullableString(conversion.StringValueToPointer(model.IPv6Gateway))
		}
	}

	modelIPv4Nameservers := []string{}
	var modelIPv4List []attr.Value

	if !(model.IPv4Nameservers.IsNull() || model.IPv4Nameservers.IsUnknown()) {
		modelIPv4List = model.IPv4Nameservers.Elements()
	} else {
		modelIPv4List = model.Nameservers.Elements()
	}
	for _, ipv4ns := range modelIPv4List {
		ipv4NameserverString, ok := ipv4ns.(types.String)
		if !ok {
			return nil, fmt.Errorf("type assertion failed")
		}
		modelIPv4Nameservers = append(modelIPv4Nameservers, ipv4NameserverString.ValueString())
	}

	var ipv4Body *iaasalpha.UpdateNetworkIPv4Body
	if !model.IPv4Nameservers.IsNull() || !model.Nameservers.IsNull() {
		ipv4Body = &iaasalpha.UpdateNetworkIPv4Body{
			Nameservers: &modelIPv4Nameservers,
		}

		if model.NoIPv4Gateway.ValueBool() {
			ipv4Body.Gateway = iaasalpha.NewNullableString(nil)
		} else if !(model.IPv4Gateway.IsUnknown() || model.IPv4Gateway.IsNull()) {
			ipv4Body.Gateway = iaasalpha.NewNullableString(conversion.StringValueToPointer(model.IPv4Gateway))
		}
	}
	currentLabels := stateModel.Labels
	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	payload := iaasalpha.PartialUpdateNetworkPayload{
		Name:           conversion.StringValueToPointer(model.Name),
		Labels:         &labels,
		Ipv4:           ipv4Body,
		Ipv6:           ipv6Body,
		RoutingTableId: conversion.StringValueToPointer(model.RoutingTableID),
	}

	return &payload, nil
}
