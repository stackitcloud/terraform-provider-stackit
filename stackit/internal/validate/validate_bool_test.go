package validate

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func enablesTestHelper[T fieldInterface](target path.Expression, callback func(*diag.Diagnostics, path.Path, path.Path), _ T) *BoolValidator {
	return enables[T](target, callback)
}

func TestEnables(t *testing.T) {
	tests := []struct {
		description     string
		value           *bool
		targetValue     attr.Value
		targetAttribute schema.Attribute
		isValid         bool
	}{
		{
			description:     "enabled, target string set",
			value:           new(true),
			targetValue:     types.StringValue("Hello World"),
			targetAttribute: schema.StringAttribute{},
			isValid:         true,
		},
		{
			description:     "enabled, target bool set",
			value:           new(true),
			targetValue:     types.BoolValue(true),
			targetAttribute: schema.BoolAttribute{},
			isValid:         true,
		},
		{
			description: "enabled, target list set",
			value:       new(true),
			targetValue: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("foo"),
				types.StringValue("bar"),
			}),
			targetAttribute: schema.ListAttribute{
				ElementType: types.StringType,
			},
			isValid: true,
		},
		{
			description:     "enabled, target not set",
			value:           new(true),
			targetValue:     types.StringNull(),
			targetAttribute: schema.StringAttribute{},
			isValid:         true,
		},
		{
			description:     "enabled, target unknown",
			value:           new(true),
			targetValue:     types.StringUnknown(),
			targetAttribute: schema.StringAttribute{},
			isValid:         true,
		},
		{
			description:     "disabled, target string set",
			value:           new(false),
			targetValue:     types.StringValue("Hello World"),
			targetAttribute: schema.StringAttribute{},
			isValid:         false,
		},
		{
			description:     "disabled, target bool set",
			value:           new(false),
			targetValue:     types.BoolValue(true),
			targetAttribute: schema.BoolAttribute{},
			isValid:         false,
		},
		{
			description: "disabled, target list set",
			value:       new(false),
			targetValue: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("foo"),
				types.StringValue("bar"),
			}),
			targetAttribute: schema.ListAttribute{
				ElementType: types.StringType,
			},
			isValid: false,
		},
		{
			description:     "disabled, target not set",
			value:           new(false),
			targetValue:     types.StringNull(),
			targetAttribute: schema.StringAttribute{},
			isValid:         true,
		},
		{
			description:     "disabled, target unknown",
			value:           new(false),
			targetValue:     types.StringUnknown(),
			targetAttribute: schema.StringAttribute{},
			isValid:         true,
		},
		{
			description:     "not set, target string set",
			value:           nil,
			targetValue:     types.StringValue("Hello World"),
			targetAttribute: schema.StringAttribute{},
			isValid:         false,
		},
		{
			description:     "not set, target not set",
			value:           nil,
			targetValue:     types.StringNull(),
			targetAttribute: schema.StringAttribute{},
			isValid:         true,
		},
		{
			description:     "not set, target null",
			value:           nil,
			targetValue:     types.StringNull(),
			targetAttribute: schema.StringAttribute{},
			isValid:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			ctx := context.Background()

			targetVal, err := tt.targetValue.ToTerraformValue(ctx)
			if err != nil {
				t.Fatalf("Failed to convert target value to tftypes.Value: %s", err)
			}

			objType := tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"target": tftypes.DynamicPseudoType,
				},
			}
			rawConfig := tftypes.NewValue(objType, map[string]tftypes.Value{
				"target": targetVal,
			})

			req := validator.BoolRequest{
				ConfigValue: types.BoolPointerValue(tt.value),
				Config: tfsdk.Config{
					Raw: rawConfig,
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"target": tt.targetAttribute,
						},
					},
				},
			}

			resp := &validator.BoolResponse{}
			customErrorSummary := "My custom error"

			enablesTestHelper(path.MatchRoot("target"), func(diagnostics *diag.Diagnostics, ownPath path.Path, targetPath path.Path) {
				diagnostics.AddAttributeError(
					targetPath,
					customErrorSummary,
					fmt.Sprintf("This can not be set. Target: %q; Self: %q", targetPath.String(), ownPath.String()),
				)
			}, tt.targetValue).ValidateBool(ctx, req, resp)

			if tt.isValid {
				if resp.Diagnostics.HasError() {
					t.Fatalf("did not expect validation error, got: %v", resp.Diagnostics)
				}
			} else {
				hasExpectedError := false

				for _, diag := range resp.Diagnostics {
					if diag.Summary() == customErrorSummary {
						hasExpectedError = true
					} else {
						t.Fatalf("expected validation error, got %q", diag.Summary())
					}
				}

				if !hasExpectedError {
					t.Fatalf("expected %q error, got: %v", customErrorSummary, resp.Diagnostics)
				}
			}
		})
	}
}
