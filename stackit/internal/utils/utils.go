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
// This function recursively processes nested objects, lists, sets, and maps.
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

		// If the field is Unknown or Null at the top level, convert it to Null
		if isUnknown || isNull {
			if err := setFieldToNull(ctx, field, fieldValue, fieldType); err != nil {
				return err
			}
			continue
		}

		// If the field is Known and not Null, recursively process it
		if err := processKnownField(ctx, field, fieldValue, fieldType); err != nil {
			return err
		}
	}

	return nil
}

// setFieldToNull sets a field to its appropriate Null value based on type
func setFieldToNull(ctx context.Context, field reflect.Value, fieldValue any, fieldType reflect.StructField) error {
	switch v := fieldValue.(type) {
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
		elemType := v.ElementType(ctx)
		field.Set(reflect.ValueOf(types.ListNull(elemType)))

	case basetypes.SetValue:
		elemType := v.ElementType(ctx)
		field.Set(reflect.ValueOf(types.SetNull(elemType)))

	case basetypes.MapValue:
		elemType := v.ElementType(ctx)
		field.Set(reflect.ValueOf(types.MapNull(elemType)))

	case basetypes.ObjectValue:
		attrTypes := v.AttributeTypes(ctx)
		field.Set(reflect.ValueOf(types.ObjectNull(attrTypes)))

	default:
		tflog.Debug(ctx, fmt.Sprintf("SetModelFieldsToNull: skipping field %s of unsupported type %T", fieldType.Name, fieldValue))
	}
	return nil
}

// processKnownField recursively processes known (non-null, non-unknown) fields
// to handle nested structures like objects within lists, maps, etc.
func processKnownField(ctx context.Context, field reflect.Value, fieldValue any, fieldType reflect.StructField) error {
	switch v := fieldValue.(type) {
	case basetypes.ObjectValue:
		// Recursively process object fields
		return processObjectValue(ctx, field, v, fieldType)

	case basetypes.ListValue:
		// Recursively process list elements
		return processListValue(ctx, field, v, fieldType)

	case basetypes.SetValue:
		// Recursively process set elements
		return processSetValue(ctx, field, v, fieldType)

	case basetypes.MapValue:
		// Recursively process map values
		return processMapValue(ctx, field, v, fieldType)

	default:
		// Primitive types (String, Bool, Int64, etc.) don't need recursion
		return nil
	}
}

// processObjectValue recursively processes fields within an ObjectValue
func processObjectValue(ctx context.Context, field reflect.Value, objValue basetypes.ObjectValue, fieldType reflect.StructField) error {
	attrs := objValue.Attributes()
	attrTypes := objValue.AttributeTypes(ctx)
	modified := false
	newAttrs := make(map[string]attr.Value, len(attrs))

	for key, attrVal := range attrs {
		// Check if the attribute has IsUnknown and IsNull methods
		attrValReflect := reflect.ValueOf(attrVal)
		isUnknownMethod := attrValReflect.MethodByName("IsUnknown")
		isNullMethod := attrValReflect.MethodByName("IsNull")

		if !isUnknownMethod.IsValid() || !isNullMethod.IsValid() {
			newAttrs[key] = attrVal
			continue
		}

		isUnknownResult := isUnknownMethod.Call(nil)
		isNullResult := isNullMethod.Call(nil)

		if len(isUnknownResult) == 0 || len(isNullResult) == 0 {
			newAttrs[key] = attrVal
			continue
		}

		isUnknown := isUnknownResult[0].Bool()
		isNull := isNullResult[0].Bool()

		// Convert Unknown or Null attributes to Null
		if isUnknown || isNull {
			nullVal := createNullValue(ctx, attrVal, attrTypes[key])
			if nullVal != nil {
				newAttrs[key] = nullVal
				modified = true
			} else {
				newAttrs[key] = attrVal
			}
		} else {
			// Recursively process known attributes
			processedVal, wasModified, err := processAttributeValueWithFlag(ctx, attrVal, attrTypes[key])
			if err != nil {
				return err
			}
			newAttrs[key] = processedVal
			if wasModified {
				modified = true
			}
		}
	}

	// Only update the field if something changed
	if modified {
		newObj, diags := types.ObjectValue(attrTypes, newAttrs)
		if diags.HasError() {
			return fmt.Errorf("creating new object value for field %s: %v", fieldType.Name, diags.Errors())
		}
		field.Set(reflect.ValueOf(newObj))
	}

	return nil
}

