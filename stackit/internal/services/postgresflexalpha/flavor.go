package postgresflex

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var _ basetypes.ObjectTypable = FlavorType{}

type FlavorType struct {
	basetypes.ObjectType
}

func (t FlavorType) Equal(o attr.Type) bool {
	other, ok := o.(FlavorType)

	if !ok {
		return false
	}

	return t.ObjectType.Equal(other.ObjectType)
}

func (t FlavorType) String() string {
	return "FlavorType"
}

func (t FlavorType) ValueFromObject(_ context.Context, in basetypes.ObjectValue) (basetypes.ObjectValuable, diag.Diagnostics) {
	var diags diag.Diagnostics

	attributes := in.Attributes()

	cpuAttribute, ok := attributes["cpu"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`cpu is missing from object`)

		return nil, diags
	}

	cpuVal, ok := cpuAttribute.(basetypes.Int64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`cpu expected to be basetypes.Int64Value, was: %T`, cpuAttribute))
	}

	descriptionAttribute, ok := attributes["description"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`description is missing from object`)

		return nil, diags
	}

	descriptionVal, ok := descriptionAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`description expected to be basetypes.StringValue, was: %T`, descriptionAttribute))
	}

	idAttribute, ok := attributes["id"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`id is missing from object`)

		return nil, diags
	}

	idVal, ok := idAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`id expected to be basetypes.StringValue, was: %T`, idAttribute))
	}

	memoryAttribute, ok := attributes["memory"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`memory is missing from object`)

		return nil, diags
	}

	ramVal, ok := memoryAttribute.(basetypes.Int64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`memory expected to be basetypes.Int64Value, was: %T`, memoryAttribute))
	}

	if diags.HasError() {
		return nil, diags
	}

	return FlavorValue{
		Cpu:         cpuVal,
		Description: descriptionVal,
		Id:          idVal,
		Ram:         ramVal,
		state:       attr.ValueStateKnown,
	}, diags
}

func NewFlavorValueNull() FlavorValue {
	return FlavorValue{
		state: attr.ValueStateNull,
	}
}

func NewFlavorValueUnknown() FlavorValue {
	return FlavorValue{
		state: attr.ValueStateUnknown,
	}
}

func NewFlavorValue(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) (FlavorValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Reference: https://github.com/hashicorp/terraform-plugin-framework/issues/521
	ctx := context.Background()

	for name, attributeType := range attributeTypes {
		attribute, ok := attributes[name]

		if !ok {
			diags.AddError(
				"Missing FlavorValue Attribute Value",
				"While creating a FlavorValue value, a missing attribute value was detected. "+
					"A FlavorValue must contain values for all attributes, even if null or unknown. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("FlavorValue Attribute Name (%s) Expected Type: %s", name, attributeType.String()),
			)

			continue
		}

		if !attributeType.Equal(attribute.Type(ctx)) {
			diags.AddError(
				"Invalid FlavorValue Attribute Type",
				"While creating a FlavorValue value, an invalid attribute value was detected. "+
					"A FlavorValue must use a matching attribute type for the value. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("FlavorValue Attribute Name (%s) Expected Type: %s\n", name, attributeType.String())+
					fmt.Sprintf("FlavorValue Attribute Name (%s) Given Type: %s", name, attribute.Type(ctx)),
			)
		}
	}

	for name := range attributes {
		_, ok := attributeTypes[name]

		if !ok {
			diags.AddError(
				"Extra FlavorValue Attribute Value",
				"While creating a FlavorValue value, an extra attribute value was detected. "+
					"A FlavorValue must not contain values beyond the expected attribute types. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("Extra FlavorValue Attribute Name: %s", name),
			)
		}
	}

	if diags.HasError() {
		return NewFlavorValueUnknown(), diags
	}

	cpuAttribute, ok := attributes["cpu"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`cpu is missing from object`)

		return NewFlavorValueUnknown(), diags
	}

	cpuVal, ok := cpuAttribute.(basetypes.Int64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`cpu expected to be basetypes.Int64Value, was: %T`, cpuAttribute))
	}

	descriptionAttribute, ok := attributes["description"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`description is missing from object`)

		return NewFlavorValueUnknown(), diags
	}

	descriptionVal, ok := descriptionAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`description expected to be basetypes.StringValue, was: %T`, descriptionAttribute))
	}

	idAttribute, ok := attributes["id"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`id is missing from object`)

		return NewFlavorValueUnknown(), diags
	}

	idVal, ok := idAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`id expected to be basetypes.StringValue, was: %T`, idAttribute))
	}

	memoryAttribute, ok := attributes["memory"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`memory is missing from object`)

		return NewFlavorValueUnknown(), diags
	}

	memoryVal, ok := memoryAttribute.(basetypes.Int64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`memory expected to be basetypes.Int64Value, was: %T`, memoryAttribute))
	}

	if diags.HasError() {
		return NewFlavorValueUnknown(), diags
	}

	return FlavorValue{
		Cpu:         cpuVal,
		Description: descriptionVal,
		Id:          idVal,
		Ram:         memoryVal,
		state:       attr.ValueStateKnown,
	}, diags
}

