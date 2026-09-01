package valkey

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	valkey "github.com/stackitcloud/stackit-sdk-go/services/valkey/v2api"
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
	"min_replicas_to_write":   types.Int32Value(1),
	"monitoring_instance_id":  types.StringValue("mid"),
	"notify_keyspace_events":  types.StringValue("events"),
	"repl_backlog_size":       types.StringValue("1mb"),
	"snapshot":                types.StringValue("snapshot"),
	"syslog": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("syslog"),
		types.StringValue("syslog2"),
	}),
})

var fixtureInstanceParameters = valkey.InstanceParameters{
	SgwAcl:                new("acl"),
	DownAfterMilliseconds: new(int32(10)),
	EnableMonitoring:      new(true),
	FailoverTimeout:       new(int32(10)),
	Graphite:              new("1.1.1.1:91"),
	LazyfreeLazyEviction:  new(valkey.INSTANCEPARAMETERSLAZYFREELAZYEVICTION_NO),
	LazyfreeLazyExpire:    new(valkey.INSTANCEPARAMETERSLAZYFREELAZYEXPIRE_NO),
	LuaTimeLimit:          new(int32(10)),
	MaxDiskThreshold:      new(int32(100)),
	Maxclients:            new(int32(10)),
	MaxmemoryPolicy:       new(valkey.INSTANCEPARAMETERSMAXMEMORYPOLICY_VOLATILE_LRU),
	MaxmemorySamples:      new(int32(10)),
	MetricsFrequency:      new(int32(10)),
	MetricsPrefix:         new("prefix"),
	MinReplicasMaxLag:     new(int32(10)),
	MinReplicasToWrite:    new(int32(1)),
	MonitoringInstanceId:  new("mid"),
	NotifyKeyspaceEvents:  new("events"),
	ReplBacklogSize:       new("1mb"),
	Snapshot:              new("snapshot"),
	Syslog:                []string{"syslog", "syslog2"},
}

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	tests := []struct {
		description string
		input       *valkey.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&valkey.Instance{},
			Model{
				Id:                 types.StringValue(fmt.Sprintf("pid,%s,iid", testRegion)),
				InstanceId:         types.StringValue("iid"),
				ProjectId:          types.StringValue("pid"),
				Region:             types.StringValue(testRegion),
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
			&valkey.Instance{
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
					"min-replicas-to-write":   int32(1),
					"monitoring_instance_id":  "mid",
					"notify-keyspace-events":  "events",
					"repl-backlog-size":       "1mb",
					"snapshot":                "snapshot",
					"syslog":                  []string{"syslog", "syslog2"},
				},
			},
			Model{
				Id:                 types.StringValue(fmt.Sprintf("pid,%s,iid", testRegion)),
				InstanceId:         types.StringValue("iid"),
				ProjectId:          types.StringValue("pid"),
				Region:             types.StringValue(testRegion),
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
			&valkey.Instance{},
			Model{},
			false,
		},
		{
			"wrong_param_types_1",
			&valkey.Instance{
				Parameters: map[string]any{
					"sgw_acl": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&valkey.Instance{
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
			err := mapFields(tt.input, state, testRegion)
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
		expected    *valkey.CreateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&valkey.CreateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:       types.StringValue("name"),
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&valkey.CreateInstancePayload{
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
			&valkey.CreateInstancePayload{
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
			&valkey.CreateInstancePayload{
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
		expected    *valkey.PartialUpdateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&valkey.PartialUpdateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&valkey.PartialUpdateInstancePayload{
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
			&valkey.PartialUpdateInstancePayload{
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
			&valkey.PartialUpdateInstancePayload{
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
