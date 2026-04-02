package alertgroup

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	observabilitySdk "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		name      string
		input     *Model
		expect    *observabilitySdk.CreateAlertgroupsPayload
		expectErr bool
	}{
		{
			name:      "Nil Model",
			input:     nil,
			expect:    nil,
			expectErr: true,
		},
		{
			name: "Empty Model",
			input: &Model{
				Name:     types.StringNull(),
				Interval: types.StringNull(),
				Rules:    types.ListNull(types.StringType),
			},
			expect:    &observabilitySdk.CreateAlertgroupsPayload{},
			expectErr: false,
		},
		{
			name: "Model with Name and Interval",
			input: &Model{
				Name:     types.StringValue("test-alertgroup"),
				Interval: types.StringValue("5m"),
			},
			expect: &observabilitySdk.CreateAlertgroupsPayload{
				Name:     "test-alertgroup",
				Interval: new("5m"),
			},
			expectErr: false,
		},
		{
			name: "Model with Full Information",
			input: &Model{
				Name:     types.StringValue("full-alertgroup"),
				Interval: types.StringValue("10m"),
				Rules: types.ListValueMust(
					types.ObjectType{AttrTypes: ruleTypes},
					[]attr.Value{
						types.ObjectValueMust(
							ruleTypes,
							map[string]attr.Value{
								"alert":      types.StringValue("alert"),
								"expression": types.StringValue("expression"),
								"for":        types.StringValue("10s"),
								"labels": types.MapValueMust(
									types.StringType,
									map[string]attr.Value{
										"k": types.StringValue("v"),
									},
								),
								"annotations": types.MapValueMust(
									types.StringType,
									map[string]attr.Value{
										"k": types.StringValue("v"),
									},
								),
								"record": types.StringValue("record"),
							},
						),
					},
				),
			},
			expect: &observabilitySdk.CreateAlertgroupsPayload{
				Name:     "full-alertgroup",
				Interval: new("10m"),
				Rules: []observabilitySdk.CreateAlertgroupsPayloadRulesInner{
					{
						Alert: new("alert"),
						Annotations: map[string]any{
							"k": "v",
						},
						Expr: "expression",
						For:  new("10s"),
						Labels: map[string]any{
							"k": "v",
						},
						Record: new("record"),
					},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := toCreatePayload(ctx, tt.input)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if diff := cmp.Diff(got, tt.expect); diff != "" {
				t.Errorf("unexpected result (-got +want):\n%s", diff)
			}
		})
	}
}