func NewFlavorValueMust(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) FlavorValue {
	object, diags := NewFlavorValue(attributeTypes, attributes)

	if diags.HasError() {
		// This could potentially be added to the diag package.
		diagsStrings := make([]string, 0, len(diags))

		for _, diagnostic := range diags {
			diagsStrings = append(diagsStrings, fmt.Sprintf(
				"%s | %s | %s",
				diagnostic.Severity(),
				diagnostic.Summary(),
				diagnostic.Detail()))
		}

		panic("NewFlavorValueMust received error(s): " + strings.Join(diagsStrings, "\n"))
	}

	return object
}

func (t FlavorType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	if in.Type() == nil {
		return NewFlavorValueNull(), nil
	}

	if !in.Type().Equal(t.TerraformType(ctx)) {
		return nil, fmt.Errorf("expected %s, got %s", t.TerraformType(ctx), in.Type())
	}

	if !in.IsKnown() {
		return NewFlavorValueUnknown(), nil
	}

	if in.IsNull() {
		return NewFlavorValueNull(), nil
	}

	attributes := map[string]attr.Value{}

	val := map[string]tftypes.Value{}

	err := in.As(&val)

	if err != nil {
		return nil, err
	}

	for k, v := range val {
		a, err := t.AttrTypes[k].ValueFromTerraform(ctx, v)

		if err != nil {
			return nil, err
		}

		attributes[k] = a
	}

	return NewFlavorValueMust(FlavorValue{}.AttributeTypes(ctx), attributes), nil
}

func (t FlavorType) ValueType(_ context.Context) attr.Value {
	return FlavorValue{}
}

var _ basetypes.ObjectValuable = FlavorValue{}

type FlavorValue struct {
	Cpu         basetypes.Int64Value  `tfsdk:"cpu"`
	Description basetypes.StringValue `tfsdk:"description"`
	Id          basetypes.StringValue `tfsdk:"id"`
	Ram         basetypes.Int64Value  `tfsdk:"ram"`
	state       attr.ValueState
}

func (v FlavorValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	attrTypes := make(map[string]tftypes.Type, 4)

	var val tftypes.Value
	var err error

	attrTypes["cpu"] = basetypes.Int64Type{}.TerraformType(ctx)
	attrTypes["description"] = basetypes.StringType{}.TerraformType(ctx)
	attrTypes["id"] = basetypes.StringType{}.TerraformType(ctx)
	attrTypes["memory"] = basetypes.Int64Type{}.TerraformType(ctx)

	objectType := tftypes.Object{AttributeTypes: attrTypes}

	switch v.state {
	case attr.ValueStateKnown:
		vals := make(map[string]tftypes.Value, 4)

		val, err = v.Cpu.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["cpu"] = val

		val, err = v.Description.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["description"] = val

		val, err = v.Id.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["id"] = val

		val, err = v.Ram.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["memory"] = val

		if err := tftypes.ValidateValue(objectType, vals); err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		return tftypes.NewValue(objectType, vals), nil
	case attr.ValueStateNull:
		return tftypes.NewValue(objectType, nil), nil
	case attr.ValueStateUnknown:
		return tftypes.NewValue(objectType, tftypes.UnknownValue), nil
	default:
		panic(fmt.Sprintf("unhandled Object state in ToTerraformValue: %s", v.state))
	}
}

func (v FlavorValue) IsNull() bool {
	return v.state == attr.ValueStateNull
}

func (v FlavorValue) IsUnknown() bool {
	return v.state == attr.ValueStateUnknown
}

func (v FlavorValue) String() string {
	return "FlavorValue"
}

func (v FlavorValue) ToObjectValue(_ context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	attributeTypes := map[string]attr.Type{
		"cpu":         basetypes.Int64Type{},
		"description": basetypes.StringType{},
		"id":          basetypes.StringType{},
		"memory":      basetypes.Int64Type{},
	}

	if v.IsNull() {
		return types.ObjectNull(attributeTypes), diags
	}

	if v.IsUnknown() {
		return types.ObjectUnknown(attributeTypes), diags
	}

	objVal, diags := types.ObjectValue(
		attributeTypes,
		map[string]attr.Value{
			"cpu":         v.Cpu,
			"description": v.Description,
			"id":          v.Id,
			"memory":      v.Ram,
		})

	return objVal, diags
}

func (v FlavorValue) Equal(o attr.Value) bool {
	other, ok := o.(FlavorValue)

	if !ok {
		return false
	}

	if v.state != other.state {
		return false
	}

	if v.state != attr.ValueStateKnown {
		return true
	}

	if !v.Cpu.Equal(other.Cpu) {
		return false
	}

	if !v.Description.Equal(other.Description) {
		return false
	}

	if !v.Id.Equal(other.Id) {
		return false
	}

	if !v.Ram.Equal(other.Ram) {
		return false
	}

	return true
}

func (v FlavorValue) Type(ctx context.Context) attr.Type {
	return FlavorType{
		basetypes.ObjectType{
			AttrTypes: v.AttributeTypes(ctx),
		},
	}
}

func (v FlavorValue) AttributeTypes(_ context.Context) map[string]attr.Type {
	return map[string]attr.Type{
		"cpu":         basetypes.Int64Type{},
		"description": basetypes.StringType{},
		"id":          basetypes.StringType{},
		"memory":      basetypes.Int64Type{},
	}
}
