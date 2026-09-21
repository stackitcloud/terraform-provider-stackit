package volume

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"

	automation "github.com/stackitcloud/stackit-sdk-go/services/automation/v1betaapi"
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
		//{
		//	"full_values_count",
		//	&automation.VolumeAutomation{
		//		Id:          "automation_uid",
		//		TemplateId:  new("template_uid"),
		//		Name:        new("name1"),
		//		Description: new("desc1"),
		//		Input: &automation.VolumeAutomationInput{
		//			VolumeRecoveryPointManagementInput: &automation.VolumeRecoveryPointManagementInput{
		//				Kind:                "VolumeRecoveryPointManagement",
		//				InheritVolumeLabels: new(true),
		//				RecoveryPointLabels: &map[string]string{"k": "v"},
		//				VolumeLabelSelector: new("sel"),
		//				SnapshotRetentionPolicy: automation.SnapshotRetentionPolicyCountAsSnapshotRetentionPolicy(&automation.SnapshotRetentionPolicyCount{
		//					Kind:  automation.SNAPSHOTRETENTIONPOLICYCOUNTKIND_COUNT,
		//					Value: 7,
		//				}),
		//			},
		//		},
		//		Triggers: &automation.AutomationTriggers{
		//			Schedule: &automation.AutomationScheduleTrigger{
		//				Rrule: "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1",
		//			},
		//		},
		//	},
		//	Model{
		//		ID:           types.StringValue("project_uid,eu01,automation_uid"),
		//		ProjectId:    types.StringValue("project_uid"),
		//		Region:       types.StringValue("eu01"),
		//		AutomationId: types.StringValue("automation_uid"),
		//		TemplateId:   types.StringValue("template_uid"),
		//		Name:         types.StringValue("name1"),
		//		Description:  types.StringValue("desc1"),
		//		Input: &inputModel{
		//			VolumeRecoveryPointManagement: &volumeRecoveryPointManagementModel{
		//				InheritVolumeLabels: types.BoolValue(true),
		//				RecoveryPointLabels: types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")}),
		//				VolumeLabelSelector: types.StringValue("sel"),
		//				SnapshotRetentionPolicy: &snapshotRetentionPolicyModel{
		//					Kind:  types.StringValue("count"),
		//					Value: types.Int32Value(7),
		//				},
		//			},
		//		},
		//		Triggers: &triggersModel{
		//			Schedule: &scheduleTriggerModel{
		//				Rrule: types.StringValue("DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"),
		//			},
		//		},
		//	},
		//	true,
		//},
		//{
		//	"full_values_indefinitely",
		//	&automation.VolumeAutomation{
		//		Id: "automation_uid",
		//		Input: &automation.VolumeAutomationInput{
		//			VolumeRecoveryPointManagementInput: &automation.VolumeRecoveryPointManagementInput{
		//				Kind: "VolumeRecoveryPointManagement",
		//				SnapshotRetentionPolicy: automation.SnapshotRetentionPolicyIndefinitelyAsSnapshotRetentionPolicy(&automation.SnapshotRetentionPolicyIndefinitely{
		//					Kind: automation.SNAPSHOTRETENTIONPOLICYINDEFINITELYKIND_INDEFINITELY,
		//				}),
		//			},
		//		},
		//	},
		//	Model{
		//		ID:           types.StringValue("project_uid,eu01,automation_uid"),
		//		ProjectId:    types.StringValue("project_uid"),
		//		Region:       types.StringValue("eu01"),
		//		AutomationId: types.StringValue("automation_uid"),
		//		Name:         types.StringNull(),
		//		Description:  types.StringNull(),
		//		Input: &inputModel{
		//			VolumeRecoveryPointManagement: &volumeRecoveryPointManagementModel{
		//				InheritVolumeLabels: types.BoolNull(),
		//				RecoveryPointLabels: types.MapNull(types.StringType),
		//				VolumeLabelSelector: types.StringNull(),
		//				SnapshotRetentionPolicy: &snapshotRetentionPolicyModel{
		//					Kind:  types.StringValue("indefinitely"),
		//					Value: types.Int32Null(),
		//				},
		//			},
		//		},
		//	},
		//	true,
		//},
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
				TemplateId: "template_uid",
				Input:      *automation.NewNullableVolumeAutomationInput(nil),
				Triggers:   *automation.NewNullableAutomationTriggers(nil),
			},
			true,
		},
		//{
		//	"full_values_count",
		//	&Model{
		//		TemplateId:  types.StringValue("template_uid"),
		//		Name:        types.StringValue("name1"),
		//		Description: types.StringValue("desc1"),
		//		Input: &inputModel{
		//			VolumeRecoveryPointManagement: &volumeRecoveryPointManagementModel{
		//				InheritVolumeLabels: types.BoolValue(true),
		//				RecoveryPointLabels: types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")}),
		//				VolumeLabelSelector: types.StringValue("sel"),
		//				SnapshotRetentionPolicy: &snapshotRetentionPolicyModel{
		//					Kind:  types.StringValue("count"),
		//					Value: types.Int32Value(7),
		//				},
		//			},
		//		},
		//		Triggers: &triggersModel{
		//			Schedule: &scheduleTriggerModel{
		//				Rrule: types.StringValue("RRULE"),
		//			},
		//		},
		//	},
		//	&automation.CreateVolumeAutomationPayload{
		//		TemplateId:  "template_uid",
		//		Name:        new("name1"),
		//		Description: new("desc1"),
		//		Input: &automation.VolumeAutomationInput{
		//			VolumeRecoveryPointManagementInput: &automation.VolumeRecoveryPointManagementInput{
		//				Kind:                "VolumeRecoveryPointManagement",
		//				InheritVolumeLabels: new(true),
		//				RecoveryPointLabels: &map[string]string{"k": "v"},
		//				VolumeLabelSelector: new("sel"),
		//				SnapshotRetentionPolicy: automation.SnapshotRetentionPolicyCountAsSnapshotRetentionPolicy(&automation.SnapshotRetentionPolicyCount{
		//					Kind:  automation.SNAPSHOTRETENTIONPOLICYCOUNTKIND_COUNT,
		//					Value: 7,
		//				}),
		//			},
		//		},
		//		Triggers: &automation.AutomationTriggers{
		//			Schedule: &automation.AutomationScheduleTrigger{
		//				Rrule: "RRULE",
		//			},
		//		},
		//	},
		//	true,
		//},
		//{
		//	"count_kind_missing_value",
		//	&Model{
		//		TemplateId: types.StringValue("template_uid"),
		//		Input: &inputModel{
		//			VolumeRecoveryPointManagement: &volumeRecoveryPointManagementModel{
		//				SnapshotRetentionPolicy: &snapshotRetentionPolicyModel{
		//					Kind: types.StringValue("count"),
		//				},
		//			},
		//		},
		//	},
		//	nil,
		//	false,
		//},
		//{
		//	"missing_snapshot_retention_policy",
		//	&Model{
		//		TemplateId: types.StringValue("template_uid"),
		//		Input: &inputModel{
		//			VolumeRecoveryPointManagement: &volumeRecoveryPointManagementModel{},
		//		},
		//	},
		//	nil,
		//	false,
		//},
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
	//baseInput := &inputModel{
	//	VolumeRecoveryPointManagement: &volumeRecoveryPointManagementModel{
	//		SnapshotRetentionPolicy: &snapshotRetentionPolicyModel{
	//			Kind:  types.StringValue("count"),
	//			Value: types.Int32Value(3),
	//		},
	//	},
	//}

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
		//{
		//	"full_values",
		//	&Model{
		//		Name:        types.StringValue("n"),
		//		Description: types.StringValue("d"),
		//		Input:       baseInput,
		//		Triggers: &triggersModel{
		//			Schedule: &scheduleTriggerModel{Rrule: types.StringValue("RRULE")},
		//		},
		//	},
		//	&automation.PartialUpdateVolumeAutomationPayload{
		//		Name:        new("n"),
		//		Description: new("d"),
		//		Input: &automation.VolumeAutomationInput{
		//			VolumeRecoveryPointManagementInput: &automation.VolumeRecoveryPointManagementInput{
		//				Kind: "VolumeRecoveryPointManagement",
		//				SnapshotRetentionPolicy: automation.SnapshotRetentionPolicyCountAsSnapshotRetentionPolicy(&automation.SnapshotRetentionPolicyCount{
		//					Kind:  automation.SNAPSHOTRETENTIONPOLICYCOUNTKIND_COUNT,
		//					Value: 3,
		//				}),
		//			},
		//		},
		//		Triggers: &automation.AutomationTriggers{
		//			Schedule: &automation.AutomationScheduleTrigger{Rrule: "RRULE"},
		//		},
		//	},
		//	true,
		//},
		{
			"clears_optional_string_fields",
			&Model{
				Name:        types.StringNull(),
				Description: types.StringNull(),
			},
			&automation.PartialUpdateVolumeAutomationPayload{
				//TODO: should be updated to null as soon the API spec is updated
				Name:        new(""),
				Description: new(""),
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
