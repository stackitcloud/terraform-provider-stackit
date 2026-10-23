package volume

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"

	automation "github.com/stackitcloud/stackit-sdk-go/services/automation/v1api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *automation.VolumeAutomation
		expected    Model
		isValid     bool
	}{
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"default_values",
			&automation.VolumeAutomation{
				Id: "automation_uid",
			},
			Model{
				ID:           types.StringValue("project_uid,eu01,automation_uid"),
				ProjectId:    types.StringValue("project_uid"),
				Region:       types.StringValue("eu01"),
				AutomationId: types.StringValue("automation_uid"),
				Name:         types.StringNull(),
				Description:  types.StringNull(),
			},
			true,
		},
		{
			"empty_strings_for_name_and_description",
			&automation.VolumeAutomation{
				Id:          "automation_uid",
				Name:        new(""),
				Description: new(""),
				Input: *automation.NewNullableVolumeAutomationInput(&automation.VolumeAutomationInput{
					Kind: "VolumeRecoveryPointManagementInput",
					AdditionalProperties: map[string]interface{}{
						"volumeLabelSelector": "",
						"snapshotRetentionPolicy": map[string]interface{}{
							"kind": "indefinitely",
						},
					},
				}),
			},
			Model{
				ID:           types.StringValue("project_uid,eu01,automation_uid"),
				ProjectId:    types.StringValue("project_uid"),
				Region:       types.StringValue("eu01"),
				AutomationId: types.StringValue("automation_uid"),
				Name:         types.StringValue(""),
				Description:  types.StringValue(""),
				Input:        jsontypes.NewNormalizedValue("{\"kind\":\"VolumeRecoveryPointManagementInput\",\"snapshotRetentionPolicy\":{\"kind\":\"indefinitely\"},\"volumeLabelSelector\":\"\"}"),
			},
			true,
		},
		{
			"full_values_count",
			&automation.VolumeAutomation{
				Id:          "automation_uid",
				TemplateId:  new("template_uid"),
				Name:        new("name1"),
				Description: new("desc1"),
				Input: *automation.NewNullableVolumeAutomationInput(&automation.VolumeAutomationInput{
					Kind: "VolumeRecoveryPointManagement",
					AdditionalProperties: map[string]interface{}{
						"inheritVolumeLabels": true,
						"recoveryPointLabels": map[string]string{
							"k": "v",
						},
						"snapshotRetentionPolicy": map[string]interface{}{
							"kind":  "count",
							"value": 7,
						},
						"volumeLabelSelector": "sel",
					},
				}),
				Triggers: *automation.NewNullableAutomationTriggers(&automation.AutomationTriggers{
					Schedule: *automation.NewNullableAutomationScheduleTrigger(&automation.AutomationScheduleTrigger{
						Rrule: "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1",
					}),
				}),
			},
			Model{
				ID:           types.StringValue("project_uid,eu01,automation_uid"),
				ProjectId:    types.StringValue("project_uid"),
				Region:       types.StringValue("eu01"),
				AutomationId: types.StringValue("automation_uid"),
				TemplateId:   types.StringValue("template_uid"),
				Name:         types.StringValue("name1"),
				Description:  types.StringValue("desc1"),
				Input:        jsontypes.NewNormalizedValue("{\"inheritVolumeLabels\":true,\"kind\":\"VolumeRecoveryPointManagement\",\"recoveryPointLabels\":{\"k\":\"v\"},\"snapshotRetentionPolicy\":{\"kind\":\"count\",\"value\":7},\"volumeLabelSelector\":\"sel\"}"),
				Triggers: &triggersModel{
					Schedule: &scheduleTriggerModel{
						Rrule: types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
					},
				},
			},
			true,
		},
		{
			"full_values_indefinitely",
			&automation.VolumeAutomation{
				Id: "automation_uid",
				Input: *automation.NewNullableVolumeAutomationInput(&automation.VolumeAutomationInput{
					Kind: "VolumeRecoveryPointManagement",
					AdditionalProperties: map[string]interface{}{
						"snapshotRetentionPolicy": map[string]interface{}{
							"kind": "indefinitely",
						},
					},
				}),
			},
			Model{
				ID:           types.StringValue("project_uid,eu01,automation_uid"),
				ProjectId:    types.StringValue("project_uid"),
				Region:       types.StringValue("eu01"),
				AutomationId: types.StringValue("automation_uid"),
				Name:         types.StringNull(),
				Description:  types.StringNull(),
				Input:        jsontypes.NewNormalizedValue("{\"kind\":\"VolumeRecoveryPointManagement\",\"snapshotRetentionPolicy\":{\"kind\":\"indefinitely\"}}"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
			}
			ctx := context.TODO()
			err := mapFields(ctx, tt.input, state, "eu01")
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *automation.CreateVolumeAutomationPayload
		isValid     bool
	}{
		{
			"nil_model",
			nil,
			nil,
			false,
		},
		{
			"minimal",
			&Model{
				TemplateId: types.StringValue("template_uid"),
			},
			&automation.CreateVolumeAutomationPayload{
				TemplateId:  "template_uid",
				Name:        *automation.NewNullableString(nil),
				Description: *automation.NewNullableString(nil),
				Input:       *automation.NewNullableVolumeAutomationInput(nil),
				Triggers:    *automation.NewNullableAutomationTriggers(nil),
			},
			true,
		},
		{
			"full_values_count",
			&Model{
				TemplateId:  types.StringValue("template_uid"),
				Name:        types.StringValue("name1"),
				Description: types.StringValue("desc1"),
				Input:       jsontypes.NewNormalizedValue("{\"inheritVolumeLabels\":true,\"kind\":\"VolumeRecoveryPointManagement\",\"recoveryPointLabels\":{\"k\":\"v\"},\"snapshotRetentionPolicy\":{\"kind\":\"count\",\"value\":7},\"volumeLabelSelector\":\"sel\"}"),
				Triggers: &triggersModel{
					Schedule: &scheduleTriggerModel{
						Rrule: types.StringValue("RRULE"),
					},
				},
			},
			&automation.CreateVolumeAutomationPayload{
				TemplateId:  "template_uid",
				Name:        *automation.NewNullableString(new("name1")),
				Description: *automation.NewNullableString(new("desc1")),
				Input: *automation.NewNullableVolumeAutomationInput(&automation.VolumeAutomationInput{
					Kind: "VolumeRecoveryPointManagement",
					AdditionalProperties: map[string]interface{}{
						"inheritVolumeLabels": true,
						"recoveryPointLabels": map[string]interface{}{
							"k": "v",
						},
						"snapshotRetentionPolicy": map[string]interface{}{
							"kind":  "count",
							"value": float64(7),
						},
						"volumeLabelSelector": "sel",
					},
				}),
				Triggers: *automation.NewNullableAutomationTriggers(&automation.AutomationTriggers{
					Schedule: *automation.NewNullableAutomationScheduleTrigger(&automation.AutomationScheduleTrigger{
						Rrule: "RRULE",
					}),
				}),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				cmpOpts := cmp.Options{
					cmp.AllowUnexported(
						automation.NullableVolumeAutomationInput{},
						automation.NullableAutomationTriggers{},
						automation.NullableAutomationScheduleTrigger{},
						automation.NullableString{},
					),
				}
				diff := cmp.Diff(tt.expected, output, cmpOpts)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		plan        *Model
		expected    *automation.PartialUpdateVolumeAutomationPayload
		isValid     bool
	}{
		{
			"nil_plan",
			nil,
			nil,
			false,
		},
		{
			"full_values",
			&Model{
				Name:        types.StringValue("n"),
				Description: types.StringValue("d"),
				Input:       jsontypes.NewNormalizedValue("{\"kind\":\"VolumeRecoveryPointManagement\",\"snapshotRetentionPolicy\":{\"kind\":\"count\",\"value\":3}}\n"),
				Triggers: &triggersModel{
					Schedule: &scheduleTriggerModel{Rrule: types.StringValue("RRULE")},
				},
			},
			&automation.PartialUpdateVolumeAutomationPayload{
				Name:        *automation.NewNullableString(new("n")),
				Description: *automation.NewNullableString(new("d")),
				Input: *automation.NewNullableVolumeAutomationInput(&automation.VolumeAutomationInput{
					Kind: "VolumeRecoveryPointManagement",
					AdditionalProperties: map[string]interface{}{
						"snapshotRetentionPolicy": map[string]interface{}{
							"kind":  "count",
							"value": float64(3),
						},
					},
				}),
				Triggers: *automation.NewNullableAutomationTriggers(&automation.AutomationTriggers{
					Schedule: *automation.NewNullableAutomationScheduleTrigger(&automation.AutomationScheduleTrigger{Rrule: "RRULE"}),
				}),
			},
			true,
		},
		{
			"clears_optional_fields",
			&Model{
				Name:        types.StringNull(),
				Description: types.StringNull(),
			},
			&automation.PartialUpdateVolumeAutomationPayload{
				Name:        *automation.NewNullableString(nil),
				Description: *automation.NewNullableString(nil),
				Input:       *automation.NewNullableVolumeAutomationInput(nil),
				Triggers:    *automation.NewNullableAutomationTriggers(nil),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.plan)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				cmpOpts := cmp.Options{
					cmp.AllowUnexported(
						automation.NullableVolumeAutomationInput{},
						automation.NullableAutomationTriggers{},
						automation.NullableAutomationScheduleTrigger{},
						automation.NullableString{},
					),
				}
				diff := cmp.Diff(output, tt.expected, cmpOpts)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
