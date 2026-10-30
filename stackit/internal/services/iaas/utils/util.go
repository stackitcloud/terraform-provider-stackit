package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

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

// IsNetworkAreaRegionNotFound reports whether err means that the regional configuration of a network area does not exist.
// The IaaS API answers with 404 if the network area is gone, but with 400 "region not enabled" if the network area
// still exists and only its regional configuration was deleted. Other 400 responses are not treated as "not found",
// so that an unrelated bad request does not silently remove a resource from the state.
func IsNetworkAreaRegionNotFound(err error) bool {
	var oapiErr *oapierror.GenericOpenAPIError
	if !errors.As(err, &oapiErr) {
		return false
	}
	switch oapiErr.StatusCode {
	case http.StatusNotFound:
		return true
	case http.StatusBadRequest:
		return strings.Contains(string(oapiErr.Body), "region not enabled")
	default:
		return false
	}
}
