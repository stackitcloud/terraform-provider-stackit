package utils

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	SKEServiceId = "cloud.stackit.ske"
)

var (
	LegacyProjectRoles = []string{"project.admin", "project.auditor", "project.member", "project.owner"}
)

// ReconcileStringSlices reconciles two string lists by removing elements from the
// first list that are not in the second list and appending elements from the
// second list that are not in the first list.
// This preserves the order of the elements in the first list that are also in
// the second list, which is useful when using ListAttributes in Terraform.
// The source of truth for the order is the first list and the source of truth for the content is the second list.
func ReconcileStringSlices(list1, list2 []string) []string {
	// Create a copy of list1 to avoid modifying the original list
	list1Copy := append([]string{}, list1...)

	// Create a map to quickly check if an element is in list2
	inList2 := make(map[string]bool)
	for _, elem := range list2 {
		inList2[elem] = true
	}

	// Remove elements from list1Copy that are not in list2
	i := 0
	for _, elem := range list1Copy {
		if inList2[elem] {
			list1Copy[i] = elem
			i++
		}
	}
	list1Copy = list1Copy[:i]

	// Append elements to list1Copy that are in list2 but not in list1Copy
	inList1 := make(map[string]bool)
	for _, elem := range list1Copy {
		inList1[elem] = true
	}
	for _, elem := range list2 {
		if !inList1[elem] {
			list1Copy = append(list1Copy, elem)
		}
	}

	return list1Copy
}

func ListValuetoStringSlice(list basetypes.ListValue) ([]string, error) {
	result := []string{}
	for _, el := range list.Elements() {
		elStr, ok := el.(types.String)
		if !ok {
			return result, fmt.Errorf("expected record to be of type %T, got %T", types.String{}, elStr)
		}
		result = append(result, elStr.ValueString())
	}

	return result, nil
}

// Remove leading 0s from backup schedule numbers (e.g. "00 00 * * *" becomes "0 0 * * *")
// Needed as the API does it internally and would otherwise cause inconsistent result in Terraform
func SimplifyBackupSchedule(schedule string) string {
	regex := regexp.MustCompile(`0+\d+`) // Matches series of one or more zeros followed by a series of one or more digits
	simplifiedSchedule := regex.ReplaceAllStringFunc(schedule, func(match string) string {
		simplified := strings.TrimLeft(match, "0")
		if simplified == "" {
			simplified = "0"
		}
		return simplified
	})
	return simplifiedSchedule
}

func SupportedValuesDocumentation(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return "Supported values are: " + strings.Join(QuoteValues(values), ", ") + "."
}

func QuoteValues(values []string) []string {
	quotedValues := make([]string, len(values))
	for i, value := range values {
		quotedValues[i] = fmt.Sprintf("`%s`", value)
	}
	return quotedValues
}

func IsLegacyProjectRole(role string) bool {
	return utils.Contains(LegacyProjectRoles, role)
}

// ToJSONMApPartialUpdatePayload returns a map[string]interface{} to be used in a PATCH request payload.
// It takes a current map as it is in the terraform state and a desired map as it is in the user configuratiom
// and builds a map which sets to null keys that should be removed, updates the values of existing keys and adds new keys
// This method is needed because in partial updates, e.g. if the key is not provided it is ignored and not removed
func ToJSONMapPartialUpdatePayload(ctx context.Context, current, desired types.Map) (map[string]interface{}, error) {
	currentMap, err := conversion.ToStringInterfaceMap(ctx, current)
	if err != nil {
		return nil, fmt.Errorf("converting to Go map: %w", err)
	}

	desiredMap, err := conversion.ToStringInterfaceMap(ctx, desired)
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