func TestToRulesPayload(t *testing.T) {
	tests := []struct {
		name      string
		input     *Model
		expect    []observabilitySdk.CreateAlertgroupsPayloadRulesInner
		expectErr bool
	}{
		{
			name: "Nil Rules",
			input: &Model{
				Rules: types.ListNull(types.StringType), // Simulates a lack of rules
			},
			expect:    []observabilitySdk.CreateAlertgroupsPayloadRulesInner{},
			expectErr: false,
		},
		{
			name: "Invalid Rule Element Type",
			input: &Model{
				Rules: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("invalid"), // Should cause a conversion failure
				}),
			},
			expect:    nil,
			expectErr: true,
		},
		{
			name: "Single Valid Rule",
			input: &Model{
				Rules: types.ListValueMust(types.ObjectType{AttrTypes: ruleTypes}, []attr.Value{
					types.ObjectValueMust(ruleTypes, map[string]attr.Value{
						"alert":      types.StringValue("alert"),
						"expression": types.StringValue("expr"),
						"for":        types.StringValue("5s"),
						"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
							"key": types.StringValue("value"),
						}),
						"annotations": types.MapValueMust(types.StringType, map[string]attr.Value{
							"note": types.StringValue("important"),
						}),
						"record": types.StringValue("record"),
					}),
				}),
			},
			expect: []observabilitySdk.CreateAlertgroupsPayloadRulesInner{
				{
					Alert: new("alert"),
					Expr:  "expr",
					For:   new("5s"),
					Labels: map[string]any{
						"key": "value",
					},
					Annotations: map[string]any{
						"note": "important",
					},
					Record: new("record"),
				},
			},
			expectErr: false,
		},
		{
			name: "Multiple Valid Rules",
			input: &Model{
				Rules: types.ListValueMust(types.ObjectType{AttrTypes: ruleTypes}, []attr.Value{
					types.ObjectValueMust(ruleTypes, map[string]attr.Value{
						"alert":       types.StringValue("alert1"),
						"expression":  types.StringValue("expr1"),
						"for":         types.StringValue("5s"),
						"labels":      types.MapNull(types.StringType),
						"annotations": types.MapNull(types.StringType),
						"record":      types.StringValue("record1"),
					}),
					types.ObjectValueMust(ruleTypes, map[string]attr.Value{
						"alert":      types.StringValue("alert2"),
						"expression": types.StringValue("expr2"),
						"for":        types.StringValue("10s"),
						"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
							"key": types.StringValue("value"),
						}),
						"annotations": types.MapValueMust(types.StringType, map[string]attr.Value{
							"note": types.StringValue("important"),
						}),
						"record": types.StringValue("record2"),
					}),
				}),
			},
			expect: []observabilitySdk.CreateAlertgroupsPayloadRulesInner{
				{
					Alert:  new("alert1"),
					Expr:   "expr1",
					For:    new("5s"),
					Record: new("record1"),
				},
				{
					Alert: new("alert2"),
					Expr:  "expr2",
					For:   new("10s"),
					Labels: map[string]any{
						"key": "value",
					},
					Annotations: map[string]any{
						"note": "important",
					},
					Record: new("record2"),
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := toRulesPayload(ctx, tt.input)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if diff := cmp.Diff(got, tt.expect); diff != "" {
				t.Errorf("unexpected result (-got +want):\n%s", diff)
			}
		})
	}
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		name         string
		alertGroup   *observabilitySdk.AlertGroup
		model        *Model
		expectedName string
		expectedID   string
		expectErr    bool
	}{
		{
			name:       "Nil AlertGroup",
			alertGroup: nil,
			model:      &Model{},
			expectErr:  true,
		},
		{
			name:       "Nil Model",
			alertGroup: &observabilitySdk.AlertGroup{},
			model:      nil,
			expectErr:  true,
		},
		{
			name: "Interval Missing",
			alertGroup: &observabilitySdk.AlertGroup{
				Name: "alert-group-name",
			},
			model: &Model{
				Name:       types.StringValue("alert-group-name"),
				ProjectId:  types.StringValue("project1"),
				InstanceId: types.StringValue("instance1"),
			},
			expectedName: "alert-group-name",
			expectedID:   "project1,instance1,alert-group-name",
			expectErr:    true,
		},
		{
			name: "Name Missing",
			alertGroup: &observabilitySdk.AlertGroup{
				Interval: new("5m"),
			},
			model: &Model{
				Name:       types.StringValue("model-name"),
				InstanceId: types.StringValue("instance1"),
			},
			expectErr: true,
		},
		{
			name: "Complete Model and AlertGroup",
			alertGroup: &observabilitySdk.AlertGroup{
				Name:     "alert-group-name",
				Interval: new("10m"),
			},
			model: &Model{
				Name:       types.StringValue("alert-group-name"),
				ProjectId:  types.StringValue("project1"),
				InstanceId: types.StringValue("instance1"),
				Id:         types.StringValue("project1,instance1,alert-group-name"),
				Interval:   types.StringValue("10m"),
			},
			expectedName: "alert-group-name",
			expectedID:   "project1,instance1,alert-group-name",
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := mapFields(ctx, tt.alertGroup, tt.model)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				if diff := cmp.Diff(tt.model.Name.ValueString(), tt.expectedName); diff != "" {
					t.Errorf("unexpected name (-got +want):\n%s", diff)
				}
				if diff := cmp.Diff(tt.model.Id.ValueString(), tt.expectedID); diff != "" {
					t.Errorf("unexpected ID (-got +want):\n%s", diff)
				}
			}
		})
	}
}

func TestMapRules(t *testing.T) {
	tests := []struct {
		name       string
		alertGroup *observabilitySdk.AlertGroup
		model      *Model
		expectErr  bool
	}{
		{
			name: "Empty Rules",
			alertGroup: &observabilitySdk.AlertGroup{
				Rules: []observabilitySdk.AlertRuleRecord{},
			},
			model:     &Model{},
			expectErr: false,
		},
		{
			name: "Single Complete Rule",
			alertGroup: &observabilitySdk.AlertGroup{
				Rules: []observabilitySdk.AlertRuleRecord{
					{
						Alert:       new("HighCPUUsage"),
						Expr:        "rate(cpu_usage[5m]) > 0.9",
						For:         new("2m"),
						Labels:      &map[string]string{"severity": "critical"},
						Annotations: &map[string]string{"summary": "CPU usage high"},
						Record:      new("record1"),
					},
				},
			},
			model:     &Model{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := mapRules(ctx, tt.alertGroup, tt.model)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err != nil)
			}
		})
	}
}
