package logme

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/logme"
)

var fixtureModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                 types.StringValue("acl"),
	"enable_monitoring":       types.BoolValue(true),
	"fluentd_tcp":             types.Int64Value(10),
	"fluentd_tls":             types.Int64Value(10),
	"fluentd_tls_ciphers":     types.StringValue("ciphers"),
	"fluentd_tls_max_version": types.StringValue("max_version"),
	"fluentd_tls_min_version": types.StringValue("min_version"),
	"fluentd_tls_version":     types.StringValue("version"),
	"fluentd_udp":             types.Int64Value(10),
	"graphite":                types.StringValue("graphite"),
	"ism_deletion_after":      types.StringValue("deletion_after"),
	"ism_jitter":              types.Float64Value(10.1),
	"ism_job_interval":        types.Int64Value(10),
	"java_heapspace":          types.Int64Value(10),
	"java_maxmetaspace":       types.Int64Value(10),
	"max_disk_threshold":      types.Int64Value(10),
	"metrics_frequency":       types.Int64Value(10),
	"metrics_prefix":          types.StringValue("prefix"),
	"monitoring_instance_id":  types.StringValue("mid"),
	"opensearch_tls_ciphers": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("ciphers"),
		types.StringValue("ciphers2"),
	}),
	"opensearch_tls_protocols": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("protocols"),
		types.StringValue("protocols2"),
	}),
	"syslog": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("syslog"),
		types.StringValue("syslog2"),
	}),
	"syslog_use_udp": types.StringValue("udp"),
})

var fixtureNullModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                  types.StringNull(),
	"enable_monitoring":        types.BoolNull(),
	"fluentd_tcp":              types.Int64Null(),
	"fluentd_tls":              types.Int64Null(),
	"fluentd_tls_ciphers":      types.StringNull(),
	"fluentd_tls_max_version":  types.StringNull(),
	"fluentd_tls_min_version":  types.StringNull(),
	"fluentd_tls_version":      types.StringNull(),
	"fluentd_udp":              types.Int64Null(),
	"graphite":                 types.StringNull(),
	"ism_deletion_after":       types.StringNull(),
	"ism_jitter":               types.Float64Null(),
	"ism_job_interval":         types.Int64Null(),
	"java_heapspace":           types.Int64Null(),
	"java_maxmetaspace":        types.Int64Null(),
	"max_disk_threshold":       types.Int64Null(),
	"metrics_frequency":        types.Int64Null(),
	"metrics_prefix":           types.StringNull(),
	"monitoring_instance_id":   types.StringNull(),
	"opensearch_tls_ciphers":   types.ListNull(types.StringType),
	"opensearch_tls_protocols": types.ListNull(types.StringType),
	"syslog":                   types.ListNull(types.StringType),
	"syslog_use_udp":           types.StringNull(),
})

