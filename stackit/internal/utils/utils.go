package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

// ConvertPointerSliceToStringSlice safely converts a slice of string pointers to a slice of strings.
func ConvertPointerSliceToStringSlice(pointerSlice []*string) []string {
	if pointerSlice == nil {
		return []string{}
	}
	stringSlice := make([]string, 0, len(pointerSlice))
	for _, strPtr := range pointerSlice {
		if strPtr != nil { // Safely skip any nil pointers in the list
			stringSlice = append(stringSlice, *strPtr)
		}
	}
	return stringSlice
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

// FormatPossibleValues formats a slice into a comma-separated-list for usage in the provider docs
func FormatPossibleValues(values ...string) string {
	var formattedValues []string
	for _, value := range values {
		formattedValues = append(formattedValues, fmt.Sprintf("`%v`", value))
	}
	return fmt.Sprintf("Possible values are: %s.", strings.Join(formattedValues, ", "))
}

func BuildInternalTerraformId(idParts ...string) types.String {
	return types.StringValue(strings.Join(idParts, core.Separator))
}

// If a List was completely removed from the terraform config this is not recognized by terraform.
// This helper function checks if that is the case and adjusts the plan accordingly.
func CheckListRemoval(ctx context.Context, configModelList, planModelList types.List, destination path.Path, listType attr.Type, createEmptyList bool, resp *resource.ModifyPlanResponse) {
	if configModelList.IsNull() && !planModelList.IsNull() {
		if createEmptyList {
			emptyList, _ := types.ListValueFrom(ctx, listType, []string{})
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, destination, emptyList)...)
		} else {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, destination, types.ListNull(listType))...)
		}
	}
}

// SetAndLogStateFields writes the given map of key-value pairs to the state
func SetAndLogStateFields(ctx context.Context, diags *diag.Diagnostics, state *tfsdk.State, values map[string]any) {
	for key, val := range values {
		ctx = tflog.SetField(ctx, key, val)
		diags.Append(state.SetAttribute(ctx, path.Root(key), val)...)
	}
}

// SetModelFieldsToNull sets all Unknown or Null fields in a model struct to their appropriate Null values.
// This is useful when saving minimal state after API calls to ensure idempotency.
// The model parameter must be a pointer to a struct containing Terraform framework types.
func SetModelFieldsToNull(ctx context.Context, model any) error {
	if model == nil {
		return fmt.Errorf("model cannot be nil")
	}

	v := reflect.ValueOf(model)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("model must be a pointer, got %v", v.Kind())
	}

	v = v.Elem()
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return fmt.Errorf("model must point to a struct, got %v", v.Kind())
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanInterface() || !field.CanSet() {
			continue
		}

		fieldValue := field.Interface()

		// Check if the field implements IsUnknown and IsNull
		isUnknownMethod := field.MethodByName("IsUnknown")
		isNullMethod := field.MethodByName("IsNull")

		if !isUnknownMethod.IsValid() || !isNullMethod.IsValid() {
			continue
		}

		// Call IsUnknown() and IsNull()
		isUnknownResult := isUnknownMethod.Call(nil)
		isNullResult := isNullMethod.Call(nil)

		if len(isUnknownResult) == 0 || len(isNullResult) == 0 {
			continue
		}

		isUnknown := isUnknownResult[0].Bool()
		isNull := isNullResult[0].Bool()

		if !isUnknown && !isNull {
			continue
		}

		// Determine the type and set to appropriate Null value
		switch fieldValue.(type) {
		case basetypes.StringValue:
			field.Set(reflect.ValueOf(types.StringNull()))

		case basetypes.BoolValue:
			field.Set(reflect.ValueOf(types.BoolNull()))

		case basetypes.Int64Value:
			field.Set(reflect.ValueOf(types.Int64Null()))

		case basetypes.Float64Value:
			field.Set(reflect.ValueOf(types.Float64Null()))

		case basetypes.NumberValue:
			field.Set(reflect.ValueOf(types.NumberNull()))

		case basetypes.ListValue:
			listVal := fieldValue.(basetypes.ListValue)
			elemType := listVal.ElementType(ctx)
			field.Set(reflect.ValueOf(types.ListNull(elemType)))

		case basetypes.SetValue:
			setVal := fieldValue.(basetypes.SetValue)
			elemType := setVal.ElementType(ctx)
			field.Set(reflect.ValueOf(types.SetNull(elemType)))

		case basetypes.MapValue:
			mapVal := fieldValue.(basetypes.MapValue)
			elemType := mapVal.ElementType(ctx)
			field.Set(reflect.ValueOf(types.MapNull(elemType)))

		case basetypes.ObjectValue:
			objVal := fieldValue.(basetypes.ObjectValue)
			attrTypes := objVal.AttributeTypes(ctx)
			field.Set(reflect.ValueOf(types.ObjectNull(attrTypes)))

		default:
			tflog.Debug(ctx, fmt.Sprintf("SetModelFieldsToNull: skipping field %s of unsupported type %T", fieldType.Name, fieldValue))
		}
	}

	return nil
}

// ShouldWait checks the STACKIT_TF_WAIT_FOR_READY environment variable to determine
// if the provider should wait for resources to be ready after creation/update.
// Returns true if the variable is unset or set to "true" (case-insensitive).
// Returns false if the variable is set to any other value.
// This is typically used to skip waiting in async mode for Crossplane/Upjet.
func ShouldWait() bool {
	v := os.Getenv("STACKIT_TF_WAIT_FOR_READY")
	return v == "" || strings.EqualFold(v, "true")
}
