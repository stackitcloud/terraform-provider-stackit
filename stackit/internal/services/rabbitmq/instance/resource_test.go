package rabbitmq

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *rabbitmq.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&rabbitmq.Instance{},
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
			&rabbitmq.Instance{
				PlanId:             utils.Ptr("plan"),
				CfGuid:             utils.Ptr("cf"),
				CfSpaceGuid:        utils.Ptr("space"),
				DashboardUrl:       utils.Ptr("dashboard"),
				ImageUrl:           utils.Ptr("image"),
				InstanceId:         utils.Ptr("iid"),
				Name:               utils.Ptr("name"),
				CfOrganizationGuid: utils.Ptr("org"),
				Parameters: &map[string]interface{}{
					"sgw_acl": "acl",
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
					"sgw_acl":                types.StringValue("acl"),
					"consumer_timeout":       types.Int64Null(),
					"enable_monitoring":      types.BoolNull(),
					"graphite":               types.StringNull(),
					"max_disk_threshold":     types.Int64Null(),
					"metrics_frequency":      types.Int64Null(),
					"metrics_prefix":         types.StringNull(),
					"monitoring_instance_id": types.StringNull(),
					"plugins":                types.ListNull(types.StringType),
					"roles":                  types.ListNull(types.StringType),
					"syslog":                 types.ListNull(types.StringType),
					"tls_ciphers":            types.ListNull(types.StringType),
					"tls_protocols":          types.StringNull(),
				}),
			},
			true,
		},
		{
			"simple_values_params",
			&rabbitmq.Instance{
				PlanId:             utils.Ptr("plan"),
				CfGuid:             utils.Ptr("cf"),
				CfSpaceGuid:        utils.Ptr("space"),
				DashboardUrl:       utils.Ptr("dashboard"),
				ImageUrl:           utils.Ptr("image"),
				InstanceId:         utils.Ptr("iid"),
				Name:               utils.Ptr("name"),
				CfOrganizationGuid: utils.Ptr("org"),
				Parameters: &map[string]interface{}{
					"sgw_acl":                "acl",
					"consumer_timeout":       10,
					"enable_monitoring":      true,
					"graphite":               "1.1.1.1:91",
					"max_disk_threshold":     100,
					"metrics_frequency":      10,
					"metrics_prefix":         "prefix",
					"monitoring_instance_id": "mid",
					"plugins":                []string{"plugin1", "plugin2"},
					"roles":                  []string{"role1", "role2"},
					"syslog":                 []string{"syslog", "syslog2"},
					"tls_ciphers":            []string{"ciphers1", "ciphers2"},
					"tls_protocols":          "protocol1",
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
					"sgw_acl":                types.StringValue("acl"),
					"consumer_timeout":       types.Int64Value(10),
					"enable_monitoring":      types.BoolValue(true),
					"graphite":               types.StringValue("1.1.1.1:91"),
					"max_disk_threshold":     types.Int64Value(100),
					"metrics_frequency":      types.Int64Value(10),
					"metrics_prefix":         types.StringValue("prefix"),
					"monitoring_instance_id": types.StringValue("mid"),
					"plugins": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("plugin1"),
						types.StringValue("plugin2"),
					}),
					"roles": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("role1"),
						types.StringValue("role2"),
					}),
					"syslog": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("syslog"),
						types.StringValue("syslog2"),
					}),
					"tls_ciphers": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("ciphers1"),
						types.StringValue("ciphers2"),
					}),
					"tls_protocols": types.StringValue("protocol1"),
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
			&rabbitmq.Instance{},
			Model{},
			false,
		},
		{
			"wrong_param_types_1",
			&rabbitmq.Instance{
				Parameters: &map[string]interface{}{
					"sgw_acl": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&rabbitmq.Instance{
				Parameters: &map[string]interface{}{
					"sgw_acl": 1,
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
		description     string
		input           *Model
		inputParameters *parametersModel
		expected        *rabbitmq.CreateInstancePayload
		isValid         bool
	}{
		{
			"default_values",
			&Model{},
			&parametersModel{},
			&rabbitmq.CreateInstancePayload{
				Parameters: &rabbitmq.InstanceParameters{},
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
				SgwAcl: types.StringValue("sgw"),
			},
			&rabbitmq.CreateInstancePayload{
				InstanceName: utils.Ptr("name"),
				Parameters: &rabbitmq.InstanceParameters{
					SgwAcl: utils.Ptr("sgw"),
				},
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
		{
			"simple_values_params",
			&Model{
				Name:   types.StringValue("name"),
				PlanId: types.StringValue("plan"),
			},
			&parametersModel{
				SgwAcl:               types.StringValue("sgw"),
				ConsumerTimeout:      types.Int64Value(10),
				EnableMonitoring:     types.BoolValue(true),
				Graphite:             types.StringValue("1.1.1.1:91"),
				MaxDiskThreshold:     types.Int64Value(100),
				MetricsFrequency:     types.Int64Value(10),
				MetricsPrefix:        types.StringValue("prefix"),
				MonitoringInstanceId: types.StringValue("mid"),
				Plugins: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("plugin1"),
					types.StringValue("plugin2"),
				}),
				Roles: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("role1"),
					types.StringValue("role2"),
				}),
				Syslog: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("syslog"),
					types.StringValue("syslog2"),
				}),
				TlsCiphers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ciphers1"),
					types.StringValue("ciphers2"),
				}),
				TlsProtocols: types.StringValue("protocol1"),
			},
			&rabbitmq.CreateInstancePayload{
				InstanceName: utils.Ptr("name"),
				Parameters: &rabbitmq.InstanceParameters{
					SgwAcl:               utils.Ptr("sgw"),
					ConsumerTimeout:      utils.Ptr(int64(10)),
					EnableMonitoring:     utils.Ptr(true),
					Graphite:             utils.Ptr("1.1.1.1:91"),
					MaxDiskThreshold:     utils.Ptr(int64(100)),
					MetricsFrequency:     utils.Ptr(int64(10)),
					MetricsPrefix:        utils.Ptr("prefix"),
					MonitoringInstanceId: utils.Ptr("mid"),
					Plugins:              &[]string{"plugin1", "plugin2"},
					Roles:                &[]string{"role1", "role2"},
					Syslog:               &[]string{"syslog", "syslog2"},
					TlsCiphers:           &[]string{"ciphers1", "ciphers2"},
					TlsProtocols:         utils.Ptr("protocol1"),
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
				SgwAcl: types.StringNull(),
			},
			&rabbitmq.CreateInstancePayload{
				InstanceName: utils.Ptr(""),
				Parameters: &rabbitmq.InstanceParameters{
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
			&rabbitmq.CreateInstancePayload{
				InstanceName: utils.Ptr("name"),
				PlanId:       utils.Ptr("plan"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputParameters)
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
		description     string
		input           *Model
		inputParameters *parametersModel
		expected        *rabbitmq.PartialUpdateInstancePayload
		isValid         bool
	}{
		{
			"default_values",
			&Model{},
			&parametersModel{},
			&rabbitmq.PartialUpdateInstancePayload{
				Parameters: &rabbitmq.InstanceParameters{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				PlanId: types.StringValue("plan"),
			},
			&parametersModel{
				SgwAcl: types.StringValue("sgw"),
			},
			&rabbitmq.PartialUpdateInstancePayload{
				Parameters: &rabbitmq.InstanceParameters{
					SgwAcl: utils.Ptr("sgw"),
				},
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
		{
			"simple_values_params",
			&Model{
				Name:   types.StringValue("name"),
				PlanId: types.StringValue("plan"),
			},
			&parametersModel{
				SgwAcl:               types.StringValue("sgw"),
				ConsumerTimeout:      types.Int64Value(10),
				EnableMonitoring:     types.BoolValue(true),
				Graphite:             types.StringValue("1.1.1.1:91"),
				MaxDiskThreshold:     types.Int64Value(100),
				MetricsFrequency:     types.Int64Value(10),
				MetricsPrefix:        types.StringValue("prefix"),
				MonitoringInstanceId: types.StringValue("mid"),
				Plugins: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("plugin1"),
					types.StringValue("plugin2"),
				}),
				Roles: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("role1"),
					types.StringValue("role2"),
				}),
				Syslog: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("syslog"),
					types.StringValue("syslog2"),
				}),
				TlsCiphers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ciphers1"),
					types.StringValue("ciphers2"),
				}),
				TlsProtocols: types.StringValue("protocol1"),
			},
			&rabbitmq.PartialUpdateInstancePayload{
				Parameters: &rabbitmq.InstanceParameters{
					SgwAcl:               utils.Ptr("sgw"),
					ConsumerTimeout:      utils.Ptr(int64(10)),
					EnableMonitoring:     utils.Ptr(true),
					Graphite:             utils.Ptr("1.1.1.1:91"),
					MaxDiskThreshold:     utils.Ptr(int64(100)),
					MetricsFrequency:     utils.Ptr(int64(10)),
					MetricsPrefix:        utils.Ptr("prefix"),
					MonitoringInstanceId: utils.Ptr("mid"),
					Plugins:              &[]string{"plugin1", "plugin2"},
					Roles:                &[]string{"role1", "role2"},
					Syslog:               &[]string{"syslog", "syslog2"},
					TlsCiphers:           &[]string{"ciphers1", "ciphers2"},
					TlsProtocols:         utils.Ptr("protocol1"),
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
				SgwAcl: types.StringNull(),
			},
			&rabbitmq.PartialUpdateInstancePayload{
				Parameters: &rabbitmq.InstanceParameters{
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
			nil,
			false,
		},
		{
			"nil_parameters",
			&Model{
				PlanId: types.StringValue("plan"),
			},
			nil,
			&rabbitmq.PartialUpdateInstancePayload{
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input, tt.inputParameters)
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