// processListValue recursively processes elements within a ListValue
func processListValue(ctx context.Context, field reflect.Value, listValue basetypes.ListValue, fieldType reflect.StructField) error {
	elements := listValue.Elements()
	if len(elements) == 0 {
		return nil
	}

	elemType := listValue.ElementType(ctx)
	modified := false
	newElements := make([]attr.Value, len(elements))

	for i, elem := range elements {
		// Check if element is Unknown or Null
		elemReflect := reflect.ValueOf(elem)
		isUnknownMethod := elemReflect.MethodByName("IsUnknown")
		isNullMethod := elemReflect.MethodByName("IsNull")

		if !isUnknownMethod.IsValid() || !isNullMethod.IsValid() {
			newElements[i] = elem
			continue
		}

		isUnknownResult := isUnknownMethod.Call(nil)
		isNullResult := isNullMethod.Call(nil)

		if len(isUnknownResult) == 0 || len(isNullResult) == 0 {
			newElements[i] = elem
			continue
		}

		isUnknown := isUnknownResult[0].Bool()
		isNull := isNullResult[0].Bool()

		if isUnknown || isNull {
			nullVal := createNullValue(ctx, elem, elemType)
			if nullVal != nil {
				newElements[i] = nullVal
				modified = true
			} else {
				newElements[i] = elem
			}
		} else {
			// Recursively process known elements (objects, lists, etc.)
			processedElem, wasModified, err := processAttributeValueWithFlag(ctx, elem, elemType)
			if err != nil {
				return err
			}
			newElements[i] = processedElem
			if wasModified {
				modified = true
			}
		}
	}

	// Only update if something changed
	if modified {
		newList, diags := types.ListValue(elemType, newElements)
		if diags.HasError() {
			return fmt.Errorf("creating new list value for field %s: %v", fieldType.Name, diags.Errors())
		}
		field.Set(reflect.ValueOf(newList))
	}

	return nil
}

// processSetValue recursively processes elements within a SetValue
func processSetValue(ctx context.Context, field reflect.Value, setValue basetypes.SetValue, fieldType reflect.StructField) error {
	elements := setValue.Elements()
	if len(elements) == 0 {
		return nil
	}

	elemType := setValue.ElementType(ctx)
	modified := false
	newElements := make([]attr.Value, len(elements))

	for i, elem := range elements {
		elemReflect := reflect.ValueOf(elem)
		isUnknownMethod := elemReflect.MethodByName("IsUnknown")
		isNullMethod := elemReflect.MethodByName("IsNull")

		if !isUnknownMethod.IsValid() || !isNullMethod.IsValid() {
			newElements[i] = elem
			continue
		}

		isUnknownResult := isUnknownMethod.Call(nil)
		isNullResult := isNullMethod.Call(nil)

		if len(isUnknownResult) == 0 || len(isNullResult) == 0 {
			newElements[i] = elem
			continue
		}

		isUnknown := isUnknownResult[0].Bool()
		isNull := isNullResult[0].Bool()

		if isUnknown || isNull {
			nullVal := createNullValue(ctx, elem, elemType)
			if nullVal != nil {
				newElements[i] = nullVal
				modified = true
			} else {
				newElements[i] = elem
			}
		} else {
			processedElem, wasModified, err := processAttributeValueWithFlag(ctx, elem, elemType)
			if err != nil {
				return err
			}
			newElements[i] = processedElem
			if wasModified {
				modified = true
			}
		}
	}

	if modified {
		newSet, diags := types.SetValue(elemType, newElements)
		if diags.HasError() {
			return fmt.Errorf("creating new set value for field %s: %v", fieldType.Name, diags.Errors())
		}
		field.Set(reflect.ValueOf(newSet))
	}

	return nil
}

// processMapValue recursively processes values within a MapValue
func processMapValue(ctx context.Context, field reflect.Value, mapValue basetypes.MapValue, fieldType reflect.StructField) error {
	elements := mapValue.Elements()
	if len(elements) == 0 {
		return nil
	}

	elemType := mapValue.ElementType(ctx)
	modified := false
	newElements := make(map[string]attr.Value, len(elements))

	for key, val := range elements {
		valReflect := reflect.ValueOf(val)
		isUnknownMethod := valReflect.MethodByName("IsUnknown")
		isNullMethod := valReflect.MethodByName("IsNull")

		if !isUnknownMethod.IsValid() || !isNullMethod.IsValid() {
			newElements[key] = val
			continue
		}

		isUnknownResult := isUnknownMethod.Call(nil)
		isNullResult := isNullMethod.Call(nil)

		if len(isUnknownResult) == 0 || len(isNullResult) == 0 {
			newElements[key] = val
			continue
		}

		isUnknown := isUnknownResult[0].Bool()
		isNull := isNullResult[0].Bool()

		if isUnknown || isNull {
			nullVal := createNullValue(ctx, val, elemType)
			if nullVal != nil {
				newElements[key] = nullVal
				modified = true
			} else {
				newElements[key] = val
			}
		} else {
			processedVal, wasModified, err := processAttributeValueWithFlag(ctx, val, elemType)
			if err != nil {
				return err
			}
			newElements[key] = processedVal
			if wasModified {
				modified = true
			}
		}
	}

	if modified {
		newMap, diags := types.MapValue(elemType, newElements)
		if diags.HasError() {
			return fmt.Errorf("creating new map value for field %s: %v", fieldType.Name, diags.Errors())
		}
		field.Set(reflect.ValueOf(newMap))
	}

	return nil
}