var fixtureInstanceParameters = logme.InstanceParameters{
	SgwAcl:                 utils.Ptr("acl"),
	EnableMonitoring:       utils.Ptr(true),
	FluentdTcp:             utils.Ptr(int64(10)),
	FluentdTls:             utils.Ptr(int64(10)),
	FluentdTlsCiphers:      utils.Ptr("ciphers"),
	FluentdTlsMaxVersion:   utils.Ptr("max_version"),
	FluentdTlsMinVersion:   utils.Ptr("min_version"),
	FluentdTlsVersion:      utils.Ptr("version"),
	FluentdUdp:             utils.Ptr(int64(10)),
	Graphite:               utils.Ptr("graphite"),
	IsmDeletionAfter:       utils.Ptr("deletion_after"),
	IsmJitter:              utils.Ptr(10.1),
	IsmJobInterval:         utils.Ptr(int64(10)),
	JavaHeapspace:          utils.Ptr(int64(10)),
	JavaMaxmetaspace:       utils.Ptr(int64(10)),
	MaxDiskThreshold:       utils.Ptr(int64(10)),
	MetricsFrequency:       utils.Ptr(int64(10)),
	MetricsPrefix:          utils.Ptr("prefix"),
	MonitoringInstanceId:   utils.Ptr("mid"),
	OpensearchTlsCiphers:   &[]string{"ciphers", "ciphers2"},
	OpensearchTlsProtocols: &[]string{"protocols", "protocols2"},
	Syslog:                 &[]string{"syslog", "syslog2"},
	SyslogUseUdp:           utils.Ptr("udp"),
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *logme.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&logme.Instance{},
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
			&logme.Instance{
				PlanId:             utils.Ptr("plan"),
				CfGuid:             utils.Ptr("cf"),
				CfSpaceGuid:        utils.Ptr("space"),
				DashboardUrl:       utils.Ptr("dashboard"),
				ImageUrl:           utils.Ptr("image"),
				InstanceId:         utils.Ptr("iid"),
				Name:               utils.Ptr("name"),
				CfOrganizationGuid: utils.Ptr("org"),
				Parameters: &map[string]interface{}{
					// Using "-" on purpose on some fields because that is the API response
					"sgw_acl":                  "acl",
					"enable_monitoring":        true,
					"fluentd-tcp":              10,
					"fluentd-tls":              10,
					"fluentd-tls-ciphers":      "ciphers",
					"fluentd-tls-max-version":  "max_version",
					"fluentd-tls-min-version":  "min_version",
					"fluentd-tls-version":      "version",
					"fluentd-udp":              10,
					"graphite":                 "graphite",
					"ism_deletion_after":       "deletion_after",
					"ism_jitter":               10.1,
					"ism_job_interval":         10,
					"java_heapspace":           10,
					"java_maxmetaspace":        10,
					"max_disk_threshold":       10,
					"metrics_frequency":        10,
					"metrics_prefix":           "prefix",
					"monitoring_instance_id":   "mid",
					"opensearch-tls-ciphers":   []string{"ciphers", "ciphers2"},
					"opensearch-tls-protocols": []string{"protocols", "protocols2"},
					"syslog":                   []string{"syslog", "syslog2"},
					"syslog-use-udp":           "udp",
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
				Parameters:         fixtureModelParameters,
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
			&logme.Instance{},
			Model{},
			false,
		},
		{
			"wrong_param_types_1",
			&logme.Instance{
				Parameters: &map[string]interface{}{
					"sgw_acl": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&logme.Instance{
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
		description string
		input       *Model
		expected    *logme.CreateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&logme.CreateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:       types.StringValue("name"),
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&logme.CreateInstancePayload{
				InstanceName: utils.Ptr("name"),
				PlanId:       utils.Ptr("plan"),
				Parameters:   &fixtureInstanceParameters,
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name:       types.StringValue(""),
				PlanId:     types.StringValue(""),
				Parameters: fixtureNullModelParameters,
			},
			&logme.CreateInstancePayload{
				InstanceName: utils.Ptr(""),
				PlanId:       utils.Ptr(""),
				Parameters:   &logme.InstanceParameters{},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
		{
			"nil_parameters",
			&Model{
				Name:   types.StringValue("name"),
				PlanId: types.StringValue("plan"),
			},
			&logme.CreateInstancePayload{
				InstanceName: utils.Ptr("name"),
				PlanId:       utils.Ptr("plan"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var parameters *parametersModel
			if tt.input != nil {
				if !(tt.input.Parameters.IsNull() || tt.input.Parameters.IsUnknown()) {
					parameters = &parametersModel{}
					diags := tt.input.Parameters.As(context.Background(), parameters, basetypes.ObjectAsOptions{})
					if diags.HasError() {
						t.Fatalf("Error converting parameters: %v", diags.Errors())
					}
				}
			}
			output, err := toCreatePayload(tt.input, parameters)
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
		description string
		input       *Model
		expected    *logme.PartialUpdateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&logme.PartialUpdateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&logme.PartialUpdateInstancePayload{
				Parameters: &fixtureInstanceParameters,
				PlanId:     utils.Ptr("plan"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				PlanId:     types.StringValue(""),
				Parameters: fixtureNullModelParameters,
			},
			&logme.PartialUpdateInstancePayload{
				Parameters: &logme.InstanceParameters{},
				PlanId:     utils.Ptr(""),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
		{
			"nil_parameters",
			&Model{
				PlanId: types.StringValue("plan"),
			},
			&logme.PartialUpdateInstancePayload{
				PlanId: utils.Ptr("plan"),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var parameters *parametersModel
			if tt.input != nil {
				if !(tt.input.Parameters.IsNull() || tt.input.Parameters.IsUnknown()) {
					parameters = &parametersModel{}
					diags := tt.input.Parameters.As(context.Background(), parameters, basetypes.ObjectAsOptions{})
					if diags.HasError() {
						t.Fatalf("Error converting parameters: %v", diags.Errors())
					}
				}
			}
			output, err := toUpdatePayload(tt.input, parameters)
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
