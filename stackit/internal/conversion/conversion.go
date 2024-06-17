package conversion

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
