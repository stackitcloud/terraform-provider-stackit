package mariadb

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	mariadb "github.com/stackitcloud/stackit-sdk-go/services/mariadb/v1api"
)

var fixtureModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                types.StringValue("acl"),
	"enable_monitoring":      types.BoolValue(true),
	"graphite":               types.StringValue("graphite"),
	"max_disk_threshold":     types.Int32Value(10),
	"metrics_frequency":      types.Int32Value(10),
	"metrics_prefix":         types.StringValue("prefix"),
	"monitoring_instance_id": types.StringValue("mid"),
	"syslog": types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("syslog"),
		types.StringValue("syslog2"),
	}),
})

var fixtureNullModelParameters = types.ObjectValueMust(parametersTypes, map[string]attr.Value{
	"sgw_acl":                types.StringNull(),
	"enable_monitoring":      types.BoolNull(),
	"graphite":               types.StringNull(),
	"max_disk_threshold":     types.Int32Null(),
	"metrics_frequency":      types.Int32Null(),
	"metrics_prefix":         types.StringNull(),
	"monitoring_instance_id": types.StringNull(),
	"syslog":                 types.ListNull(types.StringType),
})

var fixtureInstanceParameters = mariadb.InstanceParameters{
	SgwAcl:               new("acl"),
	EnableMonitoring:     new(true),
	Graphite:             new("graphite"),
	MaxDiskThreshold:     new(int32(10)),
	MetricsFrequency:     new(int32(10)),
	MetricsPrefix:        new("prefix"),
	MonitoringInstanceId: new("mid"),
	Syslog:               []string{"syslog", "syslog2"},
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *mariadb.Instance
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&mariadb.Instance{},
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
			&mariadb.Instance{
				PlanId:             "plan",
				CfGuid:             "cf",
				CfSpaceGuid:        "space",
				DashboardUrl:       "dashboard",
				ImageUrl:           "image",
				InstanceId:         new("iid"),
				Name:               "name",
				CfOrganizationGuid: "org",
				Parameters: map[string]any{
					"sgw_acl":                "acl",
					"enable_monitoring":      true,
					"graphite":               "graphite",
					"max_disk_threshold":     int32(10),
					"metrics_frequency":      int32(10),
					"metrics_prefix":         "prefix",
					"monitoring_instance_id": "mid",
					"syslog":                 []string{"syslog", "syslog2"},
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
			&mariadb.Instance{},
			Model{},
			false,
		},
		{
			"wrong_param_types_1",
			&mariadb.Instance{
				Parameters: map[string]any{
					"sgw_acl": true,
				},
			},
			Model{},
			false,
		},
		{
			"wrong_param_types_2",
			&mariadb.Instance{
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
		expected    *mariadb.CreateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&mariadb.CreateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Name:       types.StringValue("name"),
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&mariadb.CreateInstancePayload{
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
			&mariadb.CreateInstancePayload{
				InstanceName: "",
				Parameters:   &mariadb.InstanceParameters{},
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
			&mariadb.CreateInstancePayload{
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
		expected    *mariadb.PartialUpdateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&mariadb.PartialUpdateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				PlanId:     types.StringValue("plan"),
				Parameters: fixtureModelParameters,
			},
			&mariadb.PartialUpdateInstancePayload{
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
			&mariadb.PartialUpdateInstancePayload{
				Parameters: &mariadb.InstanceParameters{},
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
			&mariadb.PartialUpdateInstancePayload{
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
