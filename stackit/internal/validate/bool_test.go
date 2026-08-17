package validate

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

func TestOnlyIfBoolValidator(t *testing.T) {
	tests := []struct {
		description   string
		target        types.Bool
		expectedValue *bool
		isValid       bool
	}{
		{
			description:   "target true, expect true",
			target:        types.BoolValue(true),
			expectedValue: utils.Ptr(true),
			isValid:       true,
		},
		{
			description:   "target false, expect true",
			target:        types.BoolValue(false),
			expectedValue: utils.Ptr(true),
			isValid:       false,
		},
		{
			description:   "target false, expect false",
			target:        types.BoolValue(false),
			expectedValue: utils.Ptr(false),
			isValid:       true,
		},
		{
			description:   "target true, expect false",
			target:        types.BoolValue(true),
			expectedValue: utils.Ptr(false),
			isValid:       false,
		},
		{
			description:   "target unknown, expect true",
			target:        types.BoolUnknown(),
			expectedValue: utils.Ptr(true),
			isValid:       true,
		},
		{
			description:   "target unknown, expect false",
			target:        types.BoolUnknown(),
			expectedValue: utils.Ptr(false),
			isValid:       true,
		},
		{
			description:   "target null, expect true",
			target:        types.BoolNull(),
			expectedValue: utils.Ptr(true),
			isValid:       true,
		},
		{
			description:   "target null, expect false",
			target:        types.BoolNull(),
			expectedValue: utils.Ptr(false),
			isValid:       true,
		},
		{
			description:   "target true, expect nil",
			target:        types.BoolValue(true),
			expectedValue: nil,
			isValid:       true,
		},
		{
			description:   "target false, expect nil",
			target:        types.BoolValue(false),
			expectedValue: nil,
			isValid:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			ctx := context.Background()

			boolVal, err := tt.target.ToTerraformValue(ctx)
			if err != nil {
				t.Fatalf("Failed to convert bool to tftypes.Value: %s", err)
			}

			objType := tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"target_bool": tftypes.Bool,
				},
			}
			rawConfig := tftypes.NewValue(objType, map[string]tftypes.Value{
				"target_bool": boolVal,
			})

			req := validator.StringRequest{
				Path:           path.Root("my_string"),
				PathExpression: path.MatchRoot("my_string"),
				ConfigValue:    types.StringValue("example_string"),
				Config: tfsdk.Config{
					Raw: rawConfig,
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"target_bool": schema.BoolAttribute{},
						},
					},
				},
			}

			resp := &validator.StringResponse{}

			OnlyAllowedIfBoolEquals(path.MatchRoot("target_bool"), tt.expectedValue).ValidateString(ctx, req, resp)

			if tt.isValid {
				if resp.Diagnostics.HasError() {
					t.Fatalf("did not expect validation error, got: %v", resp.Diagnostics)
				}
			} else {
				hasExpectedError := false

				for _, diag := range resp.Diagnostics {
					if diag.Summary() == "Attribute can not be set" {
						hasExpectedError = true
					} else {
						t.Fatalf("expected validation error, got %q", diag.Summary())
					}
				}

				if !hasExpectedError {
					t.Fatalf("expected 'Attribute can not be set' error, got: %v", resp.Diagnostics)
				}
			}
		})
	}
}
