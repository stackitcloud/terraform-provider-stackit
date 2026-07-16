package managed_rule_set

import (
	"context"
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	albwaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"
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
		expected *albwaf.CreateManagedRuleSetPayload
		isValid  bool
	}{
		{
			name: "default",
			model: &Model{
				Name:      testName,
				Id:        testId,
				ProjectId: testProjectId,
				Region:    testRegion,
				Type:      types.StringValue(string(albwaf.MRSTYPE_TYPE_OWASP_CRS)),
			},
			expected: &albwaf.CreateManagedRuleSetPayload{
				Name: testName.ValueStringPointer(),
				Type: new(albwaf.MRSTYPE_TYPE_OWASP_CRS),
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
		input    *albwaf.GetManagedRuleSetResponse
		expected *Model
		isValid  bool
	}{
		{
			name: "default",
			state: &Model{
				ProjectId: testProjectId,
				Region:    testRegion,
				Name:      testName,
				Type:      types.StringValue(string(albwaf.MRSTYPE_TYPE_OWASP_CRS)),
				Id:        testId,
				Groups:    types.MapValueMust(types.ObjectType{AttrTypes: ruleGroupType}, map[string]attr.Value{}),
			},
			region: testRegion.ValueString(),
			input: &albwaf.GetManagedRuleSetResponse{
				Groups: &map[string]albwaf.MRSRuleGroup{},
				Name:   testName.ValueStringPointer(),
				Type:   new(albwaf.MRSTYPE2_TYPE_OWASP_CRS),
			},
			expected: &Model{
				ProjectId: testProjectId,
				Region:    testRegion,
				Name:      testName,
				Type:      types.StringValue(string(albwaf.MRSTYPE_TYPE_OWASP_CRS)),
				Id:        testId,
				Groups:    types.MapValueMust(types.ObjectType{AttrTypes: ruleGroupType}, map[string]attr.Value{}),
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
