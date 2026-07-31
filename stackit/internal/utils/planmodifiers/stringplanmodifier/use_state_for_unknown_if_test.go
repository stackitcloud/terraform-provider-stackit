package stringplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestUseStateForUnknownIf_PlanModifyString(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		stateValue        types.String
		planValue         types.String
		configValue       types.String
		ifFunc            UseStateForUnknownIfFunc
		expectedPlanValue types.String
		expectedError     bool
	}{
		{
			name:        "State is Null (Creation)",
			stateValue:  types.StringNull(),
			planValue:   types.StringUnknown(),
			configValue: types.StringValue("some-config"),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the state is null
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.StringUnknown(),
		},
		{
			name:        "Plan is already known - (User updated the value)",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringValue("new-plan"),
			configValue: types.StringValue("new-plan"),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached because the plan is known
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.StringValue("new-plan"),
		},
		{
			name:        "Config is Unknown (Interpolation)",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringUnknown(),
			configValue: types.StringUnknown(),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				// This should not be reached
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.StringUnknown(),
		},
		{
			name:        "Condition returns False (Do not use state)",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(), // Simulating computed only
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = false
			},
			expectedPlanValue: types.StringUnknown(),
		},
		{
			name:        "Condition returns True (Use state)",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				resp.UseStateForUnknown = true
			},
			expectedPlanValue: types.StringValue("old-state"),
		},
		{
			name:        "Func returns Error",
			stateValue:  types.StringValue("old-state"),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(),
			ifFunc: func(_ context.Context, _ planmodifier.StringRequest, resp *UseStateForUnknownFuncResponse) {
				resp.Diagnostics.AddError("Test Error", "Something went wrong")
			},
			expectedPlanValue: types.StringUnknown(),
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the modifier
			modifier := UseStateForUnknownIf(tt.ifFunc, "test description")

			// Construct request
			req := planmodifier.StringRequest{
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
				ConfigValue: tt.configValue,
			}

			// Construct response
			// Note: In the framework, resp.PlanValue is initialized to req.PlanValue
			// before the modifier is called. We must simulate this.
			resp := &planmodifier.StringResponse{
				PlanValue: tt.planValue,
			}

			// Run the modifier
			modifier.PlanModifyString(ctx, req, resp)

			// Check Errors
			if tt.expectedError {
				if !resp.Diagnostics.HasError() {
					t.Error("Expected error, got none")
				}
			} else {
				if resp.Diagnostics.HasError() {
					t.Errorf("Unexpected error: %s", resp.Diagnostics)
				}
			}

			// Check Plan Value
			if !resp.PlanValue.Equal(tt.expectedPlanValue) {
				t.Errorf("PlanValue mismatch.\nExpected: %s\nGot:      %s", tt.expectedPlanValue, resp.PlanValue)
			}
		})
	}
}

func TestStringUnchangedExpressionFunction(t *testing.T) {
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"attribute_1": tftypes.String,
			"attribute_2": tftypes.String,
			"attribute_3": tftypes.String,
			"nested_attribute": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"nested_attribute_1": tftypes.String,
					"nested_attribute_2": tftypes.String,
				},
			},
		},
	}

	testSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"attribute_1": schema.StringAttribute{Optional: true},
			"attribute_2": schema.StringAttribute{Optional: true},
			"attribute_3": schema.StringAttribute{Optional: true},
			"nested_attribute": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"nested_attribute_1": schema.StringAttribute{Optional: true},
					"nested_attribute_2": schema.StringAttribute{Optional: true},
				},
			},
		},
	}

	tests := []struct {
		name            string
		pathExpressions []path.Expression
		request         planmodifier.StringRequest
		expectUseState  bool
		expectError     bool
	}{
		// TODO: Add test cases.
		{
			name: "single field without nesting - unchanged",
			pathExpressions: []path.Expression{ // single field
				path.MatchRoot("attribute_1"),
			},
			request: planmodifier.StringRequest{
				PathExpression: path.MatchRoot("base_attr"), // this is just a dummy name to simulate a field to which our plan modifier is attached to
				Config: tfsdk.Config{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "some-value"), // same as in state - unchanged
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
				State: tfsdk.State{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "some-value"), // same as in config - unchanged
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
				Plan: tfsdk.Plan{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "some-value"), // same as in sate & config - unchanged
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
			},
			expectUseState: true,
			expectError:    false,
		},
		{
			name: "single without nesting - changed",
			pathExpressions: []path.Expression{ // single field
				path.MatchRoot("attribute_1"),
			},
			request: planmodifier.StringRequest{
				PathExpression: path.MatchRoot("base_attr"), // this is just a dummy name to simulate a field to which our plan modifier is attached to
				Config: tfsdk.Config{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "new-attr1-value"), // other than in state - changed
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
				State: tfsdk.State{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "old-attr1-value"), // other than in config - changed
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
				Plan: tfsdk.Plan{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "new-attr1-value"), // same as in config but other than state - changed
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
			},
			expectUseState: false,
			expectError:    false,
		},
		{
			name: "multiple fields without nesting - changed",
			pathExpressions: []path.Expression{ // multiple fields
				path.MatchRoot("attribute_1"),
				path.MatchRoot("attribute_2"),
			},
			request: planmodifier.StringRequest{
				PathExpression: path.MatchRoot("base_attr"), // this is just a dummy name to simulate a field to which our plan modifier is attached to
				Config: tfsdk.Config{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "new-value"), // other than in state - changed
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
				State: tfsdk.State{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "old-value"), // other than in config - changed
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
				Plan: tfsdk.Plan{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "new-value"), // same as in config but other than state - changed
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
			},
			expectUseState: false,
			expectError:    false,
		},
		{
			name: "multiple fields with nesting - changed",
			pathExpressions: []path.Expression{ // single field
				path.MatchRoot("attribute_1"),
			},
			request: planmodifier.StringRequest{
				PathExpression: path.MatchRoot("base_attr"), // this is just a dummy name to simulate a field to which our plan modifier is attached to
				Config: tfsdk.Config{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "new-value"), // other than in state - changed
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
				State: tfsdk.State{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "old-value"), // other than in config - changed
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
				Plan: tfsdk.Plan{
					Schema: testSchema,
					Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
						"attribute_1": tftypes.NewValue(tftypes.String, "new-value"), // same as in config but other than state - changed
						"attribute_2": tftypes.NewValue(tftypes.String, nil),
						"attribute_3": tftypes.NewValue(tftypes.String, nil),
						"nested_attribute": tftypes.NewValue(tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"nested_attribute_1": tftypes.String,
								"nested_attribute_2": tftypes.String,
							},
						}, nil),
					}),
				},
			},
			expectUseState: false,
			expectError:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response := &UseStateForUnknownFuncResponse{}

			modifierFunc := StringUnchangedExpressionFunction(tt.pathExpressions...)
			modifierFunc(ctx, tt.request, response)

			// Assertions
			if response.Diagnostics.HasError() != tt.expectError {
				t.Fatalf("expected error: %v, got diagnostics: %v", tt.expectError, response.Diagnostics)
			}

			if response.UseStateForUnknown != tt.expectUseState {
				t.Errorf("expected UseStateForUnknown to be %v, got %v", tt.expectUseState, response.UseStateForUnknown)
			}
		})
	}
}
