package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func MapLabels(ctx context.Context, responseLabels *map[string]string, currentLabels types.Map) (basetypes.MapValue, error) { // nolint:gocritic // responseLabels needs to be a pointer
	// Labels can have a value {"foo": "bar"}, can be empty {} or can be not provided by the config.
	// The last two states are identical for the API but have a different tfstate value.
	// The goal of this function is to only apply a change to the values if they actually got changed.
	labels := types.MapValueMust(types.StringType, map[string]attr.Value{})

	if responseLabels != nil && len(*responseLabels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *responseLabels)
		if diags.HasError() {
			return labels, fmt.Errorf("convert labels to string map: %w", core.DiagsToError(diags))
		}
	} else if currentLabels.IsNull() {
		labels = types.MapNull(types.StringType)
	}

	return labels, nil
}

func LabelsToPayload(ctx context.Context, modelLabels types.Map) (map[string]string, error) {
	labels := map[string]string{}

	if !modelLabels.IsNull() {
		diags := modelLabels.ElementsAs(ctx, &labels, false)
		if diags.HasError() {
			return nil, fmt.Errorf("converting from MapValue: %w", core.DiagsToError(diags))
		}
	}

	return labels, nil
}
