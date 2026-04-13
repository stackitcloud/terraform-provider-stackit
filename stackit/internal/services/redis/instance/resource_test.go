package redis

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	redis "github.com/stackitcloud/stackit-sdk-go/services/redis/v1api"
)

var fixtureModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                 types.StringValue("acl"),
	"down_after_milliseconds": types.Int32Value(10),
	"enable_monitoring":       types.BoolValue(true),
	"failover_timeout":        types.Int32Value(10),
	"graphite":                types.StringValue("1.1.1.1:91"),
	"lazyfree_lazy_eviction":  types.StringValue("no"),
	"lazyfree_lazy_expire":    types.StringValue("no"),
	"lua_time_limit":          types.Int32Value(10),
	"max_disk_threshold":      types.Int32Value(100),
	"maxclients":              types.Int32Value(10),
	"maxmemory_policy":        types.StringValue("volatile-lru"),
	"maxmemory_samples":       types.Int32Value(10),
	"metrics_frequency":       types.Int32Value(10),
	"metrics_prefix":          types.StringValue("prefix"),
	"min_replicas_max_lag":    types.Int32Value(10),
	"monitoring_instance_id":  types.StringValue("mid"),
	"notify_keyspace_events":  types.StringValue("events"),
	"snapshot":                types.StringValue("snapshot"),
	"syslog": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("syslog"),
		types.StringValue("syslog2"),
	}),
	"tls_ciphers": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("ciphers1"),
		types.StringValue("ciphers2"),
	}),
	"tls_ciphersuites": types.StringValue("ciphersuites"),
	"tls_protocols":    types.StringValue("TLSv1.2"),
})

var fixtureInstanceParameters = redis.InstanceParameters{
	SgwAcl:                new("acl"),
	DownAfterMilliseconds: new(int32(10)),
	EnableMonitoring:      new(true),
	FailoverTimeout:       new(int32(10)),
	Graphite:              new("1.1.1.1:91"),
	LazyfreeLazyEviction:  new("no"),
	LazyfreeLazyExpire:    new("no"),
	LuaTimeLimit:          new(int32(10)),
	MaxDiskThreshold:      new(int32(100)),
	Maxclients:            new(int32(10)),
	MaxmemoryPolicy:       new("volatile-lru"),
	MaxmemorySamples:      new(int32(10)),
	MetricsFrequency:      new(int32(10)),
	MetricsPrefix:         new("prefix"),
	MinReplicasMaxLag:     new(int32(10)),
	MonitoringInstanceId:  new("mid"),
	NotifyKeyspaceEvents:  new("events"),
	Snapshot:              new("snapshot"),
	Syslog:                []string{"syslog", "syslog2"},
	TlsCiphers:            []string{"ciphers1", "ciphers2"},
	TlsCiphersuites:       new("ciphersuites"),
	TlsProtocols:          new("TLSv1.2"),
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *redis.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&redis.Instance{},
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
			&redis.Instance{
				PlanId:             "plan",
				CfGuid:             "cf",
				CfSpaceGuid:        "space",
				DashboardUrl:       "dashboard",
				ImageUrl:           "image",
				InstanceId:         new("iid"),
				Name:               "name",
				CfOrganizationGuid: "org",
				Parameters: map[string]any{
					"sgw_acl":                 "acl",
					"down-after-milliseconds": int32(10),
					"enable_monitoring":       true,
					"failover-timeout":        int32(10),
					"graphite":                "1.1.1.1:91",
					"lazyfree-lazy-eviction":  "no",
					"lazyfree-lazy-expire":    "no",
					"lua-time-limit":          int32(10),
					"max_disk_threshold":      int32(100),
					"maxclients":              int32(10),
					"maxmemory-policy":        "volatile-lru",
					"maxmemory-samples":       int32(10),
					"metrics_frequency":       int32(10),
					"metrics_prefix":          "prefix",
					"min_replicas_max_lag":    int32(10),
					"monitoring_instance_id":  "mid",
					"notify-keyspace-events":  "events",
					"snapshot":                "snapshot",
					"syslog":                  []string{"syslog", "syslog2"},
					"tls-ciphers":             []string{"ciphers1", "ciphers2"},
					"tls-ciphersuites":        "ciphersuites",
					"tls-protocols":           "TLSv1.2",
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
			&redis.Instance{},
			Model{},
			false,
		},
		{
			"wrong_param_types_1",
			&redis.Instance{
				Parameters: map[string]any{
					"sgw_acl": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&redis.Instance{
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
		expected    *redis.CreateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&redis.CreateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:       types.StringValue("name"),
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&redis.CreateInstancePayload{
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
				Parameters: fixtureModelParameters,
			},
			&redis.CreateInstancePayload{
				InstanceName: "",
				Parameters:   &fixtureInstanceParameters,
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
			&redis.CreateInstancePayload{
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
		expected    *redis.PartialUpdateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&redis.PartialUpdateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&redis.PartialUpdateInstancePayload{
				Parameters: &fixtureInstanceParameters,
				PlanId:     new("plan"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				PlanId:     types.StringValue(""),
				Parameters: fixtureModelParameters,
			},
			&redis.PartialUpdateInstancePayload{
				Parameters: &fixtureInstanceParameters,
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
			&redis.PartialUpdateInstancePayload{
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
