package logme

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	logmeSdk "github.com/stackitcloud/stackit-sdk-go/services/logme/v1api"
)

var fixtureModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                 types.StringValue("acl"),
	"enable_monitoring":       types.BoolValue(true),
	"fluentd_tcp":             types.Int32Value(10),
	"fluentd_tls":             types.Int32Value(10),
	"fluentd_tls_ciphers":     types.StringValue("ciphers"),
	"fluentd_tls_max_version": types.StringValue("max_version"),
	"fluentd_tls_min_version": types.StringValue("min_version"),
	"fluentd_tls_version":     types.StringValue("version"),
	"fluentd_udp":             types.Int32Value(10),
	"graphite":                types.StringValue("graphite"),
	"ism_deletion_after":      types.StringValue("deletion_after"),
	"ism_jitter":              types.Float32Value(10.1),
	"ism_job_interval":        types.Int32Value(10),
	"java_heapspace":          types.Int32Value(10),
	"java_maxmetaspace":       types.Int32Value(10),
	"max_disk_threshold":      types.Int32Value(10),
	"metrics_frequency":       types.Int32Value(10),
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
})

var fixtureNullModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                  types.StringNull(),
	"enable_monitoring":        types.BoolNull(),
	"fluentd_tcp":              types.Int32Null(),
	"fluentd_tls":              types.Int32Null(),
	"fluentd_tls_ciphers":      types.StringNull(),
	"fluentd_tls_max_version":  types.StringNull(),
	"fluentd_tls_min_version":  types.StringNull(),
	"fluentd_tls_version":      types.StringNull(),
	"fluentd_udp":              types.Int32Null(),
	"graphite":                 types.StringNull(),
	"ism_deletion_after":       types.StringNull(),
	"ism_jitter":               types.Float32Null(),
	"ism_job_interval":         types.Int32Null(),
	"java_heapspace":           types.Int32Null(),
	"java_maxmetaspace":        types.Int32Null(),
	"max_disk_threshold":       types.Int32Null(),
	"metrics_frequency":        types.Int32Null(),
	"metrics_prefix":           types.StringNull(),
	"monitoring_instance_id":   types.StringNull(),
	"opensearch_tls_ciphers":   types.ListNull(types.StringType),
	"opensearch_tls_protocols": types.ListNull(types.StringType),
	"syslog":                   types.ListNull(types.StringType),
})

var fixtureInstanceParameters = logmeSdk.InstanceParameters{
	SgwAcl:                 new("acl"),
	EnableMonitoring:       new(true),
	FluentdTcp:             new(int32(10)),
	FluentdTls:             new(int32(10)),
	FluentdTlsCiphers:      new("ciphers"),
	FluentdTlsMaxVersion:   new("max_version"),
	FluentdTlsMinVersion:   new("min_version"),
	FluentdTlsVersion:      new("version"),
	FluentdUdp:             new(int32(10)),
	Graphite:               new("graphite"),
	IsmDeletionAfter:       new("deletion_after"),
	IsmJitter:              new(float32(10.1)),
	IsmJobInterval:         new(int32(10)),
	JavaHeapspace:          new(int32(10)),
	JavaMaxmetaspace:       new(int32(10)),
	MaxDiskThreshold:       new(int32(10)),
	MetricsFrequency:       new(int32(10)),
	MetricsPrefix:          new("prefix"),
	MonitoringInstanceId:   new("mid"),
	OpensearchTlsCiphers:   []string{"ciphers", "ciphers2"},
	OpensearchTlsProtocols: []string{"protocols", "protocols2"},
	Syslog:                 []string{"syslog", "syslog2"},
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *logmeSdk.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&logmeSdk.Instance{},
			Model{
				Id:                 types.StringValue("pid,iid"),
				InstanceId:         types.StringValue("iid"),
				ProjectId:          types.StringValue("pid"),
				PlanId:             types.StringValue(""),
				Name:               types.StringValue(""),
				CfGuid:             types.StringValue(""),
				CfSpaceGuid:        types.StringValue(""),
				DashboardUrl:       types.StringValue(""),
				ImageUrl:           types.StringValue(""),
				CfOrganizationGuid: types.StringValue(""),
				Parameters:         types.ObjectNull(parametersTypes),
			},
			true,
		},

		{
			"simple_values",
			&logmeSdk.Instance{
				PlanId:             "plan",
				CfGuid:             "cf",
				CfSpaceGuid:        "space",
				DashboardUrl:       "dashboard",
				ImageUrl:           "image",
				InstanceId:         new("iid"),
				Name:               "name",
				CfOrganizationGuid: "org",
				Parameters: map[string]any{
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
			&logmeSdk.Instance{},
			Model{},
			false,
		},
		{
			"wrong_param_types_1",
			&logmeSdk.Instance{
				Parameters: map[string]any{
					"sgw_acl": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&logmeSdk.Instance{
				Parameters: map[string]any{
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
		expected    *logmeSdk.CreateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&logmeSdk.CreateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:       types.StringValue("name"),
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&logmeSdk.CreateInstancePayload{
				InstanceName: "name",
				PlanId:       "plan",
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
			&logmeSdk.CreateInstancePayload{
				InstanceName: "",
				PlanId:       "",
				Parameters:   &logmeSdk.InstanceParameters{},
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
			&logmeSdk.CreateInstancePayload{
				InstanceName: "name",
				PlanId:       "plan",
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
		expected    *logmeSdk.PartialUpdateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&logmeSdk.PartialUpdateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&logmeSdk.PartialUpdateInstancePayload{
				Parameters: &fixtureInstanceParameters,
				PlanId:     new("plan"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				PlanId:     types.StringValue(""),
				Parameters: fixtureNullModelParameters,
			},
			&logmeSdk.PartialUpdateInstancePayload{
				Parameters: &logmeSdk.InstanceParameters{},
				PlanId:     new(""),
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
			&logmeSdk.PartialUpdateInstancePayload{
				PlanId: new("plan"),
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
