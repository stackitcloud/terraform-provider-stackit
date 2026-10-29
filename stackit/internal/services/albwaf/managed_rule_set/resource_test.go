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

func TestRuleModes(t *testing.T) {
	model := &Model{ProjectId: testProjectId, Region: testRegion, Name: testName}
	if err := mapFields(context.Background(), testRuleSet(map[string]albWaf.Mode{
		"911100": albWaf.MODE_MODE_LOG_ONLY,
		"920450": albWaf.MODE_MODE_DISABLED,
		"920451": albWaf.MODE_MODE_ENABLED,
	}), model, testRegion.ValueString()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		groups   types.Map
		expected map[string]map[string]string
	}{
		{
			name:   "groups",
			groups: model.Groups,
			expected: map[string]map[string]string{
				"911": {"911100": "MODE_LOG_ONLY"},
				"920": {"920450": "MODE_DISABLED", "920451": "MODE_ENABLED"},
			},
		},
		{
			name:     "null groups",
			groups:   types.MapNull(types.ObjectType{AttrTypes: ruleGroupType}),
			expected: nil,
		},
		{
			name:     "unknown groups",
			groups:   types.MapUnknown(types.ObjectType{AttrTypes: ruleGroupType}),
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ruleModes(context.Background(), tt.groups)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(got, tt.expected); diff != "" {
				t.Errorf("Data does not match: %s", diff)
			}
		})
	}
}

func TestFilterRuleSet(t *testing.T) {
	ruleSet := testRuleSet(map[string]albWaf.Mode{
		"911100": albWaf.MODE_MODE_ENABLED,
		"920450": albWaf.MODE_MODE_LOG_ONLY,
		"920451": albWaf.MODE_MODE_ENABLED,
		"942100": albWaf.MODE_MODE_ENABLED,
	})
	tests := []struct {
		name            string
		keep            map[string]map[string]string
		expectedGroups  map[string][]string
		expectedMissing []string
	}{
		{
			name:           "only kept groups and rules stay",
			keep:           map[string]map[string]string{"920": {"920450": "MODE_LOG_ONLY"}},
			expectedGroups: map[string][]string{"920": {"920450"}},
		},
		{
			name: "rules and groups that do not exist are reported",
			keep: map[string]map[string]string{
				"911": {"911100": "MODE_LOG_ONLY", "911999": "MODE_LOG_ONLY"},
				"999": {"999100": "MODE_LOG_ONLY"},
			},
			expectedGroups:  map[string][]string{"911": {"911100"}},
			expectedMissing: []string{"911/911999", "999/999100"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, missing := filterRuleSet(ruleSet, tt.keep)
			groups := map[string][]string{}
			for groupKey, group := range got.GetGroups() {
				groups[groupKey] = []string{}
				for ruleKey := range group.GetRules() {
					groups[groupKey] = append(groups[groupKey], ruleKey)
				}
			}
			if diff := cmp.Diff(groups, tt.expectedGroups); diff != "" {
				t.Errorf("groups do not match: %s", diff)
			}
			if diff := cmp.Diff(missing, tt.expectedMissing); diff != "" {
				t.Errorf("missing rules do not match: %s", diff)
			}
			if len(ruleSet.GetGroups()) != 3 {
				t.Errorf("input rule set was modified")
			}
		})
	}
}

func TestToPatchPayload(t *testing.T) {
	logOnly := albWaf.MODE_MODE_LOG_ONLY
	enabled := albWaf.MODE_MODE_ENABLED
	disabled := albWaf.MODE_MODE_DISABLED
	tests := []struct {
		name     string
		wanted   map[string]map[string]string
		previous map[string]map[string]string
		expected map[string]albWaf.PatchMRSRuleGroup
		isValid  bool
	}{
		{
			name:     "set on create",
			wanted:   map[string]map[string]string{"911": {"911100": "MODE_LOG_ONLY"}, "920": {"920450": "MODE_LOG_ONLY"}},
			previous: nil,
			expected: map[string]albWaf.PatchMRSRuleGroup{
				"911": {Rules: map[string]albWaf.PatchMRSRule{"911100": {Mode: &logOnly}}},
				"920": {Rules: map[string]albWaf.PatchMRSRule{"920450": {Mode: &logOnly}}},
			},
			isValid: true,
		},
		{
			name:     "removed rule goes back to enabled, changed rule takes its new mode",
			wanted:   map[string]map[string]string{"920": {"920450": "MODE_DISABLED"}},
			previous: map[string]map[string]string{"911": {"911100": "MODE_LOG_ONLY"}, "920": {"920450": "MODE_LOG_ONLY"}},
			expected: map[string]albWaf.PatchMRSRuleGroup{
				"911": {Rules: map[string]albWaf.PatchMRSRule{"911100": {Mode: &enabled}}},
				"920": {Rules: map[string]albWaf.PatchMRSRule{"920450": {Mode: &disabled}}},
			},
			isValid: true,
		},
		{
			name:     "groups removed entirely",
			wanted:   nil,
			previous: map[string]map[string]string{"921": {"921140": "MODE_LOG_ONLY"}},
			expected: map[string]albWaf.PatchMRSRuleGroup{
				"921": {Rules: map[string]albWaf.PatchMRSRule{"921140": {Mode: &enabled}}},
			},
			isValid: true,
		},
		{
			name:     "nothing to do",
			wanted:   nil,
			previous: nil,
			expected: map[string]albWaf.PatchMRSRuleGroup{},
			isValid:  true,
		},
		{
			name:     "invalid mode",
			wanted:   map[string]map[string]string{"911": {"911100": "MODE_FOO"}},
			previous: nil,
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toPatchPayload(tt.wanted, tt.previous)
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
