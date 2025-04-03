package utils

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

const (
	SKEServiceId          = "cloud.stackit.ske"
	ModelServingServiceId = "cloud.stackit.model-serving"
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

// SimplifyBackupSchedule removes leading 0s from backup schedule numbers (e.g. "00 00 * * *" becomes "0 0 * * *")
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

type value interface {
	IsUnknown() bool
	IsNull() bool
}

// IsUndefined checks if a passed value is unknown or null
func IsUndefined(val value) bool {
	return val.IsUnknown() || val.IsNull()
}

// LogError logs errors. In descriptions different messages for http status codes can be passed. When no one matches the defaultDescription will be used
func LogError(ctx context.Context, inputDiags *diag.Diagnostics, err error, summary, defaultDescription string, descriptions map[int]string) {
	if err == nil {
		return
	}
	tflog.Error(ctx, fmt.Sprintf("%s. Err: %v", summary, err))

	var oapiErr *oapierror.GenericOpenAPIError
	ok := errors.As(err, &oapiErr)
	if !ok {
		core.LogAndAddError(ctx, inputDiags, summary, fmt.Sprintf("Calling API: %v", err))
		return
	}

	var description string
	if len(descriptions) != 0 {
		description, ok = descriptions[oapiErr.StatusCode]
	}
	if !ok || description == "" {
		description = defaultDescription
	}
	core.LogAndAddError(ctx, inputDiags, summary, description)
}