// processAttributeValueWithFlag recursively processes a single attribute value
// Returns the processed value, a flag indicating if it was modified, and an error
func processAttributeValueWithFlag(ctx context.Context, attrVal attr.Value, attrType attr.Type) (attr.Value, bool, error) {
	switch v := attrVal.(type) {
	case basetypes.ObjectValue:
		// Recursively process object attributes
		attrs := v.Attributes()
		objType, ok := attrType.(types.ObjectType)
		if !ok {
			return attrVal, false, nil
		}
		attrTypes := objType.AttrTypes
		modified := false
		newAttrs := make(map[string]attr.Value, len(attrs))

		for key, subAttr := range attrs {
			subAttrReflect := reflect.ValueOf(subAttr)
			isUnknownMethod := subAttrReflect.MethodByName("IsUnknown")
			isNullMethod := subAttrReflect.MethodByName("IsNull")

			if !isUnknownMethod.IsValid() || !isNullMethod.IsValid() {
				newAttrs[key] = subAttr
				continue
			}

			isUnknownResult := isUnknownMethod.Call(nil)
			isNullResult := isNullMethod.Call(nil)

			if len(isUnknownResult) == 0 || len(isNullResult) == 0 {
				newAttrs[key] = subAttr
				continue
			}

			isUnknown := isUnknownResult[0].Bool()
			isNull := isNullResult[0].Bool()

			if isUnknown || isNull {
				nullVal := createNullValue(ctx, subAttr, attrTypes[key])
				if nullVal != nil {
					newAttrs[key] = nullVal
					modified = true
				} else {
					newAttrs[key] = subAttr
				}
			} else {
				processedSubAttr, wasModified, err := processAttributeValueWithFlag(ctx, subAttr, attrTypes[key])
				if err != nil {
					return attrVal, false, err
				}
				newAttrs[key] = processedSubAttr
				if wasModified {
					modified = true
				}
			}
		}

		if modified {
			newObj, diags := types.ObjectValue(attrTypes, newAttrs)
			if diags.HasError() {
				return attrVal, false, fmt.Errorf("creating new object value: %v", diags.Errors())
			}
			return newObj, true, nil
		}
		return attrVal, false, nil

	case basetypes.ListValue:
		// Recursively process list elements
		elements := v.Elements()
		if len(elements) == 0 {
			return attrVal, false, nil
		}

		elemType := v.ElementType(ctx)
		modified := false
		newElements := make([]attr.Value, len(elements))

		for i, elem := range elements {
			elemReflect := reflect.ValueOf(elem)
			isUnknownMethod := elemReflect.MethodByName("IsUnknown")
			isNullMethod := elemReflect.MethodByName("IsNull")

			if !isUnknownMethod.IsValid() || !isNullMethod.IsValid() {
				newElements[i] = elem
				continue
			}

			isUnknownResult := isUnknownMethod.Call(nil)
			isNullResult := isNullMethod.Call(nil)

			if len(isUnknownResult) == 0 || len(isNullResult) == 0 {
				newElements[i] = elem
				continue
			}

			isUnknown := isUnknownResult[0].Bool()
			isNull := isNullResult[0].Bool()

			if isUnknown || isNull {
				nullVal := createNullValue(ctx, elem, elemType)
				if nullVal != nil {
					newElements[i] = nullVal
					modified = true
				} else {
					newElements[i] = elem
				}
			} else {
				processedElem, wasModified, err := processAttributeValueWithFlag(ctx, elem, elemType)
				if err != nil {
					return attrVal, false, err
				}
				newElements[i] = processedElem
				if wasModified {
					modified = true
				}
			}
		}

		if modified {
			newList, diags := types.ListValue(elemType, newElements)
			if diags.HasError() {
				return attrVal, false, fmt.Errorf("creating new list value: %v", diags.Errors())
			}
			return newList, true, nil
		}
		return attrVal, false, nil

	default:
		// Primitive types don't need further processing
		return attrVal, false, nil
	}
}

// createNullValue creates a null value of the appropriate type
func createNullValue(ctx context.Context, val attr.Value, attrType attr.Type) attr.Value {
	switch val.(type) {
	case basetypes.StringValue:
		return types.StringNull()
	case basetypes.BoolValue:
		return types.BoolNull()
	case basetypes.Int64Value:
		return types.Int64Null()
	case basetypes.Float64Value:
		return types.Float64Null()
	case basetypes.NumberValue:
		return types.NumberNull()
	case basetypes.ListValue:
		if listType, ok := attrType.(types.ListType); ok {
			return types.ListNull(listType.ElemType)
		}
		return nil
	case basetypes.SetValue:
		if setType, ok := attrType.(types.SetType); ok {
			return types.SetNull(setType.ElemType)
		}
		return nil
	case basetypes.MapValue:
		if mapType, ok := attrType.(types.MapType); ok {
			return types.MapNull(mapType.ElemType)
		}
		return nil
	case basetypes.ObjectValue:
		if objType, ok := attrType.(types.ObjectType); ok {
			return types.ObjectNull(objType.AttrTypes)
		}
		return nil
	default:
		return nil
	}
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
