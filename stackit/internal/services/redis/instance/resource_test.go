package redis

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/redis"
)

var fixtureModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                 types.StringValue("acl"),
	"down_after_milliseconds": types.Int64Value(10),
	"enable_monitoring":       types.BoolValue(true),
	"failover_timeout":        types.Int64Value(10),
	"graphite":                types.StringValue("1.1.1.1:91"),
	"lazyfree_lazy_eviction":  types.StringValue("lazy_eviction"),
	"lazyfree_lazy_expire":    types.StringValue("lazy_expire"),
	"lua_time_limit":          types.Int64Value(10),
	"max_disk_threshold":      types.Int64Value(100),
	"maxclients":              types.Int64Value(10),
	"maxmemory_policy":        types.StringValue("policy"),
	"maxmemory_samples":       types.Int64Value(10),
	"metrics_frequency":       types.Int64Value(10),
	"metrics_prefix":          types.StringValue("prefix"),
	"min_replicas_max_lag":    types.Int64Value(10),
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
	"tls_protocols":    types.StringValue("protocol1"),
})

var fixtureInstanceParameters = redis.InstanceParameters{
	SgwAcl:                utils.Ptr("acl"),
	DownAfterMilliseconds: utils.Ptr(int64(10)),
	EnableMonitoring:      utils.Ptr(true),
	FailoverTimeout:       utils.Ptr(int64(10)),
	Graphite:              utils.Ptr("1.1.1.1:91"),
	LazyfreeLazyEviction:  utils.Ptr("lazy_eviction"),
	LazyfreeLazyExpire:    utils.Ptr("lazy_expire"),
	LuaTimeLimit:          utils.Ptr(int64(10)),
	MaxDiskThreshold:      utils.Ptr(int64(100)),
	Maxclients:            utils.Ptr(int64(10)),
	MaxmemoryPolicy:       utils.Ptr("policy"),
	MaxmemorySamples:      utils.Ptr(int64(10)),
	MetricsFrequency:      utils.Ptr(int64(10)),
	MetricsPrefix:         utils.Ptr("prefix"),
	MinReplicasMaxLag:     utils.Ptr(int64(10)),
	MonitoringInstanceId:  utils.Ptr("mid"),
	NotifyKeyspaceEvents:  utils.Ptr("events"),
	Snapshot:              utils.Ptr("snapshot"),
	Syslog:                &[]string{"syslog", "syslog2"},
	TlsCiphers:            &[]string{"ciphers1", "ciphers2"},
	TlsCiphersuites:       utils.Ptr("ciphersuites"),
	TlsProtocols:          utils.Ptr("protocol1"),
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
			&redis.Instance{
				PlanId:             utils.Ptr("plan"),
				CfGuid:             utils.Ptr("cf"),
				CfSpaceGuid:        utils.Ptr("space"),
				DashboardUrl:       utils.Ptr("dashboard"),
				ImageUrl:           utils.Ptr("image"),
				InstanceId:         utils.Ptr("iid"),
				Name:               utils.Ptr("name"),
				CfOrganizationGuid: utils.Ptr("org"),
				Parameters: &map[string]interface{}{
					"sgw_acl":                 "acl",
					"down_after_milliseconds": int64(10),
					"enable_monitoring":       true,
					"failover-timeout":        int64(10),
					"graphite":                "1.1.1.1:91",
					"lazyfree-lazy-eviction":  "lazy_eviction",
					"lazyfree-lazy-expire":    "lazy_expire",
					"lua-time-limit":          int64(10),
					"max_disk_threshold":      int64(100),
					"maxclients":              int64(10),
					"maxmemory-policy":        "policy",
					"maxmemory-samples":       int64(10),
					"metrics_frequency":       int64(10),
					"metrics_prefix":          "prefix",
					"min_replicas_max_lag":    int64(10),
					"monitoring_instance_id":  "mid",
					"notify-keyspace-events":  "events",
					"snapshot":                "snapshot",
					"syslog":                  []string{"syslog", "syslog2"},
					"tls-ciphers":             []string{"ciphers1", "ciphers2"},
					"tls-ciphersuites":        "ciphersuites",
					"tls-protocols":           "protocol1",
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
				Parameters: &map[string]interface{}{
					"sgw_acl": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&redis.Instance{
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
		expected    *redis.CreateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&redis.CreateInstancePayload{
				Parameters: &redis.InstanceParameters{},
			},
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
				InstanceName: utils.Ptr("name"),
				Parameters:   &fixtureInstanceParameters,
				PlanId:       utils.Ptr("plan"),
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
				InstanceName: utils.Ptr(""),
				Parameters:   &fixtureInstanceParameters,
				PlanId:       utils.Ptr(""),
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
				InstanceName: utils.Ptr("name"),
				PlanId:       utils.Ptr("plan"),
				Parameters:   &redis.InstanceParameters{},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var parameters = &parametersModel{}
			if tt.input != nil {
				if !(tt.input.Parameters.IsNull() || tt.input.Parameters.IsUnknown()) {
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
			&redis.PartialUpdateInstancePayload{
				Parameters: &redis.InstanceParameters{},
			},
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
				PlanId:     utils.Ptr("plan"),
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
			&redis.PartialUpdateInstancePayload{
				PlanId:     utils.Ptr("plan"),
				Parameters: &redis.InstanceParameters{},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var parameters = &parametersModel{}
			if tt.input != nil {
				if !(tt.input.Parameters.IsNull() || tt.input.Parameters.IsUnknown()) {
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
