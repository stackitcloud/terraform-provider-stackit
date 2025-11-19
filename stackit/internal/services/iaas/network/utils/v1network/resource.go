package v1network

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
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	networkModel "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/network/utils/model"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, client *iaas.APIClient) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model networkModel.Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Generate API request body from model
	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Create new network

	network, err := client.CreateNetwork(ctx, projectId).CreateNetworkPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Calling API: %v", err))
		return
	}

	networkId := *network.NetworkId
	network, err = wait.CreateNetworkWaitHandler(ctx, client, projectId, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating network", fmt.Sprintf("Network creation waiting: %v", err))
		return
	}

	ctx = tflog.SetField(ctx, "network_id", networkId)

	// Map response body to schema
	err = mapFields(ctx, network, &model)
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

func Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse, client *iaas.APIClient) { // nolint:gocritic // function signature required by Terraform
	var model networkModel.Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	networkResp, err := client.GetNetwork(ctx, projectId, networkId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(ctx, networkResp, &model)
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

func Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse, client *iaas.APIClient) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from plan
	var model networkModel.Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

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
	err = client.PartialUpdateNetwork(ctx, projectId, networkId).PartialUpdateNetworkPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.UpdateNetworkWaitHandler(ctx, client, projectId, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating network", fmt.Sprintf("Network update waiting: %v", err))
		return
	}

	err = mapFields(ctx, waitResp, &model)
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

func Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse, client *iaas.APIClient) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model networkModel.Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	networkId := model.NetworkId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	// Delete existing network
	err := client.DeleteNetwork(ctx, projectId, networkId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteNetworkWaitHandler(ctx, client, projectId, networkId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting network", fmt.Sprintf("Network deletion waiting: %v", err))
		return
	}

	tflog.Info(ctx, "Network deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,network_id
func ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing network",
			fmt.Sprintf("Expected import identifier with format: [project_id],[network_id]  Got: %q", req.ID),
		)
		return
	}

	projectId := idParts[0]
	networkId := idParts[1]
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "network_id", networkId)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("network_id"), networkId)...)
	tflog.Info(ctx, "Network state imported")
}

func mapFields(ctx context.Context, networkResp *iaas.Network, model *networkModel.Model) error {
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
		model.IPv4PrefixLength = types.Int64Null()
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

	if networkResp.Gateway != nil {
		model.IPv4Gateway = types.StringPointerValue(networkResp.GetGateway())
	} else {
		model.IPv4Gateway = types.StringNull()
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

	model.IPv6PrefixLength = types.Int64Null()
	model.IPv6Prefix = types.StringNull()
	if networkResp.PrefixesV6 == nil || len(*networkResp.PrefixesV6) == 0 {
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
			} else {
				ones, _ := netmask.Mask.Size()
				model.IPv6PrefixLength = types.Int64Value(int64(ones))
			}
		}
		model.IPv6Prefixes = prefixesV6TF
	}

	if networkResp.Gatewayv6 != nil {
		model.IPv6Gateway = types.StringPointerValue(networkResp.GetGatewayv6())
	} else {
		model.IPv6Gateway = types.StringNull()
	}

	model.NetworkId = types.StringValue(networkId)
	model.Name = types.StringPointerValue(networkResp.Name)
	model.PublicIP = types.StringPointerValue(networkResp.PublicIp)
	model.Labels = labels
	model.Routed = types.BoolPointerValue(networkResp.Routed)
	model.Region = types.StringNull()
	model.RoutingTableID = types.StringNull()

	return nil
}

