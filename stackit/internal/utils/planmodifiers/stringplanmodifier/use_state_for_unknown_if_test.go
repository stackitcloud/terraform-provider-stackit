package stringplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func TestUnchangedPaths(t *testing.T) {
	ctx := t.Context()
	itemAttributeTypes := map[string]attr.Type{
		"value": types.StringType,
	}
	testSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			// the attribute for which useStateForUnknown is decided
			"anchor": schema.StringAttribute{Optional: true},
			// attributes used to make the decision
			"first":   schema.StringAttribute{Optional: true},
			"second":  schema.StringAttribute{Optional: true},
			"enabled": schema.BoolAttribute{Optional: true},
			"count":   schema.Int64Attribute{Optional: true},
			"items": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{Optional: true},
					},
				},
			},
		},
	}

	type itemModel struct {
		Value types.String `tfsdk:"value"`
	}
	type testModel struct {
		Anchor  types.String `tfsdk:"anchor"`
		First   types.String `tfsdk:"first"`
		Second  types.String `tfsdk:"second"`
		Enabled types.Bool   `tfsdk:"enabled"`
		Count   types.Int64  `tfsdk:"count"`
		Items   types.List   `tfsdk:"items"`
	}
	type testValues struct {
		anchor  string
		first   string
		second  string
		enabled bool
		count   int64
		items   []string
	}

	modelFromValues := func(t *testing.T, values testValues) testModel {
		t.Helper()

		items := make([]itemModel, 0, len(values.items))
		for _, value := range values.items {
			items = append(items, itemModel{Value: types.StringValue(value)})
		}
		itemList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: itemAttributeTypes}, items)
		if diags.HasError() {
			t.Fatalf("failed to construct item list: %v", diags.Errors())
		}

		return testModel{
			Anchor:  types.StringValue(values.anchor),
			First:   types.StringValue(values.first),
			Second:  types.StringValue(values.second),
			Enabled: types.BoolValue(values.enabled),
			Count:   types.Int64Value(values.count),
			Items:   itemList,
		}
	}

	newPlan := func(t *testing.T, values testValues) tfsdk.Plan {
		t.Helper()

		plan := tfsdk.Plan{Schema: testSchema}
		diags := plan.Set(ctx, modelFromValues(t, values))
		if diags.HasError() {
			t.Fatalf("failed to construct plan: %v", diags.Errors())
		}
		return plan
	}

	newState := func(t *testing.T, values testValues) tfsdk.State {
		t.Helper()

		state := tfsdk.State{Schema: testSchema}
		diags := state.Set(ctx, modelFromValues(t, values))
		if diags.HasError() {
			t.Fatalf("failed to construct state: %v", diags.Errors())
		}
		return state
	}

	relativeFirst := path.MatchRelative().AtParent().AtName("first")
	relativeSecond := path.MatchRelative().AtParent().AtName("second")
	allItemValues := path.MatchRoot("items").AtAnyListIndex().AtName("value")
	items := path.MatchRoot("items")

	tests := []struct {
		name        string
		paths       []path.Expression
		plan        testValues
		state       testValues
		configItems []string
		want        bool
		wantError   bool
	}{
		{
			name:  "current attribute unchanged when no paths are supplied",
			plan:  testValues{anchor: "same", first: "first", second: "second"},
			state: testValues{anchor: "same", first: "first", second: "second"},
			want:  true,
		},
		{
			name:  "current attribute changed when no paths are supplied",
			plan:  testValues{anchor: "new", first: "first", second: "second"},
			state: testValues{anchor: "old", first: "first", second: "second"},
		},
		{
			name:  "relative path unchanged",
			paths: []path.Expression{relativeFirst},
			plan:  testValues{anchor: "new", first: "same", second: "second"},
			state: testValues{anchor: "old", first: "same", second: "second"},
			want:  true,
		},
		{
			name:  "relative path changed",
			paths: []path.Expression{relativeFirst},
			plan:  testValues{anchor: "same", first: "new", second: "second"},
			state: testValues{anchor: "same", first: "old", second: "second"},
		},
		{
			name:  "multiple paths unchanged",
			paths: []path.Expression{relativeFirst, relativeSecond},
			plan:  testValues{anchor: "new", first: "first", second: "second"},
			state: testValues{anchor: "old", first: "first", second: "second"},
			want:  true,
		},
		{
			name:  "one of multiple paths changed",
			paths: []path.Expression{relativeFirst, relativeSecond},
			plan:  testValues{anchor: "same", first: "first", second: "new"},
			state: testValues{anchor: "same", first: "first", second: "old"},
		},
		{
			name:  "non-string scalar unchanged",
			paths: []path.Expression{path.MatchRoot("enabled"), path.MatchRoot("count")},
			plan:  testValues{anchor: "new", enabled: true, count: 2},
			state: testValues{anchor: "old", enabled: true, count: 2},
			want:  true,
		},
		{
			name:  "non-string scalar changed",
			paths: []path.Expression{path.MatchRoot("enabled"), path.MatchRoot("count")},
			plan:  testValues{enabled: true, count: 2},
			state: testValues{enabled: true, count: 1},
		},
		{
			name:  "composite value unchanged",
			paths: []path.Expression{items},
			plan:  testValues{anchor: "new", items: []string{"one", "two"}},
			state: testValues{anchor: "old", items: []string{"one", "two"}},
			want:  true,
		},
		{
			name:  "composite value changed",
			paths: []path.Expression{items},
			plan:  testValues{items: []string{"one", "new"}},
			state: testValues{items: []string{"one", "old"}},
		},
		{
			name:  "all wildcard matches unchanged",
			paths: []path.Expression{allItemValues},
			plan:  testValues{anchor: "new", items: []string{"one", "two"}},
			state: testValues{anchor: "old", items: []string{"one", "two"}},
			want:  true,
		},
		{
			name:  "one wildcard match changed",
			paths: []path.Expression{allItemValues},
			plan:  testValues{anchor: "same", items: []string{"one", "new"}},
			state: testValues{anchor: "same", items: []string{"one", "old"}},
		},
		{
			name:  "no wildcard matches",
			paths: []path.Expression{allItemValues},
			plan:  testValues{anchor: "new", items: []string{}},
			state: testValues{anchor: "old", items: []string{}},
			want:  true,
		},
		{
			name:      "invalid path",
			paths:     []path.Expression{path.MatchRoot("missing")},
			plan:      testValues{anchor: "same"},
			state:     testValues{anchor: "same"},
			wantError: true,
		},
		{
			name:        "matched path missing from plan is changed",
			paths:       []path.Expression{allItemValues},
			plan:        testValues{items: []string{"one"}},
			state:       testValues{items: []string{"one", "two"}},
			configItems: []string{"one", "two"},
		},
		{
			name:        "matched path missing from state is changed",
			paths:       []path.Expression{allItemValues},
			plan:        testValues{items: []string{"one", "two"}},
			state:       testValues{items: []string{"one"}},
			configItems: []string{"one", "two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newPlan(t, tt.plan)
			state := newState(t, tt.state)
			configPlan := plan
			if tt.configItems != nil {
				configValues := tt.plan
				configValues.items = tt.configItems
				configPlan = newPlan(t, configValues)
			}

			request := planmodifier.StringRequest{
				PathExpression: path.MatchRoot("anchor"),
				Config: tfsdk.Config{
					Schema: testSchema,
					Raw:    configPlan.Raw,
				},
				Plan:  plan,
				State: state,
			}
			response := &UseStateForUnknownFuncResponse{}

			UnchangedPaths(tt.paths...)(ctx, request, response)

			if response.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected diagnostics: %v", response.Diagnostics)
			}
			if response.UseStateForUnknown != tt.want {
				t.Errorf("UseStateForUnknown = %t, want %t", response.UseStateForUnknown, tt.want)
			}
		})
	}
}
