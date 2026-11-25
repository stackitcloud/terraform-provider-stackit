package utils

import (
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
)

// ValidateServiceIdAndRegionCombination validates the combination of the given service id and region
func ValidateServiceIdAndRegionCombination(availableServicesByRegion map[string][]string, serviceId, region string) *diag.ErrorDiagnostic {
	val, ok := availableServicesByRegion[region]
	if !ok {
		return new(diag.NewErrorDiagnostic("Invalid configuration", fmt.Sprintf("The region '%s' is not supported by the service enablement resource", region)))
	}

	if !slices.Contains(val, serviceId) {
		return new(diag.NewErrorDiagnostic("Invalid configuration", fmt.Sprintf("The region '%s' is not available for the service '%s'", region, serviceId)))
	}

	return nil
}

func CheckServiceEnabled(resp *serviceenablement.ServiceStatus) (bool, error) {
	if resp == nil {
		return false, fmt.Errorf("nil response")
	} else if resp.State == nil {
		return false, fmt.Errorf("nil state")
	}

	return *resp.State == serviceenablement.SERVICESTATUSSTATE_ENABLED, nil
}
