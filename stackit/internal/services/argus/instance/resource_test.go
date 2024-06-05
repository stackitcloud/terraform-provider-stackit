package argus

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description             string
		instanceResp            *argus.GetInstanceResponse
		listACLResp             *argus.ListACLResponse
		getMetricsRetentionResp *argus.GetMetricsStorageRetentionResponse
		expected                Model
		isValid                 bool
	}{
		{
			"default_ok",
			&argus.GetInstanceResponse{
				Id: utils.Ptr("iid"),
			},
			&argus.ListACLResponse{},
			&argus.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			Model{
				Id:                                 types.StringValue("pid,iid"),
				ProjectId:                          types.StringValue("pid"),
				InstanceId:                         types.StringValue("iid"),
				PlanId:                             types.StringNull(),
				PlanName:                           types.StringNull(),
				Name:                               types.StringNull(),
				Parameters:                         types.MapNull(types.StringType),
				ACL:                                types.SetNull(types.StringType),
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
			},
			true,
		},
		{
			"values_ok",
			&argus.GetInstanceResponse{
				Id:         utils.Ptr("iid"),
				Name:       utils.Ptr("name"),
				PlanName:   utils.Ptr("plan1"),
				PlanId:     utils.Ptr("planId"),
				Parameters: &map[string]string{"key": "value"},
				Instance: &argus.InstanceSensitiveData{
					MetricsRetentionTimeRaw: utils.Ptr(int64(30)),
					MetricsRetentionTime1h:  utils.Ptr(int64(15)),
					MetricsRetentionTime5m:  utils.Ptr(int64(10)),
				},
			},
			&argus.ListACLResponse{
				Acl: &[]string{
					"1.1.1.1/32",
				},
				Message: utils.Ptr("message"),
			},
			&argus.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			Model{
				Id:         types.StringValue("pid,iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				InstanceId: types.StringValue("iid"),
				PlanId:     types.StringValue("planId"),
				PlanName:   types.StringValue("plan1"),
				Parameters: toTerraformStringMapMust(context.Background(), map[string]string{"key": "value"}),
				ACL: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("1.1.1.1/32"),
				}),
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
			},
			true,
		},
		{
			"values_ok_multiple_acls",
			&argus.GetInstanceResponse{
				Id:         utils.Ptr("iid"),
				Name:       utils.Ptr("name"),
				PlanName:   utils.Ptr("plan1"),
				PlanId:     utils.Ptr("planId"),
				Parameters: &map[string]string{"key": "value"},
			},
			&argus.ListACLResponse{
				Acl: &[]string{
					"1.1.1.1/32",
					"8.8.8.8/32",
				},
				Message: utils.Ptr("message"),
			},
			&argus.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			Model{
				Id:         types.StringValue("pid,iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				InstanceId: types.StringValue("iid"),
				PlanId:     types.StringValue("planId"),
				PlanName:   types.StringValue("plan1"),
				Parameters: toTerraformStringMapMust(context.Background(), map[string]string{"key": "value"}),
				ACL: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("1.1.1.1/32"),
					types.StringValue("8.8.8.8/32"),
				}),
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
			},
			true,
		},
		{
			"nullable_fields_ok",
			&argus.GetInstanceResponse{
				Id:   utils.Ptr("iid"),
				Name: nil,
			},
			&argus.ListACLResponse{
				Acl:     &[]string{},
				Message: nil,
			},
			&argus.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			Model{
				Id:                                 types.StringValue("pid,iid"),
				ProjectId:                          types.StringValue("pid"),
				InstanceId:                         types.StringValue("iid"),
				PlanId:                             types.StringNull(),
				PlanName:                           types.StringNull(),
				Name:                               types.StringNull(),
				Parameters:                         types.MapNull(types.StringType),
				ACL:                                types.SetNull(types.StringType),
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
			},
			true,
		},
		{
			"response_nil_fail",
			nil,
			nil,
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&argus.GetInstanceResponse{},
			nil,
			nil,
			Model{},
			false,
		},
		{
			"empty metrics retention",
			&argus.GetInstanceResponse{
				Id:   utils.Ptr("iid"),
				Name: nil,
			},
			&argus.ListACLResponse{
				Acl:     &[]string{},
				Message: nil,
			},
			&argus.GetMetricsStorageRetentionResponse{},
			Model{},
			false,
		},
		{
			"nil metrics retention",
			&argus.GetInstanceResponse{
				Id:   utils.Ptr("iid"),
				Name: nil,
			},
			&argus.ListACLResponse{
				Acl:     &[]string{},
				Message: nil,
			},
			nil,
			Model{},
			false,
		},
		{
			"update metrics retention",
			&argus.GetInstanceResponse{
				Id:         utils.Ptr("iid"),
				Name:       utils.Ptr("name"),
				PlanName:   utils.Ptr("plan1"),
				PlanId:     utils.Ptr("planId"),
				Parameters: &map[string]string{"key": "value"},
				Instance: &argus.InstanceSensitiveData{
					MetricsRetentionTimeRaw: utils.Ptr(int64(30)),
					MetricsRetentionTime1h:  utils.Ptr(int64(15)),
					MetricsRetentionTime5m:  utils.Ptr(int64(10)),
				},
			},
			&argus.ListACLResponse{
				Acl: &[]string{
					"1.1.1.1/32",
				},
				Message: utils.Ptr("message"),
			},
			&argus.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			Model{
				Id:         types.StringValue("pid,iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				InstanceId: types.StringValue("iid"),
				PlanId:     types.StringValue("planId"),
				PlanName:   types.StringValue("plan1"),
				Parameters: toTerraformStringMapMust(context.Background(), map[string]string{"key": "value"}),
				ACL: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("1.1.1.1/32"),
				}),
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
				ACL:       types.SetNull(types.StringType),
			}
			err := mapFields(context.Background(), tt.instanceResp, state)
			aclErr := mapACLField(tt.listACLResp, state)
			metricsErr := mapMetricsRetentionField(tt.getMetricsRetentionResp, state)
			if !tt.isValid && err == nil && aclErr == nil && metricsErr == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && (err != nil || aclErr != nil || metricsErr != nil) {
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
		expected    *argus.CreateInstancePayload
		isValid     bool
	}{
		{
			"basic_ok",
			&Model{
				PlanId: types.StringValue("planId"),
			},
			&argus.CreateInstancePayload{
				Name:      nil,
				PlanId:    utils.Ptr("planId"),
				Parameter: &map[string]interface{}{},
			},
			true,
		},
		{
			"ok",
			&Model{
				Name:       types.StringValue("Name"),
				PlanId:     types.StringValue("planId"),
				Parameters: makeTestMap(t),
			},
			&argus.CreateInstancePayload{
				Name:      utils.Ptr("Name"),
				PlanId:    utils.Ptr("planId"),
				Parameter: &map[string]interface{}{"key": `"value"`},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
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
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToPayloadUpdate(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *argus.UpdateInstancePayload
		isValid     bool
	}{
		{
			"basic_ok",
			&Model{
				PlanId: types.StringValue("planId"),
			},
			&argus.UpdateInstancePayload{
				Name:      nil,
				PlanId:    utils.Ptr("planId"),
				Parameter: &map[string]any{},
			},
			true,
		},
		{
			"ok",
			&Model{
				Name:       types.StringValue("Name"),
				PlanId:     types.StringValue("planId"),
				Parameters: makeTestMap(t),
			},
			&argus.UpdateInstancePayload{
				Name:      utils.Ptr("Name"),
				PlanId:    utils.Ptr("planId"),
				Parameter: &map[string]any{"key": `"value"`},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input)
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

func makeTestMap(t *testing.T) basetypes.MapValue {
	p := make(map[string]attr.Value, 1)
	p["key"] = types.StringValue("value")
	params, diag := types.MapValueFrom(context.Background(), types.StringType, p)
	if diag.HasError() {
		t.Fail()
	}
	return params
}

// ToTerraformStringMapMust Silently ignores the error
func toTerraformStringMapMust(ctx context.Context, m map[string]string) basetypes.MapValue {
	labels := make(map[string]attr.Value, len(m))
	for l, v := range m {
		stringValue := types.StringValue(v)
		labels[l] = stringValue
	}
	res, diags := types.MapValueFrom(ctx, types.StringType, m)
	if diags.HasError() {
		return types.MapNull(types.StringType)
	}
	return res
}
