package conversion

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func ToPtrInt32(source types.Int64) *int32 {
	if source.IsNull() || source.IsUnknown() {
		return nil
	}
	ttlInt64 := source.ValueInt64()
	ttlInt32 := int32(ttlInt64)
	return &ttlInt32
}

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
