package managed_rule_set

import (
	"context"
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"
)

var (
	testProjectId = types.StringValue(uuid.NewString())
	testRegion    = types.StringValue("eu01")
	testName      = types.StringValue("test-managed-rule-set")
	testId        = types.StringValue(testProjectId.ValueString() + "," + testRegion.ValueString() + "," + testName.ValueString())
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		name     string
		model    *Model
		expected *albWaf.CreateManagedRuleSetPayload
		isValid  bool
	}{
		{
			name: "default",
			model: &Model{
				Name:      testName,
				Id:        testId,
				ProjectId: testProjectId,
				Region:    testRegion,
				Type:      types.StringValue(string(albWaf.TYPE_TYPE_OWASP_CRS)),
			},
			expected: &albWaf.CreateManagedRuleSetPayload{
				Name: testName.ValueString(),
				Type: albWaf.TYPE_TYPE_OWASP_CRS,
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toCreatePayload(context.Background(), tt.model)
			if (err != nil) == tt.isValid {
				t.Errorf("toCreatePayload() error = %v, isValid %v", err, tt.isValid)
				return
			}

			if tt.isValid {
				if diff := cmp.Diff(got, tt.expected); diff != "" {
					t.Errorf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		name     string
		state    *Model
		region   string
		input    *albWaf.GetManagedRuleSetResponse
		expected *Model
		isValid  bool
	}{
		{
			name: "default",
			state: &Model{
				ProjectId: testProjectId,
				Region:    testRegion,
				Name:      testName,
				Type:      types.StringValue(string(albWaf.TYPE_TYPE_OWASP_CRS)),
				Id:        testId,
				Groups:    types.MapValueMust(types.ObjectType{AttrTypes: ruleGroupType}, map[string]attr.Value{}),
			},
			region: testRegion.ValueString(),
			input: &albWaf.GetManagedRuleSetResponse{
				Groups: &map[string]albWaf.MRSRuleGroup{},
				Name:   testName.ValueString(),
				Type:   albWaf.TYPE_TYPE_OWASP_CRS,
			},
			expected: &Model{
				ProjectId: testProjectId,
				Region:    testRegion,
				Name:      testName,
				Type:      types.StringValue(string(albWaf.TYPE_TYPE_OWASP_CRS)),
				Id:        testId,
				Groups:    types.MapValueMust(types.ObjectType{AttrTypes: ruleGroupType}, map[string]attr.Value{}),
				Version:   types.StringValue(""),
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := mapFields(ctx, tt.input, tt.state, tt.region); (err == nil) != tt.isValid {
				t.Errorf("unexpected error")
			}
			if tt.isValid {
				if diff := cmp.Diff(tt.state, tt.expected); diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapResourceFields(t *testing.T) {
	tests := []struct {
		name     string
		state    types.Map
		input    *albWaf.GetManagedRuleSetResponse
		expected types.Map
	}{
		{
			name:  "drift shows, a rule that no longer exists is dropped, unmanaged rules stay out",
			state: modes(map[string]string{"911100": "MODE_LOG_ONLY", "999999": "MODE_LOG_ONLY"}),
			input: testRuleSet(map[string]albWaf.Mode{
				"911100": albWaf.MODE_MODE_ENABLED,
				"920450": albWaf.MODE_MODE_LOG_ONLY,
			}),
			expected: modes(map[string]string{"911100": "MODE_ENABLED"}),
		},
		{
			name:     "not configured stays null",
			state:    types.MapNull(types.StringType),
			input:    testRuleSet(map[string]albWaf.Mode{"911100": albWaf.MODE_MODE_LOG_ONLY}),
			expected: types.MapNull(types.StringType),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ResourceModel{
				Model:     Model{ProjectId: testProjectId, Region: testRegion, Name: testName},
				RuleModes: tt.state,
			}
			if err := mapResourceFields(context.Background(), tt.input, model, testRegion.ValueString()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(model.RuleModes, tt.expected); diff != "" {
				t.Errorf("rule_modes do not match: %s", diff)
			}
		})
	}
}

func modes(m map[string]string) types.Map {
	values := map[string]attr.Value{}
	for k, v := range m {
		values[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, values)
}

// testRuleSet returns a rule set whose rules sit in the group named by their
// first three digits, as in OWASP CRS.
func testRuleSet(rules map[string]albWaf.Mode) *albWaf.GetManagedRuleSetResponse {
	groups := map[string]albWaf.MRSRuleGroup{}
	for ruleKey, mode := range rules {
		groupKey := ruleKey[:3]
		group := groups[groupKey]
		if group.Rules == nil {
			group.Rules = &map[string]albWaf.MRSRule{}
		}
		(*group.Rules)[ruleKey] = albWaf.MRSRule{Mode: mode}
		groups[groupKey] = group
	}
	return &albWaf.GetManagedRuleSetResponse{
		Groups:  &groups,
		Name:    testName.ValueString(),
		Type:    albWaf.TYPE_TYPE_OWASP_CRS,
		Version: "4.14.0",
	}
}

func TestToPatchPayload(t *testing.T) {
	ruleSet := testRuleSet(map[string]albWaf.Mode{
		"911100": albWaf.MODE_MODE_ENABLED,
		"920450": albWaf.MODE_MODE_ENABLED,
		"921140": albWaf.MODE_MODE_LOG_ONLY,
	})
	logOnly := albWaf.MODE_MODE_LOG_ONLY
	enabled := albWaf.MODE_MODE_ENABLED
	disabled := albWaf.MODE_MODE_DISABLED
	tests := []struct {
		name     string
		wanted   types.Map
		previous types.Map
		expected map[string]albWaf.PatchMRSRuleGroup
		isValid  bool
	}{
		{
			name:     "set on create",
			wanted:   modes(map[string]string{"911100": "MODE_LOG_ONLY", "920450": "MODE_LOG_ONLY"}),
			previous: types.MapNull(types.StringType),
			expected: map[string]albWaf.PatchMRSRuleGroup{
				"911": {Rules: map[string]albWaf.PatchMRSRule{"911100": {Mode: &logOnly}}},
				"920": {Rules: map[string]albWaf.PatchMRSRule{"920450": {Mode: &logOnly}}},
			},
			isValid: true,
		},
		{
			name:     "removed rule goes back to enabled, changed rule takes its new mode",
			wanted:   modes(map[string]string{"920450": "MODE_DISABLED"}),
			previous: modes(map[string]string{"911100": "MODE_LOG_ONLY", "920450": "MODE_LOG_ONLY"}),
			expected: map[string]albWaf.PatchMRSRuleGroup{
				"911": {Rules: map[string]albWaf.PatchMRSRule{"911100": {Mode: &enabled}}},
				"920": {Rules: map[string]albWaf.PatchMRSRule{"920450": {Mode: &disabled}}},
			},
			isValid: true,
		},
		{
			name:     "attribute removed entirely",
			wanted:   types.MapNull(types.StringType),
			previous: modes(map[string]string{"921140": "MODE_LOG_ONLY"}),
			expected: map[string]albWaf.PatchMRSRuleGroup{
				"921": {Rules: map[string]albWaf.PatchMRSRule{"921140": {Mode: &enabled}}},
			},
			isValid: true,
		},
		{
			name:     "nothing to do",
			wanted:   types.MapNull(types.StringType),
			previous: types.MapNull(types.StringType),
			expected: map[string]albWaf.PatchMRSRuleGroup{},
			isValid:  true,
		},
		{
			name:     "unknown rule",
			wanted:   modes(map[string]string{"123456": "MODE_LOG_ONLY"}),
			previous: types.MapNull(types.StringType),
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toPatchPayload(context.Background(), ruleSet, tt.wanted, tt.previous)
			if (err == nil) != tt.isValid {
				t.Fatalf("toPatchPayload() error = %v, isValid %v", err, tt.isValid)
			}
			if tt.isValid {
				if diff := cmp.Diff(got.Groups, tt.expected); diff != "" {
					t.Errorf("Data does not match: %s", diff)
				}
			}
		})
	}
}
