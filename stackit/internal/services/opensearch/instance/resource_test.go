package opensearch

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	legacyOpensearch "github.com/stackitcloud/stackit-sdk-go/services/opensearch"
	opensearch "github.com/stackitcloud/stackit-sdk-go/services/opensearch/v1api"
)

var fixtureModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                types.StringValue("acl"),
	"enable_monitoring":      types.BoolValue(true),
	"graphite":               types.StringValue("graphite"),
	"java_garbage_collector": types.StringValue(string(legacyOpensearch.INSTANCEPARAMETERSJAVA_GARBAGE_COLLECTOR_USE_G1_GC)),
	"java_heapspace":         types.Int32Value(10),
	"java_maxmetaspace":      types.Int32Value(10),
	"max_disk_threshold":     types.Int32Value(10),
	"metrics_frequency":      types.Int32Value(10),
	"metrics_prefix":         types.StringValue("prefix"),
	"monitoring_instance_id": types.StringValue("mid"),
	"plugins": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("plugin"),
		types.StringValue("plugin2"),
	}),
	"syslog": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("syslog"),
		types.StringValue("syslog2"),
	}),
	"tls_ciphers": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("cipher"),
		types.StringValue("cipher2"),
	}),
	"tls_protocols": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("TLSv1.2"),
		types.StringValue("TLSv1.3"),
	}),
})

var fixtureNullModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                types.StringNull(),
	"enable_monitoring":      types.BoolNull(),
	"graphite":               types.StringNull(),
	"java_garbage_collector": types.StringNull(),
	"java_heapspace":         types.Int32Null(),
	"java_maxmetaspace":      types.Int32Null(),
	"max_disk_threshold":     types.Int32Null(),
	"metrics_frequency":      types.Int32Null(),
	"metrics_prefix":         types.StringNull(),
	"monitoring_instance_id": types.StringNull(),
	"plugins":                types.ListNull(types.StringType),
	"syslog":                 types.ListNull(types.StringType),
	"tls_ciphers":            types.ListNull(types.StringType),
	"tls_protocols":          types.ListNull(types.StringType),
})

var fixtureInstanceParameters = opensearch.InstanceParameters{
	SgwAcl:               new("acl"),
	EnableMonitoring:     new(true),
	Graphite:             new("graphite"),
	JavaGarbageCollector: new(string(legacyOpensearch.INSTANCEPARAMETERSJAVA_GARBAGE_COLLECTOR_USE_G1_GC)),
	JavaHeapspace:        new(int32(10)),
	JavaMaxmetaspace:     new(int32(10)),
	MaxDiskThreshold:     new(int32(10)),
	MetricsFrequency:     new(int32(10)),
	MetricsPrefix:        new("prefix"),
	MonitoringInstanceId: new("mid"),
	Plugins:              []string{"plugin", "plugin2"},
	Syslog:               []string{"syslog", "syslog2"},
	TlsCiphers:           []string{"cipher", "cipher2"},
	TlsProtocols:         []string{"TLSv1.2", "TLSv1.3"},
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *opensearch.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&opensearch.Instance{},
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
			&opensearch.Instance{
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
					"sgw_acl":                "acl",
					"enable_monitoring":      true,
					"graphite":               "graphite",
					"java_garbage_collector": string(legacyOpensearch.INSTANCEPARAMETERSJAVA_GARBAGE_COLLECTOR_USE_G1_GC),
					"java_heapspace":         int32(10),
					"java_maxmetaspace":      int32(10),
					"max_disk_threshold":     int32(10),
					"metrics_frequency":      int32(10),
					"metrics_prefix":         "prefix",
					"monitoring_instance_id": "mid",
					"plugins":                []string{"plugin", "plugin2"},
					"syslog":                 []string{"syslog", "syslog2"},
					"tls-ciphers":            []string{"cipher", "cipher2"},
					"tls-protocols":          []string{"TLSv1.2", "TLSv1.3"},
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
			&opensearch.Instance{},
			Model{},
			false,
		},
		{
			"wrong_param_types_1",
			&opensearch.Instance{
				Parameters: map[string]any{
					"sgw_acl": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&opensearch.Instance{
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
		expected    *opensearch.CreateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&opensearch.CreateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:       types.StringValue("name"),
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&opensearch.CreateInstancePayload{
				InstanceName: "name",
				Parameters:   &fixtureInstanceParameters,
				PlanId:       "plan",
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
			&opensearch.CreateInstancePayload{
				InstanceName: "",
				Parameters:   &opensearch.InstanceParameters{},
				PlanId:       "",
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
			&opensearch.CreateInstancePayload{
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
		expected    *opensearch.PartialUpdateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&opensearch.PartialUpdateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&opensearch.PartialUpdateInstancePayload{
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
			&opensearch.PartialUpdateInstancePayload{
				Parameters: &opensearch.InstanceParameters{},
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
			&opensearch.PartialUpdateInstancePayload{
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