func toCreatePayload(ctx context.Context, model *networkModel.Model) (*iaas.CreateNetworkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	addressFamily := &iaas.CreateNetworkAddressFamily{}

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

	if !utils.IsUndefined(model.IPv6Prefix) || !utils.IsUndefined(model.IPv6PrefixLength) || (modelIPv6Nameservers != nil) {
		addressFamily.Ipv6 = &iaas.CreateNetworkIPv6Body{
			Prefix:       conversion.StringValueToPointer(model.IPv6Prefix),
			PrefixLength: conversion.Int64ValueToPointer(model.IPv6PrefixLength),
		}
		// IPv6 nameservers should only be set, if it contains any value. If the slice is nil, it should NOT be set.
		// Setting it to a nil slice would result in a payload, where nameservers is set to null in the json payload,
		// but it should actually be unset. Setting it to "null" will result in an error, because it's NOT nullable.
		if modelIPv6Nameservers != nil {
			addressFamily.Ipv6.Nameservers = &modelIPv6Nameservers
		}

		if model.NoIPv6Gateway.ValueBool() {
			addressFamily.Ipv6.Gateway = iaas.NewNullableString(nil)
		} else if !(model.IPv6Gateway.IsUnknown() || model.IPv6Gateway.IsNull()) {
			addressFamily.Ipv6.Gateway = iaas.NewNullableString(conversion.StringValueToPointer(model.IPv6Gateway))
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

	if !model.IPv4Prefix.IsNull() || !model.IPv4PrefixLength.IsNull() || !model.IPv4Nameservers.IsNull() || !model.Nameservers.IsNull() {
		addressFamily.Ipv4 = &iaas.CreateNetworkIPv4Body{
			Nameservers:  &modelIPv4Nameservers,
			Prefix:       conversion.StringValueToPointer(model.IPv4Prefix),
			PrefixLength: conversion.Int64ValueToPointer(model.IPv4PrefixLength),
		}

		if model.NoIPv4Gateway.ValueBool() {
			addressFamily.Ipv4.Gateway = iaas.NewNullableString(nil)
		} else if !(model.IPv4Gateway.IsUnknown() || model.IPv4Gateway.IsNull()) {
			addressFamily.Ipv4.Gateway = iaas.NewNullableString(conversion.StringValueToPointer(model.IPv4Gateway))
		}
	}

	labels, err := conversion.ToStringInterfaceMap(ctx, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	payload := iaas.CreateNetworkPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: &labels,
		Routed: conversion.BoolValueToPointer(model.Routed),
	}

	if addressFamily.Ipv6 != nil || addressFamily.Ipv4 != nil {
		payload.AddressFamily = addressFamily
	}

	return &payload, nil
}

func toUpdatePayload(ctx context.Context, model, stateModel *networkModel.Model) (*iaas.PartialUpdateNetworkPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	addressFamily := &iaas.UpdateNetworkAddressFamily{}

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

	if !utils.IsUndefined(model.NoIPv6Gateway) || !utils.IsUndefined(model.IPv6Gateway) || modelIPv6Nameservers != nil {
		addressFamily.Ipv6 = &iaas.UpdateNetworkIPv6Body{}

		// IPv6 nameservers should only be set, if it contains any value. If the slice is nil, it should NOT be set.
		// Setting it to a nil slice would result in a payload, where nameservers is set to null in the json payload,
		// but it should actually be unset. Setting it to "null" will result in an error, because it's NOT nullable.
		if modelIPv6Nameservers != nil {
			addressFamily.Ipv6.Nameservers = &modelIPv6Nameservers
		}

		if model.NoIPv6Gateway.ValueBool() {
			addressFamily.Ipv6.Gateway = iaas.NewNullableString(nil)
		} else if !utils.IsUndefined(model.IPv6Gateway) {
			addressFamily.Ipv6.Gateway = iaas.NewNullableString(conversion.StringValueToPointer(model.IPv6Gateway))
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

	if !model.IPv4Nameservers.IsNull() || !model.Nameservers.IsNull() {
		addressFamily.Ipv4 = &iaas.UpdateNetworkIPv4Body{
			Nameservers: &modelIPv4Nameservers,
		}

		if model.NoIPv4Gateway.ValueBool() {
			addressFamily.Ipv4.Gateway = iaas.NewNullableString(nil)
		} else if !(model.IPv4Gateway.IsUnknown() || model.IPv4Gateway.IsNull()) {
			addressFamily.Ipv4.Gateway = iaas.NewNullableString(conversion.StringValueToPointer(model.IPv4Gateway))
		}
	}
	currentLabels := stateModel.Labels
	labels, err := conversion.ToJSONMapPartialUpdatePayload(ctx, currentLabels, model.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	payload := iaas.PartialUpdateNetworkPayload{
		Name:   conversion.StringValueToPointer(model.Name),
		Labels: &labels,
	}

	if addressFamily.Ipv6 != nil || addressFamily.Ipv4 != nil {
		payload.AddressFamily = addressFamily
	}

	return &payload, nil
}
