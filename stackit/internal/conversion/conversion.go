package conversion

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func ToString(ctx context.Context, v attr.Value) (string, error) {
	if t := v.Type(ctx); t != types.StringType {
		return "", fmt.Errorf("type mismatch. expected 'types.StringType' but got '%s'", t.String())
	}
	if v.IsNull() || v.IsUnknown() {
		return "", fmt.Errorf("value is unknown or null")
	}
	tv, err := v.ToTerraformValue(ctx)
	if err != nil {
		return "", err
	}
	var s string
	if err := tv.Copy().As(&s); err != nil {
		return "", err
	}
	return s, nil
}

func ToOptStringMap(tfMap map[string]attr.Value) (*map[string]string, error) { //nolint: gocritic //pointer needed to map optional fields
	labels := make(map[string]string, len(tfMap))
	for l, v := range tfMap {
		valueString, ok := v.(types.String)
		if !ok {
			return nil, fmt.Errorf("error converting map value: expected to string, got %v", v)
		}
		labels[l] = valueString.ValueString()
	}

	labelsPointer := &labels
	if len(labels) == 0 {
		labelsPointer = nil
	}
	return labelsPointer, nil
}

func ToTerraformStringMap(ctx context.Context, m map[string]string) (basetypes.MapValue, error) {
	labels := make(map[string]attr.Value, len(m))
	for l, v := range m {
		stringValue := types.StringValue(v)
		labels[l] = stringValue
	}
	res, diags := types.MapValueFrom(ctx, types.StringType, m)
	if diags.HasError() {
		return types.MapNull(types.StringType), fmt.Errorf("converting to MapValue: %v", diags.Errors())
	}

	return res, nil
}

// ToStringInterfaceMap converts a basetypes.MapValue of Strings to a map[string]interface{}.
func ToStringInterfaceMap(ctx context.Context, m basetypes.MapValue) (map[string]interface{}, error) {
	labels := map[string]string{}
	diags := m.ElementsAs(ctx, &labels, false)
	if diags.HasError() {
		return nil, fmt.Errorf("converting from MapValue: %w", core.DiagsToError(diags))
	}

	interfaceMap := make(map[string]interface{}, len(labels))
	for k, v := range labels {
		interfaceMap[k] = v
	}

	return interfaceMap, nil
}

// StringValueToPointer converts basetypes.StringValue to a pointer to string.
// It returns nil if the value is null or unknown.
func StringValueToPointer(s basetypes.StringValue) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	value := s.ValueString()
	return &value
}

// Int64ValueToPointer converts basetypes.Int64Value to a pointer to int64.
// It returns nil if the value is null or unknown.
func Int64ValueToPointer(s basetypes.Int64Value) *int64 {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	value := s.ValueInt64()
	return &value
}

// Float64ValueToPointer converts basetypes.Float64Value to a pointer to float64.
// It returns nil if the value is null or unknown.
func Float64ValueToPointer(s basetypes.Float64Value) *float64 {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	value := s.ValueFloat64()
	return &value
}

// BoolValueToPointer converts basetypes.BoolValue to a pointer to bool.
// It returns nil if the value is null or unknown.
func BoolValueToPointer(s basetypes.BoolValue) *bool {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	value := s.ValueBool()
	return &value
}

// StringListToPointer converts basetypes.ListValue to a pointer to a list of strings.
// It returns nil if the value is null or unknown.
func StringListToPointer(list basetypes.ListValue) (*[]string, error) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	listStr := []string{}
	for i, el := range list.Elements() {
		elStr, ok := el.(types.String)
		if !ok {
			return nil, fmt.Errorf("element %d is not a string", i)
		}
		listStr = append(listStr, elStr.ValueString())
	}

	return &listStr, nil
}

// ToJSONMApPartialUpdatePayload returns a map[string]interface{} to be used in a PATCH request payload.
// It takes a current map as it is in the terraform state and a desired map as it is in the user configuratiom
// and builds a map which sets to null keys that should be removed, updates the values of existing keys and adds new keys
// This method is needed because in partial updates, e.g. if the key is not provided it is ignored and not removed
func ToJSONMapPartialUpdatePayload(ctx context.Context, current, desired types.Map) (map[string]interface{}, error) {
	currentMap, err := ToStringInterfaceMap(ctx, current)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	desiredMap, err := ToStringInterfaceMap(ctx, desired)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	mapPayload := map[string]interface{}{}
	// Update and remove existing keys
	for k := range currentMap {
		if desiredValue, ok := desiredMap[k]; ok {
			mapPayload[k] = desiredValue
		} else {
			mapPayload[k] = nil
		}
	}

	// Add new keys
	for k, desiredValue := range desiredMap {
		if _, ok := mapPayload[k]; !ok {
			mapPayload[k] = desiredValue
		}
	}
	return mapPayload, nil
}

func ParseProviderData(ctx context.Context, providerData any, diags *diag.Diagnostics) (core.ProviderData, bool) {
	// Prevent panic if the provider has not been configured.
	if providerData == nil {
		return core.ProviderData{}, false
	}

	stackitProviderData, ok := providerData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", providerData))
		return core.ProviderData{}, false
	}
	return stackitProviderData, true
}

// TODO: write tests
func ParseEphemeralProviderData(ctx context.Context, providerData any, diags *diag.Diagnostics) (core.EphemeralProviderData, bool) {
	// Prevent panic if the provider has not been configured.
	if providerData == nil {
		return core.EphemeralProviderData{}, false
	}

	stackitProviderData, ok := providerData.(core.EphemeralProviderData)
	if !ok {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", providerData))
		return core.EphemeralProviderData{}, false
	}
	return stackitProviderData, true
}
