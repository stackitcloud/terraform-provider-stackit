package utils

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"

	//nolint:staticcheck // TODO: will be done within STACKITTPR-713

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func MapLabels(ctx context.Context, responseLabels map[string]any, currentLabels types.Map) (basetypes.MapValue, error) { //nolint:gocritic // Linter wants to have a non-pointer type for the map, but this would mean a nil check has to be done before every usage of this func.
	labelsTF, diags := types.MapValueFrom(ctx, types.StringType, map[string]any{})
	if diags.HasError() {
		return labelsTF, fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
	}

	if len(responseLabels) != 0 {
		var diags diag.Diagnostics
		labelsTF, diags = types.MapValueFrom(ctx, types.StringType, responseLabels)
		if diags.HasError() {
			return labelsTF, fmt.Errorf("convert labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else if currentLabels.IsNull() {
		labelsTF = types.MapNull(types.StringType)
	}

	return labelsTF, nil
}

// ReadXRequestId returns the X-Request-Id Header from a context, where config.ContextHTTPResponse is set with **http.Response
func ReadXRequestId(ctx context.Context) (string, error) {
	if resp, ok := ctx.Value(config.ContextHTTPResponse).(**http.Response); ok {
		if requestIdHeader, ok := (*resp).Header[wait.XRequestIDHeader]; ok && len(requestIdHeader) > 0 {
			return requestIdHeader[0], nil
		}
		return "", fmt.Errorf("no XRequestID header found in response")
	}
	return "", fmt.Errorf("no response with type `**http.Response` found in context")
}
