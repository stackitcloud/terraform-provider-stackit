package custom_rule_group

import (
	"context"
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"
)

var (
	testProjectId = types.StringValue(uuid.NewString())
	testRegion    = types.StringValue("eu01")
	testName      = types.StringValue("test-custom-rule-group")
	testId        = types.StringValue(testProjectId.ValueString() + "," + testRegion.ValueString() + "," + testName.ValueString())
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		name     string
		model    *Model
		expected *albWaf.CreateCustomRuleGroupPayload
		isValid  bool
	}{
		{
			name: "default",
			model: &Model{
				Name:      testName,
				Id:        testId,
				ProjectId: testProjectId,
				Region:    testRegion,
				Rules: types.ListValueMust(types.ObjectType{AttrTypes: ruleType}, []attr.Value{
					types.ObjectValueMust(ruleType, map[string]attr.Value{
						"behaviour": types.ObjectValueMust(behaviourType, map[string]attr.Value{
							"action":   types.StringValue("some-action"),
							"log":      types.BoolValue(true),
							"log_msg":  types.StringValue("Log: something happened"),
							"severity": types.StringNull(),
						}),
						"conditions": types.ListValueMust(types.ObjectType{AttrTypes: conditionType}, []attr.Value{
							types.ObjectValueMust(conditionType, map[string]attr.Value{
								"operator": types.ObjectValueMust(operatorType, map[string]attr.Value{
									"type":  types.StringValue("operator-type"),
									"value": types.StringValue("operator-value"),
								}),
								"transformations": types.ListValueMust(types.StringType, []attr.Value{
									types.StringValue("foo"),
									types.StringValue("bar"),
								}),
								"variable": types.ObjectValueMust(variableType, map[string]attr.Value{
									"type":  types.StringValue("variable-type"),
									"value": types.StringValue("variable-value"),
								}),
							}),
						}),
						"description": types.StringValue("foo-bar"),
						"id":          types.Int32Null(),
					}),
				}),
			},
			expected: &albWaf.CreateCustomRuleGroupPayload{
				Name: testName.ValueStringPointer(),
				Rules: []albWaf.CreateCustomRule{
					{
						Behaviour: &albWaf.Behaviour{
							Action: new(albWaf.BehaviourAction("some-action")),
							Log:    new(true),
							LogMsg: new("Log: something happened"),
						},
						Conditions: []albWaf.Condition{
							{
								Operator: &albWaf.ConditionOperator{
									Type:  new(albWaf.ConditionOperatorType("operator-type")),
									Value: new("operator-value"),
								},
								Transformations: []albWaf.ConditionTransformationsInner{
									"foo",
									"bar",
								},
								Variable: &albWaf.ConditionVariable{
									Type:  new(albWaf.ConditionVariableType("variable-type")),
									Value: new("variable-value"),
								},
							},
						},
						Description: new("foo-bar"),
					},
				},
			},
			isValid: true,
		},
		{
			name: "null values",
			model: &Model{
				Name:      testName,
				Id:        testId,
				ProjectId: testProjectId,
				Region:    testRegion,
				Rules: types.ListValueMust(types.ObjectType{AttrTypes: ruleType}, []attr.Value{
					types.ObjectValueMust(ruleType, map[string]attr.Value{
						"behaviour": types.ObjectValueMust(behaviourType, map[string]attr.Value{
							"action":   types.StringNull(),
							"log":      types.BoolNull(),
							"log_msg":  types.StringNull(),
							"severity": types.StringNull(),
						}),
						"conditions": types.ListValueMust(types.ObjectType{AttrTypes: conditionType}, []attr.Value{
							types.ObjectValueMust(conditionType, map[string]attr.Value{
								"operator": types.ObjectValueMust(operatorType, map[string]attr.Value{
									"type":  types.StringNull(),
									"value": types.StringNull(),
								}),
								"transformations": types.ListValueMust(types.StringType, []attr.Value{}),
								"variable": types.ObjectValueMust(variableType, map[string]attr.Value{
									"type":  types.StringNull(),
									"value": types.StringNull(),
								}),
							}),
						}),
						"description": types.StringNull(),
						"id":          types.Int32Null(),
					}),
				}),
			},
			expected: &albWaf.CreateCustomRuleGroupPayload{
				Name: testName.ValueStringPointer(),
				Rules: []albWaf.CreateCustomRule{
					{
						Behaviour: &albWaf.Behaviour{},
						Conditions: []albWaf.Condition{
							{
								Operator:        &albWaf.ConditionOperator{},
								Transformations: []albWaf.ConditionTransformationsInner{},
								Variable:        &albWaf.ConditionVariable{},
							},
						},
					},
				},
			},
			isValid: true,
		},
		{
			name: "no rules",
			model: &Model{
				Name:      testName,
				Id:        testId,
				ProjectId: testProjectId,
				Region:    testRegion,
			},
			expected: &albWaf.CreateCustomRuleGroupPayload{
				Name:  testName.ValueStringPointer(),
				Rules: []albWaf.CreateCustomRule{},
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
		input    *albWaf.GetCustomRuleGroupResponse
		expected *Model
		isValid  bool
	}{
		{
			name: "default",
			state: &Model{
				ProjectId: testProjectId,
				Region:    testRegion,
				Name:      testName,
				Id:        testId,
				Rules:     types.ListNull(types.ObjectType{AttrTypes: ruleType}),
			},
			region: testRegion.ValueString(),
			input: &albWaf.GetCustomRuleGroupResponse{
				Name: testName.ValueStringPointer(),
				Rules: []albWaf.GetCustomRule{
					{
						Behaviour: &albWaf.GetBehaviour{
							Action:   new(albWaf.GetBehaviourAction("some-action")),
							Log:      new(true),
							LogMsg:   new("Log: something happened"),
							Severity: new(albWaf.GetBehaviourSeverity("critical")),
						},
						Conditions: []albWaf.Condition{
							{
								Operator: &albWaf.ConditionOperator{
									Type:  new(albWaf.ConditionOperatorType("operator-type")),
									Value: new("operator-value"),
								},
								Transformations: []albWaf.ConditionTransformationsInner{
									"foo",
									"bar",
								},
								Variable: &albWaf.ConditionVariable{
									Type:  new(albWaf.ConditionVariableType("variable-type")),
									Value: new("variable-value"),
								},
							},
						},
						Description: new("foo-bar"),
						Id:          new(int32(42)),
					},
				},
				Usage: &albWaf.CRGUsage{
					Count: new(int32(42)),
					Items: []string{
						"one",
						"two",
						"three",
					},
				},
			},
			expected: &Model{
				ProjectId: testProjectId,
				Region:    testRegion,
				Name:      testName,
				Id:        testId,
				Rules: types.ListValueMust(types.ObjectType{AttrTypes: ruleType}, []attr.Value{
					types.ObjectValueMust(ruleType, map[string]attr.Value{
						"behaviour": types.ObjectValueMust(behaviourType, map[string]attr.Value{
							"action":   types.StringValue("some-action"),
							"log":      types.BoolValue(true),
							"log_msg":  types.StringValue("Log: something happened"),
							"severity": types.StringValue("critical"),
						}),
						"conditions": types.ListValueMust(types.ObjectType{AttrTypes: conditionType}, []attr.Value{
							types.ObjectValueMust(conditionType, map[string]attr.Value{
								"operator": types.ObjectValueMust(operatorType, map[string]attr.Value{
									"type":  types.StringValue("operator-type"),
									"value": types.StringValue("operator-value"),
								}),
								"transformations": types.ListValueMust(types.StringType, []attr.Value{
									types.StringValue("foo"),
									types.StringValue("bar"),
								}),
								"variable": types.ObjectValueMust(variableType, map[string]attr.Value{
									"type":  types.StringValue("variable-type"),
									"value": types.StringValue("variable-value"),
								}),
							}),
						}),
						"description": types.StringValue("foo-bar"),
						"id":          types.Int32Value(42),
					}),
				}),
				Usage: types.ObjectValueMust(usageType, map[string]attr.Value{
					"count": types.Int32Value(42),
					"items": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("one"),
						types.StringValue("two"),
						types.StringValue("three"),
					}),
				}),
			},
			isValid: true,
		},
		{
			name: "empty rule",
			state: &Model{
				ProjectId: testProjectId,
				Region:    testRegion,
				Name:      testName,
				Id:        testId,
				Rules:     types.ListNull(types.ObjectType{AttrTypes: ruleType}),
			},
			region: testRegion.ValueString(),
			input: &albWaf.GetCustomRuleGroupResponse{
				Rules: []albWaf.GetCustomRule{
					{},
				},
			},
			expected: &Model{
				Name:      testName,
				Id:        testId,
				ProjectId: testProjectId,
				Region:    testRegion,
				Rules: types.ListValueMust(types.ObjectType{AttrTypes: ruleType}, []attr.Value{
					types.ObjectValueMust(ruleType, map[string]attr.Value{
						"behaviour":   types.ObjectNull(behaviourType),
						"conditions":  types.ListNull(types.ObjectType{AttrTypes: conditionType}),
						"description": types.StringNull(),
						"id":          types.Int32Null(),
					}),
				}),
			},
			isValid: true,
		},
		{
			name: "no rules",
			state: &Model{
				ProjectId: testProjectId,
				Region:    testRegion,
				Name:      testName,
				Id:        testId,
			},
			region: testRegion.ValueString(),
			input:  &albWaf.GetCustomRuleGroupResponse{},
			expected: &Model{
				Name:      testName,
				Id:        testId,
				ProjectId: testProjectId,
				Region:    testRegion,
				Rules:     types.ListValueMust(types.ObjectType{AttrTypes: ruleType}, []attr.Value{}),
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
