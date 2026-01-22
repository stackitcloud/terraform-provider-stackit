// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: Apache-2.0

package sqlserverflexalpha

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

func TestMapDataSourceFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *sqlserverflexalpha.GetUserResponse
		region      string
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			&sqlserverflexalpha.GetUserResponse{},
			testRegion,
			DataSourceModel{
				Id:              types.StringValue("pid,region,iid,1"),
				UserId:          types.Int64Value(1),
				InstanceId:      types.StringValue("iid"),
				ProjectId:       types.StringValue("pid"),
				Username:        types.StringNull(),
				Roles:           types.SetNull(types.StringType),
				Host:            types.StringNull(),
				Port:            types.Int64Null(),
				Region:          types.StringValue(testRegion),
				Status:          types.StringNull(),
				DefaultDatabase: types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&sqlserverflexalpha.GetUserResponse{

				Roles: &[]sqlserverflexalpha.UserRole{
					"role_1",
					"role_2",
					"",
				},
				Username:        utils.Ptr("username"),
				Host:            utils.Ptr("host"),
				Port:            utils.Ptr(int64(1234)),
				Status:          utils.Ptr("active"),
				DefaultDatabase: utils.Ptr("default_db"),
			},
			testRegion,
			DataSourceModel{
				Id:         types.StringValue("pid,region,iid,1"),
				UserId:     types.Int64Value(1),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringValue("username"),
				Roles: types.SetValueMust(
					types.StringType, []attr.Value{
						types.StringValue("role_1"),
						types.StringValue("role_2"),
						types.StringValue(""),
					},
				),
				Host:            types.StringValue("host"),
				Port:            types.Int64Value(1234),
				Region:          types.StringValue(testRegion),
				Status:          types.StringValue("active"),
				DefaultDatabase: types.StringValue("default_db"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&sqlserverflexalpha.GetUserResponse{
				Id:       utils.Ptr(int64(1)),
				Roles:    &[]sqlserverflexalpha.UserRole{},
				Username: nil,
				Host:     nil,
				Port:     utils.Ptr(int64(2123456789)),
			},
			testRegion,
			DataSourceModel{
				Id:         types.StringValue("pid,region,iid,1"),
				UserId:     types.Int64Value(1),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Username:   types.StringNull(),
				Roles:      types.SetValueMust(types.StringType, []attr.Value{}),
				Host:       types.StringNull(),
				Port:       types.Int64Value(2123456789),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			nil,
			testRegion,
			DataSourceModel{},
			false,
		},
		{
			"nil_response_2",
			&sqlserverflexalpha.GetUserResponse{},
			testRegion,
			DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			&sqlserverflexalpha.GetUserResponse{},
			testRegion,
			DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.description, func(t *testing.T) {
				state := &DataSourceModel{
					ProjectId:  tt.expected.ProjectId,
					InstanceId: tt.expected.InstanceId,
					UserId:     tt.expected.UserId,
				}
				err := mapDataSourceFields(tt.input, state, tt.region)
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
			},
		)
	}
}
