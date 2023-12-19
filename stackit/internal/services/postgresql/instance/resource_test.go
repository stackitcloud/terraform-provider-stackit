package postgresql

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresql"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *postgresql.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&postgresql.Instance{},
			Model{
				Id:                 types.StringValue("pid,iid"),
				InstanceId:         types.StringValue("iid"),
				ProjectId:          types.StringValue("pid"),
				PlanId:             types.StringNull(),
				Name:               types.StringNull(),
				CfGuid:             types.StringNull(),
				CfSpaceGuid:        types.StringNull(),
				DashboardUrl:       types.StringNull(),
				ImageUrl:           types.StringNull(),
				CfOrganizationGuid: types.StringNull(),
				Parameters:         types.ObjectNull(parametersTypes),
			},
			true,
		},
		{
			"simple_values",
			&postgresql.Instance{
				PlanId:             utils.Ptr("plan"),
				CfGuid:             utils.Ptr("cf"),
				CfSpaceGuid:        utils.Ptr("space"),
				DashboardUrl:       utils.Ptr("dashboard"),
				ImageUrl:           utils.Ptr("image"),
				InstanceId:         utils.Ptr("iid"),
				Name:               utils.Ptr("name"),
				CfOrganizationGuid: utils.Ptr("org"),
				Parameters: &map[string]interface{}{
					"enable_monitoring": true,
					"metrics_frequency": 1234,
					"plugins": []string{
						"plugin_1",
						"plugin_2",
						"",
					},
				},
			},
			Model{
				Id:                 types.StringValue("pid,iid"),
				InstanceId:         types.StringValue("iid"),
				ProjectId:          types.StringValue("pid"),
				PlanId:             types.StringValue("plan"),
				Name:               types.StringValue("name"),
				CfGuid:             types.StringValue("cf"),
				CfSpaceGuid:        types.StringValue("space"),
				DashboardUrl:       types.StringValue("dashboard"),
				ImageUrl:           types.StringValue("image"),
				CfOrganizationGuid: types.StringValue("org"),
				Parameters: types.ObjectValueMust(parametersTypes, map[string]attr.Value{
					"enable_monitoring":      types.BoolValue(true),
					"metrics_frequency":      types.Int64Value(1234),
					"metrics_prefix":         types.StringNull(),
					"monitoring_instance_id": types.StringNull(),
					"plugins": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("plugin_1"),
						types.StringValue("plugin_2"),
						types.StringValue(""),
					}),
					"sgw_acl": types.StringNull(),
				}),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&postgresql.Instance{},
			Model{},
			false,
		},
		{
			"wrong_param_types_1",
			&postgresql.Instance{
				Parameters: &map[string]interface{}{
					"enable_monitoring": "true",
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&postgresql.Instance{
				Parameters: &map[string]interface{}{
					"metrics_frequency": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_3",
			&postgresql.Instance{
				Parameters: &map[string]interface{}{
					"metrics_frequency": 12.34,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_4",
			&postgresql.Instance{
				Parameters: &map[string]interface{}{
					"plugins": "foo",
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_5",
			&postgresql.Instance{
				Parameters: &map[string]interface{}{
					"plugins": []bool{
						true,
					},
				},
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(tt.input, state)
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
		description            string
		input                  *Model
		inputParameters        *parametersModel
		inputParametersPlugins *[]string
		expected               *postgresql.CreateInstancePayload
		isValid                bool
	}{
		{
			"default_values",
			&Model{},
			&parametersModel{},
			&[]string{},
			&postgresql.CreateInstancePayload{
				Parameters: &postgresql.InstanceParameters{
					Plugins: &[]string{},
				},
			},
			true,
		},
		{
			"nil_values",
			&Model{},
			&parametersModel{},
			nil,
			&postgresql.CreateInstancePayload{
				Parameters: &postgresql.InstanceParameters{
					Plugins: nil,
				},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:   types.StringValue("name"),
				PlanId: types.StringValue("plan"),
			},
			&parametersModel{
				EnableMonitoring:     types.BoolValue(true),
				MetricsFrequency:     types.Int64Value(123),
				MetricsPrefix:        types.StringValue("prefix"),
				MonitoringInstanceId: types.StringValue("monitoring"),
				SgwAcl:               types.StringValue("sgw"),
			},
			&[]string{
				"plugin_1",
				"plugin_2",
			},
			&postgresql.CreateInstancePayload{
				InstanceName: utils.Ptr("name"),
				Parameters: &postgresql.InstanceParameters{
					EnableMonitoring:     utils.Ptr(true),
					MetricsFrequency:     utils.Ptr(int64(123)),
					MetricsPrefix:        utils.Ptr("prefix"),
					MonitoringInstanceId: utils.Ptr("monitoring"),
					Plugins: &[]string{
						"plugin_1",
						"plugin_2",
					},
					SgwAcl: utils.Ptr("sgw"),
				},
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name:   types.StringValue(""),
				PlanId: types.StringValue(""),
			},
			&parametersModel{
				EnableMonitoring:     types.BoolNull(),
				MetricsFrequency:     types.Int64Value(2123456789),
				MetricsPrefix:        types.StringNull(),
				MonitoringInstanceId: types.StringNull(),
				SgwAcl:               types.StringNull(),
			},
			&[]string{
				"",
			},
			&postgresql.CreateInstancePayload{
				InstanceName: utils.Ptr(""),
				Parameters: &postgresql.InstanceParameters{
					EnableMonitoring:     nil,
					MetricsFrequency:     utils.Ptr(int64(2123456789)),
					MetricsPrefix:        nil,
					MonitoringInstanceId: nil,
					Plugins: &[]string{
						"",
					},
					SgwAcl: nil,
				},
				PlanId: utils.Ptr(""),
			},
			true,
		},
		{
			"nil_model",
			nil,
			&parametersModel{},
			&[]string{},
			nil,
			false,
		},
		{
			"nil_parameters",
			&Model{
				Name:   types.StringValue("name"),
				PlanId: types.StringValue("plan"),
			},
			nil,
			nil,
			&postgresql.CreateInstancePayload{
				InstanceName: utils.Ptr("name"),
				PlanId:       utils.Ptr("plan"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputParameters, tt.inputParametersPlugins)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description            string
		input                  *Model
		inputParameters        *parametersModel
		inputParametersPlugins *[]string
		expected               *postgresql.PartialUpdateInstancePayload
		isValid                bool
	}{
		{
			"default_values",
			&Model{},
			&parametersModel{},
			&[]string{},
			&postgresql.PartialUpdateInstancePayload{
				Parameters: &postgresql.InstanceParameters{
					Plugins: &[]string{},
				},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				PlanId: types.StringValue("plan"),
			},
			&parametersModel{
				EnableMonitoring:     types.BoolValue(true),
				MetricsFrequency:     types.Int64Value(123),
				MetricsPrefix:        types.StringValue("prefix"),
				MonitoringInstanceId: types.StringValue("monitoring"),
				SgwAcl:               types.StringValue("sgw"),
			},
			&[]string{
				"plugin_1",
				"plugin_2",
			},
			&postgresql.PartialUpdateInstancePayload{
				Parameters: &postgresql.InstanceParameters{
					EnableMonitoring:     utils.Ptr(true),
					MetricsFrequency:     utils.Ptr(int64(123)),
					MetricsPrefix:        utils.Ptr("prefix"),
					MonitoringInstanceId: utils.Ptr("monitoring"),
					Plugins: &[]string{
						"plugin_1",
						"plugin_2",
					},
					SgwAcl: utils.Ptr("sgw"),
				},
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				PlanId: types.StringValue(""),
			},
			&parametersModel{
				EnableMonitoring:     types.BoolNull(),
				MetricsFrequency:     types.Int64Value(2123456789),
				MetricsPrefix:        types.StringNull(),
				MonitoringInstanceId: types.StringNull(),
				SgwAcl:               types.StringNull(),
			},
			&[]string{
				"",
			},
			&postgresql.PartialUpdateInstancePayload{
				Parameters: &postgresql.InstanceParameters{
					EnableMonitoring:     nil,
					MetricsFrequency:     utils.Ptr(int64(2123456789)),
					MetricsPrefix:        nil,
					MonitoringInstanceId: nil,
					Plugins: &[]string{
						"",
					},
					SgwAcl: nil,
				},
				PlanId: utils.Ptr(""),
			},
			true,
		},
		{
			"nil_model",
			nil,
			&parametersModel{},
			&[]string{},
			nil,
			false,
		},
		{
			"nil_parameters",
			&Model{
				PlanId: types.StringValue("plan"),
			},
			nil,
			nil,
			&postgresql.PartialUpdateInstancePayload{
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input, tt.inputParameters, tt.inputParametersPlugins)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
