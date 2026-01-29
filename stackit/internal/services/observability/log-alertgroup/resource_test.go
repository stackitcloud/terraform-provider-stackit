package logalertgroup

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		name      string
		input     *Model
		expect    *observability.CreateLogsAlertgroupsPayload
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
			expect:    &observability.CreateLogsAlertgroupsPayload{},
			expectErr: false,
		},
		{
			name: "Model with Name and Interval",
			input: &Model{
				Name:     types.StringValue("test-alertgroup"),
				Interval: types.StringValue("5m"),
			},
			expect: &observability.CreateLogsAlertgroupsPayload{
				Name:     utils.Ptr("test-alertgroup"),
				Interval: utils.Ptr("5m"),
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
							},
						),
					},
				),
			},
			expect: &observability.CreateLogsAlertgroupsPayload{
				Name:     utils.Ptr("full-alertgroup"),
				Interval: utils.Ptr("10m"),
				Rules: &[]observability.CreateLogsAlertgroupsPayloadRulesInner{
					{
						Alert: utils.Ptr("alert"),
						Annotations: &map[string]interface{}{
							"k": "v",
						},
						Expr: utils.Ptr("expression"),
						For:  utils.Ptr("10s"),
						Labels: &map[string]interface{}{
							"k": "v",
						},
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
		expect    []observability.UpdateAlertgroupsRequestInnerRulesInner
		expectErr bool
	}{
		{
			name: "Nil Rules",
			input: &Model{
				Rules: types.ListNull(types.StringType), // Simulates a lack of rules
			},
			expect:    []observability.UpdateAlertgroupsRequestInnerRulesInner{},
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
					}),
				}),
			},
			expect: []observability.UpdateAlertgroupsRequestInnerRulesInner{
				{
					Alert: utils.Ptr("alert"),
					Expr:  utils.Ptr("expr"),
					For:   utils.Ptr("5s"),
					Labels: &map[string]interface{}{
						"key": "value",
					},
					Annotations: &map[string]interface{}{
						"note": "important",
					},
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
					}),
				}),
			},
			expect: []observability.UpdateAlertgroupsRequestInnerRulesInner{
				{
					Alert: utils.Ptr("alert1"),
					Expr:  utils.Ptr("expr1"),
					For:   utils.Ptr("5s"),
				},
				{
					Alert: utils.Ptr("alert2"),
					Expr:  utils.Ptr("expr2"),
					For:   utils.Ptr("10s"),
					Labels: &map[string]interface{}{
						"key": "value",
					},
					Annotations: &map[string]interface{}{
						"note": "important",
					},
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
		alertGroup   *observability.AlertGroup
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
			alertGroup: &observability.AlertGroup{},
			model:      nil,
			expectErr:  true,
		},
		{
			name: "Interval Missing",
			alertGroup: &observability.AlertGroup{
				Name: utils.Ptr("alert-group-name"),
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
			alertGroup: &observability.AlertGroup{
				Interval: utils.Ptr("5m"),
			},
			model: &Model{
				Name:       types.StringValue("model-name"),
				InstanceId: types.StringValue("instance1"),
			},
			expectErr: true,
		},
		{
			name: "Complete Model and AlertGroup",
			alertGroup: &observability.AlertGroup{
				Name:     utils.Ptr("alert-group-name"),
				Interval: utils.Ptr("10m"),
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
		alertGroup *observability.AlertGroup
		model      *Model
		expectErr  bool
	}{
		{
			name: "Empty Rules",
			alertGroup: &observability.AlertGroup{
				Rules: &[]observability.AlertRuleRecord{},
			},
			model:     &Model{},
			expectErr: false,
		},
		{
			name: "Single Complete Rule",
			alertGroup: &observability.AlertGroup{
				Rules: &[]observability.AlertRuleRecord{
					{
						Alert:       utils.Ptr("HighCPUUsage"),
						Expr:        utils.Ptr("rate(cpu_usage[5m]) > 0.9"),
						For:         utils.Ptr("2m"),
						Labels:      &map[string]string{"severity": "critical"},
						Annotations: &map[string]string{"summary": "CPU usage high"},
						Record:      utils.Ptr("record1"),
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
